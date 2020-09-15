package types

import (
	"math"
	"testing"

	"github.com/tendermint/tendermint/libs/rand"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tmtime "github.com/tendermint/tendermint/types/time"
)

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
		assert.True(t, areSame(valSet, voters))
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
				valSet, _ := randValidatorSetWithMinMax(validators, 100, 100*int64(minMaxRate))
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

func TestCalNumOfVoterToElect(t *testing.T) {
	// result of CalNumOfVoterToElect(1, 0.2, 0.99999) ~ CalNumOfVoterToElect(260, 0.2, 0.99999)
	result := []int64{1, 1, 1, 1, 4, 4, 4, 4, 4, 7, 7, 7, 7, 7, 10, 10, 10, 10, 10, 13,
		13, 13, 13, 13, 16, 16, 16, 16, 16, 19, 19, 19, 19, 19, 22, 22, 22, 22, 22, 25,
		25, 25, 25, 25, 28, 28, 28, 28, 28, 31, 31, 31, 31, 31, 34, 34, 34, 34, 34, 37,
		37, 37, 37, 37, 40, 40, 40, 40, 40, 43, 43, 43, 43, 43, 46, 46, 46, 46, 46, 49,
		49, 49, 49, 49, 52, 52, 52, 52, 49, 55, 52, 52, 52, 52, 55, 55, 55, 55, 55, 58,
		58, 58, 58, 58, 61, 61, 58, 58, 58, 61, 61, 61, 61, 61, 64, 64, 64, 64, 61, 67,
		67, 64, 64, 64, 67, 67, 67, 67, 67, 70, 70, 70, 67, 67, 70, 70, 70, 70, 70, 73,
		73, 73, 70, 70, 73, 73, 73, 73, 73, 76, 76, 76, 76, 73, 79, 76, 76, 76, 76, 79,
		79, 79, 76, 76, 79, 79, 79, 79, 79, 82, 82, 82, 79, 79, 82, 82, 82, 82, 82, 85,
		85, 82, 82, 82, 85, 85, 85, 85, 85, 88, 88, 85, 85, 85, 88, 88, 88, 88, 85, 88,
		88, 88, 88, 88, 91, 91, 88, 88, 88, 91, 91, 91, 91, 88, 94, 91, 91, 91, 91, 94,
		94, 94, 91, 91, 94, 94, 94, 94, 94, 97, 94, 94, 94, 94, 97, 97, 97, 94, 94, 97,
		97, 97, 97, 97, 100, 97, 97, 97, 97, 100, 100, 100, 97, 97, 100, 100, 100, 100, 97, 103}

	for i := 1; i <= len(result); i++ {
		assert.True(t, CalNumOfVoterToElect(int64(i), 0.2, 0.99999) == result[i-1])
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
	byzantines map[string]bool, loopCount int, byzantinePercent, accuracy int32) {
	byzantineFault := 0
	totalVoters := 0
	totalByzantines := 0
	for i := 0; i < loopCount; i++ {
		voterSet := SelectVoter(valSet, hash, &VoterParams{1, byzantinePercent})
		byzantineThreshold := int64(float64(voterSet.TotalVotingPower())*0.33) + 1
		if byzantinesPower(voterSet.Voters, byzantines) >= byzantineThreshold {
			byzantineFault++
		}
		totalVoters += voterSet.Size()
		totalByzantines += countByzantines(voterSet.Voters, byzantines)
		proposer := valSet.SelectProposer(hash, int64(i), 0)
		message := MakeRoundHash(hash, int64(i), 0)
		proof, _ := privMap[proposer.Address.String()].GenerateVRFProof(message)
		pubKey, _ := privMap[proposer.Address.String()].GetPubKey()
		hash, _ = pubKey.VRFVerify(proof, message)
	}
	t.Logf("voters=%d, fault=%d, avg byzantines=%f",
		totalVoters/loopCount, byzantineFault, float64(totalByzantines)/float64(loopCount))
	assert.True(t, float64(byzantineFault) < float64(loopCount))
}

func TestCalVotersNum2(t *testing.T) {
	valSet, privMap := randValidatorSetWithMinMax(100, 100, 10000)
	byzantinePercent := int32(20)
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
		{"false nil", nil, false, false},
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
	testVotingPower(t, randValidatorSet(100))
	vals := make([]*Validator, 100)
	for i := 0; i < len(vals); i++ {
		vals[i] = newValidator(rand.Bytes(32), 100)
	}
	testVotingPower(t, NewValidatorSet(vals))
	vals2 := make([]*Validator, 100)
	for i := 0; i < len(vals2); i++ {
		vals2[i] = newValidator(rand.Bytes(32), MaxTotalStakingPower/100)
	}
	testVotingPower(t, NewValidatorSet(vals2))
}
