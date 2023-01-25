package privval

import (
	"fmt"

	cryptoproto "github.com/tendermint/tendermint/proto/tendermint/crypto"
	tmprivvalproto "github.com/tendermint/tendermint/proto/tendermint/privval"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/line/ostracon/crypto"
	cryptoenc "github.com/line/ostracon/crypto/encoding"
	privvalproto "github.com/line/ostracon/proto/ostracon/privval"
	"github.com/line/ostracon/types"
)

func DefaultValidationRequestHandler(
	privVal types.PrivValidator,
	req privvalproto.Message,
	chainID string,
) (privvalproto.Message, error) {
	var (
		res privvalproto.Message
		err error
	)

	switch r := req.Sum.(type) {
	case *privvalproto.Message_PubKeyRequest:
		if r.PubKeyRequest.GetChainId() != chainID {
			res = mustWrapMsg(&tmprivvalproto.PubKeyResponse{
				PubKey: cryptoproto.PublicKey{}, Error: &tmprivvalproto.RemoteSignerError{
					Code: 0, Description: "unable to provide pubkey"}})
			return res, fmt.Errorf("want chainID: %s, got chainID: %s", r.PubKeyRequest.GetChainId(), chainID)
		}

		var pubKey crypto.PubKey
		pubKey, err = privVal.GetPubKey()
		if err != nil {
			return res, err
		}
		pk, err := cryptoenc.PubKeyToProto(pubKey)
		if err != nil {
			return res, err
		}

		if err != nil {
			res = mustWrapMsg(&tmprivvalproto.PubKeyResponse{
				PubKey: cryptoproto.PublicKey{}, Error: &tmprivvalproto.RemoteSignerError{Code: 0, Description: err.Error()}})
		} else {
			res = mustWrapMsg(&tmprivvalproto.PubKeyResponse{PubKey: pk, Error: nil})
		}

	case *privvalproto.Message_SignVoteRequest:
		if r.SignVoteRequest.ChainId != chainID {
			res = mustWrapMsg(&tmprivvalproto.SignedVoteResponse{
				Vote: tmproto.Vote{}, Error: &tmprivvalproto.RemoteSignerError{
					Code: 0, Description: "unable to sign vote"}})
			return res, fmt.Errorf("want chainID: %s, got chainID: %s", r.SignVoteRequest.GetChainId(), chainID)
		}

		vote := r.SignVoteRequest.Vote

		err = privVal.SignVote(chainID, vote)
		if err != nil {
			res = mustWrapMsg(&tmprivvalproto.SignedVoteResponse{
				Vote: tmproto.Vote{}, Error: &tmprivvalproto.RemoteSignerError{Code: 0, Description: err.Error()}})
		} else {
			res = mustWrapMsg(&tmprivvalproto.SignedVoteResponse{Vote: *vote, Error: nil})
		}

	case *privvalproto.Message_SignProposalRequest:
		if r.SignProposalRequest.GetChainId() != chainID {
			res = mustWrapMsg(&tmprivvalproto.SignedProposalResponse{
				Proposal: tmproto.Proposal{}, Error: &tmprivvalproto.RemoteSignerError{
					Code:        0,
					Description: "unable to sign proposal"}})
			return res, fmt.Errorf("want chainID: %s, got chainID: %s", r.SignProposalRequest.GetChainId(), chainID)
		}

		proposal := r.SignProposalRequest.Proposal

		err = privVal.SignProposal(chainID, proposal)
		if err != nil {
			res = mustWrapMsg(&tmprivvalproto.SignedProposalResponse{
				Proposal: tmproto.Proposal{}, Error: &tmprivvalproto.RemoteSignerError{Code: 0, Description: err.Error()}})
		} else {
			res = mustWrapMsg(&tmprivvalproto.SignedProposalResponse{Proposal: *proposal, Error: nil})
		}
	case *privvalproto.Message_PingRequest:
		err, res = nil, mustWrapMsg(&tmprivvalproto.PingResponse{})

	case *privvalproto.Message_VrfProofRequest:
		proof, err := privVal.GenerateVRFProof(r.VrfProofRequest.Message)
		if err != nil {
			err := tmprivvalproto.RemoteSignerError{Code: 0, Description: err.Error()}
			res = mustWrapMsg(&privvalproto.VRFProofResponse{Proof: nil, Error: &err})
		} else {
			res = mustWrapMsg(&privvalproto.VRFProofResponse{Proof: proof[:], Error: nil})
		}

	default:
		err = fmt.Errorf("unknown msg: %v", r)
	}

	return res, err
}
