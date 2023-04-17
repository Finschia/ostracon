package privval

import (
	"fmt"

	"github.com/gogo/protobuf/proto"

	privvalproto "github.com/tendermint/tendermint/proto/tendermint/privval"

	ocprivvalproto "github.com/Finschia/ostracon/proto/ostracon/privval"
)

// TODO: Add ChainIDRequest

func mustWrapMsg(pb proto.Message) ocprivvalproto.Message {
	msg := ocprivvalproto.Message{}

	switch pb := pb.(type) {
	case *ocprivvalproto.Message:
		msg = *pb
	case *privvalproto.PubKeyRequest:
		msg.Sum = &ocprivvalproto.Message_PubKeyRequest{PubKeyRequest: pb}
	case *privvalproto.PubKeyResponse:
		msg.Sum = &ocprivvalproto.Message_PubKeyResponse{PubKeyResponse: pb}
	case *privvalproto.SignVoteRequest:
		msg.Sum = &ocprivvalproto.Message_SignVoteRequest{SignVoteRequest: pb}
	case *privvalproto.SignedVoteResponse:
		msg.Sum = &ocprivvalproto.Message_SignedVoteResponse{SignedVoteResponse: pb}
	case *privvalproto.SignedProposalResponse:
		msg.Sum = &ocprivvalproto.Message_SignedProposalResponse{SignedProposalResponse: pb}
	case *privvalproto.SignProposalRequest:
		msg.Sum = &ocprivvalproto.Message_SignProposalRequest{SignProposalRequest: pb}
	case *ocprivvalproto.VRFProofRequest:
		msg.Sum = &ocprivvalproto.Message_VrfProofRequest{VrfProofRequest: pb}
	case *ocprivvalproto.VRFProofResponse:
		msg.Sum = &ocprivvalproto.Message_VrfProofResponse{VrfProofResponse: pb}
	case *privvalproto.PingRequest:
		msg.Sum = &ocprivvalproto.Message_PingRequest{PingRequest: pb}
	case *privvalproto.PingResponse:
		msg.Sum = &ocprivvalproto.Message_PingResponse{PingResponse: pb}
	default:
		panic(fmt.Errorf("unknown message type %T", pb))
	}

	return msg
}
