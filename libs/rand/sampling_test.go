package rand

import (
	"fmt"
	"math"
	s "sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Element struct {
	id       uint32
	winPoint int64
	weight   uint64
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

func (e *Element) SetWinPoint(winPoint int64) {
	e.winPoint += winPoint
}

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

func resetWinPoint(candidate []Candidate) {
	for _, c := range candidate {
		c.(*Element).winPoint = 0
	}
}

func TestRandomSamplingWithoutReplacement1Candidate(t *testing.T) {
	candidates := newCandidates(1, func(i int) uint64 { return uint64(1000 * (i + 1)) })

	winners := RandomSamplingWithoutReplacement(0, candidates, 1, 1000)
	assert.True(t, len(winners) == 1)
	assert.True(t, candidates[0] == winners[0])
	assert.True(t, winners[0].(*Element).winPoint == 1000)
	resetWinPoint(candidates)

	winners2 := RandomSamplingWithoutReplacement(0, candidates, 0, 1000)
	assert.True(t, len(winners2) == 0)
	resetWinPoint(candidates)

	winners4 := RandomSamplingWithoutReplacement(0, candidates, 0, 1000)
	assert.True(t, len(winners4) == 0)
	resetWinPoint(candidates)
}

// test samplingThreshold
func TestRandomSamplingWithoutReplacementSamplingThreshold(t *testing.T) {
	candidates := newCandidates(100, func(i int) uint64 { return uint64(1000 * (i + 1)) })

	for i := 1; i <= 100; i++ {
		winners := RandomSamplingWithoutReplacement(0, candidates, i, 1000)
		assert.True(t, len(winners) == i)
		resetWinPoint(candidates)
	}
}

// test random election should be deterministic
func TestRandomSamplingWithoutReplacementDeterministic(t *testing.T) {
	candidates1 := newCandidates(100, func(i int) uint64 { return uint64(i + 1) })
	candidates2 := newCandidates(100, func(i int) uint64 { return uint64(i + 1) })
	for i := 1; i <= 100; i++ {
		winners1 := RandomSamplingWithoutReplacement(uint64(i), candidates1, 50, 1000)
		winners2 := RandomSamplingWithoutReplacement(uint64(i), candidates2, 50, 1000)
		sameCandidates(winners1, winners2)
		resetWinPoint(candidates1)
		resetWinPoint(candidates2)
	}
}

func TestRandomSamplingWithoutReplacementIncludingZeroStakingPower(t *testing.T) {
	// first candidate's priority is 0
	candidates1 := newCandidates(100, func(i int) uint64 { return uint64(i) })
	winners1 := RandomSamplingWithoutReplacement(0, candidates1, 100, 1000)
	assert.True(t, len(winners1) == 99)

	candidates2 := newCandidates(100, func(i int) uint64 {
		if i < 10 {
			return 0
		}
		return uint64(i)
	})
	winners2 := RandomSamplingWithoutReplacement(0, candidates2, 95, 1000)
	assert.True(t, len(winners2) == 90)
}

func accumulateAndResetReward(candidate []Candidate, acc []uint64) {
	for i, c := range candidate {
		acc[i] += uint64(c.(*Element).winPoint)
		c.(*Element).winPoint = 0
	}
}

// test reward fairness
func TestRandomSamplingWithoutReplacementReward(t *testing.T) {
	candidates := newCandidates(100, func(i int) uint64 { return uint64(i + 1) })

	accumulatedRewards := make([]uint64, 100)
	for i := 0; i < 100000; i++ {
		// 24 samplingThreshold is minimum to pass this test
		// If samplingThreshold is less than 24, the result says the reward is not fair
		RandomSamplingWithoutReplacement(uint64(i), candidates, 24, 10)
		accumulateAndResetReward(candidates, accumulatedRewards)
	}
	for i := 0; i < 99; i++ {
		assert.True(t, accumulatedRewards[i] < accumulatedRewards[i+1])
	}

	accumulatedRewards = make([]uint64, 100)
	for i := 0; i < 50000; i++ {
		RandomSamplingWithoutReplacement(uint64(i), candidates, 50, 10)
		accumulateAndResetReward(candidates, accumulatedRewards)
	}
	for i := 0; i < 99; i++ {
		assert.True(t, accumulatedRewards[i] < accumulatedRewards[i+1])
	}

	accumulatedRewards = make([]uint64, 100)
	for i := 0; i < 10000; i++ {
		RandomSamplingWithoutReplacement(uint64(i), candidates, 100, 10)
		accumulateAndResetReward(candidates, accumulatedRewards)
	}
	for i := 0; i < 99; i++ {
		assert.True(t, accumulatedRewards[i] < accumulatedRewards[i+1])
	}
}

func TestRandomSamplingWithoutReplacementPanic(t *testing.T) {
	type Case struct {
		Candidates        []Candidate
		SamplingThreshold int
	}

	cases := [...]*Case{
		// samplingThreshold is greater than the number of candidates
		{newCandidates(9, func(i int) uint64 { return 10 }), 10},
	}

	for i, c := range cases {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("expected panic didn't happen in case %d", i+1)
				}
			}()
			RandomSamplingWithoutReplacement(0, c.Candidates, c.SamplingThreshold, 1000)
		}()
	}
}

func newCandidates(length int, prio func(int) uint64) (candidates []Candidate) {
	candidates = make([]Candidate, length)
	for i := 0; i < length; i++ {
		candidates[i] = &Element{uint32(i), 0, prio(i)}
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
