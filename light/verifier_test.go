package light_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/line/ostracon/crypto/tmhash"
	"github.com/line/ostracon/crypto/vrf"

	"github.com/stretchr/testify/assert"

	tmmath "github.com/line/ostracon/libs/math"
	"github.com/line/ostracon/light"
	"github.com/line/ostracon/types"
)

const (
	maxClockDrift = 10 * time.Second
)

func TestVerifyAdjacentHeaders(t *testing.T) {
	const (
		chainID    = "TestVerifyAdjacentHeaders"
		lastHeight = 1
		nextHeight = 2
	)

	var (
		keys = genPrivKeys(4)
		// 20, 30, 40, 50 - the first 3 don't have 2/3, the last 3 do!
		vals     = keys.ToValidators(20, 10)
		bTime, _ = time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
		header   = keys.GenSignedHeader(chainID, lastHeight, bTime, nil,
			vals, vals,
			hash("app_hash"), hash("cons_hash"), hash("results_hash"), 0, len(keys),
			types.DefaultVoterParams())
	)

	otherVals := keys.ToValidators(10, 1)

	testCases := []struct {
		newHeader      *types.SignedHeader
		newVals        *types.ValidatorSet
		trustingPeriod time.Duration
		now            time.Time
		expErr         error
		expErrText     string
	}{
		// same header -> no error
		0: {
			header,
			vals,
			3 * time.Hour,
			bTime.Add(2 * time.Hour),
			nil,
			"headers must be adjacent in height",
		},
		// different chainID -> error
		1: {
			keys.GenSignedHeader("different-chainID", nextHeight, bTime.Add(1*time.Hour), nil,
				vals, vals,
				hash("app_hash"), hash("cons_hash"), hash("results_hash"), 0, len(keys),
				types.DefaultVoterParams()),
			vals,
			3 * time.Hour,
			bTime.Add(2 * time.Hour),
			nil,
			"header belongs to another chain",
		},
		// new header's time is before old header's time -> error
		2: {
			keys.GenSignedHeader(chainID, nextHeight, bTime.Add(-1*time.Hour), nil,
				vals, vals,
				hash("app_hash"), hash("cons_hash"), hash("results_hash"), 0, len(keys),
				types.DefaultVoterParams()),
			vals,
			3 * time.Hour,
			bTime.Add(2 * time.Hour),
			nil,
			"to be after old header time",
		},
		// new header's time is from the future -> error
		3: {
			keys.GenSignedHeader(chainID, nextHeight, bTime.Add(3*time.Hour), nil,
				vals, vals,
				hash("app_hash"), hash("cons_hash"), hash("results_hash"), 0, len(keys),
				types.DefaultVoterParams()),
			vals,
			3 * time.Hour,
			bTime.Add(2 * time.Hour),
			nil,
			"new header has a time from the future",
		},
		// new header's time is from the future, but it's acceptable (< maxClockDrift) -> no error
		4: {
			keys.GenSignedHeader(chainID, nextHeight,
				bTime.Add(2*time.Hour).Add(maxClockDrift).Add(-1*time.Millisecond), nil,
				vals, vals,
				hash("app_hash"), hash("cons_hash"), hash("results_hash"), 0, len(keys),
				types.DefaultVoterParams()),
			vals,
			3 * time.Hour,
			bTime.Add(2 * time.Hour),
			nil,
			"",
		},
		// 3/3 signed -> no error
		5: {
			keys.GenSignedHeader(chainID, nextHeight, bTime.Add(1*time.Hour), nil,
				vals, vals,
				hash("app_hash"), hash("cons_hash"), hash("results_hash"), 0, len(keys),
				types.DefaultVoterParams()),
			vals,
			3 * time.Hour,
			bTime.Add(2 * time.Hour),
			nil,
			"",
		},
		// 2/3 signed -> no error
		6: {
			keys.GenSignedHeader(chainID, nextHeight, bTime.Add(1*time.Hour), nil,
				vals, vals,
				hash("app_hash"), hash("cons_hash"), hash("results_hash"), 1, len(keys),
				types.DefaultVoterParams()),
			vals,
			3 * time.Hour,
			bTime.Add(2 * time.Hour),
			nil,
			"",
		},
		// 1/3 signed -> error
		7: {
			keys.GenSignedHeader(chainID, nextHeight, bTime.Add(1*time.Hour), nil,
				vals, vals,
				hash("app_hash"), hash("cons_hash"), hash("results_hash"), len(keys)-1, len(keys),
				types.DefaultVoterParams()),
			vals,
			3 * time.Hour,
			bTime.Add(2 * time.Hour),
			light.ErrInvalidHeader{Reason: types.ErrNotEnoughVotingWeightSigned{Got: 50, Needed: 93}},
			"",
		},
		// voters does not match with what we have -> error
		8: {
			keys.GenSignedHeader(chainID, nextHeight, bTime.Add(1*time.Hour), nil,
				otherVals, vals, hash("app_hash"), hash("cons_hash"),
				hash("results_hash"), 0, len(keys), types.DefaultVoterParams()),
			keys.ToValidators(10, 1),
			3 * time.Hour,
			bTime.Add(2 * time.Hour),
			nil,
			"to match those from new header",
		},
		// voters are inconsistent with newHeader -> error
		9: {
			keys.GenSignedHeader(chainID, nextHeight, bTime.Add(1*time.Hour), nil,
				vals, vals,
				hash("app_hash"), hash("cons_hash"), hash("results_hash"), 0, len(keys),
				types.DefaultVoterParams()),
			keys.ToValidators(10, 1),
			3 * time.Hour,
			bTime.Add(2 * time.Hour),
			nil,
			"to match those that were supplied",
		},
		// old header has expired -> error
		10: {
			keys.GenSignedHeader(chainID, nextHeight, bTime.Add(1*time.Hour), nil,
				vals, vals,
				hash("app_hash"), hash("cons_hash"), hash("results_hash"), 0, len(keys),
				types.DefaultVoterParams()),
			keys.ToValidators(10, 1),
			1 * time.Hour,
			bTime.Add(1 * time.Hour),
			nil,
			"old header has expired",
		},
	}

	for i, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("#%d", i), func(t *testing.T) {
			proofHash, err := vrf.ProofToHash(header.Proof.Bytes())
			if err != nil {
				assert.NoError(t, err)
			}
			voters := types.SelectVoter(tc.newVals, proofHash, types.DefaultVoterParams())
			err = light.VerifyAdjacent(
				header,
				tc.newHeader,
				voters,
				tc.trustingPeriod,
				tc.now,
				maxClockDrift,
			)
			switch {
			case tc.expErr != nil && assert.Error(t, err):
				assert.Equal(t, tc.expErr, err)
			case tc.expErrText != "":
				assert.Contains(t, err.Error(), tc.expErrText)
			default:
				assert.NoError(t, err)
			}
		})
	}
}

func TestVerifyAdjacentHeadersWithVoterSampling(t *testing.T) {
	const (
		chainID    = "TestVerifyAdjacentHeaders"
		lastHeight = 1
		nextHeight = 2
	)

	var (
		voterParamsHalf = &types.VoterParams{
			VoterElectionThreshold:          5,
			MaxTolerableByzantinePercentage: 10,
		}
		keys = genPrivKeys(10)
		// 100, 110, ..., 200
		vals     = keys.ToValidators(100, 10)
		bTime, _ = time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
		header   = keys.GenSignedHeader(chainID, lastHeight, bTime, nil,
			vals, vals,
			[]byte("app_hash"), tmhash.Sum([]byte("cons_hash")), []byte("results_hash"), 0, len(keys), voterParamsHalf)
	)

	otherVals := keys.ToValidators(200, 1)

	consHash := tmhash.Sum([]byte("cons_hash"))
	resultHash := tmhash.Sum([]byte("results_hash"))

	testCases := []struct {
		newHeader      *types.SignedHeader
		newVals        *types.ValidatorSet
		trustingPeriod time.Duration
		now            time.Time
		expErr         error
		expErrText     string
	}{
		// same header -> no error
		0: {
			header,
			vals,
			3 * time.Hour,
			bTime.Add(2 * time.Hour),
			nil,
			"headers must be adjacent in height",
		},
		// different chainID -> error
		1: {
			keys.GenSignedHeader("different-chainID", nextHeight, bTime.Add(1*time.Hour), nil,
				vals, vals, []byte("app_hash"), consHash, resultHash, 0, len(keys),
				voterParamsHalf),
			vals,
			3 * time.Hour,
			bTime.Add(2 * time.Hour),
			nil,
			"untrustedHeader.ValidateBasic failed: header belongs to another chain \"different-chainID\", not" +
				" \"TestVerifyAdjacentHeaders\"",
		},
		// new header's time is before old header's time -> error
		2: {
			keys.GenSignedHeader(chainID, nextHeight, bTime.Add(-1*time.Hour), nil,
				vals, vals, []byte("app_hash"), consHash, resultHash, 0, len(keys),
				voterParamsHalf),
			vals,
			3 * time.Hour,
			bTime.Add(2 * time.Hour),
			nil,
			"to be after old header time",
		},
		// new header's time is from the future -> error
		3: {
			keys.GenSignedHeader(chainID, nextHeight, bTime.Add(3*time.Hour), nil,
				vals, vals, []byte("app_hash"), consHash, resultHash, 0, len(keys),
				voterParamsHalf),
			vals,
			3 * time.Hour,
			bTime.Add(2 * time.Hour),
			nil,
			"new header has a time from the future",
		},
		// new header's time is from the future, but it's acceptable (< maxClockDrift) -> no error
		4: {
			keys.GenSignedHeader(chainID, nextHeight,
				bTime.Add(2*time.Hour).Add(maxClockDrift).Add(-1*time.Millisecond), nil,
				vals, vals, []byte("app_hash"), consHash, resultHash, 0, len(keys),
				voterParamsHalf),
			vals,
			3 * time.Hour,
			bTime.Add(2 * time.Hour),
			nil,
			"",
		},
		// 3/3 signed -> no error
		5: {
			keys.GenSignedHeaderByRate(chainID, nextHeight, bTime.Add(1*time.Hour), nil,
				vals, vals, []byte("app_hash"), consHash, resultHash, 1.0,
				voterParamsHalf),
			vals,
			3 * time.Hour,
			bTime.Add(2 * time.Hour),
			nil,
			"",
		},
		// 2/3 signed -> no error
		6: {
			keys.GenSignedHeaderByRate(chainID, nextHeight, bTime.Add(1*time.Hour), nil,
				vals, vals, []byte("app_hash"), consHash, resultHash, 0.67,
				voterParamsHalf),
			vals,
			3 * time.Hour,
			bTime.Add(2 * time.Hour),
			nil,
			"",
		},
		// 1/3 signed -> error
		7: {
			keys.GenSignedHeaderByRate(chainID, nextHeight, bTime.Add(1*time.Hour), nil,
				vals, vals,
				[]byte("app_hash"), consHash, resultHash, 0.33,
				voterParamsHalf),
			vals,
			3 * time.Hour,
			bTime.Add(2 * time.Hour),
			nil,
			"invalid commit -- insufficient voting weight",
		},
		// vals does not match with what we have -> error
		8: {
			keys.GenSignedHeader(chainID, nextHeight, bTime.Add(1*time.Hour), nil,
				otherVals, vals, []byte("app_hash"), consHash,
				resultHash, 0, len(keys), voterParamsHalf),
			otherVals,
			3 * time.Hour,
			bTime.Add(2 * time.Hour),
			nil,
			"to match those from new header",
		},
		// vals are inconsistent with newHeader -> error
		9: {
			keys.GenSignedHeader(chainID, nextHeight, bTime.Add(1*time.Hour), nil,
				vals, vals, []byte("app_hash"), consHash, resultHash, 0, len(keys),
				voterParamsHalf),
			keys.ToValidators(10, 1),
			3 * time.Hour,
			bTime.Add(2 * time.Hour),
			nil,
			"to match those that were supplied",
		},
		// old header has expired -> error
		10: {
			keys.GenSignedHeader(chainID, nextHeight, bTime.Add(1*time.Hour), nil,
				vals, vals, []byte("app_hash"), consHash, resultHash, 0, len(keys),
				voterParamsHalf),
			keys.ToValidators(10, 1),
			1 * time.Hour,
			bTime.Add(1 * time.Hour),
			nil,
			"old header has expired",
		},
	}

	for i, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("#%d", i), func(t *testing.T) {
			proofHash, err := vrf.ProofToHash(tc.newHeader.Proof.Bytes())
			if err != nil {
				assert.NoError(t, err)
			}
			voters := types.SelectVoter(tc.newVals, proofHash, voterParamsHalf)
			err = light.VerifyAdjacent(header, tc.newHeader, voters, tc.trustingPeriod, tc.now, maxClockDrift)
			switch {
			case tc.expErr != nil && assert.Error(t, err):
				assert.Equal(t, tc.expErr, err)
			case tc.expErrText != "":
				if !assert.Contains(t, err.Error(), tc.expErrText) {
					fmt.Printf("%s\n%s\n", err.Error(), tc.expErrText)
				}
			default:
				assert.NoError(t, err)
			}
		})
	}
}

func TestVerifyNonAdjacentHeaders(t *testing.T) {
	const (
		chainID    = "TestVerifyNonAdjacentHeaders"
		lastHeight = 1
	)

	var (
		keys = genPrivKeys(4)
		// 20, 30, 40, 50 - the first 3 don't have 2/3, the last 3 do!
		vals     = keys.ToValidators(20, 10)
		bTime, _ = time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
		header   = keys.GenSignedHeader(chainID, lastHeight, bTime, nil,
			vals, vals,
			hash("app_hash"), hash("cons_hash"), hash("results_hash"), 0, len(keys),
			types.DefaultVoterParams())

		// 30, 40, 50
		twoThirds     = keys[1:]
		twoThirdsVals = twoThirds.ToValidators(30, 10)

		// 50
		oneThird     = keys[len(keys)-1:]
		oneThirdVals = oneThird.ToValidators(50, 10)

		// 20
		lessThanOneThird     = keys[0:1]
		lessThanOneThirdVals = lessThanOneThird.ToValidators(20, 10)
	)

	testCases := []struct {
		newHeader      *types.SignedHeader
		newVals        *types.ValidatorSet
		trustingPeriod time.Duration
		now            time.Time
		expErr         error
		expErrText     string
	}{
		// 3/3 new voters signed, 3/3 old voters present -> no error
		0: {
			keys.GenSignedHeader(chainID, 3, bTime.Add(1*time.Hour), nil,
				vals, vals,
				hash("app_hash"), hash("cons_hash"), hash("results_hash"), 0, len(keys),
				types.DefaultVoterParams()),
			vals,
			3 * time.Hour,
			bTime.Add(2 * time.Hour),
			nil,
			"",
		},
		// 2/3 new voters signed, 3/3 old voters present -> no error
		1: {
			keys.GenSignedHeader(chainID, 4, bTime.Add(1*time.Hour), nil,
				vals, vals,
				hash("app_hash"), hash("cons_hash"), hash("results_hash"), 1, len(keys),
				types.DefaultVoterParams()),
			vals,
			3 * time.Hour,
			bTime.Add(2 * time.Hour),
			nil,
			"",
		},
		// 1/3 new voters signed, 3/3 old voters present -> error
		2: {
			keys.GenSignedHeader(chainID, 5, bTime.Add(1*time.Hour), nil,
				vals, vals,
				hash("app_hash"), hash("cons_hash"), hash("results_hash"), len(keys)-1, len(keys),
				types.DefaultVoterParams()),
			vals,
			3 * time.Hour,
			bTime.Add(2 * time.Hour),
			light.ErrInvalidHeader{types.ErrNotEnoughVotingWeightSigned{Got: 50, Needed: 93}},
			"",
		},
		// 3/3 new voters signed, 2/3 old voters present -> no error
		3: {
			twoThirds.GenSignedHeader(chainID, 5, bTime.Add(1*time.Hour), nil,
				twoThirdsVals, twoThirdsVals,
				hash("app_hash"), hash("cons_hash"), hash("results_hash"), 0, len(twoThirds),
				types.DefaultVoterParams()),
			twoThirdsVals,
			3 * time.Hour,
			bTime.Add(2 * time.Hour),
			nil,
			"",
		},
		// 3/3 new voters signed, 1/3 old voters present -> no error
		4: {
			oneThird.GenSignedHeader(chainID, 5, bTime.Add(1*time.Hour), nil,
				oneThirdVals, oneThirdVals,
				hash("app_hash"), hash("cons_hash"), hash("results_hash"), 0, len(oneThird),
				types.DefaultVoterParams()),
			oneThirdVals,
			3 * time.Hour,
			bTime.Add(2 * time.Hour),
			nil,
			"",
		},
		// 3/3 new voters signed, less than 1/3 old voters present -> error
		5: {
			lessThanOneThird.GenSignedHeader(chainID, 5, bTime.Add(1*time.Hour), nil,
				lessThanOneThirdVals, lessThanOneThirdVals,
				hash("app_hash"), hash("cons_hash"), hash("results_hash"), 0, len(lessThanOneThird),
				types.DefaultVoterParams()),
			lessThanOneThirdVals,
			3 * time.Hour,
			bTime.Add(2 * time.Hour),
			light.ErrNewValSetCantBeTrusted{types.ErrNotEnoughVotingWeightSigned{Got: 20, Needed: 46}},
			"",
		},
	}

	for i, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("#%d", i), func(t *testing.T) {
			proofHash, err := vrf.ProofToHash(header.Proof.Bytes())
			if err != nil {
				assert.NoError(t, err)
			}
			trustedVoters := types.SelectVoter(vals, proofHash, types.DefaultVoterParams())
			proofHash, err = vrf.ProofToHash(tc.newHeader.Proof.Bytes())
			if err != nil {
				assert.NoError(t, err)
			}
			voters := types.SelectVoter(tc.newVals, proofHash, types.DefaultVoterParams())
			err = light.VerifyNonAdjacent(header, trustedVoters, tc.newHeader, voters, tc.trustingPeriod, tc.now,
				maxClockDrift, light.DefaultTrustLevel)

			switch {
			case tc.expErr != nil && assert.Error(t, err):
				assert.Equal(t, tc.expErr, err)
			case tc.expErrText != "":
				assert.Contains(t, err.Error(), tc.expErrText)
			default:
				assert.NoError(t, err)
			}
		})
	}
}

func TestVerifyReturnsErrorIfTrustLevelIsInvalid(t *testing.T) {
	const (
		chainID    = "TestVerifyReturnsErrorIfTrustLevelIsInvalid"
		lastHeight = 1
	)

	var (
		keys = genPrivKeys(4)
		// 20, 30, 40, 50 - the first 3 don't have 2/3, the last 3 do!
		vals     = keys.ToValidators(20, 10)
		bTime, _ = time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
		header   = keys.GenSignedHeader(chainID, lastHeight, bTime, nil,
			vals, vals,
			hash("app_hash"), hash("cons_hash"), hash("results_hash"), 0, len(keys),
			types.DefaultVoterParams())
	)

	err := light.Verify(header, vals, header, vals, 2*time.Hour, time.Now(), maxClockDrift,
		tmmath.Fraction{Numerator: 2, Denominator: 1}, types.DefaultVoterParams())
	assert.Error(t, err)
}

func TestValidateTrustLevel(t *testing.T) {
	testCases := []struct {
		lvl   tmmath.Fraction
		valid bool
	}{
		// valid
		0: {tmmath.Fraction{Numerator: 1, Denominator: 1}, true},
		1: {tmmath.Fraction{Numerator: 1, Denominator: 3}, true},
		2: {tmmath.Fraction{Numerator: 2, Denominator: 3}, true},
		3: {tmmath.Fraction{Numerator: 3, Denominator: 3}, true},
		4: {tmmath.Fraction{Numerator: 4, Denominator: 5}, true},

		// invalid
		5: {tmmath.Fraction{Numerator: 6, Denominator: 5}, false},
		6: {tmmath.Fraction{Numerator: 0, Denominator: 1}, false},
		7: {tmmath.Fraction{Numerator: 0, Denominator: 0}, false},
		8: {tmmath.Fraction{Numerator: 1, Denominator: 0}, false},
	}

	for _, tc := range testCases {
		err := light.ValidateTrustLevel(tc.lvl)
		if !tc.valid {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}
}
