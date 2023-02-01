package privval

import (
	"fmt"

	cryptoproto "github.com/tendermint/tendermint/proto/tendermint/crypto"
	privvalproto "github.com/tendermint/tendermint/proto/tendermint/privval"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/line/ostracon/crypto"
	cryptoenc "github.com/line/ostracon/crypto/encoding"
	ocprivvalproto "github.com/line/ostracon/proto/ostracon/privval"
	"github.com/line/ostracon/types"
)

func DefaultValidationRequestHandler(
	privVal types.PrivValidator,
	req ocprivvalproto.Message,
	chainID string,
) (ocprivvalproto.Message, error) {
	var (
		res ocprivvalproto.Message
		err error
	)

	switch r := req.Sum.(type) {
	case *ocprivvalproto.Message_PubKeyRequest:
		if r.PubKeyRequest.GetChainId() != chainID {
			res = mustWrapMsg(&privvalproto.PubKeyResponse{
				PubKey: cryptoproto.PublicKey{}, Error: &privvalproto.RemoteSignerError{
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
			res = mustWrapMsg(&privvalproto.PubKeyResponse{
				PubKey: cryptoproto.PublicKey{}, Error: &privvalproto.RemoteSignerError{Code: 0, Description: err.Error()}})
		} else {
			res = mustWrapMsg(&privvalproto.PubKeyResponse{PubKey: pk, Error: nil})
		}

	case *ocprivvalproto.Message_SignVoteRequest:
		if r.SignVoteRequest.ChainId != chainID {
			res = mustWrapMsg(&privvalproto.SignedVoteResponse{
				Vote: tmproto.Vote{}, Error: &privvalproto.RemoteSignerError{
					Code: 0, Description: "unable to sign vote"}})
			return res, fmt.Errorf("want chainID: %s, got chainID: %s", r.SignVoteRequest.GetChainId(), chainID)
		}

		vote := r.SignVoteRequest.Vote

		err = privVal.SignVote(chainID, vote)
		if err != nil {
			res = mustWrapMsg(&privvalproto.SignedVoteResponse{
				Vote: tmproto.Vote{}, Error: &privvalproto.RemoteSignerError{Code: 0, Description: err.Error()}})
		} else {
			res = mustWrapMsg(&privvalproto.SignedVoteResponse{Vote: *vote, Error: nil})
		}

	case *ocprivvalproto.Message_SignProposalRequest:
		if r.SignProposalRequest.GetChainId() != chainID {
			res = mustWrapMsg(&privvalproto.SignedProposalResponse{
				Proposal: tmproto.Proposal{}, Error: &privvalproto.RemoteSignerError{
					Code:        0,
					Description: "unable to sign proposal"}})
			return res, fmt.Errorf("want chainID: %s, got chainID: %s", r.SignProposalRequest.GetChainId(), chainID)
		}

		proposal := r.SignProposalRequest.Proposal

		err = privVal.SignProposal(chainID, proposal)
		if err != nil {
			res = mustWrapMsg(&privvalproto.SignedProposalResponse{
				Proposal: tmproto.Proposal{}, Error: &privvalproto.RemoteSignerError{Code: 0, Description: err.Error()}})
		} else {
			res = mustWrapMsg(&privvalproto.SignedProposalResponse{Proposal: *proposal, Error: nil})
		}
	case *ocprivvalproto.Message_PingRequest:
		err, res = nil, mustWrapMsg(&privvalproto.PingResponse{})

	case *ocprivvalproto.Message_VrfProofRequest:
		proof, err := privVal.GenerateVRFProof(r.VrfProofRequest.Message)
		if err != nil {
			err := privvalproto.RemoteSignerError{Code: 0, Description: err.Error()}
			res = mustWrapMsg(&ocprivvalproto.VRFProofResponse{Proof: nil, Error: &err})
		} else {
			res = mustWrapMsg(&ocprivvalproto.VRFProofResponse{Proof: proof[:], Error: nil})
		}

	default:
		err = fmt.Errorf("unknown msg: %v", r)
	}

	return res, err
}
