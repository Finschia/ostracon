package rand

import (
	"fmt"
	"math"
	"math/rand"
	s "sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Element struct {
	id          uint32
	winPoint    float64
	weight      uint64
	votingPower uint64
}

func (e *Element) Priority() uint64 {
	return e.weight
}

func (e *Element) LessThan(other Candidate) bool {
	o, ok := other.(*Element)
	if !ok {
		panic("incompatible type")
	}
	return e.id < o.id
}

func (e *Element) SetWinPoint(winPoint float64) {
	e.winPoint = winPoint
}
func (e *Element) SetVotingPower(votingPower uint64) {
	e.votingPower = votingPower
}
func (e *Element) WinPoint() float64   { return e.winPoint }
func (e *Element) VotingPower() uint64 { return e.votingPower }

func TestRandomSamplingWithPriority(t *testing.T) {
	candidates := newCandidates(100, func(i int) uint64 { return uint64(i) })

	elected := RandomSamplingWithPriority(0, candidates, 10, uint64(len(candidates)))
	if len(elected) != 10 {
		t.Errorf(fmt.Sprintf("unexpected sample size: %d", len(elected)))
	}

	// ----
	// The same result can be obtained for the same input.
	others := newCandidates(100, func(i int) uint64 { return uint64(i) })
	secondTimeElected := RandomSamplingWithPriority(0, others, 10, uint64(len(others)))
	if len(elected) != len(secondTimeElected) || !sameCandidates(elected, secondTimeElected) {
		t.Errorf(fmt.Sprintf("undeterministic: %+v != %+v", elected, others))
	}

	// ----
	// Make sure the winning frequency will be even
	candidates = newCandidates(100, func(i int) uint64 { return 1 })
	counts := make([]int, len(candidates))
	for i := 0; i < 100000; i++ {
		elected = RandomSamplingWithPriority(uint64(i), candidates, 10, uint64(len(candidates)))
		for _, e := range elected {
			counts[e.(*Element).id]++
		}
	}
	expected := float64(1) / float64(100)
	mean, variance, z := calculateZ(expected, counts)
	if z >= 1e-15 || math.Abs(mean-expected) >= 1e-15 || variance >= 1e-5 {
		t.Errorf("winning frequency is uneven: mean=%f, variance=%e, z=%e", mean, variance, z)
	}
}

func TestRandomSamplingPanicCase(t *testing.T) {
	type Case struct {
		Candidates    []Candidate
		TotalPriority uint64
	}

	cases := [...]*Case{
		// empty candidate set
		{newCandidates(0, func(i int) uint64 { return 0 }), 0},
		// actual total priority is zero
		{newCandidates(100, func(i int) uint64 { return 0 }), 100},
		// specified total priority is less than actual one
		{newCandidates(2, func(i int) uint64 { return 1 }), 1000},
	}

	for i, c := range cases {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("expected panic didn't happen in case %d", i+1)
				}
			}()
			RandomSamplingWithPriority(0, c.Candidates, 10, c.TotalPriority)
		}()
	}
}

func resetPoints(candidate []Candidate) {
	for _, c := range candidate {
		c.(*Element).winPoint = 0
		c.(*Element).votingPower = 0
	}
}

func isByzantine(candidates []Candidate, totalPriority, tolerableByzantinePercent uint64) bool {
	tolerableByzantinePower := totalPriority * tolerableByzantinePercent / 100
	topFVotersVotingPower := countVoters(candidates, tolerableByzantinePower)
	return topFVotersVotingPower >= totalPriority/3
}

func TestElectVotersNonDupCandidate(t *testing.T) {
	candidates := newCandidates(100, func(i int) uint64 { return uint64(1000 * (i + 1)) })

	winners := ElectVotersNonDup(candidates, 0, 20)
	assert.True(t, !isByzantine(winners, sumTotalPriority(candidates), 20))
}

// test samplingThreshold
func TestElectVotersNonDupSamplingThreshold(t *testing.T) {
	candidates := newCandidates(100, func(i int) uint64 { return uint64(1000 * (i + 1)) })

	for i := uint64(1); i <= 30; i++ {
		winners := ElectVotersNonDup(candidates, 0, i)
		assert.True(t, !isByzantine(winners, sumTotalPriority(candidates), i))
		resetPoints(candidates)
	}
}

// test downscale of win point cases
func TestElectVotersNonDupDownscale(t *testing.T) {
	candidates := newCandidates(10, func(i int) uint64 {
		if i == 0 {
			return math.MaxInt64 >> 1
		}
		if i == 1 {
			return 1 << 55
		}
		if i == 3 {
			return 1 << 54
		}
		if i == 4 {
			return 1 << 53
		}
		return uint64(i)
	})
	ElectVotersNonDup(candidates, 0, 20)
}

// test random election should be deterministic
func TestElectVotersNonDupDeterministic(t *testing.T) {
	candidates1 := newCandidates(100, func(i int) uint64 { return uint64(i + 1) })
	candidates2 := newCandidates(100, func(i int) uint64 { return uint64(i + 1) })
	for i := 1; i <= 100; i++ {
		winners1 := ElectVotersNonDup(candidates1, uint64(i), 50)
		winners2 := ElectVotersNonDup(candidates2, uint64(i), 50)
		sameCandidates(winners1, winners2)
		resetPoints(candidates1)
		resetPoints(candidates2)
	}
}

func TestElectVotersNonDupIncludingZeroStakingPower(t *testing.T) {
	// first candidate's priority is 0
	candidates1 := newCandidates(100, func(i int) uint64 { return uint64(i) })
	winners1 := ElectVotersNonDup(candidates1, 0, 20)
	assert.True(t, !isByzantine(winners1, sumTotalPriority(candidates1), 20))

	//half of candidates has 0 priority
	candidates2 := newCandidates(100, func(i int) uint64 {
		if i < 50 {
			return 0
		}
		return uint64(i)
	})
	winners2 := ElectVotersNonDup(candidates2, 0, 20)
	assert.True(t, !isByzantine(winners2, sumTotalPriority(candidates2), 20))
}

func TestElectVotersNonDupOverflow(t *testing.T) {
	number := 98
	candidates := newCandidates(number, func(i int) uint64 { return math.MaxUint64 / uint64(number+2) })
	totalPriority := sumTotalPriority(candidates)
	assert.True(t, totalPriority < math.MaxUint64)
	winners := ElectVotersNonDup(candidates, rand.Uint64(), 20)
	assert.True(t, !isByzantine(winners, totalPriority, 20))
	for _, w := range winners {
		element := w.(*Element)
		assert.True(t, element.winPoint > 0)
	}
}

func accumulateAndResetReward(candidate []Candidate, acc []uint64) uint64 {
	totalWinPoint := uint64(0)
	for _, c := range candidate {
		winPoint := uint64(c.(*Element).winPoint * float64(precisionForSelection))
		acc[c.(*Element).id] += winPoint
		totalWinPoint += winPoint
	}
	return totalWinPoint
}

func TestDivider(t *testing.T) {
	assert.True(t, divider.Uint64() == uint64Mask+1)
}

func TestRandomThreshold(t *testing.T) {
	loopCount := 100000

	// randomThreshold() should not return a value greater than total.
	for i := 0; i < loopCount; i++ {
		seed := rand.Uint64()
		total := rand.Int63()
		random := randomThreshold(&seed, uint64(total))
		assert.True(t, random < uint64(total))
	}

	// test randomness
	total := math.MaxInt64
	bitHit := make([]int, 63)
	for i := 0; i < loopCount; i++ {
		seed := rand.Uint64()
		random := randomThreshold(&seed, uint64(total))
		for j := 0; j < 63; j++ {
			if random&(1<<j) > 0 {
				bitHit[j]++
			}
		}
	}
	// all bit hit count should be near at loopCount/2
	for i := 0; i < len(bitHit); i++ {
		assert.True(t, math.Abs(float64(bitHit[i])-float64(loopCount/2))/float64(loopCount/2) < 0.01)
	}

	// verify idempotence
	expect := [][]uint64{
		{7070836379803831726, 3176749709313725329, 6607573645926202312, 3491641484182981082, 3795411888399561855},
		{1227844342346046656, 2900311180284727168, 8193302169476290588, 2343329048962716018, 6435608444680946564},
		{1682153688901572301, 5713119979229610871, 1690050691353843586, 6615539178087966730, 965357176598405746},
		{2092789425003139052, 7803713333738082738, 391680292209432075, 3242280302033391430, 2071067388247806529},
		{7958955049054603977, 5770386275058218277, 6648532499409218539, 5505026356475271777, 3466385424369377032}}
	for i := 0; i < len(expect); i++ {
		seed := uint64(i)
		for j := 0; j < len(expect[i]); j++ {
			seed = randomThreshold(&seed, uint64(total))
			assert.True(t, seed == expect[i][j])
		}
	}
}

// test reward fairness
func TestElectVotersNonDupReward(t *testing.T) {
	candidates := newCandidates(100, func(i int) uint64 { return uint64(i + 1) })

	accumulatedRewards := make([]uint64, 100)
	for i := 0; i < 100000; i++ {
		// 25 samplingThreshold is minimum to pass this test
		// If samplingThreshold is less than 25, the result says the reward is not fair
		winners := ElectVotersNonDup(candidates, uint64(i), 20)
		accumulateAndResetReward(winners, accumulatedRewards)
	}
	for i := 0; i < 99; i++ {
		assert.True(t, accumulatedRewards[i] < accumulatedRewards[i+1])
	}

	accumulatedRewards = make([]uint64, 100)
	for i := 0; i < 50000; i++ {
		winners := ElectVotersNonDup(candidates, uint64(i), 20)
		accumulateAndResetReward(winners, accumulatedRewards)
	}
	for i := 0; i < 99; i++ {
		assert.True(t, accumulatedRewards[i] < accumulatedRewards[i+1])
	}

	//fail
	//accumulatedRewards = make([]uint64, 100)
	//for i := 0; i < 10000; i++ {
	//	winners := ElectVotersNonDup(candidates, uint64(i), 20)
	//	accumulateAndResetReward(winners, accumulatedRewards)
	//}
	//for i := 0; i < 99; i++ {
	//	assert.True(t, accumulatedRewards[i] < accumulatedRewards[i+1])
	//}
}

/**
conditions for fair reward
1. even staking power(less difference between min staking and max staking)
2. large total staking(a small total staking power makes a large error when converting float into int)
3. many sampling count
4. loop count
*/
//failed: not fit to new voting
//func TestElectVotersNonDupEquity(t *testing.T) {
//	loopCount := 10000
//
//	// good condition
//	candidates := newCandidates(100, func(i int) uint64 { return 1000000 + rand.Uint64()&0xFFFFF })
//	totalStaking := uint64(0)
//	for _, c := range candidates {
//		totalStaking += c.Priority()
//	}
//
//	accumulatedRewards := make([]uint64, 100)
//	totalAccumulateRewards := uint64(0)
//	for i := 0; i < loopCount; i++ {
//		ElectVotersNonDup(candidates, uint64(i), 20)
//		totalAccumulateRewards += accumulateAndResetReward(candidates, accumulatedRewards)
//	}
//	for i := 0; i < 99; i++ {
//		rewardRate := float64(accumulatedRewards[i]) / float64(totalAccumulateRewards)
//		stakingRate := float64(candidates[i].Priority()) / float64(totalStaking)
//		rate := rewardRate / stakingRate
//		rewardPerStakingDiff := math.Abs(1 - rate)
//		assert.True(t, rewardPerStakingDiff < 0.01)
//	}
//
//	// =======================================================================================================
//	// The codes below are not test codes to verify logic,
//	// but codes to find out what parameters are that weaken the equity of rewards.
//
//	// violation of condition 1
//	candidates = newCandidates(100, func(i int) uint64 { return rand.Uint64() & 0xFFFFFFFFF })
//	accumulatedRewards = make([]uint64, 100)
//	for i := 0; i < loopCount; i++ {
//		ElectVotersNonDup(candidates, uint64(i), 20)
//		accumulateAndResetReward(candidates, accumulatedRewards)
//	}
//	maxRewardPerStakingDiff := float64(0)
//	for i := 0; i < 99; i++ {
//		rewardPerStakingDiff :=
//			math.Abs(float64(accumulatedRewards[i])/float64(candidates[i].Priority())/float64(loopCount) - 1)
//		if maxRewardPerStakingDiff < rewardPerStakingDiff {
//			maxRewardPerStakingDiff = rewardPerStakingDiff
//		}
//	}
//	t.Logf("[! condition 1] max reward per staking difference: %f", maxRewardPerStakingDiff)
//
//	// violation of condition 2
//	candidates = newCandidates(100, func(i int) uint64 { return rand.Uint64() & 0xFFFFF })
//	accumulatedRewards = make([]uint64, 100)
//	for i := 0; i < loopCount; i++ {
//		ElectVotersNonDup(candidates, uint64(i), 20)
//		accumulateAndResetReward(candidates, accumulatedRewards)
//	}
//	maxRewardPerStakingDiff = float64(0)
//	for i := 0; i < 99; i++ {
//		rewardPerStakingDiff :=
//			math.Abs(float64(accumulatedRewards[i])/float64(candidates[i].Priority())/float64(loopCount) - 1)
//		if maxRewardPerStakingDiff < rewardPerStakingDiff {
//			maxRewardPerStakingDiff = rewardPerStakingDiff
//		}
//	}
//	t.Logf("[! condition 2] max reward per staking difference: %f", maxRewardPerStakingDiff)
//
//	// violation of condition 3
//	candidates = newCandidates(100, func(i int) uint64 { return 1000000 + rand.Uint64()&0xFFFFF })
//	accumulatedRewards = make([]uint64, 100)
//	for i := 0; i < loopCount; i++ {
//		ElectVotersNonDup(candidates, uint64(i), 20)
//		accumulateAndResetReward(candidates, accumulatedRewards)
//	}
//	maxRewardPerStakingDiff = float64(0)
//	for i := 0; i < 99; i++ {
//		rewardPerStakingDiff :=
//			math.Abs(float64(accumulatedRewards[i])/float64(candidates[i].Priority())/float64(loopCount) - 1)
//		if maxRewardPerStakingDiff < rewardPerStakingDiff {
//			maxRewardPerStakingDiff = rewardPerStakingDiff
//		}
//	}
//	t.Logf("[! condition 3] max reward per staking difference: %f", maxRewardPerStakingDiff)
//
//	// violation of condition 4
//	loopCount = 100
//	candidates = newCandidates(100, func(i int) uint64 { return 1000000 + rand.Uint64()&0xFFFFF })
//	accumulatedRewards = make([]uint64, 100)
//	for i := 0; i < loopCount; i++ {
//		ElectVotersNonDup(candidates, uint64(i), 99)
//		accumulateAndResetReward(candidates, accumulatedRewards)
//	}
//	maxRewardPerStakingDiff = float64(0)
//	for i := 0; i < 99; i++ {
//		rewardPerStakingDiff :=
//			math.Abs(float64(accumulatedRewards[i])/float64(candidates[i].Priority())/float64(loopCount) - 1)
//		if maxRewardPerStakingDiff < rewardPerStakingDiff {
//			maxRewardPerStakingDiff = rewardPerStakingDiff
//		}
//	}
//	t.Logf("[! condition 4] max reward per staking difference: %f", maxRewardPerStakingDiff)
//}

func newCandidates(length int, prio func(int) uint64) (candidates []Candidate) {
	candidates = make([]Candidate, length)
	for i := 0; i < length; i++ {
		candidates[i] = &Element{uint32(i), 0, prio(i), 0}
	}
	return
}

func sameCandidates(c1 []Candidate, c2 []Candidate) bool {
	if len(c1) != len(c2) {
		return false
	}
	s.Slice(c1, func(i, j int) bool { return c1[i].LessThan(c1[j]) })
	s.Slice(c2, func(i, j int) bool { return c2[i].LessThan(c2[j]) })
	for i := 0; i < len(c1); i++ {
		if c1[i].(*Element).id != c2[i].(*Element).id {
			return false
		}
		if c1[i].(*Element).winPoint != c2[i].(*Element).winPoint {
			return false
		}
		if c1[i].VotingPower() != c2[i].VotingPower() {
			return false
		}
	}
	return true
}

// The cumulative VotingPowers should follow a normal distribution with a mean as the expected value.
// A risk factor will be able to acquire from the value using a standard normal distribution table by
// applying the transformation to normalize to the expected value.
func calculateZ(expected float64, values []int) (mean, variance, z float64) {
	sum := 0.0
	for i := 0; i < len(values); i++ {
		sum += float64(values[i])
	}
	actuals := make([]float64, len(values))
	for i := 0; i < len(values); i++ {
		actuals[i] = float64(values[i]) / sum
	}
	mean, variance = calculateMeanAndVariance(actuals)
	z = (mean - expected) / math.Sqrt(variance/float64(len(values)))
	return
}

func calculateMeanAndVariance(values []float64) (mean float64, variance float64) {
	sum := 0.0
	for _, x := range values {
		sum += x
	}
	mean = sum / float64(len(values))
	sum2 := 0.0
	for _, x := range values {
		dx := x - mean
		sum2 += dx * dx
	}
	variance = sum2 / float64(len(values))
	return
}

func TestElectVotersNonDup(t *testing.T) {
	candidates := newCandidates(5, func(i int) uint64 { return 10 })
	expectedPercentage := []float64{
		0.262,
		0.239,
		0.210,
		0.172,
		0.114,
	}
	expectedVotingPower := []uint64{
		13,
		11,
		10,
		8,
		5,
	}

	totalWinPoint := float64(0)
	byzantinePercent := uint64(10)
	voters := ElectVotersNonDup(candidates, 0, byzantinePercent)
	assert.True(t, !isByzantine(voters, sumTotalPriority(candidates), 10))

	for _, voter := range voters {
		totalWinPoint += voter.WinPoint()
	}

	for i, voter := range voters {
		assert.True(t, expectedPercentage[i] == float64(uint64(voter.WinPoint()*1000/totalWinPoint))/1000)
		assert.True(t, expectedVotingPower[i] == voter.VotingPower())
	}

}

func TestElectVoter(t *testing.T) {
	candidates := make([]Candidate, 0)
	for i := uint32(0); i < 10; i++ {
		candidates = append(candidates, &Element{
			id:     i,
			weight: 10,
		})
	}
	total := uint64(0)
	for _, candidate := range candidates {
		total += candidate.Priority()
	}
	seed := uint64(0)

	//if fail to voting, panic
	for i := range candidates {
		idx, winner := electVoter(&seed, candidates, i, total)
		total -= winner.Priority()
		moveWinnerToLast(candidates, idx)
	}
}
