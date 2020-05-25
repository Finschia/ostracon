package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/crypto/merkle"
	"github.com/tendermint/tendermint/crypto/tmhash"
	tmmath "github.com/tendermint/tendermint/libs/math"
	tmrand "github.com/tendermint/tendermint/libs/rand"
)

var MaxVoters = 20

// VoterSet represent a set of *Validator at a given height.
type VoterSet struct {
	// NOTE: persisted via reflect, must be exported.
	Voters []*Validator `json:"voters"`

	// cached (unexported)
	totalVotingPower int64
}

func NewVoterSet(valz []*Validator) *VoterSet {
	sort.Sort(ValidatorsByAddress(valz))
	vals := &VoterSet{Voters: copyValidatorListShallow(valz), totalVotingPower: 0}
	vals.updateTotalVotingPower()
	return vals
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
// Panics if total voting power is bigger than MaxTotalStakingPower.
func (voters *VoterSet) updateTotalVotingPower() {
	sum := int64(0)
	for _, val := range voters.Voters {
		// mind overflow
		sum = safeAddClip(sum, val.StakingPower)
		if sum > MaxTotalStakingPower {
			panic(fmt.Sprintf(
				"Total voting power should be guarded to not exceed %v; got: %v",
				MaxTotalStakingPower,
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

		// Validate signature.
		voteSignBytes := commit.VoteSignBytes(chainID, idx)
		if !val.PubKey.VerifyBytes(voteSignBytes, commitSig.Signature) {
			return fmt.Errorf("wrong signature (#%d): %X", idx, commitSig.Signature)
		}
		// Good!
		if blockID.Equals(commitSig.BlockID(commit.BlockID)) {
			talliedVotingPower += val.StakingPower
		}
		// else {
		// It's OK that the BlockID doesn't match.  We include stray
		// signatures (~votes for nil) to measure validator availability.
		// }

		// return as soon as +2/3 of the signatures are verified
		if talliedVotingPower > votingPowerNeeded {
			return nil
		}
	}

	// talliedVotingPower <= needed, thus return error
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

	for idx, commitSig := range commit.Signatures {
		if commitSig.Absent() {
			continue // OK, some signatures can be absent.
		}

		// See if this validator is in oldVals.
		oldIdx, val := oldVoters.GetByAddress(commitSig.ValidatorAddress)
		if val == nil || seen[oldIdx] {
			continue // missing or double vote...
		}
		seen[oldIdx] = true

		// Validate signature.
		voteSignBytes := commit.VoteSignBytes(chainID, idx)
		if !val.PubKey.VerifyBytes(voteSignBytes, commitSig.Signature) {
			return errors.Errorf("wrong signature (#%d): %X", idx, commitSig.Signature)
		}
		// Good!
		if blockID.Equals(commitSig.BlockID(commit.BlockID)) {
			oldVotingPower += val.StakingPower
		}
		// else {
		// It's OK that the BlockID doesn't match.  We include stray
		// signatures (~votes for nil) to measure validator availability.
		// }
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
		votingPowerNeeded  = (voters.TotalVotingPower() * trustLevel.Numerator) / trustLevel.Denominator
	)

	for idx, commitSig := range commit.Signatures {
		if commitSig.Absent() {
			continue // OK, some signatures can be absent.
		}

		// We don't know the validators that committed this block, so we have to
		// check for each vote if its validator is already known.
		valIdx, val := voters.GetByAddress(commitSig.ValidatorAddress)

		if firstIndex, ok := seenVals[valIdx]; ok { // double vote
			secondIndex := idx
			return errors.Errorf("double vote from %v (%d and %d)", val, firstIndex, secondIndex)
		}

		if val != nil {
			seenVals[valIdx] = idx

			// Validate signature.
			voteSignBytes := commit.VoteSignBytes(chainID, idx)
			if !val.PubKey.VerifyBytes(voteSignBytes, commitSig.Signature) {
				return errors.Errorf("wrong signature (#%d): %X", idx, commitSig.Signature)
			}

			// Good!
			if blockID.Equals(commitSig.BlockID(commit.BlockID)) {
				talliedVotingPower += val.StakingPower
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

	return ErrNotEnoughVotingPowerSigned{Got: talliedVotingPower, Needed: votingPowerNeeded}
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

func SelectVoter(validators *ValidatorSet, proofHash []byte) *VoterSet {
	// TODO: decide MaxVoters; make it to config
	if len(proofHash) == 0 || validators.Size() <= MaxVoters {
		// height 1 has voter set that is same to validator set
		result := &VoterSet{Voters: copyValidatorListShallow(validators.Validators), totalVotingPower: 0}
		result.updateTotalVotingPower()
		return result
	}

	seed := hashToSeed(proofHash)
	candidates := make([]tmrand.Candidate, len(validators.Validators))
	for i, val := range validators.Validators {
		candidates[i] = &candidate{idx: i, win: 0, val: val}
	}
	totalSampling := tmrand.RandomSamplingToMax(seed, candidates, MaxVoters, uint64(validators.TotalStakingPower()))
	voters := 0
	for _, candi := range candidates {
		if candi.(*candidate).win > 0 {
			voters += 1
		}
	}

	vals := make([]*Validator, voters)
	index := 0
	for _, candi := range candidates {
		if candi.(*candidate).win > 0 {
			vals[index] = &Validator{Address: candi.(*candidate).val.Address,
				PubKey: candi.(*candidate).val.PubKey,
				// StakingPower = TotalStakingPower * win / totalSampling : can be overflow
				StakingPower: validators.TotalStakingPower()/int64(totalSampling)*int64(candi.(*candidate).win) +
					int64(math.Ceil(float64(validators.TotalStakingPower()%int64(totalSampling))/float64(int64(totalSampling))*
						float64(candi.(*candidate).win)))}
			index++
		}
	}
	return NewVoterSet(vals)
}

// This should be used in only test
func ToVoterAll(validators *ValidatorSet) *VoterSet {
	return NewVoterSet(validators.Validators)
}

// candidate save simple validator data for selecting proposer
type candidate struct {
	idx int
	win uint64
	val *Validator
}

func (c *candidate) Priority() uint64 {
	// TODO Is it possible to have a negative StakingPower?
	if c.val.StakingPower < 0 {
		return 0
	}
	return uint64(c.val.StakingPower)
}

func (c *candidate) LessThan(other tmrand.Candidate) bool {
	o, ok := other.(*candidate)
	if !ok {
		panic("incompatible type")
	}
	return bytes.Compare(c.val.Address, o.val.Address) < 0
}

func (c *candidate) IncreaseWin() {
	c.win++
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
	return vals, SelectVoter(vals, []byte{}), privValidators
}
