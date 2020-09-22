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
	WinPoint() float64
	VotingPower() uint64
	SetWinPoint(winPoint float64)
	SetVotingPower(votingPower uint64)
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

// precisionForSelection is a value to be corrected to increase precision when calculating voting power as an integer.
const precisionForSelection = uint64(1000)

// precisionCorrectionForSelection is a value corrected for accuracy of voting power
const precisionCorrectionForSelection = uint64(1000)

var divider *big.Int

func init() {
	divider = big.NewInt(int64(uint64Mask))
	divider.Add(divider, big.NewInt(1))
}

func randomThreshold(seed *uint64, total uint64) uint64 {
	totalBig := new(big.Int).SetUint64(total)
	a := new(big.Int).SetUint64(nextRandom(seed) & uint64Mask)
	a.Mul(a, totalBig)
	a.Div(a, divider)
	return a.Uint64()
}

// sumTotalPriority calculate the sum of all candidate's priority(weight)
// and the sum should be less then or equal to MaxUint64
// TODO We need to check the total weight doesn't over MaxUint64 in somewhere not here.
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

func electVoter(
	seed *uint64, candidates []Candidate, voterNum int, totalPriority uint64) (
	winnerIdx int, winner Candidate) {
	threshold := randomThreshold(seed, totalPriority)
	found := false
	cumulativePriority := uint64(0)
	for i, candidate := range candidates[:len(candidates)-voterNum] {
		if threshold < cumulativePriority+candidate.Priority() {
			winner = candidates[i]
			winnerIdx = i
			found = true
			break
		}
		cumulativePriority += candidate.Priority()
	}

	if !found {
		panic(fmt.Sprintf("Cannot find random sample. voterNum=%d, "+
			"totalPriority=%d, threshold=%d",
			voterNum, totalPriority, threshold))
	}

	return winnerIdx, winner
}

func ElectVotersNonDup(candidates []Candidate, seed, tolerableByzantinePercent uint64) (voters []Candidate) {
	totalPriority := sumTotalPriority(candidates)
	tolerableByzantinePower := totalPriority * tolerableByzantinePercent / 100
	voters = make([]Candidate, 0)
	candidates = sort(candidates)

	zeroPriorities := 0
	for i := len(candidates); candidates[i-1].Priority() == 0; i-- {
		zeroPriorities++
	}

	losersPriorities := totalPriority
	for len(voters)+zeroPriorities < len(candidates) {
		//accumulateWinPoints(voters)
		for _, voter := range voters {
			//i = v1 ... vt
			//stakingPower(i) * 1000 / (stakingPower(vt+1 ... vn) + stakingPower(i))
			additionalWinPoint := new(big.Int).Mul(new(big.Int).SetUint64(voter.Priority()),
				new(big.Int).SetUint64(precisionForSelection))
			additionalWinPoint.Div(additionalWinPoint, new(big.Int).Add(new(big.Int).SetUint64(losersPriorities),
				new(big.Int).SetUint64(voter.Priority())))
			voter.SetWinPoint(voter.WinPoint() + float64(additionalWinPoint.Uint64())/float64(precisionCorrectionForSelection))
		}
		//electVoter
		winnerIdx, winner := electVoter(&seed, candidates, len(voters)+zeroPriorities, losersPriorities)

		//add 1 winPoint to winner
		winner.SetWinPoint(1)

		moveWinnerToLast(candidates, winnerIdx)
		voters = append(voters, winner)
		losersPriorities -= winner.Priority()

		//sort voters in ascending votingPower/stakingPower
		voters = sortVoters(voters)
		totalWinPoint := float64(0)

		//calculateVotingPowers(voters)
		for _, voter := range voters {
			totalWinPoint += voter.WinPoint()
		}
		totalVotingPower := uint64(0)
		for _, voter := range voters {
			bigWinPoint := new(big.Int).SetUint64(
				uint64(voter.WinPoint() * float64(precisionForSelection*precisionForSelection)))
			bigTotalWinPoint := new(big.Int).SetUint64(uint64(totalWinPoint * float64(precisionForSelection)))
			bigVotingPower := new(big.Int).Mul(new(big.Int).Div(bigWinPoint, bigTotalWinPoint),
				new(big.Int).SetUint64(totalPriority))
			votingPower := new(big.Int).Div(bigVotingPower, new(big.Int).SetUint64(precisionForSelection)).Uint64()
			voter.SetVotingPower(votingPower)
			totalVotingPower += votingPower
		}

		topFVotersVotingPower := countVoters(voters, tolerableByzantinePower)
		if topFVotersVotingPower < totalVotingPower/3 {
			break
		}
	}
	return voters
}

func countVoters(voters []Candidate, tolerableByzantinePower uint64) uint64 {
	topFVotersStakingPower := uint64(0)
	topFVotersVotingPower := uint64(0)
	for _, voter := range voters {
		prev := topFVotersStakingPower
		topFVotersStakingPower += voter.Priority()
		topFVotersVotingPower += voter.VotingPower()
		if prev < tolerableByzantinePower && topFVotersStakingPower >= tolerableByzantinePower {
			break
		}
	}
	return topFVotersVotingPower
}

// sortVoters is function to sort voters in descending votingPower/stakingPower
func sortVoters(candidates []Candidate) []Candidate {
	temp := make([]Candidate, len(candidates))
	copy(temp, candidates)
	s.Slice(temp, func(i, j int) bool {
		a := temp[i].VotingPower() / temp[i].Priority()
		b := temp[j].VotingPower() / temp[j].Priority()
		return a > b
	})
	return temp
}
