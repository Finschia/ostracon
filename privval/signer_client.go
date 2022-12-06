package privval

import (
	"fmt"
	"time"

	"github.com/line/ostracon/crypto"
	cryptoenc "github.com/line/ostracon/crypto/encoding"
	privvalproto "github.com/line/ostracon/proto/ostracon/privval"
	tmproto "github.com/line/ostracon/proto/ostracon/types"
	"github.com/line/ostracon/types"
)

// SignerClient implements PrivValidator.
// Handles remote validator connections that provide signing services
type SignerClient struct {
	endpoint *SignerListenerEndpoint
	chainID  string
}

var _ types.PrivValidator = (*SignerClient)(nil)

// NewSignerClient returns an instance of SignerClient.
// it will start the endpoint (if not already started)
func NewSignerClient(endpoint *SignerListenerEndpoint, chainID string) (*SignerClient, error) {
	if !endpoint.IsRunning() {
		if err := endpoint.Start(); err != nil {
			return nil, fmt.Errorf("failed to start listener endpoint: %w", err)
		}
	}

	return &SignerClient{endpoint: endpoint, chainID: chainID}, nil
}

// Close closes the underlying connection
func (sc *SignerClient) Close() error {
	return sc.endpoint.Close()
}

// IsConnected indicates with the signer is connected to a remote signing service
func (sc *SignerClient) IsConnected() bool {
	return sc.endpoint.IsConnected()
}

// WaitForConnection waits maxWait for a connection or returns a timeout error
func (sc *SignerClient) WaitForConnection(maxWait time.Duration) error {
	return sc.endpoint.WaitForConnection(maxWait)
}

//--------------------------------------------------------
// Implement PrivValidator

// GetPubKey retrieves a public key from a remote signer
// returns an error if client is not able to provide the key
func (sc *SignerClient) GetPubKey() (crypto.PubKey, error) {
	response, err := sc.endpoint.SendRequest(mustWrapMsg(&privvalproto.PubKeyRequest{ChainId: sc.chainID}))
	if err != nil {
		return nil, fmt.Errorf("send: %w", err)
	}

	resp := response.GetPubKeyResponse()
	if resp == nil {
		return nil, ErrUnexpectedResponse
	}
	if resp.Error != nil {
		return nil, &RemoteSignerError{Code: int(resp.Error.Code), Description: resp.Error.Description}
	}

	pk, err := cryptoenc.PubKeyFromProto(&resp.PubKey)
	if err != nil {
		return nil, err
	}

	return pk, nil
}

// SignVote requests a remote signer to sign a vote
func (sc *SignerClient) SignVote(chainID string, vote *tmproto.Vote) error {
	response, err := sc.endpoint.SendRequest(mustWrapMsg(&privvalproto.SignVoteRequest{Vote: vote, ChainId: chainID}))
	if err != nil {
		return err
	}

	resp := response.GetSignedVoteResponse()
	if resp == nil {
		return ErrUnexpectedResponse
	}
	if resp.Error != nil {
		return &RemoteSignerError{Code: int(resp.Error.Code), Description: resp.Error.Description}
	}

	*vote = resp.Vote

	return nil
}

// SignProposal requests a remote signer to sign a proposal
func (sc *SignerClient) SignProposal(chainID string, proposal *tmproto.Proposal) error {
	response, err := sc.endpoint.SendRequest(mustWrapMsg(
		&privvalproto.SignProposalRequest{Proposal: proposal, ChainId: chainID},
	))
	if err != nil {
		return err
	}

	resp := response.GetSignedProposalResponse()
	if resp == nil {
		return ErrUnexpectedResponse
	}
	if resp.Error != nil {
		return &RemoteSignerError{Code: int(resp.Error.Code), Description: resp.Error.Description}
	}

	*proposal = resp.Proposal

	return nil
}

// GenerateVRFProof requests a remote signer to generate a VRF proof
func (sc *SignerClient) GenerateVRFProof(message []byte) (crypto.Proof, error) {
	msg := &privvalproto.VRFProofRequest{Message: message}
	response, err := sc.endpoint.SendRequest(mustWrapMsg(msg))
	if err != nil {
		sc.endpoint.Logger.Error("SignerClient::GenerateVRFProof", "err", err)
		return nil, err
	}

	switch r := response.Sum.(type) {
	case *privvalproto.Message_VrfProofResponse:
		if r.VrfProofResponse.Error != nil {
			return nil, fmt.Errorf(r.VrfProofResponse.Error.Description)
		}
		return r.VrfProofResponse.Proof, nil
	default:
		sc.endpoint.Logger.Error("SignerClient::GenerateVRFProof", "err", "response != VRFProofResponse")
		return nil, ErrUnexpectedResponse
	}
}
