package types

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/line/ostracon/crypto"
	tmversion "github.com/line/ostracon/proto/ostracon/version"
	"github.com/line/ostracon/version"
)

func TestLightBlockValidateBasic(t *testing.T) {
	header := makeRandHeader()
	commit := randCommit(time.Now())
	vals, voters, _ := RandVoterSet(5, 1)
	header.Height = commit.Height
	header.LastBlockID = commit.BlockID
	header.ValidatorsHash = vals.Hash()
	header.VotersHash = voters.Hash()
	header.Version.Block = version.BlockProtocol
	vals2, voters2, _ := RandVoterSet(3, 1)
	vals3 := vals.Copy()
	vals3.Validators[2] = &Validator{}
	voters3 := voters.Copy()
	voters3.Voters[2] = &Validator{}
	commit.BlockID.Hash = header.Hash()

	sh := &SignedHeader{
		Header: &header,
		Commit: commit,
	}

	testCases := []struct {
		name      string
		sh        *SignedHeader
		vals      *ValidatorSet
		voters    *VoterSet
		expectErr bool
	}{
		{"valid light block", sh, vals, voters, false},
		{"validators hashes don't match", sh, vals2, voters, true},
		{"voters hashes don't match", sh, vals, voters2, true},
		{"invalid validator set", sh, vals3, voters, true},
		{"invalid voter set", sh, vals, voters3, true},
		{"invalid signed header", &SignedHeader{Header: &header, Commit: randCommit(time.Now())}, vals, voters, true},
		{"empty signed header", nil, vals, voters, true},
		{"empty validator set", sh, nil, voters, true},
		{"empty validator set", sh, vals, nil, true},
	}

	for _, tc := range testCases {
		lightBlock := LightBlock{
			SignedHeader: tc.sh,
			ValidatorSet: tc.vals,
			VoterSet:     tc.voters,
		}
		err := lightBlock.ValidateBasic(header.ChainID)
		if tc.expectErr {
			assert.Error(t, err, tc.name)
		} else {
			assert.NoError(t, err, tc.name)
		}
	}

}

func TestLightBlockProtobuf(t *testing.T) {
	header := makeRandHeader()
	commit := randCommit(time.Now())
	vals, voters, _ := RandVoterSet(5, 1)
	header.Height = commit.Height
	header.LastBlockID = commit.BlockID
	header.Version.Block = version.BlockProtocol
	header.ValidatorsHash = vals.Hash()
	header.VotersHash = voters.Hash()
	commit.BlockID.Hash = header.Hash()

	sh := &SignedHeader{
		Header: &header,
		Commit: commit,
	}

	testCases := []struct {
		name       string
		sh         *SignedHeader
		vals       *ValidatorSet
		voters     *VoterSet
		toProtoErr bool
		toBlockErr bool
	}{
		{"valid light block", sh, vals, voters, false, false},
		{"empty signed header", &SignedHeader{}, vals, voters, false, false},
		{"empty validator set", sh, &ValidatorSet{}, voters, false, true},
		{"empty voter set", sh, vals, &VoterSet{}, false, true},
		{"empty light block", &SignedHeader{}, &ValidatorSet{}, &VoterSet{}, false, true},
	}

	for _, tc := range testCases {
		lightBlock := &LightBlock{
			SignedHeader: tc.sh,
			ValidatorSet: tc.vals,
			VoterSet:     tc.voters,
		}
		lbp, err := lightBlock.ToProto()
		if tc.toProtoErr {
			assert.Error(t, err, tc.name)
		} else {
			assert.NoError(t, err, tc.name)
		}

		lb, err := LightBlockFromProto(lbp)
		if tc.toBlockErr {
			assert.Error(t, err, tc.name)
		} else {
			assert.NoError(t, err, tc.name)
			assert.Equal(t, lightBlock, lb)
		}
	}

}

func TestSignedHeaderValidateBasic(t *testing.T) {
	commit := randCommit(time.Now())
	chainID := "𠜎"
	timestamp := time.Date(math.MaxInt64, 0, 0, 0, 0, 0, math.MaxInt64, time.UTC)
	h := Header{
		Version:            tmversion.Consensus{Block: version.BlockProtocol, App: math.MaxInt64},
		ChainID:            chainID,
		Height:             commit.Height,
		Time:               timestamp,
		LastBlockID:        commit.BlockID,
		LastCommitHash:     commit.Hash(),
		DataHash:           commit.Hash(),
		VotersHash:         commit.Hash(),
		ValidatorsHash:     commit.Hash(),
		NextValidatorsHash: commit.Hash(),
		ConsensusHash:      commit.Hash(),
		AppHash:            commit.Hash(),
		LastResultsHash:    commit.Hash(),
		EvidenceHash:       commit.Hash(),
		ProposerAddress:    crypto.AddressHash([]byte("proposer_address")),
	}

	validSignedHeader := SignedHeader{Header: &h, Commit: commit}
	validSignedHeader.Commit.BlockID.Hash = validSignedHeader.Hash()
	invalidSignedHeader := SignedHeader{}

	testCases := []struct {
		testName  string
		shHeader  *Header
		shCommit  *Commit
		expectErr bool
	}{
		{"Valid Signed Header", validSignedHeader.Header, validSignedHeader.Commit, false},
		{"Invalid Signed Header", invalidSignedHeader.Header, validSignedHeader.Commit, true},
		{"Invalid Signed Header", validSignedHeader.Header, invalidSignedHeader.Commit, true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			sh := SignedHeader{
				Header: tc.shHeader,
				Commit: tc.shCommit,
			}
			err := sh.ValidateBasic(validSignedHeader.Header.ChainID)
			assert.Equalf(
				t,
				tc.expectErr,
				err != nil,
				"Validate Basic had an unexpected result",
				err,
			)
		})
	}
}
