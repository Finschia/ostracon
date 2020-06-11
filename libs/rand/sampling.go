package rand

import (
	"fmt"
	"math"
	"math/big"
	s "sort"
)

// Interface for performing weighted deterministic random selection.
type Candidate interface {
	Priority() uint64
	LessThan(other Candidate) bool
	SetWinPoint(winPoint int64)
}

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
		thresholds[i] = randomThreshold(&seed, totalPriority)
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

const uint64Mask = uint64(0x7FFFFFFFFFFFFFFF)

var divider *big.Int

func init() {
	divider = big.NewInt(int64(uint64Mask))
	divider.Add(divider, big.NewInt(1))
}

func randomThreshold(seed *uint64, total uint64) uint64 {
	if int64(total) < 0 {
		panic(fmt.Sprintf("total priority is overflow: %d", total))
	}
	totalBig := big.NewInt(int64(total))
	a := big.NewInt(int64(nextRandom(seed) & uint64Mask))
	a.Mul(a, totalBig)
	a.Div(a, divider)
	return a.Uint64()
}

// `RandomSamplingWithoutReplacement` elects winners among candidates without replacement
// so it updates rewards of winners. This function continues to elect winners until the both of two
// conditions(minSamplingCount, minPriorityPercent) are met.
func RandomSamplingWithoutReplacement(
	seed uint64, candidates []Candidate, minSamplingCount int, winPointUnit uint64) (
	winners []Candidate) {

	if len(candidates) < minSamplingCount {
		panic(fmt.Sprintf("The number of candidates(%d) cannot be less minSamplingCount %d",
			len(candidates), minSamplingCount))
	}

	totalPriority := sumTotalPriority(candidates)
	candidates = sort(candidates)
	winnersPriority := uint64(0)
	losersPriorities := make([]uint64, len(candidates))
	winnerNum := 0
	for winnerNum < minSamplingCount {
		if totalPriority-winnersPriority == 0 {
			// it's possible if some candidates have zero priority
			// if then, we can't elect voter any more; we should holt electing not to fall in infinity loop
			break
		}
		threshold := randomThreshold(&seed, totalPriority-winnersPriority)
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
			panic(fmt.Sprintf("Cannot find random sample. winnerNum=%d, minSamplingCount=%d, "+
				"winnersPriority=%d, totalPriority=%d, threshold=%d",
				winnerNum, minSamplingCount, winnersPriority, totalPriority, threshold))
		}
	}
	compensationProportions := make([]float64, winnerNum)
	for i := winnerNum - 2; i >= 0; i-- { // last winner doesn't get compensation reward
		compensationProportions[i] = compensationProportions[i+1] + 1/float64(losersPriorities[i])
	}
	winners = candidates[len(candidates)-winnerNum:]
	winPoints := make([]float64, len(winners))
	downscaleNeeded := false
	for i, winner := range winners {
		winPoints[i] = float64(winPointUnit) +
			float64(winner.Priority())*compensationProportions[i]*float64(winPointUnit)
		if int64(winPoints[i]) < 0 {
			downscaleNeeded = true
		}
	}
	for downscaleNeeded {
		downscaleNeeded = false
		for i := range winPoints {
			winPoints[i] /= 10
			if int64(winPoints[i]) < 0 {
				downscaleNeeded = true
			}
		}
	}
	for i, winner := range winners {
		if winPoints[i] > math.MaxInt64 || winPoints[i] < 0 {
			panic(fmt.Sprintf("winPoint is invalid: %f", winPoints[i]))
		}
		winner.SetWinPoint(int64(winPoints[i]))
	}
	return winners
}

func sumTotalPriority(candidates []Candidate) (sum uint64) {
	for _, candi := range candidates {
		sum += candi.Priority()
	}
	if sum == 0 {
		panic("all candidates have zero priority")
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
