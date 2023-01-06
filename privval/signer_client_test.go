package privval

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/line/ostracon/crypto"
	"github.com/line/ostracon/crypto/ed25519"
	"github.com/line/ostracon/crypto/tmhash"
	"github.com/line/ostracon/crypto/vrf"
	tmrand "github.com/line/ostracon/libs/rand"
	cryptoproto "github.com/line/ostracon/proto/ostracon/crypto"
	privvalproto "github.com/line/ostracon/proto/ostracon/privval"
	"github.com/line/ostracon/types"
)

type signerTestCase struct {
	chainID      string
	mockPV       types.PrivValidator
	signerClient *SignerClient
	signerServer *SignerServer
}

func getSignerTestCases(t *testing.T, mockPV types.PrivValidator, start bool) []signerTestCase {
	testCases := make([]signerTestCase, 0)

	// Get test cases for each possible dialer (DialTCP / DialUnix / etc)
	for _, dtc := range getDialerTestCases(t) {
		chainID := tmrand.Str(12)
		mockKey := ed25519.GenPrivKey()
		if mockPV == nil {
			mockPV = types.NewMockPVWithParams(mockKey, false, false)
		}

		// get a pair of signer listener, signer dialer endpoints
		sl, sd := getMockEndpoints(t, dtc.addr, dtc.dialer)
		sc, err := NewSignerClient(sl, chainID)
		require.NoError(t, err)
		ss := NewSignerServer(sd, chainID, mockPV)

		if start {
			err = ss.Start()
			require.NoError(t, err)
		}

		tc := signerTestCase{
			chainID:      chainID,
			mockPV:       mockPV,
			signerClient: sc,
			signerServer: ss,
		}

		testCases = append(testCases, tc)
	}

	return testCases
}

func TestSignerClose(t *testing.T) {
	for _, tc := range getSignerTestCases(t, nil, true) {
		err := tc.signerClient.Close()
		assert.NoError(t, err)

		err = tc.signerServer.Stop()
		assert.NoError(t, err)
	}
}

func TestSignerGetPubKey(t *testing.T) {
	for _, tc := range getSignerTestCases(t, nil, true) {
		tc := tc
		t.Cleanup(func() {
			if err := tc.signerServer.Stop(); err != nil {
				t.Error(err)
			}
		})
		t.Cleanup(func() {
			if err := tc.signerClient.Close(); err != nil {
				t.Error(err)
			}
		})

		pubKey, err := tc.signerClient.GetPubKey()
		require.NoError(t, err)
		expectedPubKey, err := tc.mockPV.GetPubKey()
		require.NoError(t, err)

		assert.Equal(t, expectedPubKey, pubKey)

		pubKey, err = tc.signerClient.GetPubKey()
		require.NoError(t, err)
		expectedpk, err := tc.mockPV.GetPubKey()
		require.NoError(t, err)
		expectedAddr := expectedpk.Address()

		assert.Equal(t, expectedAddr, pubKey.Address())
	}
}

func TestSignerProposal(t *testing.T) {
	for _, tc := range getSignerTestCases(t, nil, true) {
		ts := time.Now()
		hash := tmrand.Bytes(tmhash.Size)
		have := &types.Proposal{
			Type:      tmproto.ProposalType,
			Height:    1,
			Round:     2,
			POLRound:  2,
			BlockID:   types.BlockID{Hash: hash, PartSetHeader: types.PartSetHeader{Hash: hash, Total: 2}},
			Timestamp: ts,
		}
		want := &types.Proposal{
			Type:      tmproto.ProposalType,
			Height:    1,
			Round:     2,
			POLRound:  2,
			BlockID:   types.BlockID{Hash: hash, PartSetHeader: types.PartSetHeader{Hash: hash, Total: 2}},
			Timestamp: ts,
		}

		tc := tc
		t.Cleanup(func() {
			if err := tc.signerServer.Stop(); err != nil {
				t.Error(err)
			}
		})
		t.Cleanup(func() {
			if err := tc.signerClient.Close(); err != nil {
				t.Error(err)
			}
		})

		require.NoError(t, tc.mockPV.SignProposal(tc.chainID, want.ToProto()))
		require.NoError(t, tc.signerClient.SignProposal(tc.chainID, have.ToProto()))

		assert.Equal(t, want.Signature, have.Signature)
	}
}

func TestSignerGenerateVRFProof(t *testing.T) {
	message := []byte("hello, world")
	for _, tc := range getSignerTestCases(t, nil, true) {
		tc := tc
		t.Cleanup(func() {
			if err := tc.signerServer.Stop(); err != nil {
				t.Error(err)
			}
		})
		t.Cleanup(func() {
			if err := tc.signerClient.Close(); err != nil {
				t.Error(err)
			}
		})

		proof, err := tc.signerClient.GenerateVRFProof(message)
		require.Nil(t, err)
		require.True(t, len(proof) > 0)
		output, err := vrf.ProofToHash(vrf.Proof(proof))
		require.Nil(t, err)
		require.NotNil(t, output)
		pubKey, err := tc.signerClient.GetPubKey()
		require.Nil(t, err)
		ed25519PubKey, ok := pubKey.(ed25519.PubKey)
		require.True(t, ok)
		expected, err := vrf.Verify(ed25519PubKey, vrf.Proof(proof), message)
		require.Nil(t, err)
		assert.True(t, expected)
	}
}

func TestSignerVote(t *testing.T) {
	for _, tc := range getSignerTestCases(t, nil, true) {
		ts := time.Now()
		hash := tmrand.Bytes(tmhash.Size)
		valAddr := tmrand.Bytes(crypto.AddressSize)
		want := &types.Vote{
			Type:             tmproto.PrecommitType,
			Height:           1,
			Round:            2,
			BlockID:          types.BlockID{Hash: hash, PartSetHeader: types.PartSetHeader{Hash: hash, Total: 2}},
			Timestamp:        ts,
			ValidatorAddress: valAddr,
			ValidatorIndex:   1,
		}

		have := &types.Vote{
			Type:             tmproto.PrecommitType,
			Height:           1,
			Round:            2,
			BlockID:          types.BlockID{Hash: hash, PartSetHeader: types.PartSetHeader{Hash: hash, Total: 2}},
			Timestamp:        ts,
			ValidatorAddress: valAddr,
			ValidatorIndex:   1,
		}

		tc := tc
		t.Cleanup(func() {
			if err := tc.signerServer.Stop(); err != nil {
				t.Error(err)
			}
		})
		t.Cleanup(func() {
			if err := tc.signerClient.Close(); err != nil {
				t.Error(err)
			}
		})

		require.NoError(t, tc.mockPV.SignVote(tc.chainID, want.ToProto()))
		require.NoError(t, tc.signerClient.SignVote(tc.chainID, have.ToProto()))

		assert.Equal(t, want.Signature, have.Signature)
	}
}

func TestSignerVoteResetDeadline(t *testing.T) {
	for _, tc := range getSignerTestCases(t, nil, true) {
		ts := time.Now()
		hash := tmrand.Bytes(tmhash.Size)
		valAddr := tmrand.Bytes(crypto.AddressSize)
		want := &types.Vote{
			Type:             tmproto.PrecommitType,
			Height:           1,
			Round:            2,
			BlockID:          types.BlockID{Hash: hash, PartSetHeader: types.PartSetHeader{Hash: hash, Total: 2}},
			Timestamp:        ts,
			ValidatorAddress: valAddr,
			ValidatorIndex:   1,
		}

		have := &types.Vote{
			Type:             tmproto.PrecommitType,
			Height:           1,
			Round:            2,
			BlockID:          types.BlockID{Hash: hash, PartSetHeader: types.PartSetHeader{Hash: hash, Total: 2}},
			Timestamp:        ts,
			ValidatorAddress: valAddr,
			ValidatorIndex:   1,
		}

		tc := tc
		t.Cleanup(func() {
			if err := tc.signerServer.Stop(); err != nil {
				t.Error(err)
			}
		})
		t.Cleanup(func() {
			if err := tc.signerClient.Close(); err != nil {
				t.Error(err)
			}
		})

		time.Sleep(testTimeoutReadWrite2o3)

		require.NoError(t, tc.mockPV.SignVote(tc.chainID, want.ToProto()))
		require.NoError(t, tc.signerClient.SignVote(tc.chainID, have.ToProto()))
		assert.Equal(t, want.Signature, have.Signature)

		// TODO(jleni): Clarify what is actually being tested

		// This would exceed the deadline if it was not extended by the previous message
		time.Sleep(testTimeoutReadWrite2o3)

		require.NoError(t, tc.mockPV.SignVote(tc.chainID, want.ToProto()))
		require.NoError(t, tc.signerClient.SignVote(tc.chainID, have.ToProto()))
		assert.Equal(t, want.Signature, have.Signature)
	}
}

func TestSignerVoteKeepAlive(t *testing.T) {
	for _, tc := range getSignerTestCases(t, nil, true) {
		ts := time.Now()
		hash := tmrand.Bytes(tmhash.Size)
		valAddr := tmrand.Bytes(crypto.AddressSize)
		want := &types.Vote{
			Type:             tmproto.PrecommitType,
			Height:           1,
			Round:            2,
			BlockID:          types.BlockID{Hash: hash, PartSetHeader: types.PartSetHeader{Hash: hash, Total: 2}},
			Timestamp:        ts,
			ValidatorAddress: valAddr,
			ValidatorIndex:   1,
		}

		have := &types.Vote{
			Type:             tmproto.PrecommitType,
			Height:           1,
			Round:            2,
			BlockID:          types.BlockID{Hash: hash, PartSetHeader: types.PartSetHeader{Hash: hash, Total: 2}},
			Timestamp:        ts,
			ValidatorAddress: valAddr,
			ValidatorIndex:   1,
		}

		tc := tc
		t.Cleanup(func() {
			if err := tc.signerServer.Stop(); err != nil {
				t.Error(err)
			}
		})
		t.Cleanup(func() {
			if err := tc.signerClient.Close(); err != nil {
				t.Error(err)
			}
		})

		// Check that even if the client does not request a
		// signature for a long time. The service is still available

		// in this particular case, we use the dialer logger to ensure that
		// test messages are properly interleaved in the test logs
		tc.signerServer.Logger.Debug("TEST: Forced Wait -------------------------------------------------")
		time.Sleep(testTimeoutReadWrite * 3)
		tc.signerServer.Logger.Debug("TEST: Forced Wait DONE---------------------------------------------")

		require.NoError(t, tc.mockPV.SignVote(tc.chainID, want.ToProto()))
		require.NoError(t, tc.signerClient.SignVote(tc.chainID, have.ToProto()))

		assert.Equal(t, want.Signature, have.Signature)
	}
}

func TestSignerSignProposalErrors(t *testing.T) {
	for _, tc := range getSignerTestCases(t, nil, true) {
		// Replace service with a mock that always fails
		tc.signerServer.privVal = types.NewErroringMockPV()
		tc.mockPV = types.NewErroringMockPV()

		tc := tc
		t.Cleanup(func() {
			if err := tc.signerServer.Stop(); err != nil {
				t.Error(err)
			}
		})
		t.Cleanup(func() {
			if err := tc.signerClient.Close(); err != nil {
				t.Error(err)
			}
		})

		ts := time.Now()
		hash := tmrand.Bytes(tmhash.Size)
		proposal := &types.Proposal{
			Type:      tmproto.ProposalType,
			Height:    1,
			Round:     2,
			POLRound:  2,
			BlockID:   types.BlockID{Hash: hash, PartSetHeader: types.PartSetHeader{Hash: hash, Total: 2}},
			Timestamp: ts,
			Signature: []byte("signature"),
		}

		err := tc.signerClient.SignProposal(tc.chainID, proposal.ToProto())
		require.Equal(t, err.(*RemoteSignerError).Description, types.ErroringMockPVErr.Error())

		err = tc.mockPV.SignProposal(tc.chainID, proposal.ToProto())
		require.Error(t, err)

		err = tc.signerClient.SignProposal(tc.chainID, proposal.ToProto())
		require.Error(t, err)
	}
}

func TestSignerSignVoteErrors(t *testing.T) {
	for _, tc := range getSignerTestCases(t, nil, true) {
		ts := time.Now()
		hash := tmrand.Bytes(tmhash.Size)
		valAddr := tmrand.Bytes(crypto.AddressSize)
		vote := &types.Vote{
			Type:             tmproto.PrecommitType,
			Height:           1,
			Round:            2,
			BlockID:          types.BlockID{Hash: hash, PartSetHeader: types.PartSetHeader{Hash: hash, Total: 2}},
			Timestamp:        ts,
			ValidatorAddress: valAddr,
			ValidatorIndex:   1,
		}

		// Replace signer service privval with one that always fails
		tc.signerServer.privVal = types.NewErroringMockPV()
		tc.mockPV = types.NewErroringMockPV()

		tc := tc
		t.Cleanup(func() {
			if err := tc.signerServer.Stop(); err != nil {
				t.Error(err)
			}
		})
		t.Cleanup(func() {
			if err := tc.signerClient.Close(); err != nil {
				t.Error(err)
			}
		})

		err := tc.signerClient.SignVote(tc.chainID, vote.ToProto())
		require.Equal(t, err.(*RemoteSignerError).Description, types.ErroringMockPVErr.Error())

		err = tc.mockPV.SignVote(tc.chainID, vote.ToProto())
		require.Error(t, err)

		err = tc.signerClient.SignVote(tc.chainID, vote.ToProto())
		require.Error(t, err)
	}
}

func brokenHandler(privVal types.PrivValidator, request privvalproto.Message,
	chainID string) (privvalproto.Message, error) {
	var res privvalproto.Message
	var err error

	switch r := request.Sum.(type) {
	// This is broken and will answer most requests with a pubkey response
	case *privvalproto.Message_PubKeyRequest:
		res = mustWrapMsg(&privvalproto.PubKeyResponse{PubKey: cryptoproto.PublicKey{}, Error: nil})
	case *privvalproto.Message_SignVoteRequest:
		res = mustWrapMsg(&privvalproto.PubKeyResponse{PubKey: cryptoproto.PublicKey{}, Error: nil})
	case *privvalproto.Message_SignProposalRequest:
		res = mustWrapMsg(&privvalproto.PubKeyResponse{PubKey: cryptoproto.PublicKey{}, Error: nil})
	case *privvalproto.Message_PingRequest:
		err, res = nil, mustWrapMsg(&privvalproto.PingResponse{})
	default:
		err = fmt.Errorf("unknown msg: %v", r)
	}

	return res, err
}

func TestSignerUnexpectedResponse(t *testing.T) {
	for _, tc := range getSignerTestCases(t, types.NewMockPV(), false) {
		tc.signerServer.SetRequestHandler(brokenHandler)
		err := tc.signerServer.Start()
		if err != nil {
			panic(err)
		}

		tc := tc
		t.Cleanup(func() {
			if err := tc.signerServer.Stop(); err != nil {
				t.Error(err)
			}
		})
		t.Cleanup(func() {
			if err := tc.signerClient.Close(); err != nil {
				t.Error(err)
			}
		})

		ts := time.Now()
		want := &types.Vote{Timestamp: ts, Type: tmproto.PrecommitType}

		e := tc.signerClient.SignVote(tc.chainID, want.ToProto())
		assert.EqualError(t, e, "empty response")
	}
}
