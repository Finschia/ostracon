package rand

import (
	"fmt"
	"math/big"
	s "sort"
)

// Interface for performing weighted deterministic random selection.
type Candidate interface {
	Priority() uint64
	LessThan(other Candidate) bool
}

// Select a specified number of candidates randomly from the candidate set based on each priority. This function is
// deterministic and will produce the same result for the same input.
//
// Inputs:
// seed - 64bit integer used for random selection.
// candidates - A set of candidates. You get different results depending on the order.
// sampleSize - The number of candidates to select at random.
// totalPriority - The exact sum of the priorities of each candidate.
//
// Returns:
// samples - A randomly selected candidate from a set of candidates. NOTE that the same candidate may have been
// selected in duplicate.
func RandomSamplingWithPriority(
	seed uint64, candidates []Candidate, sampleSize int, totalPriority uint64) (samples []Candidate) {

	// This step is performed if and only if the parameter is invalid. The reasons are as stated in the message:
	err := checkInvalidPriority(candidates, totalPriority)
	if err != nil {
		panic(err)
	}

	// generates a random selection threshold for candidates' cumulative priority
	thresholds := make([]uint64, sampleSize)
	for i := 0; i < sampleSize; i++ {
		// calculating [gross weights] × [(0,1] random number]
		thresholds[i] = RandomThreshold(&seed, totalPriority)
	}
	s.Slice(thresholds, func(i, j int) bool { return thresholds[i] < thresholds[j] })

	// extract candidates with a cumulative priority threshold
	samples = make([]Candidate, sampleSize)
	cumulativePriority := uint64(0)
	undrawn := 0
	for _, candidate := range candidates {
		for thresholds[undrawn] < cumulativePriority+candidate.Priority() {
			samples[undrawn] = candidate
			undrawn++
			if undrawn == len(samples) {
				return
			}
		}
		cumulativePriority += candidate.Priority()
	}

	// We're assuming you never get to this code
	panic(fmt.Sprintf("Cannot select samples; "+
		"totalPriority=%d, seed=%d, sampleSize=%d, undrawn=%d, threshold[%d]=%d, len(candidates)=%d",
		totalPriority, seed, sampleSize, undrawn, undrawn, thresholds[undrawn], len(candidates)))
}

const uint64Mask = uint64(0x7FFFFFFFFFFFFFFF)

var divider *big.Int

func init() {
	divider = big.NewInt(int64(uint64Mask))
	divider.Add(divider, big.NewInt(1))
}

func RandomThreshold(seed *uint64, total uint64) uint64 {
	totalBig := new(big.Int).SetUint64(total)
	a := new(big.Int).SetUint64(nextRandom(seed) & uint64Mask)
	a.Mul(a, totalBig)
	a.Div(a, divider)
	return a.Uint64()
}

// SplitMix64
// http://xoshiro.di.unimi.it/splitmix64.c
//
// The PRNG used for this random selection:
//   1. must be deterministic.
//   2. should easily portable, independent of language or library
//   3. is not necessary to keep a long period like MT, since there aren't many random numbers to generate and
//      we expect a certain amount of randomness in the seed.
func nextRandom(rand *uint64) uint64 {
	*rand += uint64(0x9e3779b97f4a7c15)
	var z = *rand
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

func checkInvalidPriority(candidates []Candidate, totalPriority uint64) error {
	actualTotalPriority := uint64(0)
	for i := 0; i < len(candidates); i++ {
		actualTotalPriority += candidates[i].Priority()
	}

	if len(candidates) == 0 {
		return fmt.Errorf("candidates is empty; "+
			"totalPriority=%d, actualTotalPriority=%d, len(candidates)=%d",
			totalPriority, actualTotalPriority, len(candidates))

	} else if totalPriority == 0 || actualTotalPriority == 0 {
		return fmt.Errorf("either total priority or actual priority is zero; "+
			"totalPriority=%d, actualTotalPriority=%d, len(candidates)=%d",
			totalPriority, actualTotalPriority, len(candidates))

	} else if actualTotalPriority != totalPriority {
		return fmt.Errorf("total priority not equal to actual priority; "+
			"totalPriority=%d, actualTotalPriority=%d, len(candidates)=%d",
			totalPriority, actualTotalPriority, len(candidates))

	}
	return nil
}
