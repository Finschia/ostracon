package types

import (
	"bytes"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/crypto/merkle"
	tmmath "github.com/tendermint/tendermint/libs/math"
	"github.com/tendermint/tendermint/libs/rand"
	tmrand "github.com/tendermint/tendermint/libs/rand"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmtime "github.com/tendermint/tendermint/types/time"
)

//-------------------------------------------------------------------

// Check VerifyCommit, VerifyCommitLight and VerifyCommitLightTrusting basic
// verification.
func TestVoterSet_VerifyCommit_All(t *testing.T) {
	var (
		privKey = ed25519.GenPrivKey()
		pubKey  = privKey.PubKey()
		v1      = NewValidator(pubKey, 1000)
		vset    = ToVoterAll([]*Validator{v1})

		chainID = "Lalande21185"
	)

	vote := examplePrecommit()
	vote.ValidatorAddress = pubKey.Address()
	v := vote.ToProto()
	sig, err := privKey.Sign(VoteSignBytes(chainID, v))
	require.NoError(t, err)
	vote.Signature = sig

	commit := NewCommit(vote.Height, vote.Round, vote.BlockID, []CommitSig{vote.CommitSig()})

	vote2 := *vote
	sig2, err := privKey.Sign(VoteSignBytes("EpsilonEridani", v))
	require.NoError(t, err)
	vote2.Signature = sig2

	testCases := []struct {
		description string
		chainID     string
		blockID     BlockID
		height      int64
		commit      *Commit
		expErr      bool
	}{
		{"good", chainID, vote.BlockID, vote.Height, commit, false},

		{"wrong signature (#0)", "EpsilonEridani", vote.BlockID, vote.Height, commit, true},
		{"wrong block ID", chainID, makeBlockIDRandom(), vote.Height, commit, true},
		{"wrong height", chainID, vote.BlockID, vote.Height - 1, commit, true},

		{"wrong set size: 1 vs 0", chainID, vote.BlockID, vote.Height,
			NewCommit(vote.Height, vote.Round, vote.BlockID, []CommitSig{}), true},

		{"wrong set size: 1 vs 2", chainID, vote.BlockID, vote.Height,
			NewCommit(vote.Height, vote.Round, vote.BlockID,
				[]CommitSig{vote.CommitSig(), {BlockIDFlag: BlockIDFlagAbsent}}), true},

		{"insufficient voting power: got 0, needed more than 666", chainID, vote.BlockID, vote.Height,
			NewCommit(vote.Height, vote.Round, vote.BlockID, []CommitSig{{BlockIDFlag: BlockIDFlagAbsent}}), true},

		{"wrong signature (#0)", chainID, vote.BlockID, vote.Height,
			NewCommit(vote.Height, vote.Round, vote.BlockID, []CommitSig{vote2.CommitSig()}), true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			err := vset.VerifyCommit(tc.chainID, tc.blockID, tc.height, tc.commit)
			if tc.expErr {
				if assert.Error(t, err, "VerifyCommit") {
					assert.Contains(t, err.Error(), tc.description, "VerifyCommit")
				}
			} else {
				assert.NoError(t, err, "VerifyCommit")
			}

			err = vset.VerifyCommitLight(tc.chainID, tc.blockID, tc.height, tc.commit)
			if tc.expErr {
				if assert.Error(t, err, "VerifyCommitLight") {
					assert.Contains(t, err.Error(), tc.description, "VerifyCommitLight")
				}
			} else {
				assert.NoError(t, err, "VerifyCommitLight")
			}
		})
	}
}

func TestVoterSet_VerifyCommit_CheckAllSignatures(t *testing.T) {
	var (
		chainID = "test_chain_id"
		h       = int64(3)
		blockID = makeBlockIDRandom()
	)

	voteSet, _, voterSet, vals := randVoteSet(h, 0, tmproto.PrecommitType, 4, 10)
	commit, err := MakeCommit(blockID, h, 0, voteSet, vals, time.Now())
	require.NoError(t, err)

	// malleate 4th signature
	vote := voteSet.GetByIndex(3)
	v := vote.ToProto()
	err = vals[3].SignVote("CentaurusA", v)
	require.NoError(t, err)
	vote.Signature = v.Signature
	commit.Signatures[3] = vote.CommitSig()

	err = voterSet.VerifyCommit(chainID, blockID, h, commit)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "wrong signature (#3)")
	}
}

func TestVoterSet_VerifyCommitLight_ReturnsAsSoonAsMajorityOfVotingPowerSigned(t *testing.T) {
	var (
		chainID = "test_chain_id"
		h       = int64(3)
		blockID = makeBlockIDRandom()
	)

	voteSet, _, voterSet, vals := randVoteSet(h, 0, tmproto.PrecommitType, 4, 10)
	commit, err := MakeCommit(blockID, h, 0, voteSet, vals, time.Now())
	require.NoError(t, err)

	// malleate 4th signature (3 signatures are enough for 2/3+)
	vote := voteSet.GetByIndex(3)
	v := vote.ToProto()
	err = vals[3].SignVote("CentaurusA", v)
	require.NoError(t, err)
	vote.Signature = v.Signature
	commit.Signatures[3] = vote.CommitSig()

	err = voterSet.VerifyCommitLight(chainID, blockID, h, commit)
	assert.NoError(t, err)
}

func TestVoterSet_VerifyCommitLightTrusting_ReturnsAsSoonAsTrustLevelOfVotingPowerSigned(t *testing.T) {
	var (
		chainID = "test_chain_id"
		h       = int64(3)
		blockID = makeBlockIDRandom()
	)

	voteSet, _, voterSet, vals := randVoteSet(h, 0, tmproto.PrecommitType, 4, 10)
	commit, err := MakeCommit(blockID, h, 0, voteSet, vals, time.Now())
	require.NoError(t, err)

	// malleate 3rd signature (2 signatures are enough for 1/3+ trust level)
	vote := voteSet.GetByIndex(2)
	v := vote.ToProto()
	err = vals[2].SignVote("CentaurusA", v)
	require.NoError(t, err)
	vote.Signature = v.Signature
	commit.Signatures[2] = vote.CommitSig()

	err = voterSet.VerifyCommitLightTrusting(chainID, commit, tmmath.Fraction{Numerator: 1, Denominator: 3})
	assert.NoError(t, err)
}

func TestValidatorSet_VerifyCommitLightTrusting(t *testing.T) {
	var (
		blockID                    = makeBlockIDRandom()
		voteSet, _, voterSet, vals = randVoteSet(1, 1, tmproto.PrecommitType, 6, 1)
		commit, err                = MakeCommit(blockID, 1, 1, voteSet, vals, time.Now())
		_, newVoterSet, _          = RandVoterSet(2, 1)
	)
	require.NoError(t, err)

	testCases := []struct {
		voterSet *VoterSet
		err      bool
	}{
		// good
		0: {
			voterSet: voterSet,
			err:      false,
		},
		// bad - no overlap between voter sets
		1: {
			voterSet: newVoterSet,
			err:      true,
		},
		// good - first two are different but the rest of the same -> >1/3
		2: {
			voterSet: WrapValidatorsToVoterSet(append(newVoterSet.Voters, voterSet.Voters...)),
			err:      false,
		},
	}

	for _, tc := range testCases {
		err = tc.voterSet.VerifyCommitLightTrusting("test_chain_id", commit,
			tmmath.Fraction{Numerator: 1, Denominator: 3})
		if tc.err {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}
}

func TestValidatorSet_VerifyCommitLightTrustingErrorsOnOverflow(t *testing.T) {
	var (
		blockID                    = makeBlockIDRandom()
		voteSet, _, voterSet, vals = randVoteSet(1, 1, tmproto.PrecommitType, 1, MaxTotalStakingPower)
		commit, err                = MakeCommit(blockID, 1, 1, voteSet, vals, time.Now())
	)
	require.NoError(t, err)

	err = voterSet.VerifyCommitLightTrusting("test_chain_id", commit,
		tmmath.Fraction{Numerator: 25, Denominator: 55})
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "int64 overflow")
	}
}

func countZeroStakingPower(vals []*Validator) int {
	count := 0
	for _, v := range vals {
		if v.StakingPower == 0 {
			count++
		}
	}
	return count
}

func verifyVoterSetSame(t *testing.T, vset1, vset2 *VoterSet) {
	assert.True(t, vset1.Size() == vset2.Size())
	for i, v1 := range vset1.Voters {
		v2 := vset2.Voters[i]
		assert.True(t, v1.Address.String() == v2.Address.String())
		assert.True(t, v1.VotingPower == v2.VotingPower)
		assert.True(t, v1.StakingPower == v2.StakingPower)
	}
}

func verifyVoterSetDifferent(t *testing.T, vset1, vset2 *VoterSet) {
	result := vset1.Size() != vset2.Size()
	if !result {
		for i, v1 := range vset1.Voters {
			v2 := vset2.Voters[i]
			if v1.Address.String() != v2.Address.String() ||
				v1.StakingPower != v2.StakingPower ||
				v1.VotingPower != v2.VotingPower {
				result = true
				break
			}
		}
	}
	assert.True(t, result)
}

func TestSelectVoter(t *testing.T) {
	valSet := newValidatorSet(30, func(i int) int64 { return int64(rand.Int()%10000 + 100) })
	valSet.Validators[0].StakingPower = 0

	zeroVals := countZeroStakingPower(valSet.Validators)
	genDoc := &GenesisDoc{
		GenesisTime: tmtime.Now(),
		ChainID:     "tendermint-test",
		VoterParams: &VoterParams{10, 20},
		Validators:  toGenesisValidators(valSet.Validators),
	}
	hash := genDoc.Hash()

	// verifying determinism
	voterSet1 := SelectVoter(valSet, hash, genDoc.VoterParams)
	voterSet2 := SelectVoter(valSet, hash, genDoc.VoterParams)
	verifyVoterSetSame(t, voterSet1, voterSet2)

	// verifying randomness
	hash[0] = (hash[0] & 0xFE) | (^(hash[0] & 0x01) & 0x01) // reverse 1 bit of hash
	voterSet3 := SelectVoter(valSet, hash, genDoc.VoterParams)
	verifyVoterSetDifferent(t, voterSet1, voterSet3)

	// verifying zero-staking removed
	assert.True(t, countZeroStakingPower(voterSet1.Voters) == 0)

	// case that all validators are voters
	voterSet := SelectVoter(valSet, hash, &VoterParams{30, 1})
	assert.True(t, voterSet.Size() == 30-zeroVals)
	voterSet = SelectVoter(valSet, nil, genDoc.VoterParams)
	assert.True(t, voterSet.Size() == 30-zeroVals)
}

func zeroIncluded(valSet *ValidatorSet) bool {
	for _, v := range valSet.Validators {
		if v.StakingPower == 0 {
			return true
		}
	}
	return false
}

func areSame(a *ValidatorSet, b *VoterSet) bool {
	if a.Size() != b.Size() {
		return false
	}
	for i, v := range a.Validators {
		if !v.PubKey.Equals(b.Voters[i].PubKey) {
			return false
		}
		if v.Address.String() != b.Voters[i].Address.String() {
			return false
		}
		if v.StakingPower != b.Voters[i].StakingPower {
			return false
		}
	}
	return true
}

func TestToVoterAll(t *testing.T) {
	valSet := randValidatorSet(30)
	vals := valSet.Validators
	vals[0].StakingPower = 0
	vals[5].StakingPower = 0
	vals[28].StakingPower = 0
	zeroRemovedVoters := ToVoterAll(vals)
	assert.True(t, zeroRemovedVoters.Size() == 27)

	valSet = randValidatorSet(3)
	vals = valSet.Validators
	vals[0].StakingPower = 0
	vals[1].StakingPower = 0
	vals[2].StakingPower = 0
	zeroRemovedVoters = ToVoterAll(vals)
	assert.True(t, zeroRemovedVoters.Size() == 0)

	for i := 0; i < 100; i++ {
		valSet = randValidatorSet(10)
		if zeroIncluded(valSet) {
			continue
		}
		voters := ToVoterAll(valSet.Validators)
		assert.True(t, areSame(valSet, voters), "[%d] %+v != %+v", i, valSet, voters)
	}
}

func toGenesisValidators(vals []*Validator) []GenesisValidator {
	genVals := make([]GenesisValidator, len(vals))
	for i, val := range vals {
		genVals[i] = GenesisValidator{Address: val.Address, PubKey: val.PubKey, Power: val.StakingPower, Name: "name"}
	}
	return genVals
}

/**
The result when we set LoopCount to 10000
  << min power=100, max power=100, actual average voters=10, max voters=10 >> largest gap: 0.040000
  << min power=100, max power=100, actual average voters=20, max voters=20 >> largest gap: 0.030000
  << min power=100, max power=100, actual average voters=29, max voters=29 >> largest gap: 0.010000
  << min power=100, max power=10000, actual average voters=10, max voters=10 >> largest gap: 0.183673
  << min power=100, max power=10000, actual average voters=20, max voters=20 >> largest gap: 0.128788
  << min power=100, max power=10000, actual average voters=28, max voters=29 >> largest gap: 0.304348
  << min power=100, max power=1000000, actual average voters=10, max voters=10 >> largest gap: 0.093158
  << min power=100, max power=1000000, actual average voters=20, max voters=20 >> largest gap: 0.094404
  << min power=100, max power=1000000, actual average voters=28, max voters=29 >> largest gap: 0.194133
  << min power=100, max power=100000000, actual average voters=10, max voters=10 >> largest gap: 0.076536
  << min power=100, max power=100000000, actual average voters=20, max voters=20 >> largest gap: 0.076547
  << min power=100, max power=100000000, actual average voters=29, max voters=29 >> largest gap: 0.147867
*/
func TestSelectVoterReasonableStakingPower(t *testing.T) {
	// Raise LoopCount to get smaller gap over 10000. But large LoopCount takes a lot of time
	const LoopCount = 100
	for minMaxRate := 1; minMaxRate <= 1000000; minMaxRate *= 100 {
		findLargestStakingPowerGap(t, LoopCount, minMaxRate, 10)
		findLargestStakingPowerGap(t, LoopCount, minMaxRate, 20)
		findLargestStakingPowerGap(t, LoopCount, minMaxRate, 29)
	}
}

func findLargestStakingPowerGap(t *testing.T, loopCount int, minMaxRate int, maxVoters int) {
	valSet, privMap := randValidatorSetWithMinMax(PrivKeyEd25519, 30, 100, 100*int64(minMaxRate))
	genDoc := &GenesisDoc{
		GenesisTime: tmtime.Now(),
		ChainID:     "tendermint-test",
		VoterParams: DefaultVoterParams(),
		Validators:  toGenesisValidators(valSet.Validators),
	}
	hash := genDoc.Hash()
	accumulation := make(map[string]int64)
	totalVoters := 0
	for i := 0; i < loopCount; i++ {
		voterSet := SelectVoter(valSet, hash, genDoc.VoterParams)
		for _, voter := range voterSet.Voters {
			accumulation[voter.Address.String()] += voter.StakingPower
		}
		proposer := valSet.SelectProposer(hash, int64(i), 0)
		message := MakeRoundHash(hash, int64(i), 0)
		proof, _ := privMap[proposer.Address.String()].GenerateVRFProof(message)
		pubKey, _ := privMap[proposer.Address.String()].GetPubKey()
		hash, _ = pubKey.VRFVerify(proof, message)
		totalVoters += voterSet.Size()
	}
	largestGap := float64(0)
	for _, val := range valSet.Validators {
		acc := accumulation[val.Address.String()] / int64(loopCount)
		if math.Abs(float64(val.StakingPower-acc))/float64(val.StakingPower) > largestGap {
			largestGap = math.Abs(float64(val.StakingPower-acc)) / float64(val.StakingPower)
		}
	}
	t.Logf("<< min power=100, max power=%d, actual average voters=%d, max voters=%d >> largest gap: %f",
		100*minMaxRate, totalVoters/loopCount, maxVoters, largestGap)
}

/**
  This test is a test to see the difference between MaxVoters and the actual number of elected voters.
  This test is to identify the minimum MaxVoters that cannot be selected as much as MaxVoters by fixing
	MaxSamplingLoopTry.
  If MaxSamplingLoopTry is very large then actual elected voters is up to MaxVoters,
  but large MaxSamplingLoopTry takes too much time.
*/
func TestSelectVoterMaxVarious(t *testing.T) {
	hash := 0
	for minMaxRate := 1; minMaxRate <= 100000000; minMaxRate *= 10000 {
		t.Logf("<<< min: 100, max: %d >>>", 100*minMaxRate)
		for validators := 16; validators <= 256; validators *= 4 {
			for voters := 1; voters <= validators; voters += 10 {
				valSet, _ := randValidatorSetWithMinMax(PrivKeyEd25519, validators, 100, 100*int64(minMaxRate))
				voterSet := SelectVoter(valSet, []byte{byte(hash)}, &VoterParams{int32(voters), 20})
				if voterSet.Size() < voters {
					t.Logf("Cannot elect voters up to MaxVoters: validators=%d, MaxVoters=%d, actual voters=%d",
						validators, voters, voterSet.Size())
					break
				}
				hash++
			}
		}
	}
}

func makeByzantine(valSet *ValidatorSet, rate float64) map[string]bool {
	result := make(map[string]bool)
	byzantinePower := int64(0)
	threshold := int64(float64(valSet.TotalStakingPower()) * rate)
	for _, v := range valSet.Validators {
		if byzantinePower+v.StakingPower > threshold {
			break
		}
		result[v.Address.String()] = true
		byzantinePower += v.StakingPower
	}
	return result
}

func TestVoterSetProtoBuf(t *testing.T) {
	_, voterSet, _ := RandVoterSet(10, 100)
	_, voterSet2, _ := RandVoterSet(10, 100)
	voterSet2.Voters[0] = &Validator{}

	testCase := []struct {
		msg      string
		v1       *VoterSet
		expPass1 bool
		expPass2 bool
	}{
		{"success", voterSet, true, true},
		{"fail voterSet2, pubkey empty", voterSet2, false, false},
		{"fail empty voterSet", &VoterSet{}, true, false},
		{"false nil", nil, true, false},
	}
	for _, tc := range testCase {
		protoVoterSet, err := tc.v1.ToProto()
		if tc.expPass1 {
			require.NoError(t, err, tc.msg)
		} else {
			require.Error(t, err, tc.msg)
		}

		vSet, err := VoterSetFromProto(protoVoterSet)
		if tc.expPass2 {
			require.NoError(t, err, tc.msg)
			require.EqualValues(t, tc.v1, vSet, tc.msg)
		} else {
			require.Error(t, err, tc.msg)
		}
	}
}

func testVotingPower(t *testing.T, valSet *ValidatorSet) {
	voterParams := &VoterParams{
		VoterElectionThreshold:          100,
		MaxTolerableByzantinePercentage: 20,
	}

	voterSetNoSampling := SelectVoter(valSet, []byte{0}, voterParams)
	for _, v := range voterSetNoSampling.Voters {
		assert.True(t, v.StakingPower == v.VotingPower)
	}

	for i := 90; i > 50; i-- {
		voterParams.VoterElectionThreshold = int32(i)
		voterSetSampling := SelectVoter(valSet, []byte{0}, voterParams)
		allSame := true
		for _, v := range voterSetSampling.Voters {
			if v.StakingPower != v.VotingPower {
				allSame = false
				break
			}
		}
		assert.False(t, allSame)
		assert.True(t, valSet.TotalStakingPower() > voterSetSampling.TotalVotingPower())
		// total voting power can not be less than total staking power - precisionForSelection(1000)

		//TODO: make test code for new voting power
		//assert.True(t, valSet.TotalStakingPower()-voterSetSampling.TotalVotingPower() <= 1000)
	}
}

func TestVotingPower(t *testing.T) {
	vals := make([]*Validator, 100)
	for i := 0; i < len(vals); i++ {
		vals[i] = newValidator(tmrand.Bytes(32), 100)
	}
	testVotingPower(t, NewValidatorSet(vals))
	vals2 := make([]*Validator, 100)
	for i := 0; i < len(vals2); i++ {
		vals2[i] = newValidator(rand.Bytes(32), MaxTotalStakingPower/100)
	}
	testVotingPower(t, NewValidatorSet(vals2))
}

func resetPoints(validators *ValidatorSet) {
	for _, v := range validators.Validators {
		v.VotingPower = 0
	}
}

func isByzantineTolerable(validators []*Validator, tolerableByzantinePercentage int) bool {
	totalStakingPower := int64(0)
	totalVotingPower := int64(0)
	for _, v := range validators {
		totalStakingPower += v.StakingPower
		totalVotingPower += v.VotingPower
	}
	voters := make([]*voter, len(validators))
	for i, v := range validators {
		voters[i] = &voter{val: v}
	}
	tolerableByzantinePower := getTolerableByzantinePower(totalStakingPower, tolerableByzantinePercentage)
	topFVotersVotingPower := getTopByzantineVotingPower(voters, tolerableByzantinePower)
	return topFVotersVotingPower < totalVotingPower/3
}

func pickRandomVoter(voters []*Validator) (target *Validator, remain []*Validator) {
	if len(voters) == 0 {
		return nil, voters
	}
	idx := int(rand.Uint() % uint(len(voters)))
	remain = make([]*Validator, len(voters)-1)
	count := 0
	for i, v := range voters {
		if i == idx {
			continue
		}
		remain[count] = v
		count++
	}
	return voters[idx], remain
}

func TestElectVotersNonDupByzantineTolerable(t *testing.T) {
	seed := time.Now().UnixNano()
	rand.Seed(seed)
	t.Logf("used seed=%d", seed)
	validatorSet := newValidatorSet(100, func(i int) int64 { return int64(rand.Uint32()%10000 + 100) })
	tolerableByzantinePercentage := int(rand.Uint() % 33)
	tolerableByzantinePower := getTolerableByzantinePower(validatorSet.TotalStakingPower(),
		tolerableByzantinePercentage)
	voters := electVotersNonDup(validatorSet.Validators, rand.Uint64(), tolerableByzantinePercentage, int(rand.Uint()%100))
	totalVoting := int64(0)
	for _, v := range voters {
		totalVoting += v.VotingPower
	}
	for i := 0; i < 100; i++ {
		copied := copyValidatorListShallow(voters)
		sumStaking := int64(0)
		sumVoting := int64(0)
		for {
			var one *Validator
			one, copied = pickRandomVoter(copied)
			if one == nil {
				break
			}
			sumStaking += one.StakingPower
			sumVoting += one.VotingPower
			if sumStaking >= tolerableByzantinePower {
				break
			}
		}
		assert.True(t, sumVoting < totalVoting/3)
	}
}

func TestElectVotersNonDupMinVoters(t *testing.T) {
	seed := time.Now().UnixNano()
	rand.Seed(seed)
	t.Logf("used seed=%d", seed)
	validatorSet := newValidatorSet(100, func(i int) int64 { return int64(rand.Uint32()%10000 + 100) })
	tolerableByzantinePercentage := int(rand.Uint() % 33)
	for i := 0; i <= 100; i++ {
		voters := electVotersNonDup(validatorSet.Validators, rand.Uint64(), tolerableByzantinePercentage, i)
		assert.True(t, len(voters) >= i, "%d < %d", len(voters), i)
	}
}

func TestElectVotersNonDupVoterCountHardCode(t *testing.T) {
	validatorSet := newValidatorSet(100, func(i int) int64 { return int64(i) })
	expected := [][]int{
		{6, 12, 15, 21, 21, 26, 29, 34, 36, 39, 41, 44, 48, 54, 54, 57, 65, 65, 69, 71, 76, 80, 82, 84, 87, 91,
			100, 100, 100, 100, 100, 100, 100},
		{10, 12, 15, 21, 21, 26, 29, 34, 36, 39, 41, 44, 48, 54, 54, 57, 65, 65, 69, 71, 76, 80, 82, 84, 87, 91,
			100, 100, 100, 100, 100, 100, 100},
		{20, 20, 20, 21, 21, 26, 29, 34, 36, 39, 41, 44, 48, 54, 54, 57, 65, 65, 69, 71, 76, 80, 82, 84, 87, 91,
			100, 100, 100, 100, 100, 100, 100},
		{30, 30, 30, 30, 30, 30, 30, 34, 36, 39, 41, 44, 48, 54, 54, 57, 65, 65, 69, 71, 76, 80, 82, 84, 87, 91,
			100, 100, 100, 100, 100, 100, 100},
		{40, 40, 40, 40, 40, 40, 40, 40, 40, 40, 41, 44, 48, 54, 54, 57, 65, 65, 69, 71, 76, 80, 82, 84, 87, 91,
			100, 100, 100, 100, 100, 100, 100},
		{50, 50, 50, 50, 50, 50, 50, 50, 50, 50, 50, 50, 50, 54, 54, 57, 65, 65, 69, 71, 76, 80, 82, 84, 87, 91,
			100, 100, 100, 100, 100, 100, 100},
		{60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 65, 65, 69, 71, 76, 80, 82, 84, 87, 91,
			100, 100, 100, 100, 100, 100, 100},
		{70, 70, 70, 70, 70, 70, 70, 70, 70, 70, 70, 70, 70, 70, 70, 70, 70, 70, 70, 71, 76, 80, 82, 84, 87, 91,
			100, 100, 100, 100, 100, 100, 100},
		{80, 80, 80, 80, 80, 80, 80, 80, 80, 80, 80, 80, 80, 80, 80, 80, 80, 80, 80, 80, 80, 80, 82, 84, 87, 91,
			100, 100, 100, 100, 100, 100, 100},
		{90, 90, 90, 90, 90, 90, 90, 90, 90, 90, 90, 90, 90, 90, 90, 90, 90, 90, 90, 90, 90, 90, 90, 90, 90, 91,
			100, 100, 100, 100, 100, 100, 100},
		{100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100,
			100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100},
	}
	for i := 0; i <= 100; i += 10 {
		for j := 1; j <= 33; j++ {
			voters := electVotersNonDup(validatorSet.Validators, 0, j, i)
			assert.True(t, len(voters) == expected[i/10][j-1])
		}
	}

	validatorSet = newValidatorSet(100, func(i int) int64 { return int64(100) })
	expected2 := []int{4, 7, 10, 13, 16, 20, 23, 27, 30, 34, 37, 41, 45, 49, 53, 57, 61, 66, 70, 74, 78, 82, 86, 90,
		93, 96, 99, 100, 100, 100, 100, 100, 100}
	for j := 1; j <= 33; j++ {
		voters := electVotersNonDup(validatorSet.Validators, 0, j, 0)
		assert.True(t, len(voters) == expected2[j-1])
	}

	validatorSet = newValidatorSet(100, func(i int) int64 { return int64((i + 1) * (i + 1)) })
	expected3 := []int{6, 9, 15, 17, 20, 25, 27, 30, 34, 34, 39, 41, 44, 44, 51, 53, 56, 56, 62, 65, 65, 68, 69, 73,
		100, 100, 100, 100, 100, 100, 100, 100, 100}
	for j := 1; j <= 33; j++ {
		voters := electVotersNonDup(validatorSet.Validators, 0, j, 0)
		assert.True(t, len(voters) == expected3[j-1])
	}

	staking := []int64{150000, 80000, 70000, 60000, 50000, 40000, 30000, 20000, 10000, 10000, 10000, 10000, 10000,
		10000, 10000, 10000, 10000, 10000, 10000, 10000, 10000, 10000, 10000, 10000, 10000, 10000, 10000, 10000, 5000,
		5000, 5000, 5000, 5000, 5000, 5000, 5000, 5000, 5000, 5000, 5000, 5000, 5000, 5000, 5000, 5000, 5000, 5000,
		5000, 5000, 5000, 5000, 5000, 5000, 5000, 5000, 5000, 5000, 5000, 5000, 5000, 5000, 5000, 5000, 5000, 5000,
		5000, 5000, 5000, 4200, 4200, 4000, 3800, 3500, 3500, 3500, 3500, 3500, 3500, 3500, 3500, 3500, 3500, 3500,
		3500, 3500, 3500, 3500, 3500, 3500, 3500, 3500, 3500, 3200, 3200, 3200, 3200, 400, 300, 200, 100}
	validatorSet = newValidatorSet(100, func(i int) int64 { return staking[i] })
	expected4 := []int{9, 13, 17, 21, 26, 29, 34, 37, 40, 44, 46, 51, 55, 58, 61, 64, 70, 72, 76, 82, 87,
		100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100}
	for j := 1; j <= 33; j++ {
		voters := electVotersNonDup(validatorSet.Validators, 0, j, 0)
		assert.True(t, len(voters) == expected4[j-1])
	}

	cosmosStaking := []int64{1234042, 31110, 246201, 1831691, 1523490, 107361, 1236356, 161896, 19927, 938209, 3272742,
		100307, 12304849, 94597, 31575, 324626, 133737, 4312343, 2194691, 6972384, 692945, 1286209, 4798646, 1570259,
		5898004, 386084, 429788, 178651, 3665192, 2552662, 532779, 445203, 109041, 5194266, 1093331, 35002, 1090943,
		1134566, 10149041, 897491, 458723, 1382301, 112657, 661520, 181534, 4734863, 385250, 2680449, 298927, 110777,
		570288, 2739273, 1717590, 1241054, 2607278, 25851, 1159428, 20716, 878196, 1869523, 4938663, 137206, 67957,
		5279435, 333170, 91352, 137427, 59996, 804791, 3154819, 732246, 6222939, 918502, 278327, 223664, 508546,
		10218349, 1025785, 710880, 2375372, 625513, 1162888, 1741665, 851293, 2332027, 601971, 67995, 442813, 908473,
		26882, 586028, 542899, 594937, 517893, 487137, 5962613, 781432, 20063, 763681, 665929, 194212, 439620, 3295816,
		3738661, 520012, 185377, 520152, 801032, 162559, 785938, 250053, 608602, 245002, 300770, 117118, 595488, 111634,
		1753346, 345997, 25211, 68514, 1004207, 11955082, 525223, 276736}
	validatorSet = newValidatorSet(len(cosmosStaking), func(i int) int64 { return cosmosStaking[i] })
	expected5 := []int{9, 14, 19, 23, 26, 31, 34, 38, 41, 43, 45, 48, 54, 55, 60, 61, 66, 73, 78, 78, 82, 90,
		125, 125, 125, 125, 125, 125, 125, 125, 125, 125, 125}
	for j := 1; j <= 33; j++ {
		voters := electVotersNonDup(validatorSet.Validators, 0, j, 0)
		assert.True(t, len(voters) == expected5[j-1])
	}
}

func TestElectVotersNonDupCandidate(t *testing.T) {
	validatorSet := newValidatorSet(100, func(i int) int64 { return int64(1000 * (i + 1)) })

	winners := electVotersNonDup(validatorSet.Validators, 0, 20, 0)
	assert.True(t, isByzantineTolerable(winners, 20))
}

// test samplingThreshold
func TestElectVotersNonDupSamplingThreshold(t *testing.T) {
	candidates := newValidatorSet(100, func(i int) int64 { return int64(1000 * (i + 1)) })

	for i := 1; i <= 33; i++ {
		winners := electVotersNonDup(candidates.Validators, 0, i, 0)
		if len(winners) < 100 {
			assert.True(t, isByzantineTolerable(winners, i))
		}
		resetPoints(candidates)
	}
}

// test downscale of win point cases
func TestElectVotersNonDupDownscale(t *testing.T) {
	candidates := newValidatorSet(10, func(i int) int64 {
		if i == 0 {
			return MaxTotalStakingPower >> 1
		}
		if i == 1 {
			return 1 << 55
		}
		if i == 3 {
			return 1 << 54
		}
		if i == 4 {
			return 1 << 55
		}
		return int64(i)
	})
	electVotersNonDup(candidates.Validators, 0, 20, 0)
}

// test random election should be deterministic
func TestElectVotersNonDupDeterministic(t *testing.T) {
	candidates1 := newValidatorSet(100, func(i int) int64 { return int64(i + 1) })
	candidates2 := newValidatorSet(100, func(i int) int64 { return int64(i + 1) })
	for i := 1; i <= 100; i++ {
		winners1 := electVotersNonDup(candidates1.Validators, uint64(i), 24, 0)
		winners2 := electVotersNonDup(candidates2.Validators, uint64(i), 24, 0)
		sameVoters(winners1, winners2)
		resetPoints(candidates1)
		resetPoints(candidates2)
	}
}

func TestElectVotersNonDupIncludingZeroStakingPower(t *testing.T) {
	// first candidate's priority is 0
	candidates1 := newValidatorSet(100, func(i int) int64 { return int64(i) })
	winners1 := electVotersNonDup(candidates1.Validators, 0, 20, 0)
	assert.True(t, isByzantineTolerable(winners1, 20))

	//half of candidates has 0 priority
	candidates2 := newValidatorSet(100, func(i int) int64 {
		if i < 50 {
			return 0
		}
		return int64(i)
	})
	winners2 := electVotersNonDup(candidates2.Validators, 0, 20, 0)
	assert.True(t, isByzantineTolerable(winners2, 20))
}

func TestElectVotersNonDupOverflow(t *testing.T) {
	number := 98
	candidates := newValidatorSet(number, func(i int) int64 { return MaxTotalStakingPower / int64(number+2) })
	totalPriority := candidates.totalStakingPower
	assert.True(t, totalPriority < math.MaxInt64)
	winners := electVotersNonDup(candidates.Validators, rand.Uint64(), 20, 0)
	assert.True(t, isByzantineTolerable(winners, 20))
}

func accumulateAndResetReward(voters []*Validator, acc map[string]int64) int64 {
	total := int64(0)
	for _, v := range voters {
		acc[v.Address.String()] += v.VotingPower
		total += v.VotingPower
	}
	return total
}

// test reward fairness
func TestElectVotersNonDupReward(t *testing.T) {
	candidates := newValidatorSet(30, func(i int) int64 { return int64(i + 1) })

	accumulatedRewards := make(map[string]int64, 30)
	for i := 0; i < 3000; i++ {
		winners := electVotersNonDup(candidates.Validators, uint64(i), 20, 0)
		accumulateAndResetReward(winners, accumulatedRewards)
	}
	sortValidators(candidates.Validators)
	for i := 0; i < 29; i++ {
		assert.True(t, accumulatedRewards[candidates.Validators[i].Address.String()] >
			accumulatedRewards[candidates.Validators[i+1].Address.String()])
	}
}

/**
conditions for fair reward
1. even staking power(less difference between min staking and max staking)
2. large total staking(a small total staking power makes a large error when converting float into int)
3. many sampling count
4. loop count
*/

func TestElectVotersNonDupEquity(t *testing.T) {
	t.Skip("this test case need a new reward rule")
	loopCount := 10000

	// good condition
	candidates := newValidatorSet(100, func(i int) int64 { return 1000000 + rand.Int64()&0xFFFFF })
	totalStaking := int64(0)
	for _, c := range candidates.Validators {
		totalStaking += c.StakingPower
	}

	accumulatedRewards := make(map[string]int64, 100)
	totalAccumulateRewards := int64(0)
	for i := 0; i < loopCount; i++ {
		electVotersNonDup(candidates.Validators, uint64(i), 20, 0)
		totalAccumulateRewards += accumulateAndResetReward(candidates.Validators, accumulatedRewards)
	}
	for i := 0; i < 99; i++ {
		rewardRate := float64(accumulatedRewards[candidates.Validators[i].Address.String()]) /
			float64(totalAccumulateRewards)
		stakingRate := float64(candidates.Validators[i].StakingPower) / float64(totalStaking)
		rate := rewardRate / stakingRate
		rewardPerStakingDiff := math.Abs(1 - rate)
		assert.True(t, rewardPerStakingDiff < 0.01)
	}

	// =======================================================================================================
	// The codes below are not test codes to verify logic,
	// but codes to find out what parameters are that weaken the equity of rewards.

	// violation of condition 1
	candidates = newValidatorSet(100, func(i int) int64 { return rand.Int64() & 0xFFFFFFFFF })
	accumulatedRewards = make(map[string]int64, 100)
	for i := 0; i < loopCount; i++ {
		electVotersNonDup(candidates.Validators, uint64(i), 20, 0)
		accumulateAndResetReward(candidates.Validators, accumulatedRewards)
	}
	maxRewardPerStakingDiff := float64(0)
	for i := 0; i < 99; i++ {
		rewardPerStakingDiff :=
			math.Abs(float64(accumulatedRewards[candidates.Validators[i].Address.String()])/
				float64(candidates.Validators[i].StakingPower)/float64(loopCount) - 1)
		if maxRewardPerStakingDiff < rewardPerStakingDiff {
			maxRewardPerStakingDiff = rewardPerStakingDiff
		}
	}
	t.Logf("[! condition 1] max reward per staking difference: %f", maxRewardPerStakingDiff)

	// violation of condition 2
	candidates = newValidatorSet(100, func(i int) int64 { return rand.Int64() & 0xFFFFF })
	accumulatedRewards = make(map[string]int64, 100)
	for i := 0; i < loopCount; i++ {
		electVotersNonDup(candidates.Validators, uint64(i), 20, 0)
		accumulateAndResetReward(candidates.Validators, accumulatedRewards)
	}
	maxRewardPerStakingDiff = float64(0)
	for i := 0; i < 99; i++ {
		rewardPerStakingDiff :=
			math.Abs(float64(accumulatedRewards[candidates.Validators[i].Address.String()])/
				float64(candidates.Validators[i].StakingPower)/float64(loopCount) - 1)
		if maxRewardPerStakingDiff < rewardPerStakingDiff {
			maxRewardPerStakingDiff = rewardPerStakingDiff
		}
	}
	t.Logf("[! condition 2] max reward per staking difference: %f", maxRewardPerStakingDiff)

	// violation of condition 3
	candidates = newValidatorSet(100, func(i int) int64 { return 1000000 + rand.Int64()&0xFFFFF })
	accumulatedRewards = make(map[string]int64, 100)
	for i := 0; i < loopCount; i++ {
		electVotersNonDup(candidates.Validators, uint64(i), 20, 0)
		accumulateAndResetReward(candidates.Validators, accumulatedRewards)
	}
	maxRewardPerStakingDiff = float64(0)
	for i := 0; i < 99; i++ {
		rewardPerStakingDiff :=
			math.Abs(float64(accumulatedRewards[candidates.Validators[i].Address.String()])/
				float64(candidates.Validators[i].StakingPower)/float64(loopCount) - 1)
		if maxRewardPerStakingDiff < rewardPerStakingDiff {
			maxRewardPerStakingDiff = rewardPerStakingDiff
		}
	}
	t.Logf("[! condition 3] max reward per staking difference: %f", maxRewardPerStakingDiff)

	// violation of condition 4
	loopCount = 100
	candidates = newValidatorSet(100, func(i int) int64 { return 1000000 + rand.Int64()&0xFFFFF })
	accumulatedRewards = make(map[string]int64, 100)
	for i := 0; i < loopCount; i++ {
		electVotersNonDup(candidates.Validators, uint64(i), 33, 0)
		accumulateAndResetReward(candidates.Validators, accumulatedRewards)
	}
	maxRewardPerStakingDiff = float64(0)
	for i := 0; i < 99; i++ {
		rewardPerStakingDiff :=
			math.Abs(float64(accumulatedRewards[candidates.Validators[i].Address.String()])/
				float64(candidates.Validators[i].StakingPower)/float64(loopCount) - 1)
		if maxRewardPerStakingDiff < rewardPerStakingDiff {
			maxRewardPerStakingDiff = rewardPerStakingDiff
		}
	}
	t.Logf("[! condition 4] max reward per staking difference: %f", maxRewardPerStakingDiff)
}

func newValidatorSet(length int, prio func(int) int64) *ValidatorSet {
	validators := make([]*Validator, length)
	totalStakingPower := int64(0)
	for i := 0; i < length; i++ {
		stakingPower := prio(i)
		validators[i] = &Validator{
			Address:      crypto.AddressHash([]byte(strconv.Itoa(i))),
			StakingPower: stakingPower,
			VotingPower:  0,
		}
		totalStakingPower += stakingPower
	}

	return &ValidatorSet{
		Validators:        validators,
		totalStakingPower: totalStakingPower,
	}
}

func sameVoters(c1 []*Validator, c2 []*Validator) bool {
	if len(c1) != len(c2) {
		return false
	}
	sort.Slice(c1, func(i, j int) bool {
		return bytes.Compare(c1[i].Address.Bytes(), c1[j].Address.Bytes()) == -1
	})
	sort.Slice(c2, func(i, j int) bool {
		return bytes.Compare(c2[i].Address.Bytes(), c2[j].Address.Bytes()) == -1
	})
	for i := 0; i < len(c1); i++ {
		if bytes.Compare(c1[i].Address.Bytes(), c2[i].Address.Bytes()) == 1 {
			return false
		}
		if c1[i].StakingPower != c2[i].StakingPower {
			return false
		}
		if c1[i].VotingPower != c2[i].VotingPower {
			return false
		}
	}
	return true
}

func TestMyMy(t *testing.T) {
	a := new(big.Int).Mul(new(big.Int).Div(big.NewInt(100000000000), big.NewInt(1000000000000)), big.NewInt(1000000000000))
	b := new(big.Int).Div(new(big.Int).Mul(big.NewInt(100000000000), big.NewInt(1000000000000)), big.NewInt(1000000000000))
	t.Logf("a=%v, b=%v", a, b)
}

func TestElectVotersNonDup(t *testing.T) {
	for n := 100; n <= 1000; n += 100 {
		rand.Seed(int64(n))
		validators := newValidatorSet(n, func(i int) int64 {
			return rand.Int63n(100) + 1
		})

		winners := electVotersNonDup(validators.Validators, 0, 30, 0)

		if !isByzantineTolerable(winners, 30) {
			for i, v := range winners {
				fmt.Printf("%d: voting power: %d, staking power: %d\n", i, v.VotingPower, v.StakingPower)
			}
			break
		}
		assert.True(t, isByzantineTolerable(winners, 30))
	}
}

func TestElectVotersNonDupStaticVotingPower(t *testing.T) {
	candidates := newValidatorSet(5, func(i int) int64 { return 10 })
	expectedVotingPower := []int64{
		13,
		11,
		10,
		8,
		5,
	}

	byzantinePercent := 10
	voters := electVotersNonDup(candidates.Validators, 0, byzantinePercent, 0)
	assert.True(t, isByzantineTolerable(voters, byzantinePercent))

	for i, voter := range voters {
		assert.True(t, expectedVotingPower[i] == voter.VotingPower)
	}

}

func TestElectVoter(t *testing.T) {
	validators := newValidatorSet(10, func(i int) int64 { return int64(i + 1) })
	total := int64(0)
	for _, val := range validators.Validators {
		total += val.StakingPower
	}
	seed := uint64(0)

	candidates := validators.Validators

	//if fail to voting, panic
	for i := range validators.Validators {
		idx, winner := electVoter(&seed, candidates, i, total)
		total -= winner.StakingPower
		moveWinnerToLast(candidates, idx)
	}
}

func TestElectVotersNonDupWithDifferentSeed(t *testing.T) {
	validators := newValidatorSet(100, func(i int) int64 {
		return rand.Int63n(1000) + 1
	})

	voters := electVotersNonDup(validators.Validators, 0, 25, 0)
	for n := int64(1); n <= 100; n++ {
		rand.Seed(n)
		seed := rand.Int63n(100000) + 1
		otherVoters := electVotersNonDup(validators.Validators, uint64(seed), 25, 0)

		assert.False(t, sameVoters(voters, otherVoters))
	}
}

func TestElectVotersNonDupValidatorsNotSorting(t *testing.T) {
	validators := newValidatorSet(1000, func(i int) int64 {
		return int64(i + 1)
	})

	shuffled := validators.Copy()
	for i := range shuffled.Validators {
		r := rand.Intn(len(shuffled.Validators))
		shuffled.Validators[i], shuffled.Validators[r] = shuffled.Validators[r], shuffled.Validators[i]
	}

	winners := electVotersNonDup(validators.Validators, 0, 30, 0)
	shuffledWinners := electVotersNonDup(shuffled.Validators, 0, 30, 0)

	assert.True(t, sameVoters(winners, shuffledWinners))
}

func TestElectVotersNonDupVotingPower(t *testing.T) {
	validators := newValidatorSet(100, func(i int) int64 {
		return 1000
	})

	winners := electVotersNonDup(validators.Validators, 0, 25, 0)

	winPoints := make([]*big.Int, 0)
	total := int64(0)
	totalWinPoint := new(big.Int)
	for n := 0; n < len(winners); n++ {
		for i := range winPoints {
			winPoint := big.NewInt(validators.totalStakingPower - total + 1000)
			winPoint.Div(big.NewInt(1000*precisionForSelection), winPoint)
			totalWinPoint.Add(totalWinPoint, winPoint)
			winPoints[i] = new(big.Int).Add(winPoints[i], winPoint)
		}
		winPoints = append(winPoints, big.NewInt(1000))
		totalWinPoint.Add(totalWinPoint, big.NewInt(1000))
		total += 1000
	}

	for i, w := range winners {
		winPoint := new(big.Int).Mul(winPoints[i], big.NewInt(precisionForSelection))
		votingPower := new(big.Int).Mul(winPoint, big.NewInt(validators.totalStakingPower))
		votingPower.Div(votingPower, totalWinPoint)
		votingPower.Div(votingPower, big.NewInt(precisionCorrectionForSelection))

		assert.True(t, w.VotingPower == votingPower.Int64())
	}
}

func TestElectVotersNonDupWithOverflow(t *testing.T) {
	expectedPanic := "Total staking power should be guarded to not exceed"
	validators := newValidatorSet(101, func(i int) int64 {
		return math.MaxInt64 / 100
	})

	defer func() {
		pnc := recover()
		if pncStr, ok := pnc.(string); ok {
			assert.True(t, strings.HasPrefix(pncStr, expectedPanic))
		} else {
			t.Fatal("panic expected, but doesn't panic")
		}
	}()
	validators.updateTotalStakingPower() // it will be panic
	// electVotersNonDup does not call updateTotalStakingPower() any more
	electVotersNonDup(validators.Validators, 0, 30, 0)
}

func TestElectVotersNonDupDistribution(t *testing.T) {
	validators := newValidatorSet(100, func(i int) int64 {
		return 1000
	})
	scores := make(map[string]int)
	for i := 0; i < 100000; i++ {
		//hash is distributed well
		hash := merkle.HashFromByteSlices([][]byte{
			[]byte(strconv.Itoa(i)),
		})
		seed := hashToSeed(hash)
		winners := electVotersNonDup(validators.Validators, seed, 1, 0)
		scores[winners[0].Address.String()]++
	}

	for _, v := range scores {
		assert.True(t, v >= 900 && v <= 1100)
	}
}

func TestElectVoterPanic(t *testing.T) {

	validators := newValidatorSet(10, func(i int) int64 { return int64(i + 1) })
	total := int64(0)
	for _, val := range validators.Validators {
		total += val.StakingPower
	}
	seed := uint64(0)

	candidates := validators.Validators

	//vote when there is no candidates
	expectedResult := "Cannot find random sample."
	defer func() {
		pnc := recover()
		if pncStr, ok := pnc.(string); ok {
			assert.True(t, strings.HasPrefix(pncStr, expectedResult))
		} else {
			t.Fatal("panic expected, but doesn't panic")
		}

	}()
	for i := 0; i < 11; i++ {
		idx, winner := electVoter(&seed, candidates, i, total)
		total -= winner.StakingPower
		moveWinnerToLast(candidates, idx)
	}
}

func newVotersWithRandomVotingPowerDescending(seed, max, numerator, stakingPower int64) []*voter {
	voters := make([]*voter, 0)

	// random voters descending
	random := int64(0)
	rand.Seed(seed)
	for votingPower := max; votingPower > 0; votingPower -= random {
		random = rand.Int63n(max/numerator) + 1
		voters = append(voters, &voter{
			val: &Validator{
				StakingPower: stakingPower,
				VotingPower:  votingPower,
			},
		})
	}
	return voters
}

func TestSortVoters(t *testing.T) {
	for n := int64(0); n < 100; n++ {

		// random voters descending
		voters := newVotersWithRandomVotingPowerDescending(n, 100000, 100, 10)

		//shuffle the voters
		shuffled := make([]*voter, len(voters))
		copy(shuffled, voters)
		for i := range shuffled {
			target := rand.Intn(len(shuffled) - 1)
			shuffled[i], shuffled[target] = shuffled[target], shuffled[i]
		}

		sortVoters(shuffled)
		for i := range shuffled {
			assert.True(t, shuffled[i].val.VotingPower == voters[i].val.VotingPower)
			if i > 0 {
				assert.True(t, shuffled[i-1].val.VotingPower >= voters[i].val.VotingPower)
			}
		}
	}
}

func TestSortVotersWithSameValue(t *testing.T) {
	for n := 0; n < 100; n++ {

		voters := make([]*voter, 0)

		// random voters descending
		random := int64(0)
		rand.Seed(int64(n))
		n := 0
		for votingPower := int64(100000); votingPower > 0; votingPower -= random {
			random = rand.Int63n(100000/100) + 1
			voters = append(voters, &voter{
				val: &Validator{
					StakingPower: 10,
					VotingPower:  votingPower,
					Address:      []byte(strconv.Itoa(n)),
				},
			})
			voters = append(voters, &voter{
				val: &Validator{
					StakingPower: 10,
					VotingPower:  votingPower,
					Address:      []byte(strconv.Itoa(n + 1)),
				},
			})
			n += 2
		}

		//shuffle the voters
		shuffled := make([]*voter, len(voters))
		copy(shuffled, voters)
		for i := range shuffled {
			target := rand.Intn(len(shuffled) - 1)
			shuffled[i], shuffled[target] = shuffled[target], shuffled[i]
		}

		sortVoters(shuffled)
		for i := range shuffled {
			a := shuffled[i].val
			b := voters[i].val
			assert.True(t, bytes.Equal(a.Address, b.Address))
			assert.True(t, a.VotingPower == b.VotingPower)
		}
	}
}

func TestGetTolerableByzantinePower(t *testing.T) {
	assert.True(t, getTolerableByzantinePower(100, 20) == 20)
	assert.True(t, getTolerableByzantinePower(101, 20) == 21)
	assert.True(t, getTolerableByzantinePower(102, 20) == 21)
	assert.True(t, getTolerableByzantinePower(103, 20) == 21)
	assert.True(t, getTolerableByzantinePower(104, 20) == 21)
	assert.True(t, getTolerableByzantinePower(105, 20) == 21)
	assert.True(t, getTolerableByzantinePower(106, 20) == 22)
	assert.True(t, getTolerableByzantinePower(120, 20) == 24)
	assert.True(t, getTolerableByzantinePower(100000, 20) == 20000)

	assert.True(t, getTolerableByzantinePower(math.MaxInt64, 10) == math.MaxInt64/10+1)
	assert.True(t, getTolerableByzantinePower(math.MaxInt64, 50) == math.MaxInt64/2+1)
	assert.True(t, getTolerableByzantinePower(math.MaxInt64-1, 50) == (math.MaxInt64-1)/2)
}

func TestMoveWinnerToLast(t *testing.T) {
	validators := newValidatorSet(10, func(i int) int64 {
		return int64(i + 1)
	})

	target := validators.Validators[3]
	nextOfTarget := validators.Validators[4]
	moveWinnerToLast(validators.Validators, 3)
	assert.True(t, target == validators.Validators[9])
	assert.True(t, nextOfTarget == validators.Validators[3])

}
