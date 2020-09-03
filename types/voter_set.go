package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"
	"sort"
	"strings"

	tmproto "github.com/tendermint/tendermint/proto/types"

	"github.com/pkg/errors"

	"github.com/tendermint/tendermint/crypto/merkle"
	"github.com/tendermint/tendermint/crypto/tmhash"
	tmmath "github.com/tendermint/tendermint/libs/math"
	tmrand "github.com/tendermint/tendermint/libs/rand"
)

// VoterSet represent a set of *Validator at a given height.
type VoterSet struct {
	// NOTE: persisted via reflect, must be exported.
	Voters []*Validator `json:"voters"`

	// cached (unexported)
	totalVotingPower int64
}

func WrapValidatorsToVoterSet(vals []*Validator) *VoterSet {
	sort.Sort(ValidatorsByAddress(vals))
	voterSet := &VoterSet{Voters: vals, totalVotingPower: 0}
	voterSet.updateTotalVotingPower()
	return voterSet
}

// IsNilOrEmpty returns true if validator set is nil or empty.
func (voters *VoterSet) IsNilOrEmpty() bool {
	return voters == nil || len(voters.Voters) == 0
}

// HasAddress returns true if address given is in the validator set, false -
// otherwise.
func (voters *VoterSet) HasAddress(address []byte) bool {
	idx := sort.Search(len(voters.Voters), func(i int) bool {
		return bytes.Compare(address, voters.Voters[i].Address) <= 0
	})
	return idx < len(voters.Voters) && bytes.Equal(voters.Voters[idx].Address, address)
}

// GetByAddress returns an index of the validator with address and validator
// itself if found. Otherwise, -1 and nil are returned.
func (voters *VoterSet) GetByAddress(address []byte) (index int, val *Validator) {
	idx := sort.Search(len(voters.Voters), func(i int) bool {
		return bytes.Compare(address, voters.Voters[i].Address) <= 0
	})
	if idx < len(voters.Voters) && bytes.Equal(voters.Voters[idx].Address, address) {
		return idx, voters.Voters[idx].Copy()
	}
	return -1, nil
}

// GetByIndex returns the validator's address and validator itself by index.
// It returns nil values if index is less than 0 or greater or equal to
// len(VoterSet.Validators).
func (voters *VoterSet) GetByIndex(index int) (address []byte, val *Validator) {
	if index < 0 || index >= len(voters.Voters) {
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
	return &VoterSet{
		Voters:           copyValidatorListShallow(voters.Voters),
		totalVotingPower: voters.totalVotingPower,
	}
}

// Forces recalculation of the set's total voting power.
// Panics if total voting power is bigger than MaxTotalVotingPower.
func (voters *VoterSet) updateTotalVotingPower() {
	sum := int64(0)
	for _, val := range voters.Voters {
		// mind overflow
		sum = safeAddClip(sum, val.VotingPower)
		if sum > MaxTotalVotingPower {
			panic(fmt.Sprintf(
				"Total voting power should be guarded to not exceed %v; got: %v",
				MaxTotalVotingPower,
				sum))
		}
	}
	voters.totalVotingPower = sum
}

func (voters *VoterSet) TotalVotingPower() int64 {
	if voters.totalVotingPower == 0 {
		voters.updateTotalVotingPower()
	}
	return voters.totalVotingPower
}

// Hash returns the Merkle root hash build using validators (as leaves) in the
// set.
func (voters *VoterSet) Hash() []byte {
	if len(voters.Voters) == 0 {
		return nil
	}
	bzs := make([][]byte, len(voters.Voters))
	for i, val := range voters.Voters {
		bzs[i] = val.Bytes()
	}
	return merkle.SimpleHashFromByteSlices(bzs)
}

// VerifyCommit verifies +2/3 of the set had signed the given commit.
//
// It checks all the signatures! While it's safe to exit as soon as we have
// 2/3+ signatures, doing so would impact incentivization logic in the ABCI
// application that depends on the LastCommitInfo sent in BeginBlock, which
// includes which validators signed. For instance, Gaia incentivizes proposers
// with a bonus for including more than +2/3 of the signatures.
func (voters *VoterSet) VerifyCommit(chainID string, blockID BlockID,
	height int64, commit *Commit) error {

	if voters.Size() != len(commit.Signatures) {
		return NewErrInvalidCommitSignatures(voters.Size(), len(commit.Signatures))
	}
	if err := verifyCommitBasic(commit, height, blockID); err != nil {
		return err
	}

	talliedVotingPower := int64(0)
	votingPowerNeeded := voters.TotalVotingPower() * 2 / 3
	for idx, commitSig := range commit.Signatures {
		if commitSig.Absent() {
			continue // OK, some signatures can be absent.
		}

		// The vals and commit have a 1-to-1 correspondance.
		// This means we don't need the validator address or to do any lookup.
		val := voters.Voters[idx]

		// Good!
		if blockID.Equals(commitSig.BlockID(commit.BlockID)) {
			talliedVotingPower += val.VotingPower
		}
		// else {
		// It's OK that the BlockID doesn't match.  We include stray
		// signatures (~votes for nil) to measure validator availability.
		// }
	}

	// Validate signature.
	if err := commit.VerifySignatures(chainID, voters.Voters); err != nil {
		return err
	}

	if got, needed := talliedVotingPower, votingPowerNeeded; got <= needed {
		return ErrNotEnoughVotingPowerSigned{Got: got, Needed: needed}
	}

	return nil
}

///////////////////////////////////////////////////////////////////////////////
// LIGHT CLIENT VERIFICATION METHODS
///////////////////////////////////////////////////////////////////////////////

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

	talliedVotingPower := int64(0)
	votingPowerNeeded := voters.TotalVotingPower() * 2 / 3
	for idx, commitSig := range commit.Signatures {
		// No need to verify absent or nil votes.
		if !commitSig.ForBlock() {
			continue
		}

		// The vals and commit have a 1-to-1 correspondance.
		// This means we don't need the validator address or to do any lookup.
		val := voters.Voters[idx]

		talliedVotingPower += val.VotingPower

		// return as soon as +2/3 of the signatures are verified
		if talliedVotingPower > votingPowerNeeded {
			// Validate signature.
			if err := commit.VerifySignatures(chainID, voters.Voters); err != nil {
				return err
			}
			return nil
		}
	}

	return ErrNotEnoughVotingPowerSigned{Got: talliedVotingPower, Needed: votingPowerNeeded}
}

// VerifyFutureCommit will check to see if the set would be valid with a different
// validator set.
//
// vals is the old validator set that we know.  Over 2/3 of the power in old
// signed this block.
//
// In Tendermint, 1/3 of the voting power can halt or fork the chain, but 1/3
// can't make arbitrary state transitions.  You still need > 2/3 Byzantine to
// make arbitrary state transitions.
//
// To preserve this property in the light client, we also require > 2/3 of the
// old vals to sign the future commit at H, that way we preserve the property
// that if they weren't being truthful about the validator set at H (block hash
// -> vals hash) or about the app state (block hash -> app hash) we can slash
// > 2/3.  Otherwise, the lite client isn't providing the same security
// guarantees.
//
// Even if we added a slashing condition that if you sign a block header with
// the wrong validator set, then we would only need > 1/3 of signatures from
// the old vals on the new commit, it wouldn't be sufficient because the new
// vals can be arbitrary and commit some arbitrary app hash.
//
// newSet is the validator set that signed this block.  Only votes from new are
// sufficient for 2/3 majority in the new set as well, for it to be a valid
// commit.
//
// NOTE: This doesn't check whether the commit is a future commit, because the
// current height isn't part of the VoterSet.  Caller must check that the
// commit height is greater than the height for this validator set.
func (voters *VoterSet) VerifyFutureCommit(newSet *VoterSet, chainID string,
	blockID BlockID, height int64, commit *Commit) error {
	oldVoters := voters

	// Commit must be a valid commit for newSet.
	err := newSet.VerifyCommit(chainID, blockID, height, commit)
	if err != nil {
		return err
	}

	// Check old voting power.
	oldVotingPower := int64(0)
	seen := map[int]bool{}

	vals := make([]*Validator, 0, len(commit.Signatures))
	for _, commitSig := range commit.Signatures {
		if commitSig.Absent() {
			continue // OK, some signatures can be absent.
		}

		// See if this validator is in oldVals.
		oldIdx, val := oldVoters.GetByAddress(commitSig.ValidatorAddress)
		vals = append(vals, val)
		if val == nil || seen[oldIdx] {
			continue // missing or double vote...
		}
		seen[oldIdx] = true

		// Good!
		if blockID.Equals(commitSig.BlockID(commit.BlockID)) {
			oldVotingPower += val.VotingPower
		}
		// else {
		// It's OK that the BlockID doesn't match.  We include stray
		// signatures (~votes for nil) to measure validator availability.
		// }
	}

	// Validate signature.
	if err := commit.VerifySignatures(chainID, vals); err != nil {
		return err
	}

	if got, needed := oldVotingPower, oldVoters.TotalVotingPower()*2/3; got <= needed {
		return ErrNotEnoughVotingPowerSigned{Got: got, Needed: needed}
	}
	return nil
}

// VerifyCommitTrusting verifies that trustLevel ([1/3, 1]) of the validator
// set signed this commit.
// NOTE the given validators do not necessarily correspond to the validator set
// for this commit, but there may be some intersection.
func (voters *VoterSet) VerifyCommitTrusting(chainID string, blockID BlockID,
	height int64, commit *Commit, trustLevel tmmath.Fraction) error {

	if trustLevel.Numerator*3 < trustLevel.Denominator || // < 1/3
		trustLevel.Numerator > trustLevel.Denominator { // > 1
		panic(fmt.Sprintf("trustLevel must be within [1/3, 1], given %v", trustLevel))
	}

	if err := verifyCommitBasic(commit, height, blockID); err != nil {
		return err
	}

	var (
		talliedVotingPower int64
		seenVals           = make(map[int]int, len(commit.Signatures)) // validator index -> commit index
	)

	totalVotingPowerMulByNumerator, overflow := safeMul(voters.TotalVotingPower(), trustLevel.Numerator)
	if overflow {
		return errors.New("int64 overflow while calculating voting power needed. please provide smaller trustLevel numerator")
	}
	votingPowerNeeded := totalVotingPowerMulByNumerator / trustLevel.Denominator

	vals := make([]*Validator, 0, len(commit.Signatures))
	for idx, commitSig := range commit.Signatures {
		if commitSig.Absent() {
			continue // OK, some signatures can be absent.
		}

		// We don't know the validators that committed this block, so we have to
		// check for each vote if its validator is already known.
		valIdx, val := voters.GetByAddress(commitSig.ValidatorAddress)
		vals = append(vals, val)

		if firstIndex, ok := seenVals[valIdx]; ok { // double vote
			secondIndex := idx
			return errors.Errorf("double vote from %v (%d and %d)", val, firstIndex, secondIndex)
		}

		if val != nil {
			seenVals[valIdx] = idx

			// Good!
			if blockID.Equals(commitSig.BlockID(commit.BlockID)) {
				talliedVotingPower += val.VotingPower
			}
			// else {
			// It's OK that the BlockID doesn't match.  We include stray
			// signatures (~votes for nil) to measure validator availability.
			// }

			if talliedVotingPower > votingPowerNeeded {
				return nil
			}
		}
	}

	// Validate signature.
	if err := commit.VerifySignatures(chainID, vals); err != nil {
		return err
	}

	return ErrNotEnoughVotingPowerSigned{Got: talliedVotingPower, Needed: votingPowerNeeded}
}

// VerifyCommitLightTrusting verifies that trustLevel of the validator set signed
// this commit.
//
// This method is primarily used by the light client and does not check all the
// signatures.
//
// NOTE the given validators do not necessarily correspond to the validator set
// for this commit, but there may be some intersection.
//
// Panics if trustLevel is invalid.
func (voters *VoterSet) VerifyCommitLightTrusting(chainID string, blockID BlockID,
	height int64, commit *Commit, trustLevel tmmath.Fraction) error {

	// sanity check
	if trustLevel.Numerator*3 < trustLevel.Denominator || // < 1/3
		trustLevel.Numerator > trustLevel.Denominator { // > 1
		panic(fmt.Sprintf("trustLevel must be within [1/3, 1], given %v", trustLevel))
	}

	if err := verifyCommitBasic(commit, height, blockID); err != nil {
		return err
	}

	var (
		talliedVotingPower int64
		seenVals           = make(map[int]int, len(commit.Signatures)) // validator index -> commit index
	)

	// Safely calculate voting power needed.
	totalVotingPowerMulByNumerator, overflow := safeMul(voters.TotalVotingPower(), trustLevel.Numerator)
	if overflow {
		return errors.New("int64 overflow while calculating voting power needed. please provide smaller trustLevel numerator")
	}
	votingPowerNeeded := totalVotingPowerMulByNumerator / trustLevel.Denominator

	for idx, commitSig := range commit.Signatures {
		// No need to verify absent or nil votes.
		if !commitSig.ForBlock() {
			continue
		}

		// We don't know the validators that committed this block, so we have to
		// check for each vote if its validator is already known.
		valIdx, val := voters.GetByAddress(commitSig.ValidatorAddress)

		if val != nil {
			// check for double vote of validator on the same commit
			if firstIndex, ok := seenVals[valIdx]; ok {
				secondIndex := idx
				return errors.Errorf("double vote from %v (%d and %d)", val, firstIndex, secondIndex)
			}
			seenVals[valIdx] = idx

			talliedVotingPower += val.VotingPower

			if talliedVotingPower > votingPowerNeeded {
				// Validate signature.
				vals := make([]*Validator, 0, len(commit.Signatures))
				for _, cs := range commit.Signatures {
					_, val := voters.GetByAddress(cs.ValidatorAddress)
					vals = append(vals, val)
				}
				if err := commit.VerifySignatures(chainID, vals); err != nil {
					return err
				}
				return nil
			}
		}
	}

	return ErrNotEnoughVotingPowerSigned{Got: talliedVotingPower, Needed: votingPowerNeeded}
}

// ToProto converts VoterSet to protobuf
func (voters *VoterSet) ToProto() (*tmproto.VoterSet, error) {
	if voters == nil {
		return nil, errors.New("nil voter set")
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
	vsp.TotalVotingPower = voters.totalVotingPower

	return vsp, nil
}

// VoterSetFromProto sets a protobuf VoterSEt to the give pointer.
// It returns an error if any of the voters from the set or the proposer is invalid
func VoterSetFromProto(vp *tmproto.VoterSet) (*VoterSet, error) {
	if vp == nil {
		return nil, errors.New("nil voter set")
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
	voters.totalVotingPower = vp.GetTotalVotingPower()

	return voters, nil
}

func verifyCommitBasic(commit *Commit, height int64, blockID BlockID) error {
	if err := commit.ValidateBasic(); err != nil {
		return err
	}
	if height != commit.Height {
		return NewErrInvalidCommitHeight(height, commit.Height)
	}
	if !blockID.Equals(commit.BlockID) {
		return fmt.Errorf("invalid commit -- wrong block ID: want %v, got %v",
			blockID, commit.BlockID)
	}
	return nil
}

//-----------------

// IsErrNotEnoughVotingPowerSigned returns true if err is
// ErrNotEnoughVotingPowerSigned.
func IsErrNotEnoughVotingPowerSigned(err error) bool {
	_, ok := errors.Cause(err).(ErrNotEnoughVotingPowerSigned)
	return ok
}

// ErrNotEnoughVotingPowerSigned is returned when not enough validators signed
// a commit.
type ErrNotEnoughVotingPowerSigned struct {
	Got    int64
	Needed int64
}

func (e ErrNotEnoughVotingPowerSigned) Error() string {
	return fmt.Sprintf("invalid commit -- insufficient voting power: got %d, needed more than %d", e.Got, e.Needed)
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
	var valStrings []string
	voters.Iterate(func(index int, val *Validator) bool {
		valStrings = append(valStrings, val.String())
		return false
	})
	return fmt.Sprintf(`VoterSet{
%s  Validators:
%s    %v
%s}`,
		indent, indent, strings.Join(valStrings, "\n"+indent+"    "),
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
	newVoters := make([]*Validator, len(validators))
	voterCount := 0
	for _, val := range validators {
		if val.StakingPower == 0 {
			// remove the validator with the staking power of 0 from the voter set
			continue
		}
		newVoters[voterCount] = &Validator{
			Address:          val.Address,
			PubKey:           val.PubKey,
			StakingPower:     val.StakingPower,
			VotingPower:      val.StakingPower,
			ProposerPriority: val.ProposerPriority,
		}
		voterCount++
	}
	if voterCount < len(newVoters) {
		zeroRemoved := make([]*Validator, voterCount)
		copy(zeroRemoved, newVoters[:voterCount])
		newVoters = zeroRemoved
	}
	sort.Sort(ValidatorsByAddress(newVoters))
	return WrapValidatorsToVoterSet(newVoters)
}

func hashToSeed(hash []byte) uint64 {
	for len(hash) < 8 {
		hash = append(hash, byte(0))
	}
	return binary.LittleEndian.Uint64(hash[:8])
}

// MakeRoundHash combines the VRF hash, block height, and round to create a hash value for each round. This value is
// used for random sampling of the Proposer.
func MakeRoundHash(proofHash []byte, height int64, round int) []byte {
	b := make([]byte, 16)
	binary.LittleEndian.PutUint64(b, uint64(height))
	binary.LittleEndian.PutUint64(b[8:], uint64(round))
	hash := tmhash.New()
	hash.Write(proofHash)
	hash.Write(b[:8])
	hash.Write(b[8:16])
	return hash.Sum(nil)
}

// RandValidatorSet returns a randomized validator set, useful for testing.
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
		if threshold < uint64(cumulativePriority+candidate.StakingPower) {
			winner = candidates[i]
			winnerIdx = i
			found = true
			break
		}
		cumulativePriority += candidate.StakingPower
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
	totalStakingPower := getTotalStakingPower(candidates)
	tolerableByzantinePower := getTolerableByzantinePower(totalStakingPower, tolerableByzantinePercent)
	voters := make([]*voter, 0)
	sortValidators(candidates)

	zeroValidators := 0
	for i := len(candidates); candidates[i-1].StakingPower == 0; i-- {
		zeroValidators++
	}

	losersStakingPower := totalStakingPower
	for {
		// accumulateWinPoints(voters)
		for _, voter := range voters {
			// i = v1 ... vt
			// stakingPower(i) * 1000 / (stakingPower(vt+1 ... vn) + stakingPower(i))
			additionalWinPoint := new(big.Int).Mul(big.NewInt(voter.val.StakingPower),
				big.NewInt(precisionForSelection))
			additionalWinPoint.Div(additionalWinPoint, new(big.Int).Add(big.NewInt(losersStakingPower),
				big.NewInt(voter.val.StakingPower)))
			voter.winPoint.Add(voter.winPoint, additionalWinPoint)
		}
		// electVoter
		winnerIdx, winner := electVoter(&seed, candidates, len(voters)+zeroValidators, losersStakingPower)

		moveWinnerToLast(candidates, winnerIdx)
		voters = append(voters, &voter{
			val:      winner,
			winPoint: big.NewInt(precisionForSelection),
		})
		losersStakingPower -= winner.StakingPower

		// calculateVotingPowers(voters)
		totalWinPoint := new(big.Int)
		for _, voter := range voters {
			totalWinPoint.Add(totalWinPoint, voter.winPoint)
		}
		totalVotingPower := int64(0)
		for _, voter := range voters {
			winPoint := new(big.Int).Mul(voter.winPoint, big.NewInt(precisionForSelection))
			bigVotingPower := new(big.Int).Div(new(big.Int).Mul(winPoint, big.NewInt(totalStakingPower)), totalWinPoint)
			votingPower := new(big.Int).Div(bigVotingPower, big.NewInt(precisionCorrectionForSelection)).Int64()
			voter.val.VotingPower = votingPower
			totalVotingPower += votingPower
		}

		if len(voters) >= minVoters {
			// sort voters in ascending votingPower/stakingPower
			sortVoters(voters)

			topFVotersVotingPower := getTopByzantineVotingPower(voters, tolerableByzantinePower)
			if topFVotersVotingPower < totalVotingPower/3 {
				break
			}
		}

		if len(voters)+zeroValidators == len(candidates) {
			// there is no voter group satisfying the finality
			// cannot do sampling voters
			for _, c := range candidates {
				c.VotingPower = c.StakingPower
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

func getTotalStakingPower(validators []*Validator) int64 {
	totalStaking := int64(0)
	for _, v := range validators {
		totalStaking += v.StakingPower
	}
	return totalStaking
}

func getTopByzantineVotingPower(voters []*voter, tolerableByzantinePower int64) int64 {
	topFVotersStakingPower := int64(0)
	topFVotersVotingPower := int64(0)
	for _, voter := range voters {
		prev := topFVotersStakingPower
		topFVotersStakingPower += voter.val.StakingPower
		topFVotersVotingPower += voter.val.VotingPower
		if prev < tolerableByzantinePower && topFVotersStakingPower >= tolerableByzantinePower {
			break
		}
	}
	return topFVotersVotingPower
}

// sort validators in-place
func sortValidators(validators []*Validator) {
	sort.Slice(validators, func(i, j int) bool {
		if validators[i].StakingPower == validators[j].StakingPower {
			return bytes.Compare(validators[i].Address, validators[j].Address) == -1
		}
		return validators[i].StakingPower > validators[j].StakingPower
	})
}

// sortVoters is function to sort voters in descending votingPower/stakingPower in-place
func sortVoters(candidates []*voter) {
	sort.Slice(candidates, func(i, j int) bool {
		bigA := new(big.Int).Mul(big.NewInt(candidates[i].val.VotingPower), big.NewInt(candidates[j].val.StakingPower))
		bigB := new(big.Int).Mul(big.NewInt(candidates[j].val.VotingPower), big.NewInt(candidates[i].val.StakingPower))
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

func getTolerableByzantinePower(totalStakingPower int64, tolerableByzantinePercent int) int64 {
	// `totalStakingPower * tolerableByzantinePercent` may be overflow for int64 type
	bigMultiplied := new(big.Int).Mul(big.NewInt(totalStakingPower), big.NewInt(int64(tolerableByzantinePercent)))
	tolerableByzantinePower := new(big.Int).Div(bigMultiplied, big.NewInt(100))

	// ceiling
	if new(big.Int).Mul(tolerableByzantinePower, big.NewInt(100)).Cmp(bigMultiplied) < 0 {
		tolerableByzantinePower = new(big.Int).Add(tolerableByzantinePower, big.NewInt(1))
	}
	return tolerableByzantinePower.Int64()
}
