package rand

import (
	"fmt"
	"math"
	s "sort"
)

// Interface for performing weighted deterministic random selection.
type Candidate interface {
	Priority() uint64
	LessThan(other Candidate) bool
	Reward(rewards uint64)
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

func moveWinnerToLast(candidates []Candidate, winner int) {
	winnerCandidate := candidates[winner]
	copy(candidates[winner:], candidates[winner+1:])
	candidates[len(candidates)-1] = winnerCandidate
}

// `RandomSamplingWithoutReplacement` elects winners among candidates without replacement so it updates rewards of winners.
// This function continues to elect winners until the both of two conditions(samplingThreshold, priorityRateThreshold) are met.
func RandomSamplingWithoutReplacement(
	seed uint64, candidates []Candidate, samplingThreshold int, priorityRateThreshold float64, rewardUnit uint64) (
	winners []Candidate) {

	if len(candidates) < samplingThreshold {
		panic(fmt.Sprintf("The number of candidates(%d) cannot be less samplingThreshold %d",
			len(candidates), samplingThreshold))
	}

	if priorityRateThreshold > 1 {
		panic(fmt.Sprintf("priorityRateThreshold cannot be greater than 1.0: %f", priorityRateThreshold))
	}

	totalPriority := sumTotalPriority(candidates)
	priorityThreshold := uint64(math.Ceil(float64(totalPriority) * priorityRateThreshold))
	if priorityThreshold > totalPriority {
		// This can be possible because of float64's precision when priorityRateThreshold is 1 and totalPriority is very big
		priorityThreshold = totalPriority
	}
	candidates = sort(candidates)
	winnersPriority := uint64(0)
	losersPriorities := make([]uint64, len(candidates))
	winnerNum := 0
	for winnerNum < samplingThreshold || winnersPriority < priorityThreshold {
		threshold := uint64(float64(nextRandom(&seed)&uint64Mask) / float64(uint64Mask+1) * float64(totalPriority-winnersPriority))
		cumulativePriority := uint64(0)
		found := false
		for i, candidate := range candidates[:len(candidates)-winnerNum] {
			if threshold < cumulativePriority+candidate.Priority() {
				moveWinnerToLast(candidates, i)
				winnersPriority += candidate.Priority()
				losersPriorities[winnerNum] = totalPriority - winnersPriority
				winnerNum++
				found = true
				break
			}
			cumulativePriority += candidate.Priority()
		}

		if !found {
			panic(fmt.Sprintf("Cannot find random sample. winnerNum=%d, samplingThreshold=%d, "+
				"winnersPriority=%d, priorityThreshold=%d, totalPriority=%d, threshold=%d",
				winnerNum, samplingThreshold, winnersPriority, priorityThreshold, totalPriority, threshold))
		}
	}

	compensationRewardProportions := make([]float64, winnerNum)
	for i := winnerNum - 2; i >= 0; i-- { // last winner doesn't get compensation reward
		compensationRewardProportions[i] = compensationRewardProportions[i+1] + 1/float64(losersPriorities[i])
	}
	winners = candidates[len(candidates)-winnerNum:]
	for i, winner := range winners {
		// TODO protect overflow and verify the accuracy of the calculations.
		// voter.Priority()*rewardUnit can be overflow, so we multiply voter.Priority() and rewardProportion at first
		winner.Reward(rewardUnit + uint64((float64(winner.Priority())*compensationRewardProportions[i])*float64(rewardUnit)))
	}
	return
}

func sumTotalPriority(candidates []Candidate) (sum uint64) {
	for _, candi := range candidates {
		if candi.Priority() == 0 {
			panic("candidate(%d) priority must not be 0")
		}
		sum += candi.Priority()
	}
	return
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
		return temp[i].LessThan(temp[j])
	})
	return temp
}
