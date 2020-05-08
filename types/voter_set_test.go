package types

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSelectVoter(t *testing.T) {
	MaxVoters = 29
	valSet := randValidatorSet(30)
	accumulation := make(map[string]int64)
	for i := 0; i < 10000; i++ {
		voterSet := SelectVoter(valSet, []byte{byte(i)})
		assert.True(t, math.Abs(float64(valSet.TotalVotingPower() - voterSet.TotalVotingPower())) <= 10)
		for _, voter := range voterSet.Voters {
			accumulation[voter.Address.String()] += voter.VotingPower / 10000
		}
	}
}

func TestSelectVoterVarious(t *testing.T) {
	hash := 0
	for minMaxRate := 10; minMaxRate < 1000000; minMaxRate *= 10 {
		t.Logf("<<< min: 100, max: %d >>>", 100 * minMaxRate)
		for validators := 1; validators <= 100; validators++ {
			for voters := 1; voters < validators; voters++ {
				MaxVoters = voters
				valSet := randValidatorSetWithMinMax(validators, 100, 100 * int64(minMaxRate))
				voterSet := SelectVoter(valSet, []byte{byte(hash)})
				assert.True(t, int(math.Abs(float64(valSet.TotalVotingPower() - voterSet.TotalVotingPower()))) <= voters)
				if voterSet.Size() < MaxVoters {
					t.Logf("Cannot elect voters up to MaxVoters: validators=%d, MaxVoters=%d, actual voters=%d", validators, voters, voterSet.Size())
				}
				hash++
			}
		}
	}
}
