package blockchain

import (
	"errors"
	"fmt"

	"github.com/gogo/protobuf/proto"
	bcproto "github.com/tendermint/tendermint/proto/tendermint/blockchain"

	ocbcproto "github.com/Finschia/ostracon/proto/ostracon/blockchain"
	"github.com/Finschia/ostracon/types"
)

const (
	// NOTE: keep up to date with ocbcproto.BlockResponse
	BlockResponseMessagePrefixSize   = 4
	BlockResponseMessageFieldKeySize = 1
	MaxMsgSize                       = types.MaxBlockSizeBytes +
		BlockResponseMessagePrefixSize +
		BlockResponseMessageFieldKeySize
)

// EncodeMsg encodes a Protobuf message
func EncodeMsg(pb proto.Message) ([]byte, error) {
	msg := ocbcproto.Message{}

	switch pb := pb.(type) {
	case *bcproto.BlockRequest:
		msg.Sum = &ocbcproto.Message_BlockRequest{BlockRequest: pb}
	case *ocbcproto.BlockResponse:
		msg.Sum = &ocbcproto.Message_BlockResponse{BlockResponse: pb}
	case *bcproto.NoBlockResponse:
		msg.Sum = &ocbcproto.Message_NoBlockResponse{NoBlockResponse: pb}
	case *bcproto.StatusRequest:
		msg.Sum = &ocbcproto.Message_StatusRequest{StatusRequest: pb}
	case *bcproto.StatusResponse:
		msg.Sum = &ocbcproto.Message_StatusResponse{StatusResponse: pb}
	default:
		return nil, fmt.Errorf("unknown message type %T", pb)
	}

	bz, err := proto.Marshal(&msg)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal %T: %w", pb, err)
	}

	return bz, nil
}

// DecodeMsg decodes a Protobuf message.
func DecodeMsg(bz []byte) (proto.Message, error) {
	pb := &ocbcproto.Message{}

	err := proto.Unmarshal(bz, pb)
	if err != nil {
		return nil, err
	}

	switch msg := pb.Sum.(type) {
	case *ocbcproto.Message_BlockRequest:
		return msg.BlockRequest, nil
	case *ocbcproto.Message_BlockResponse:
		return msg.BlockResponse, nil
	case *ocbcproto.Message_NoBlockResponse:
		return msg.NoBlockResponse, nil
	case *ocbcproto.Message_StatusRequest:
		return msg.StatusRequest, nil
	case *ocbcproto.Message_StatusResponse:
		return msg.StatusResponse, nil
	default:
		return nil, fmt.Errorf("unknown message type %T", msg)
	}
}

// ValidateMsg validates a message.
func ValidateMsg(pb proto.Message) error {
	if pb == nil {
		return errors.New("message cannot be nil")
	}

	switch msg := pb.(type) {
	case *bcproto.BlockRequest:
		if msg.Height < 0 {
			return errors.New("negative Height")
		}
	case *ocbcproto.BlockResponse:
		_, err := types.BlockFromProto(msg.Block)
		if err != nil {
			return err
		}
	case *bcproto.NoBlockResponse:
		if msg.Height < 0 {
			return errors.New("negative Height")
		}
	case *bcproto.StatusResponse:
		if msg.Base < 0 {
			return errors.New("negative Base")
		}
		if msg.Height < 0 {
			return errors.New("negative Height")
		}
		if msg.Base > msg.Height {
			return fmt.Errorf("base %v cannot be greater than height %v", msg.Base, msg.Height)
		}
	case *bcproto.StatusRequest:
		return nil
	default:
		return fmt.Errorf("unknown message type %T", msg)
	}
	return nil
}
