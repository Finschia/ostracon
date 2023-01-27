package types

import (
	"bufio"
	"encoding/binary"
	"io"

	"github.com/gogo/protobuf/proto"

	"github.com/tendermint/tendermint/abci/types"
)

const (
	maxMsgSize = 104857600 // 100MB
)

// WriteMessage writes a varint length-delimited protobuf message.
func WriteMessage(msg proto.Message, w io.Writer) error {
	bz, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	return encodeByteSlice(w, bz)
}

// ReadMessage reads a varint length-delimited protobuf message.
func ReadMessage(r io.Reader, msg proto.Message) error {
	return readProtoMsg(r, msg, maxMsgSize)
}

func readProtoMsg(r io.Reader, msg proto.Message, maxSize int) error {
	// binary.ReadVarint takes an io.ByteReader, eg. a bufio.Reader
	reader, ok := r.(*bufio.Reader)
	if !ok {
		reader = bufio.NewReader(r)
	}
	length64, err := binary.ReadVarint(reader)
	if err != nil {
		return err
	}
	length := int(length64)
	if length < 0 || length > maxSize {
		return io.ErrShortBuffer
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(reader, buf); err != nil {
		return err
	}
	return proto.Unmarshal(buf, msg)
}

//-----------------------------------------------------------------------
// NOTE: we copied wire.EncodeByteSlice from go-wire rather than keep
// go-wire as a dep

func encodeByteSlice(w io.Writer, bz []byte) (err error) {
	err = encodeVarint(w, int64(len(bz)))
	if err != nil {
		return
	}
	_, err = w.Write(bz)
	return
}

func encodeVarint(w io.Writer, i int64) (err error) {
	var buf [10]byte
	n := binary.PutVarint(buf[:], i)
	_, err = w.Write(buf[0:n])
	return
}

//----------------------------------------

func ToRequestEcho(message string) *Request {
	return &Request{
		Value: &Request_Echo{&types.RequestEcho{Message: message}},
	}
}

func ToRequestFlush() *Request {
	return &Request{
		Value: &Request_Flush{&types.RequestFlush{}},
	}
}

func ToRequestInfo(req types.RequestInfo) *Request {
	return &Request{
		Value: &Request_Info{&req},
	}
}

func ToRequestSetOption(req types.RequestSetOption) *Request {
	return &Request{
		Value: &Request_SetOption{&req},
	}
}

func ToRequestDeliverTx(req types.RequestDeliverTx) *Request {
	return &Request{
		Value: &Request_DeliverTx{&req},
	}
}

func ToRequestCheckTx(req types.RequestCheckTx) *Request {
	return &Request{
		Value: &Request_CheckTx{&req},
	}
}

func ToRequestCommit() *Request {
	return &Request{
		Value: &Request_Commit{&types.RequestCommit{}},
	}
}

func ToRequestQuery(req types.RequestQuery) *Request {
	return &Request{
		Value: &Request_Query{&req},
	}
}

func ToRequestInitChain(req types.RequestInitChain) *Request {
	return &Request{
		Value: &Request_InitChain{&req},
	}
}

func ToRequestBeginBlock(req RequestBeginBlock) *Request {
	return &Request{
		Value: &Request_BeginBlock{&req},
	}
}

func ToRequestEndBlock(req types.RequestEndBlock) *Request {
	return &Request{
		Value: &Request_EndBlock{&req},
	}
}

func ToRequestBeginRecheckTx(req RequestBeginRecheckTx) *Request {
	return &Request{
		Value: &Request_BeginRecheckTx{&req},
	}
}

func ToRequestEndRecheckTx(req RequestEndRecheckTx) *Request {
	return &Request{
		Value: &Request_EndRecheckTx{&req},
	}
}

func ToRequestListSnapshots(req types.RequestListSnapshots) *Request {
	return &Request{
		Value: &Request_ListSnapshots{&req},
	}
}

func ToRequestOfferSnapshot(req types.RequestOfferSnapshot) *Request {
	return &Request{
		Value: &Request_OfferSnapshot{&req},
	}
}

func ToRequestLoadSnapshotChunk(req types.RequestLoadSnapshotChunk) *Request {
	return &Request{
		Value: &Request_LoadSnapshotChunk{&req},
	}
}

func ToRequestApplySnapshotChunk(req types.RequestApplySnapshotChunk) *Request {
	return &Request{
		Value: &Request_ApplySnapshotChunk{&req},
	}
}

//----------------------------------------

func ToResponseException(errStr string) *Response {
	return &Response{
		Value: &Response_Exception{&types.ResponseException{Error: errStr}},
	}
}

func ToResponseEcho(message string) *Response {
	return &Response{
		Value: &Response_Echo{&types.ResponseEcho{Message: message}},
	}
}

func ToResponseFlush() *Response {
	return &Response{
		Value: &Response_Flush{&types.ResponseFlush{}},
	}
}

func ToResponseInfo(res types.ResponseInfo) *Response {
	return &Response{
		Value: &Response_Info{&res},
	}
}

func ToResponseSetOption(res types.ResponseSetOption) *Response {
	return &Response{
		Value: &Response_SetOption{&res},
	}
}

func ToResponseDeliverTx(res types.ResponseDeliverTx) *Response {
	return &Response{
		Value: &Response_DeliverTx{&res},
	}
}

func ToResponseCheckTx(res ResponseCheckTx) *Response {
	return &Response{
		Value: &Response_CheckTx{&res},
	}
}

func ToResponseCommit(res types.ResponseCommit) *Response {
	return &Response{
		Value: &Response_Commit{&res},
	}
}

func ToResponseQuery(res types.ResponseQuery) *Response {
	return &Response{
		Value: &Response_Query{&res},
	}
}

func ToResponseInitChain(res types.ResponseInitChain) *Response {
	return &Response{
		Value: &Response_InitChain{&res},
	}
}

func ToResponseBeginBlock(res types.ResponseBeginBlock) *Response {
	return &Response{
		Value: &Response_BeginBlock{&res},
	}
}

func ToResponseEndBlock(res types.ResponseEndBlock) *Response {
	return &Response{
		Value: &Response_EndBlock{&res},
	}
}

func ToResponseBeginRecheckTx(res ResponseBeginRecheckTx) *Response {
	return &Response{
		Value: &Response_BeginRecheckTx{&res},
	}
}

func ToResponseEndRecheckTx(res ResponseEndRecheckTx) *Response {
	return &Response{
		Value: &Response_EndRecheckTx{&res},
	}
}

func ToResponseListSnapshots(res types.ResponseListSnapshots) *Response {
	return &Response{
		Value: &Response_ListSnapshots{&res},
	}
}

func ToResponseOfferSnapshot(res types.ResponseOfferSnapshot) *Response {
	return &Response{
		Value: &Response_OfferSnapshot{&res},
	}
}

func ToResponseLoadSnapshotChunk(res types.ResponseLoadSnapshotChunk) *Response {
	return &Response{
		Value: &Response_LoadSnapshotChunk{&res},
	}
}

func ToResponseApplySnapshotChunk(res types.ResponseApplySnapshotChunk) *Response {
	return &Response{
		Value: &Response_ApplySnapshotChunk{&res},
	}
}
