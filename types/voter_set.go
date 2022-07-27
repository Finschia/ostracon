package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/line/ostracon/crypto/bls"
	"github.com/line/ostracon/crypto/merkle"
	"github.com/line/ostracon/crypto/tmhash"
	tmmath "github.com/line/ostracon/libs/math"
	tmrand "github.com/line/ostracon/libs/rand"
	tmproto "github.com/line/ostracon/proto/ostracon/types"
)

// VoterSet represent a set of *Validator at a given height.
type VoterSet struct {
	// NOTE: persisted via reflect, must be exported.
	Voters []*Validator `json:"voters"`

	// cached (unexported)
	totalStakingPower int64
}

func WrapValidatorsToVoterSet(vals []*Validator) *VoterSet {
	sort.Sort(ValidatorsByVotingPower(vals))
	voterSet := &VoterSet{Voters: vals, totalStakingPower: 0}
	voterSet.updateTotalStakingPower()
	return voterSet
}

func (voters *VoterSet) ValidateBasic() error {
	if voters.IsNilOrEmpty() {
		return errors.New("voter set is nil or empty")
	}

	for idx, val := range voters.Voters {
		if err := val.ValidateBasic(); err != nil {
			return fmt.Errorf("invalid validator #%d: %w", idx, err)
		}
	}

	return nil
}

// IsNilOrEmpty returns true if validator set is nil or empty.
func (voters *VoterSet) IsNilOrEmpty() bool {
	return voters == nil || len(voters.Voters) == 0
}

// HasAddress returns true if address given is in the validator set, false -
// otherwise.
func (voters *VoterSet) HasAddress(address []byte) bool {
	for _, voter := range voters.Voters {
		if bytes.Equal(voter.Address, address) {
			return true
		}
	}
	return false
}

// GetByAddress returns an index of the validator with address and validator
// itself if found. Otherwise, -1 and nil are returned.
func (voters *VoterSet) GetByAddress(address []byte) (index int32, val *Validator) {
	for idx, voter := range voters.Voters {
		if bytes.Equal(voter.Address, address) {
			return int32(idx), voter.Copy()
		}
	}
	return -1, nil
}

// GetByIndex returns the validator's address and validator itself by index.
// It returns nil values if index is less than 0 or greater or equal to
// len(VoterSet.Validators).
func (voters *VoterSet) GetByIndex(index int32) (address []byte, val *Validator) {
	if index < 0 || int(index) >= len(voters.Voters) {
		return nil, nil
	}
	val = voters.Voters[index]
	return val.Address, val.Copy()
}

// Size returns the length of the validator set.
func (voters *VoterSet) Size() int {
	return len(voters.Voters)
}

func copyValidatorListShallow(vals []*Validator) []*Validator {
	result := make([]*Validator, len(vals))
	copy(result, vals)
	return result
}

// VoterSet.Copy() copies validator list shallow
func (voters *VoterSet) Copy() *VoterSet {
	if voters == nil {
		return nil
	}
	return &VoterSet{
		Voters:            copyValidatorListShallow(voters.Voters),
		totalStakingPower: voters.totalStakingPower,
	}
}

// Forces recalculation of the set's total staking power.
// Panics if total staking power is bigger than MaxTotalStakingPower.
func (voters *VoterSet) updateTotalStakingPower() {
	sum := int64(0)
	for _, val := range voters.Voters {
		// mind overflow
		sum = safeAddClip(sum, val.StakingPower)
		if sum > MaxTotalStakingPower {
			panic(fmt.Sprintf(
				"Total staking power should be guarded to not exceed %v; got: %v",
				MaxTotalStakingPower,
				sum))
		}
	}
	voters.totalStakingPower = sum
}

func (voters *VoterSet) TotalStakingPower() int64 {
	if voters.totalStakingPower == 0 {
		voters.updateTotalStakingPower()
	}
	return voters.totalStakingPower
}

// Hash returns the Merkle root hash build using voters (as leaves) in the
// set.
func (voters *VoterSet) Hash() []byte {
	if len(voters.Voters) == 0 {
		return nil
	}
	bzs := make([][]byte, len(voters.Voters))
	for i, voter := range voters.Voters {
		bzs[i] = voter.Bytes()
	}
	return merkle.HashFromByteSlices(bzs)
}

// VerifyCommit verifies +2/3 of the set had signed the given commit.
//
// It checks all the signatures! While it's safe to exit as soon as we have
// 2/3+ signatures, doing so would impact incentivization logic in the ABCI
// application that depends on the LastCommitInfo sent in BeginBlock, which
// includes which voters signed. For instance, Gaia incentivizes proposers
// with a bonus for including more than +2/3 of the signatures.
func (voters *VoterSet) VerifyCommit(chainID string, blockID BlockID, height int64, commit *Commit) error {

	if voters.Size() != len(commit.Signatures) {
		return NewErrInvalidCommitSignatures(voters.Size(), len(commit.Signatures))
	}

	// Validate Height and BlockID.
	if height != commit.Height {
		return NewErrInvalidCommitHeight(height, commit.Height)
	}
	if !blockID.Equals(commit.BlockID) {
		return fmt.Errorf("invalid commit -- wrong block ID: want %v, got %v",
			blockID, commit.BlockID)
	}

	talliedStakingPower := int64(0)
	stakingPowerNeeded := voters.TotalStakingPower() * 2 / 3 // FIXME: 🏺 arithmetic overflow
	blsPubKeys := make([]bls.PubKey, 0, len(commit.Signatures))
	messages := make([][]byte, 0, len(commit.Signatures))
	for idx, commitSig := range commit.Signatures {
		if commitSig.Absent() {
			continue // OK, some signatures can be absent.
		}

		// The voters and commit have a 1-to-1 correspondance.
		// This means we don't need the voter address or to do any lookup.
		voter := voters.Voters[idx]

		// Validate signature.
		voteSignBytes := commit.VoteSignBytes(chainID, int32(idx))
		verifiedStakingPower, unverifiedStakingPower, err := verifySignatureOrCollectBlsPubKeysAndGetStakingPower(
			idx, commitSig, voter, voteSignBytes, &blsPubKeys, &messages)
		if err != nil {
			return err
		}

		// Good!
		if commitSig.ForBlock() {
			talliedStakingPower += verifiedStakingPower + unverifiedStakingPower
		}

		// else {
		// It's OK. We include stray signatures (~votes for nil) to measure
		// voter availability.
		// }
	}

	// Validate aggregate signature
	if err := bls.VerifyAggregatedSignature(commit.AggregatedSignature, blsPubKeys, messages); err != nil {
		return fmt.Errorf("wrong aggregated signature: %X; %s", commit.AggregatedSignature, err)
	}

	// add staking power for BLS batch verification and return without error if trust-level of the signatures are verified
	if got, needed := talliedStakingPower, stakingPowerNeeded; got <= needed {
		return ErrNotEnoughStakingPowerSigned{Got: got, Needed: needed}
	}

	return nil
}

// LIGHT CLIENT VERIFICATION METHODS

// VerifyCommitLight verifies +2/3 of the set had signed the given commit.
//
// This method is primarily used by the light client and does not check all the
// signatures.
func (voters *VoterSet) VerifyCommitLight(chainID string, blockID BlockID,
	height int64, commit *Commit) error {

	if voters.Size() != len(commit.Signatures) {
		return NewErrInvalidCommitSignatures(voters.Size(), len(commit.Signatures))
	}

	// Validate Height and BlockID.
	if height != commit.Height {
		return NewErrInvalidCommitHeight(height, commit.Height)
	}
	if !blockID.Equals(commit.BlockID) {
		return fmt.Errorf("invalid commit -- wrong block ID: want %v, got %v",
			blockID, commit.BlockID)
	}

	talliedStakingPower := int64(0)
	talliedUnverifiedStakingPower := int64(0)
	stakingPowerNeeded := voters.TotalStakingPower() * 2 / 3 // FIXME: 🏺 arithmetic overflow
	blsPubKeys := make([]bls.PubKey, 0, len(commit.Signatures))
	messages := make([][]byte, 0, len(commit.Signatures))
	for idx, commitSig := range commit.Signatures {
		// No need to verify absent or nil votes.
		if !commitSig.ForBlock() {
			continue
		}

		// The voters and commit have a 1-to-1 correspondence.
		// This means we don't need the voter address or to do any lookup.
		// voter := voters.Voters[idx]
		index, voter := voters.GetByAddress(commitSig.ValidatorAddress)
		if index == -1 && voter == nil {
			continue
		}

		// Validate signature.
		voteSignBytes := commit.VoteSignBytes(chainID, int32(idx))
		verifiedStakingPower, unverifiedStakingPower, err := verifySignatureOrCollectBlsPubKeysAndGetStakingPower(
			idx, commitSig, voter, voteSignBytes, &blsPubKeys, &messages)
		if err != nil {
			return err
		}

		talliedStakingPower += verifiedStakingPower
		talliedUnverifiedStakingPower += unverifiedStakingPower

		// return as soon as +2/3 of the signatures are verified by individual verification
		if talliedStakingPower > stakingPowerNeeded {
			return nil
		}
	}

	// Validate aggregate signature
	if err := bls.VerifyAggregatedSignature(commit.AggregatedSignature, blsPubKeys, messages); err != nil {
		return fmt.Errorf("wrong aggregated signature: %X; %s", commit.AggregatedSignature, err)
	}
	// add staking power for BLS batch verification and return without error if +2/3 of the signatures are verified
	talliedStakingPower += talliedUnverifiedStakingPower
	if talliedStakingPower > stakingPowerNeeded {
		return nil
	}

	return ErrNotEnoughStakingPowerSigned{Got: talliedStakingPower, Needed: stakingPowerNeeded}
}

// VerifyCommitLightTrusting verifies that trustLevel of the voter set signed
// this commit.
//
// NOTE the given voters do not necessarily correspond to the voter set
// for this commit, but there may be some intersection.
//
// This method is primarily used by the light client and does not check all the
// signatures.
func (voters *VoterSet) VerifyCommitLightTrusting(chainID string, commit *Commit, trustLevel tmmath.Fraction) error {
	// sanity check
	if trustLevel.Denominator == 0 {
		return errors.New("trustLevel has zero Denominator")
	}

	var (
		talliedStakingPower           int64
		talliedUnverifiedStakingPower int64
		seenVoters                    = make(map[int32]int, len(commit.Signatures)) // voter index -> commit index
	)

	// Safely calculate staking power needed.
	totalStakingPowerMulByNumerator, overflow := safeMul(voters.TotalStakingPower(), int64(trustLevel.Numerator))
	if overflow {
		return errors.New("int64 overflow while calculating staking power needed. " +
			"please provide smaller trustLevel numerator")
	}
	stakingPowerNeeded := totalStakingPowerMulByNumerator / int64(trustLevel.Denominator)

	blsPubKeys := make([]bls.PubKey, 0, len(commit.Signatures))
	messages := make([][]byte, 0, len(commit.Signatures))
	for idx, commitSig := range commit.Signatures {
		// No need to verify absent or nil votes.
		if !commitSig.ForBlock() {
			continue
		}

		// We don't know the voters that committed this block, so we have to
		// check for each vote if its voter is already known.
		voterIdx, voter := voters.GetByAddress(commitSig.ValidatorAddress)

		if voter != nil {
			// check for double vote of voter on the same commit
			if firstIndex, ok := seenVoters[voterIdx]; ok {
				secondIndex := idx
				return fmt.Errorf("double vote from %v (%d and %d)", voter, firstIndex, secondIndex)
			}
			seenVoters[voterIdx] = idx

			// Verify Signature
			voteSignBytes := commit.VoteSignBytes(chainID, int32(idx))
			verifiedStakingPower, unverifiedStakingPower, err := verifySignatureOrCollectBlsPubKeysAndGetStakingPower(
				idx, commitSig, voter, voteSignBytes, &blsPubKeys, &messages)
			if err != nil {
				return err
			}

			talliedStakingPower += verifiedStakingPower
			talliedUnverifiedStakingPower += unverifiedStakingPower

			if talliedStakingPower > stakingPowerNeeded {
				return nil
			}
		}
	}

	// Validate aggregate signature
	if err := bls.VerifyAggregatedSignature(commit.AggregatedSignature, blsPubKeys, messages); err != nil {
		return fmt.Errorf("wrong aggregated signature: %X; %s", commit.AggregatedSignature, err)
	}
	// add staking power for BLS batch verification and return without error if trust-level of the signatures are verified
	talliedStakingPower += talliedUnverifiedStakingPower
	if talliedStakingPower > stakingPowerNeeded {
		return nil
	}

	return ErrNotEnoughStakingPowerSigned{Got: talliedStakingPower, Needed: stakingPowerNeeded}
}

func verifySignatureOrCollectBlsPubKeysAndGetStakingPower(
	idx int, commitSig CommitSig, val *Validator, voteSignBytes []byte,
	blsPubKeys *[]bls.PubKey, messages *[][]byte) (int64, int64, error) {
	verifiedStakingPower := int64(0)
	unverifiedStakingPower := int64(0)
	if commitSig.Signature != nil {
		if !val.PubKey.VerifySignature(voteSignBytes, commitSig.Signature) {
			return verifiedStakingPower, unverifiedStakingPower, fmt.Errorf(
				"wrong signature (#%d): %X",
				idx,
				commitSig.Signature,
			)
		}
		verifiedStakingPower = val.StakingPower
	} else {
		blsPubKey := GetSignatureKey(val.PubKey)
		if blsPubKey == nil {
			return verifiedStakingPower, unverifiedStakingPower, fmt.Errorf(
				"signature %d has been omitted, even though it is not a BLS key",
				idx,
			)
		}
		*blsPubKeys = append(*blsPubKeys, *blsPubKey)
		*messages = append(*messages, voteSignBytes)
		unverifiedStakingPower = val.StakingPower
	}
	return verifiedStakingPower, unverifiedStakingPower, nil
}

// ToProto converts VoterSet to protobuf
func (voters *VoterSet) ToProto() (*tmproto.VoterSet, error) {
	if voters.IsNilOrEmpty() {
		return &tmproto.VoterSet{}, nil // validator set should never be nil
	}

	vsp := new(tmproto.VoterSet)
	votersProto := make([]*tmproto.Validator, len(voters.Voters))
	for i := 0; i < len(voters.Voters); i++ {
		voterp, err := voters.Voters[i].ToProto()
		if err != nil {
			return nil, err
		}
		votersProto[i] = voterp
	}
	vsp.Voters = votersProto
	vsp.TotalStakingPower = voters.totalStakingPower

	return vsp, nil
}

// VoterSetFromProto sets a protobuf VoterSet to the given pointer.
// It returns an error if any of the validators from the set or the proposer
// is invalid
func VoterSetFromProto(vp *tmproto.VoterSet) (*VoterSet, error) {
	if vp == nil {
		return nil, errors.New("nil voter set") // voter set should never be nil, bigger issues are at play if empty
	}
	voters := new(VoterSet)

	valsProto := make([]*Validator, len(vp.Voters))
	for i := 0; i < len(vp.Voters); i++ {
		v, err := ValidatorFromProto(vp.Voters[i])
		if err != nil {
			return nil, err
		}
		valsProto[i] = v
	}
	voters.Voters = valsProto
	voters.totalStakingPower = vp.GetTotalStakingPower()

	return voters, voters.ValidateBasic()
}

//-----------------

// IsErrNotEnoughStakingPowerSigned returns true if err is
// ErrNotEnoughStakingPowerSigned.
func IsErrNotEnoughStakingPowerSigned(err error) bool {
	_, ok := errors.Cause(err).(ErrNotEnoughStakingPowerSigned)
	return ok
}

// ErrNotEnoughStakingPowerSigned is returned when not enough voters signed
// a commit.
type ErrNotEnoughStakingPowerSigned struct {
	Got    int64
	Needed int64
}

func (e ErrNotEnoughStakingPowerSigned) Error() string {
	return fmt.Sprintf("invalid commit -- insufficient staking power: got %d, needed more than %d", e.Got, e.Needed)
}

//----------------

// Iterate will run the given function over the set.
func (voters *VoterSet) Iterate(fn func(index int, val *Validator) bool) {
	for i, val := range voters.Voters {
		stop := fn(i, val)
		if stop {
			break
		}
	}
}

func (voters *VoterSet) String() string {
	return voters.StringIndented("")
}

// StringIndented returns an intended string representation of VoterSet.
func (voters *VoterSet) StringIndented(indent string) string {
	if voters == nil {
		return "nil-VoterSet"
	}
	var voterStrings []string
	voters.Iterate(func(index int, voter *Validator) bool {
		voterStrings = append(voterStrings, voter.String())
		return false
	})
	return fmt.Sprintf(`VoterSet{
%s  Voters:
%s    %v
%s}`,
		indent, indent, strings.Join(voterStrings, "\n"+indent+"    "),
		indent)

}

func SelectVoter(validators *ValidatorSet, proofHash []byte, voterParams *VoterParams) *VoterSet {
	if len(proofHash) == 0 || validators.Size() <= int(voterParams.VoterElectionThreshold) ||
		voterParams.MaxTolerableByzantinePercentage > BftMaxTolerableByzantinePercentage {
		return ToVoterAll(validators.Validators)
	}
	seed := hashToSeed(proofHash)
	voters := electVotersNonDup(validators.Validators, seed, int(voterParams.MaxTolerableByzantinePercentage),
		int(voterParams.VoterElectionThreshold))
	return WrapValidatorsToVoterSet(voters)
}

func ToVoterAll(validators []*Validator) *VoterSet {
	newVoters := make([]*Validator, 0, len(validators))
	for _, val := range validators {
		if val.VotingPower == 0 {
			// remove the validator with the voting power of 0 from the voter set
			continue
		}
		newVoters = append(newVoters, &Validator{
			Address:          val.Address,
			PubKey:           val.PubKey,
			VotingPower:      val.VotingPower,
			StakingPower:     val.VotingPower,
			ProposerPriority: val.ProposerPriority,
		})
	}
	return WrapValidatorsToVoterSet(newVoters) // They will be sorted in this function.
}

func hashToSeed(hash []byte) uint64 {
	for len(hash) < 8 {
		hash = append(hash, byte(0))
	}
	return binary.LittleEndian.Uint64(hash[:8])
}

// MakeRoundHash combines the VRF hash, block height, and round to create a hash value for each round. This value is
// used for random sampling of the Proposer.
func MakeRoundHash(proofHash []byte, height int64, round int32) []byte {
	b := make([]byte, 16)
	binary.LittleEndian.PutUint64(b, uint64(height))
	binary.LittleEndian.PutUint64(b[8:], uint64(round))
	hash := tmhash.New()
	if _, err := hash.Write(proofHash); err != nil {
		panic(err)
	}
	if _, err := hash.Write(b[:8]); err != nil {
		panic(err)
	}
	if _, err := hash.Write(b[8:16]); err != nil {
		panic(err)
	}
	return hash.Sum(nil)
}

// RandVoterSet returns a randomized validator set, useful for testing.
// NOTE: PrivValidator are in order.
// UNSTABLE
func RandVoterSet(numVoters int, votingPower int64) (*ValidatorSet, *VoterSet, []PrivValidator) {
	valz := make([]*Validator, numVoters)
	privValidators := make([]PrivValidator, numVoters)
	for i := 0; i < numVoters; i++ {
		val, privValidator := RandValidator(false, votingPower)
		valz[i] = val
		privValidators[i] = privValidator
	}
	vals := NewValidatorSet(valz)
	sort.Sort(PrivValidatorsByAddress(privValidators))
	return vals, SelectVoter(vals, []byte{}, DefaultVoterParams()), privValidators
}

func electVoter(
	seed *uint64, candidates []*Validator, voterNum int, totalPriority int64) (
	winnerIdx int, winner *Validator) {
	threshold := tmrand.RandomThreshold(seed, uint64(totalPriority))
	found := false
	cumulativePriority := int64(0)
	for i, candidate := range candidates[:len(candidates)-voterNum] {
		if threshold < uint64(cumulativePriority+candidate.VotingPower) {
			winner = candidates[i]
			winnerIdx = i
			found = true
			break
		}
		cumulativePriority += candidate.VotingPower
	}

	if !found {
		panic(fmt.Sprintf("Cannot find random sample. voterNum=%d, "+
			"totalPriority=%d, threshold=%d",
			voterNum, totalPriority, threshold))
	}

	return winnerIdx, winner
}

const precisionForSelection = int64(1000)
const precisionCorrectionForSelection = int64(1000)
const BftMaxTolerableByzantinePercentage = 33

type voter struct {
	val      *Validator
	winPoint *big.Int
}

func electVotersNonDup(validators []*Validator, seed uint64, tolerableByzantinePercent, minVoters int) []*Validator {
	// validators is read-only
	if tolerableByzantinePercent > BftMaxTolerableByzantinePercentage {
		panic(fmt.Sprintf("tolerableByzantinePercent cannot exceed 33: %d", tolerableByzantinePercent))
	}

	candidates := validatorListCopy(validators)
	totalVotingPower := getTotalVotingPower(candidates)
	tolerableByzantinePower := getTolerableByzantinePower(totalVotingPower, tolerableByzantinePercent)
	voters := make([]*voter, 0)
	sortValidators(candidates)

	zeroValidators := 0
	for i := len(candidates); candidates[i-1].VotingPower == 0; i-- {
		zeroValidators++
	}

	losersVotingPower := totalVotingPower
	for {
		// accumulateWinPoints(voters)
		for _, voter := range voters {
			// i = v1 ... vt
			// votingPower(i) * 1000 / (votingPower(vt+1 ... vn) + votingPower(i))
			additionalWinPoint := new(big.Int).Mul(big.NewInt(voter.val.VotingPower),
				big.NewInt(precisionForSelection))
			additionalWinPoint.Div(additionalWinPoint, new(big.Int).Add(big.NewInt(losersVotingPower),
				big.NewInt(voter.val.VotingPower)))
			voter.winPoint.Add(voter.winPoint, additionalWinPoint)
		}
		// electVoter
		winnerIdx, winner := electVoter(&seed, candidates, len(voters)+zeroValidators, losersVotingPower)

		moveWinnerToLast(candidates, winnerIdx)
		voters = append(voters, &voter{
			val:      winner,
			winPoint: big.NewInt(precisionForSelection),
		})
		losersVotingPower -= winner.VotingPower

		// calculateStakingPowers(voters)
		totalWinPoint := new(big.Int)
		for _, voter := range voters {
			totalWinPoint.Add(totalWinPoint, voter.winPoint)
		}
		totalStakingPower := int64(0)
		for _, voter := range voters {
			winPoint := new(big.Int).Mul(voter.winPoint, big.NewInt(precisionForSelection))
			bigStakingPower := new(big.Int).Div(new(big.Int).Mul(winPoint, big.NewInt(totalVotingPower)), totalWinPoint)
			stakingPower := new(big.Int).Div(bigStakingPower, big.NewInt(precisionCorrectionForSelection)).Int64()
			voter.val.StakingPower = stakingPower
			totalStakingPower += stakingPower
		}

		if len(voters) >= minVoters {
			// sort voters in ascending stakingPower/votingPower
			sortVoters(voters)

			topFVotersStakingPower := getTopByzantineStakingPower(voters, tolerableByzantinePower)
			if topFVotersStakingPower < totalStakingPower/3 {
				break
			}
		}

		if len(voters)+zeroValidators == len(candidates) {
			// there is no voter group satisfying the finality
			// cannot do sampling voters
			for _, c := range candidates {
				c.StakingPower = c.VotingPower
			}
			return candidates
		}
	}
	result := make([]*Validator, len(voters))
	for i, v := range voters {
		result[i] = v.val
	}
	return result
}

func getTotalVotingPower(validators []*Validator) int64 {
	totalVoting := int64(0)
	for _, v := range validators {
		totalVoting += v.VotingPower
	}
	return totalVoting
}

func getTopByzantineStakingPower(voters []*voter, tolerableByzantinePower int64) int64 {
	topFVotersVotingPower := int64(0)
	topFVotersStakingPower := int64(0)
	for _, voter := range voters {
		prev := topFVotersVotingPower
		topFVotersVotingPower += voter.val.VotingPower
		topFVotersStakingPower += voter.val.StakingPower
		if prev < tolerableByzantinePower && topFVotersVotingPower >= tolerableByzantinePower {
			break
		}
	}
	return topFVotersStakingPower
}

// sort validators in-place
func sortValidators(validators []*Validator) {
	sort.Slice(validators, func(i, j int) bool {
		if validators[i].VotingPower == validators[j].VotingPower {
			return bytes.Compare(validators[i].Address, validators[j].Address) == -1
		}
		return validators[i].VotingPower > validators[j].VotingPower
	})
}

// sortVoters is function to sort voters in descending stakingPower/votingPower in-place
func sortVoters(candidates []*voter) {
	sort.Slice(candidates, func(i, j int) bool {
		bigA := new(big.Int).Mul(big.NewInt(candidates[i].val.StakingPower), big.NewInt(candidates[j].val.VotingPower))
		bigB := new(big.Int).Mul(big.NewInt(candidates[j].val.StakingPower), big.NewInt(candidates[i].val.VotingPower))
		compareResult := bigA.Cmp(bigB)
		if compareResult == 0 {
			return bytes.Compare(candidates[i].val.Address, candidates[j].val.Address) == -1
		}
		return compareResult == 1
	})
}

func moveWinnerToLast(candidates []*Validator, winner int) {
	winnerCandidate := candidates[winner]
	copy(candidates[winner:], candidates[winner+1:])
	candidates[len(candidates)-1] = winnerCandidate
}

func getTolerableByzantinePower(totalVotingPower int64, tolerableByzantinePercent int) int64 {
	// `totalVotingPower * tolerableByzantinePercent` may be overflow for int64 type
	bigMultiplied := new(big.Int).Mul(big.NewInt(totalVotingPower), big.NewInt(int64(tolerableByzantinePercent)))
	tolerableByzantinePower := new(big.Int).Div(bigMultiplied, big.NewInt(100))

	// ceiling
	if new(big.Int).Mul(tolerableByzantinePower, big.NewInt(100)).Cmp(bigMultiplied) < 0 {
		tolerableByzantinePower = new(big.Int).Add(tolerableByzantinePower, big.NewInt(1))
	}
	return tolerableByzantinePower.Int64()
}
