package rand

import (
	"fmt"
	s "sort"
)

// Interface for performing weighted deterministic random selection.
type Candidate interface {
	Priority() uint64
	LessThan(other *Candidate) bool
}

const uint64Mask = uint64(0x7FFFFFFFFFFFFFFF)

// Select a specified number of candidates randomly from the candidate set based on each priority. This function is
// deterministic and will produce the same result for the same input.
//
// Inputs:
// seed - 64bit integer used for random selection.
// candidates - A set of candidates. The order is disregarded.
// sampleSize - The number of candidates to select at random.
// totalPriority - The exact sum of the priorities of each candidate.
//
// Returns:
// samples - A randomly selected candidate from a set of candidates. NOTE that the same candidate may have been
// selected in duplicate.
func RandomSamplingWithPriority(
	seed uint64, candidates []Candidate, sampleSize int, totalPriority uint64) (samples []Candidate) {

	// generates a random selection threshold for candidates' cumulative priority
	thresholds := make([]uint64, sampleSize)
	for i := 0; i < sampleSize; i++ {
		// calculating [gross weights] × [(0,1] random number]
		thresholds[i] = uint64(float64(nextRandom(&seed)&uint64Mask) / float64(uint64Mask+1) * float64(totalPriority))
	}
	s.Slice(thresholds, func(i, j int) bool { return thresholds[i] < thresholds[j] })

	// generates a copy of the set to keep the given array order
	candidates = sort(candidates)

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

	// This step is performed if and only if the parameter is invalid. The reasons are as stated in the message:
	actualTotalPriority := uint64(0)
	for i := 0; i < len(candidates); i++ {
		actualTotalPriority += candidates[i].Priority()
	}
	panic(fmt.Sprintf("Either the given candidate is an empty set, the actual cumulative priority is zero,"+
		" or the total priority is less than the actual one; totalPriority=%d, actualTotalPriority=%d,"+
		" seed=%d, sampleSize=%d, undrawn=%d, threshold[%d]=%d, len(candidates)=%d",
		totalPriority, actualTotalPriority, seed, sampleSize, undrawn, undrawn, thresholds[undrawn], len(candidates)))
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

// sort candidates in descending priority and ascending nature order
func sort(candidates []Candidate) []Candidate {
	temp := make([]Candidate, len(candidates))
	copy(temp, candidates)
	s.Slice(temp, func(i, j int) bool {
		if temp[i].Priority() != temp[j].Priority() {
			return temp[i].Priority() > temp[j].Priority()
		}
		return temp[i].LessThan(&temp[j])
	})
	return temp
}
