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

func (e *Element) SetVotingPower(votingPower uint64) {
	e.votingPower = votingPower
}
func (e *Element) WinPoint() float64   { return e.winPoint }
func (e *Element) VotingPower() uint64 { return e.votingPower }

func TestRandomSamplingWithPriority(t *testing.T) {
	candidates := newCandidates(100, func(i int) uint64 { return uint64(i) })

	elected := RandomSamplingWithPriority(0, candidates, 10, calculateTotalPriority(candidates))
	if len(elected) != 10 {
		t.Errorf(fmt.Sprintf("unexpected sample size: %d", len(elected)))
	}

	// ----
	// The same result can be obtained for the same input.
	others := newCandidates(100, func(i int) uint64 { return uint64(i) })
	secondTimeElected := RandomSamplingWithPriority(0, others, 10, calculateTotalPriority(others))
	if len(elected) != len(secondTimeElected) || !sameCandidates(elected, secondTimeElected) {
		t.Errorf(fmt.Sprintf("undeterministic: %+v != %+v", elected, others))
	}

	// ----
	// Make sure the winning frequency will be even
	candidates = newCandidates(100, func(i int) uint64 { return 1 })
	counts := make([]int, len(candidates))
	for i := 0; i < 100000; i++ {
		elected = RandomSamplingWithPriority(uint64(i), candidates, 10, calculateTotalPriority(candidates))
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

func TestDivider(t *testing.T) {
	assert.True(t, divider.Uint64() == uint64Mask+1)
}

func TestRandomThreshold(t *testing.T) {
	loopCount := 100000

	// RandomThreshold() should not return a value greater than total.
	for i := 0; i < loopCount; i++ {
		seed := rand.Uint64()
		total := rand.Int63()
		random := RandomThreshold(&seed, uint64(total))
		assert.True(t, random < uint64(total))
	}

	// test randomness
	total := math.MaxInt64
	bitHit := make([]int, 63)
	for i := 0; i < loopCount; i++ {
		seed := rand.Uint64()
		random := RandomThreshold(&seed, uint64(total))
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
			seed = RandomThreshold(&seed, uint64(total))
			assert.True(t, seed == expect[i][j])
		}
	}
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

func newCandidates(length int, prio func(int) uint64) (candidates []Candidate) {
	candidates = make([]Candidate, length)
	for i := 0; i < length; i++ {
		candidates[i] = &Element{uint32(i), 0, prio(i), 0}
	}
	return
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

func calculateTotalPriority(candidates []Candidate) uint64 {
	totalPriority := uint64(0)
	for _, candidate := range candidates {
		totalPriority += candidate.Priority()
	}
	return totalPriority
}
