package types

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/crypto/ed25519"
	tmmath "github.com/tendermint/tendermint/libs/math"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/stretchr/testify/assert"

	"github.com/tendermint/tendermint/crypto/vrf"
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
	valSet := randValidatorSet(30)
	valSet.Validators[0].StakingPower = 0

	zeroVals := countZeroStakingPower(valSet.Validators)
	genDoc := &GenesisDoc{
		GenesisTime: tmtime.Now(),
		ChainID:     "tendermint-test",
		VoterParams: &VoterParams{10, 20, 1},
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
	voterSet := SelectVoter(valSet, hash, &VoterParams{30, 1, 1})
	assert.True(t, voterSet.Size() == 30-zeroVals)
	voterSet = SelectVoter(valSet, nil, genDoc.VoterParams)
	assert.True(t, voterSet.Size() == 30-zeroVals)

	// test VoterElectionThreshold
	for i := 1; i < 100; i++ {
		voterSet := SelectVoter(valSet, hash, &VoterParams{15, i, 1})
		assert.True(t, voterSet.Size() >= 15)
	}
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
	valSet, privMap := randValidatorSetWithMinMax(30, 100, 100*int64(minMaxRate))
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
		hash, _ = vrf.ProofToHash(proof)
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
				valSet, _ := randValidatorSetWithMinMax(validators, 100, 100*int64(minMaxRate))
				voterSet := SelectVoter(valSet, []byte{byte(hash)}, &VoterParams{voters, 20, 5})
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

func TestCalVotersNum(t *testing.T) {
	total := int64(200)
	byzantine := 0.2
	accuracy := 0.99999
	selection := CalNumOfVoterToElect(total, byzantine, accuracy)
	assert.Equal(t, selection, int64(88))

	total = int64(100)
	selection = CalNumOfVoterToElect(total, byzantine, accuracy)
	assert.Equal(t, selection, int64(58))

	assert.Panics(t, func() { CalNumOfVoterToElect(total, 0.3, 10) })
	assert.Panics(t, func() { CalNumOfVoterToElect(total, 1.1, 0.9999) })
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

func byzantinesPower(voters []*Validator, byzantines map[string]bool) int64 {
	power := int64(0)
	for _, v := range voters {
		if byzantines[v.Address.String()] {
			power += v.VotingPower
		}
	}
	return power
}

func countByzantines(voters []*Validator, byzantines map[string]bool) int {
	count := 0
	for _, v := range voters {
		if byzantines[v.Address.String()] {
			count++
		}
	}
	return count
}

func electVotersForLoop(t *testing.T, hash []byte, valSet *ValidatorSet, privMap map[string]PrivValidator,
	byzantines map[string]bool, loopCount int, byzantinePercent, accuracy int) {
	byzantineFault := 0
	totalVoters := 0
	totalByzantines := 0
	for i := 0; i < loopCount; i++ {
		voterSet := SelectVoter(valSet, hash, &VoterParams{1, byzantinePercent, accuracy})
		byzantineThreshold := int64(float64(voterSet.TotalVotingPower())*0.33) + 1
		if byzantinesPower(voterSet.Voters, byzantines) >= byzantineThreshold {
			byzantineFault++
		}
		totalVoters += voterSet.Size()
		totalByzantines += countByzantines(voterSet.Voters, byzantines)
		proposer := valSet.SelectProposer(hash, int64(i), 0)
		message := MakeRoundHash(hash, int64(i), 0)
		proof, _ := privMap[proposer.Address.String()].GenerateVRFProof(message)
		hash, _ = vrf.ProofToHash(proof)
	}
	t.Logf("[accuracy=%f] voters=%d, fault=%d, avg byzantines=%f", accuracyFromElectionPrecision(accuracy),
		totalVoters/loopCount, byzantineFault, float64(totalByzantines)/float64(loopCount))
	assert.True(t, float64(byzantineFault) < float64(loopCount)*(1.0-accuracyFromElectionPrecision(accuracy)))
}

func TestCalVotersNum2(t *testing.T) {
	valSet, privMap := randValidatorSetWithMinMax(100, 100, 10000)
	byzantinePercent := 20
	byzantines := makeByzantine(valSet, float64(byzantinePercent)/100)
	genDoc := &GenesisDoc{
		GenesisTime: tmtime.Now(),
		ChainID:     "tendermint-test",
		Validators:  toGenesisValidators(valSet.Validators),
	}
	hash := genDoc.Hash()

	loopCount := 1000
	electVotersForLoop(t, hash, valSet, privMap, byzantines, loopCount, byzantinePercent, 1)
	electVotersForLoop(t, hash, valSet, privMap, byzantines, loopCount, byzantinePercent, 2)
	electVotersForLoop(t, hash, valSet, privMap, byzantines, loopCount, byzantinePercent, 3)
	electVotersForLoop(t, hash, valSet, privMap, byzantines, loopCount, byzantinePercent, 4)
	electVotersForLoop(t, hash, valSet, privMap, byzantines, loopCount, byzantinePercent, 5)
}

func TestAccuracyFromElectionPrecision(t *testing.T) {
	assert.True(t, accuracyFromElectionPrecision(2) == 0.99)
	assert.True(t, accuracyFromElectionPrecision(3) == 0.999)
	assert.True(t, accuracyFromElectionPrecision(4) == 0.9999)
	assert.True(t, accuracyFromElectionPrecision(5) == 0.99999)
	assert.True(t, accuracyFromElectionPrecision(6) == 0.999999)
	assert.True(t, accuracyFromElectionPrecision(7) == 0.9999999)
	assert.True(t, accuracyFromElectionPrecision(8) == 0.99999999)
	assert.True(t, accuracyFromElectionPrecision(9) == 0.999999999)
	assert.True(t, accuracyFromElectionPrecision(10) == 0.9999999999)
	assert.True(t, accuracyFromElectionPrecision(11) == 0.99999999999)
	assert.True(t, accuracyFromElectionPrecision(12) == 0.999999999999)
	assert.True(t, accuracyFromElectionPrecision(13) == 0.9999999999999)
	assert.True(t, accuracyFromElectionPrecision(14) == 0.99999999999999)
	assert.True(t, accuracyFromElectionPrecision(15) == 0.999999999999999)
}
