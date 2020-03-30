package rand

import (
	"fmt"
	"math"
	s "sort"
	"testing"
)

type Element struct {
	Id     uint32
	Weight uint64
}

func (e *Element) Priority() uint64 {
	return e.Weight
}

func (e *Element) LessThan(other *Candidate) bool {
	o, ok := (*other).(*Element)
	if ! ok {
		panic("incompatible type")
	}
	return e.Id < o.Id
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
			counts[e.(*Element).Id] += 1
		}
	}
	expected := float64(1) / float64(100)
	mean, variance, z := calculateZ(expected, counts)
	if z >= 1e-15 || math.Abs(mean-expected) >= 1e-15 || variance >= 1e-5 {
		t.Errorf("winning frequency is uneven: mean=%f, variance=%e, z=%e", mean, variance, z)
	}
}


func newCandidates(length int, prio func(int) uint64) (candidates []Candidate) {
	candidates = make([]Candidate, 100)
	for i := 0; i < length; i++ {
		candidates[i] = &Element{uint32(i), prio(i)}
	}
	return
}

func sameCandidates(c1 []Candidate, c2 []Candidate) bool {
	if len(c1) != len(c2) {
		return false
	}
	s.Slice(c1, func(i, j int) bool { return c1[i].LessThan(&c1[j]) })
	s.Slice(c2, func(i, j int) bool { return c2[i].LessThan(&c2[j]) })
	for i := 0; i < len(c1); i++ {
		if c1[i].(*Element).Id != c2[i].(*Element).Id {
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
		sum += float64(x)
	}
	mean = float64(sum) / float64(len(values))
	sum2 := 0.0
	for _, x := range values {
		dx := float64(x) - mean
		sum2 += dx * dx
	}
	variance = sum2 / float64(len(values))
	return
}
