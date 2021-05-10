package types

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/tendermint/tendermint/crypto"
	tmversion "github.com/tendermint/tendermint/proto/tendermint/version"
	"github.com/tendermint/tendermint/version"
)

func TestLightBlockValidateBasic(t *testing.T) {
	header := makeRandHeader()
	commit := randCommit(time.Now())
	_, voters, _ := RandVoterSet(5, 1)
	header.Height = commit.Height
	header.LastBlockID = commit.BlockID
	header.VotersHash = voters.Hash()
	header.Version.Block = version.BlockProtocol
	_, voters2, _ := RandVoterSet(3, 1)
	voters3 := voters.Copy()
	voters3.Voters[2] = &Validator{} // TODO: It should be `voters3.Proposer = &Validator{}`
	commit.BlockID.Hash = header.Hash()

	sh := &SignedHeader{
		Header: &header,
		Commit: commit,
	}

	testCases := []struct {
		name      string
		sh        *SignedHeader
		vals      *VoterSet
		expectErr bool
	}{
		{"valid light block", sh, voters, false},
		{"hashes don't match", sh, voters2, true},
		{"invalid validator set", sh, voters3, true},
		{"invalid signed header", &SignedHeader{Header: &header, Commit: randCommit(time.Now())}, voters, true},
	}

	for _, tc := range testCases {
		lightBlock := LightBlock{
			SignedHeader: tc.sh,
			VoterSet:     tc.vals,
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
	_, voters, _ := RandVoterSet(5, 1)
	header.Height = commit.Height
	header.LastBlockID = commit.BlockID
	header.Version.Block = version.BlockProtocol
	header.VotersHash = voters.Hash()
	commit.BlockID.Hash = header.Hash()

	sh := &SignedHeader{
		Header: &header,
		Commit: commit,
	}

	testCases := []struct {
		name       string
		sh         *SignedHeader
		vals       *VoterSet
		toProtoErr bool
		toBlockErr bool
	}{
		{"valid light block", sh, voters, false, false},
		{"empty signed header", &SignedHeader{}, voters, false, false},
		{"empty validator set", sh, &VoterSet{}, false, true},
		{"empty light block", &SignedHeader{}, &VoterSet{}, false, true},
	}

	for _, tc := range testCases {
		lightBlock := &LightBlock{
			SignedHeader: tc.sh,
			ValidatorSet: NewValidatorSet(tc.vals.Voters),
			VoterSet:     tc.vals,
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
		Version:         tmversion.Consensus{Block: version.BlockProtocol, App: math.MaxInt64},
		ChainID:         chainID,
		Height:          commit.Height,
		Time:            timestamp,
		LastBlockID:     commit.BlockID,
		LastCommitHash:  commit.Hash(),
		DataHash:        commit.Hash(),
		VotersHash:      commit.Hash(),
		NextVotersHash:  commit.Hash(),
		ConsensusHash:   commit.Hash(),
		AppHash:         commit.Hash(),
		LastResultsHash: commit.Hash(),
		EvidenceHash:    commit.Hash(),
		ProposerAddress: crypto.AddressHash([]byte("proposer_address")),
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
			assert.Equal(
				t,
				tc.expectErr,
				sh.ValidateBasic(validSignedHeader.Header.ChainID) != nil,
				"Validate Basic had an unexpected result",
			)
		})
	}
}
