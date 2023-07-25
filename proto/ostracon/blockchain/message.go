package blockchain

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/tendermint/tendermint/p2p"
)

var _ p2p.Wrapper = &BlockResponse{}

func (m *BlockResponse) Wrap() proto.Message {
	bm := &Message{}
	bm.Sum = &Message_BlockResponse{BlockResponse: m}
	return bm
}

// Unwrap implements the p2p Wrapper interface and unwraps a wrapped blockchain
// message.
func (m *Message) Unwrap() (proto.Message, error) {
	switch msg := m.Sum.(type) {
	case *Message_BlockRequest:
		return m.GetBlockRequest(), nil

	case *Message_BlockResponse:
		return m.GetBlockResponse(), nil

	case *Message_NoBlockResponse:
		return m.GetNoBlockResponse(), nil

	case *Message_StatusRequest:
		return m.GetStatusRequest(), nil

	case *Message_StatusResponse:
		return m.GetStatusResponse(), nil

	default:
		return nil, fmt.Errorf("unknown message: %T", msg)
	}
}
