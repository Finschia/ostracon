package privval

import (
	"fmt"

	"github.com/gogo/protobuf/proto"

	tmprivvalproto "github.com/tendermint/tendermint/proto/tendermint/privval"

	privvalproto "github.com/line/ostracon/proto/ostracon/privval"
)

// TODO: Add ChainIDRequest

func mustWrapMsg(pb proto.Message) privvalproto.Message {
	msg := privvalproto.Message{}

	switch pb := pb.(type) {
	case *privvalproto.Message:
		msg = *pb
	case *tmprivvalproto.PubKeyRequest:
		msg.Sum = &privvalproto.Message_PubKeyRequest{PubKeyRequest: pb}
	case *privvalproto.PubKeyResponse:
		msg.Sum = &privvalproto.Message_PubKeyResponse{PubKeyResponse: pb}
	case *tmprivvalproto.SignVoteRequest:
		msg.Sum = &privvalproto.Message_SignVoteRequest{SignVoteRequest: pb}
	case *tmprivvalproto.SignedVoteResponse:
		msg.Sum = &privvalproto.Message_SignedVoteResponse{SignedVoteResponse: pb}
	case *tmprivvalproto.SignedProposalResponse:
		msg.Sum = &privvalproto.Message_SignedProposalResponse{SignedProposalResponse: pb}
	case *tmprivvalproto.SignProposalRequest:
		msg.Sum = &privvalproto.Message_SignProposalRequest{SignProposalRequest: pb}
	case *privvalproto.VRFProofRequest:
		msg.Sum = &privvalproto.Message_VrfProofRequest{VrfProofRequest: pb}
	case *privvalproto.VRFProofResponse:
		msg.Sum = &privvalproto.Message_VrfProofResponse{VrfProofResponse: pb}
	case *tmprivvalproto.PingRequest:
		msg.Sum = &privvalproto.Message_PingRequest{PingRequest: pb}
	case *tmprivvalproto.PingResponse:
		msg.Sum = &privvalproto.Message_PingResponse{PingResponse: pb}
	default:
		panic(fmt.Errorf("unknown message type %T", pb))
	}

	return msg
}
