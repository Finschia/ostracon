package abcicli

import (
	"bufio"
	"container/list"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
	"time"

	"github.com/line/ostracon/abci/types"
	tmnet "github.com/line/ostracon/libs/net"
	"github.com/line/ostracon/libs/service"
	tmsync "github.com/line/ostracon/libs/sync"
	"github.com/line/ostracon/libs/timer"
)

const (
	reqQueueSize    = 256 // TODO make configurable
	flushThrottleMS = 20  // Don't wait longer than...
)

// This is goroutine-safe, but users should beware that the application in
// general is not meant to be interfaced with concurrent callers.
type socketClient struct {
	service.BaseService

	addr        string
	mustConnect bool
	conn        net.Conn

	reqQueue   chan *ReqRes
	flushTimer *timer.ThrottleTimer

	mtx     tmsync.Mutex
	err     error
	reqSent *list.List // list of requests sent, waiting for response

	globalCbMtx tmsync.Mutex
	globalCb    GlobalCallback
}

var _ Client = (*socketClient)(nil)

// NewSocketClient creates a new socket client, which connects to a given
// address. If mustConnect is true, the client will return an error upon start
// if it fails to connect.
func NewSocketClient(addr string, mustConnect bool) Client {
	cli := &socketClient{
		reqQueue:    make(chan *ReqRes, reqQueueSize),
		flushTimer:  timer.NewThrottleTimer("socketClient", flushThrottleMS),
		mustConnect: mustConnect,

		addr:     addr,
		reqSent:  list.New(),
		globalCb: nil,
	}
	cli.BaseService = *service.NewBaseService(nil, "socketClient", cli)
	return cli
}

// OnStart implements Service by connecting to the server and spawning reading
// and writing goroutines.
func (cli *socketClient) OnStart() error {
	var (
		err  error
		conn net.Conn
	)

	for {
		conn, err = tmnet.Connect(cli.addr)
		if err != nil {
			if cli.mustConnect {
				return err
			}
			cli.Logger.Error(fmt.Sprintf("abci.socketClient failed to connect to %v.  Retrying after %vs...",
				cli.addr, dialRetryIntervalSeconds), "err", err)
			time.Sleep(time.Second * dialRetryIntervalSeconds)
			continue
		}
		cli.conn = conn

		go cli.sendRequestsRoutine(conn)
		go cli.recvResponseRoutine(conn)

		return nil
	}
}

// OnStop implements Service by closing connection and flushing all queues.
func (cli *socketClient) OnStop() {
	if cli.conn != nil {
		cli.conn.Close()
	}

	cli.flushQueue()
	cli.flushTimer.Stop()
}

// Error returns an error if the client was stopped abruptly.
func (cli *socketClient) Error() error {
	cli.mtx.Lock()
	defer cli.mtx.Unlock()
	return cli.err
}

func (cli *socketClient) SetGlobalCallback(globalCb GlobalCallback) {
	cli.globalCbMtx.Lock()
	cli.globalCb = globalCb
	cli.globalCbMtx.Unlock()
}

func (cli *socketClient) GetGlobalCallback() (cb GlobalCallback) {
	cli.globalCbMtx.Lock()
	cb = cli.globalCb
	cli.globalCbMtx.Unlock()
	return cb
}

//----------------------------------------

func (cli *socketClient) sendRequestsRoutine(conn io.Writer) {
	w := bufio.NewWriter(conn)
	for {
		select {
		case reqres := <-cli.reqQueue:
			// cli.Logger.Debug("Sent request", "requestType", reflect.TypeOf(reqres.Request), "request", reqres.Request)

			cli.willSendReq(reqres)
			err := types.WriteMessage(reqres.Request, w)
			if err != nil {
				cli.stopForError(fmt.Errorf("write to buffer: %w", err))
				return
			}

			// If it's a flush request, flush the current buffer.
			if _, ok := reqres.Request.Value.(*types.Request_Flush); ok {
				err = w.Flush()
				if err != nil {
					cli.stopForError(fmt.Errorf("flush buffer: %w", err))
					return
				}
			}
		case <-cli.flushTimer.Ch: // flush queue
			select {
			case cli.reqQueue <- NewReqRes(types.ToRequestFlush(), nil):
			default:
				// Probably will fill the buffer, or retry later.
			}
		case <-cli.Quit():
			return
		}
	}
}

func (cli *socketClient) recvResponseRoutine(conn io.Reader) {
	r := bufio.NewReader(conn)
	for {
		var res = &types.Response{}
		err := types.ReadMessage(r, res)
		if err != nil {
			cli.stopForError(fmt.Errorf("read message: %w", err))
			return
		}

		// cli.Logger.Debug("Received response", "responseType", reflect.TypeOf(res), "response", res)

		switch r := res.Value.(type) {
		case *types.Response_Exception: // app responded with error
			// XXX After setting cli.err, release waiters (e.g. reqres.Done())
			cli.stopForError(errors.New(r.Exception.Error))
			return
		default:
			err := cli.didRecvResponse(res)
			if err != nil {
				cli.stopForError(err)
				return
			}
		}
	}
}

func (cli *socketClient) willSendReq(reqres *ReqRes) {
	cli.mtx.Lock()
	defer cli.mtx.Unlock()
	cli.reqSent.PushBack(reqres)
}

func (cli *socketClient) didRecvResponse(res *types.Response) error {
	cli.mtx.Lock()
	defer cli.mtx.Unlock()

	// Get the first ReqRes.
	next := cli.reqSent.Front()
	if next == nil {
		return fmt.Errorf("unexpected %v when nothing expected", reflect.TypeOf(res.Value))
	}

	reqres := next.Value.(*ReqRes)
	if !resMatchesReq(reqres.Request, res) {
		return fmt.Errorf("unexpected %v when response to %v expected",
			reflect.TypeOf(res.Value), reflect.TypeOf(reqres.Request.Value))
	}

	reqres.Response = res
	reqres.wg.Done()         // release waiters
	cli.reqSent.Remove(next) // pop first item from linked list

	// Notify client listener if set (global callback).
	if cli.globalCb != nil {
		cli.globalCb(reqres.Request, res)
	}

	// Notify reqRes listener if set (request specific callback).
	//
	// NOTE: It is possible this callback isn't set on the reqres object. At this
	// point, in which case it will be called after, when it is set.
	reqres.InvokeCallback()

	return nil
}

//----------------------------------------

func (cli *socketClient) EchoAsync(msg string, cb ResponseCallback) *ReqRes {
	return cli.queueRequest(types.ToRequestEcho(msg), cb)
}

func (cli *socketClient) FlushAsync(cb ResponseCallback) *ReqRes {
	return cli.queueRequest(types.ToRequestFlush(), cb)
}

func (cli *socketClient) InfoAsync(req types.RequestInfo, cb ResponseCallback) *ReqRes {
	return cli.queueRequest(types.ToRequestInfo(req), cb)
}

func (cli *socketClient) SetOptionAsync(req types.RequestSetOption, cb ResponseCallback) *ReqRes {
	return cli.queueRequest(types.ToRequestSetOption(req), cb)
}

func (cli *socketClient) DeliverTxAsync(req types.RequestDeliverTx, cb ResponseCallback) *ReqRes {
	return cli.queueRequest(types.ToRequestDeliverTx(req), cb)
}

func (cli *socketClient) CheckTxAsync(req types.RequestCheckTx, cb ResponseCallback) *ReqRes {
	return cli.queueRequest(types.ToRequestCheckTx(req), cb)
}

func (cli *socketClient) QueryAsync(req types.RequestQuery, cb ResponseCallback) *ReqRes {
	return cli.queueRequest(types.ToRequestQuery(req), cb)
}

func (cli *socketClient) CommitAsync(cb ResponseCallback) *ReqRes {
	return cli.queueRequest(types.ToRequestCommit(), cb)
}

func (cli *socketClient) InitChainAsync(req types.RequestInitChain, cb ResponseCallback) *ReqRes {
	return cli.queueRequest(types.ToRequestInitChain(req), cb)
}

func (cli *socketClient) BeginBlockAsync(req types.RequestBeginBlock, cb ResponseCallback) *ReqRes {
	return cli.queueRequest(types.ToRequestBeginBlock(req), cb)
}

func (cli *socketClient) EndBlockAsync(req types.RequestEndBlock, cb ResponseCallback) *ReqRes {
	return cli.queueRequest(types.ToRequestEndBlock(req), cb)
}

func (cli *socketClient) BeginRecheckTxAsync(req types.RequestBeginRecheckTx, cb ResponseCallback) *ReqRes {
	return cli.queueRequest(types.ToRequestBeginRecheckTx(req), cb)
}

func (cli *socketClient) EndRecheckTxAsync(req types.RequestEndRecheckTx, cb ResponseCallback) *ReqRes {
	return cli.queueRequest(types.ToRequestEndRecheckTx(req), cb)
}

func (cli *socketClient) ListSnapshotsAsync(req types.RequestListSnapshots, cb ResponseCallback) *ReqRes {
	return cli.queueRequest(types.ToRequestListSnapshots(req), cb)
}

func (cli *socketClient) OfferSnapshotAsync(req types.RequestOfferSnapshot, cb ResponseCallback) *ReqRes {
	return cli.queueRequest(types.ToRequestOfferSnapshot(req), cb)
}

func (cli *socketClient) LoadSnapshotChunkAsync(req types.RequestLoadSnapshotChunk, cb ResponseCallback) *ReqRes {
	return cli.queueRequest(types.ToRequestLoadSnapshotChunk(req), cb)
}

func (cli *socketClient) ApplySnapshotChunkAsync(req types.RequestApplySnapshotChunk, cb ResponseCallback) *ReqRes {
	return cli.queueRequest(types.ToRequestApplySnapshotChunk(req), cb)
}

//----------------------------------------

func (cli *socketClient) FlushSync() (*types.ResponseFlush, error) {
	reqRes := cli.queueRequest(types.ToRequestFlush(), nil)
	if err := cli.Error(); err != nil {
		return nil, err
	}
	reqRes.Wait() // NOTE: if we don't flush the queue, its possible to get stuck here
	return reqRes.Response.GetFlush(), cli.Error()
}

func (cli *socketClient) EchoSync(msg string) (*types.ResponseEcho, error) {
	reqres := cli.queueRequest(types.ToRequestEcho(msg), nil)
	if _, err := cli.FlushSync(); err != nil {
		return nil, err
	}

	return reqres.Response.GetEcho(), cli.Error()
}

func (cli *socketClient) InfoSync(req types.RequestInfo) (*types.ResponseInfo, error) {
	reqres := cli.queueRequest(types.ToRequestInfo(req), nil)
	if _, err := cli.FlushSync(); err != nil {
		return nil, err
	}

	return reqres.Response.GetInfo(), cli.Error()
}

func (cli *socketClient) SetOptionSync(req types.RequestSetOption) (*types.ResponseSetOption, error) {
	reqres := cli.queueRequest(types.ToRequestSetOption(req), nil)
	if _, err := cli.FlushSync(); err != nil {
		return nil, err
	}

	return reqres.Response.GetSetOption(), cli.Error()
}

func (cli *socketClient) DeliverTxSync(req types.RequestDeliverTx) (*types.ResponseDeliverTx, error) {
	reqres := cli.queueRequest(types.ToRequestDeliverTx(req), nil)
	if _, err := cli.FlushSync(); err != nil {
		return nil, err
	}

	return reqres.Response.GetDeliverTx(), cli.Error()
}

func (cli *socketClient) CheckTxSync(req types.RequestCheckTx) (*types.ResponseCheckTx, error) {
	reqres := cli.queueRequest(types.ToRequestCheckTx(req), nil)
	if _, err := cli.FlushSync(); err != nil {
		return nil, err
	}

	return reqres.Response.GetCheckTx(), cli.Error()
}

func (cli *socketClient) QuerySync(req types.RequestQuery) (*types.ResponseQuery, error) {
	reqres := cli.queueRequest(types.ToRequestQuery(req), nil)
	if _, err := cli.FlushSync(); err != nil {
		return nil, err
	}

	return reqres.Response.GetQuery(), cli.Error()
}

func (cli *socketClient) CommitSync() (*types.ResponseCommit, error) {
	reqres := cli.queueRequest(types.ToRequestCommit(), nil)
	if _, err := cli.FlushSync(); err != nil {
		return nil, err
	}

	return reqres.Response.GetCommit(), cli.Error()
}

func (cli *socketClient) InitChainSync(req types.RequestInitChain) (*types.ResponseInitChain, error) {
	reqres := cli.queueRequest(types.ToRequestInitChain(req), nil)
	if _, err := cli.FlushSync(); err != nil {
		return nil, err
	}

	return reqres.Response.GetInitChain(), cli.Error()
}

func (cli *socketClient) BeginBlockSync(req types.RequestBeginBlock) (*types.ResponseBeginBlock, error) {
	reqres := cli.queueRequest(types.ToRequestBeginBlock(req), nil)
	if _, err := cli.FlushSync(); err != nil {
		return nil, err
	}

	return reqres.Response.GetBeginBlock(), cli.Error()
}

func (cli *socketClient) EndBlockSync(req types.RequestEndBlock) (*types.ResponseEndBlock, error) {
	reqres := cli.queueRequest(types.ToRequestEndBlock(req), nil)
	if _, err := cli.FlushSync(); err != nil {
		return nil, err
	}

	return reqres.Response.GetEndBlock(), cli.Error()
}

func (cli *socketClient) BeginRecheckTxSync(req types.RequestBeginRecheckTx) (*types.ResponseBeginRecheckTx, error) {
	reqres := cli.queueRequest(types.ToRequestBeginRecheckTx(req), nil)
	if _, err := cli.FlushSync(); err != nil {
		return nil, err
	}

	return reqres.Response.GetBeginRecheckTx(), cli.Error()
}

func (cli *socketClient) EndRecheckTxSync(req types.RequestEndRecheckTx) (*types.ResponseEndRecheckTx, error) {
	reqres := cli.queueRequest(types.ToRequestEndRecheckTx(req), nil)
	if _, err := cli.FlushSync(); err != nil {
		return nil, err
	}

	return reqres.Response.GetEndRecheckTx(), cli.Error()
}

func (cli *socketClient) ListSnapshotsSync(req types.RequestListSnapshots) (*types.ResponseListSnapshots, error) {
	reqres := cli.queueRequest(types.ToRequestListSnapshots(req), nil)
	if _, err := cli.FlushSync(); err != nil {
		return nil, err
	}

	return reqres.Response.GetListSnapshots(), cli.Error()
}

func (cli *socketClient) OfferSnapshotSync(req types.RequestOfferSnapshot) (*types.ResponseOfferSnapshot, error) {
	reqres := cli.queueRequest(types.ToRequestOfferSnapshot(req), nil)
	if _, err := cli.FlushSync(); err != nil {
		return nil, err
	}

	return reqres.Response.GetOfferSnapshot(), cli.Error()
}

func (cli *socketClient) LoadSnapshotChunkSync(
	req types.RequestLoadSnapshotChunk) (*types.ResponseLoadSnapshotChunk, error) {
	reqres := cli.queueRequest(types.ToRequestLoadSnapshotChunk(req), nil)
	if _, err := cli.FlushSync(); err != nil {
		return nil, err
	}

	return reqres.Response.GetLoadSnapshotChunk(), cli.Error()
}

func (cli *socketClient) ApplySnapshotChunkSync(
	req types.RequestApplySnapshotChunk) (*types.ResponseApplySnapshotChunk, error) {
	reqres := cli.queueRequest(types.ToRequestApplySnapshotChunk(req), nil)
	if _, err := cli.FlushSync(); err != nil {
		return nil, err
	}
	return reqres.Response.GetApplySnapshotChunk(), cli.Error()
}

//----------------------------------------

func (cli *socketClient) queueRequest(req *types.Request, cb ResponseCallback) *ReqRes {
	reqres := NewReqRes(req, cb)

	// TODO: set cli.err if reqQueue times out
	cli.reqQueue <- reqres

	// Maybe auto-flush, or unset auto-flush
	switch req.Value.(type) {
	case *types.Request_Flush:
		cli.flushTimer.Unset()
	default:
		cli.flushTimer.Set()
	}

	return reqres
}

func (cli *socketClient) flushQueue() {
	cli.mtx.Lock()
	defer cli.mtx.Unlock()

	// mark all in-flight messages as resolved (they will get cli.Error())
	for req := cli.reqSent.Front(); req != nil; req = req.Next() {
		reqres := req.Value.(*ReqRes)
		reqres.wg.Done()
	}

	// mark all queued messages as resolved
LOOP:
	for {
		select {
		case reqres := <-cli.reqQueue:
			reqres.wg.Done()
		default:
			break LOOP
		}
	}
}

//----------------------------------------

func resMatchesReq(req *types.Request, res *types.Response) (ok bool) {
	switch req.Value.(type) {
	case *types.Request_Echo:
		_, ok = res.Value.(*types.Response_Echo)
	case *types.Request_Flush:
		_, ok = res.Value.(*types.Response_Flush)
	case *types.Request_Info:
		_, ok = res.Value.(*types.Response_Info)
	case *types.Request_SetOption:
		_, ok = res.Value.(*types.Response_SetOption)
	case *types.Request_DeliverTx:
		_, ok = res.Value.(*types.Response_DeliverTx)
	case *types.Request_CheckTx:
		_, ok = res.Value.(*types.Response_CheckTx)
	case *types.Request_Commit:
		_, ok = res.Value.(*types.Response_Commit)
	case *types.Request_Query:
		_, ok = res.Value.(*types.Response_Query)
	case *types.Request_InitChain:
		_, ok = res.Value.(*types.Response_InitChain)
	case *types.Request_BeginBlock:
		_, ok = res.Value.(*types.Response_BeginBlock)
	case *types.Request_EndBlock:
		_, ok = res.Value.(*types.Response_EndBlock)
	case *types.Request_BeginRecheckTx:
		_, ok = res.Value.(*types.Response_BeginRecheckTx)
	case *types.Request_EndRecheckTx:
		_, ok = res.Value.(*types.Response_EndRecheckTx)
	case *types.Request_ApplySnapshotChunk:
		_, ok = res.Value.(*types.Response_ApplySnapshotChunk)
	case *types.Request_LoadSnapshotChunk:
		_, ok = res.Value.(*types.Response_LoadSnapshotChunk)
	case *types.Request_ListSnapshots:
		_, ok = res.Value.(*types.Response_ListSnapshots)
	case *types.Request_OfferSnapshot:
		_, ok = res.Value.(*types.Response_OfferSnapshot)
	}
	return ok
}

func (cli *socketClient) stopForError(err error) {
	if !cli.IsRunning() {
		return
	}

	cli.mtx.Lock()
	if cli.err == nil {
		cli.err = err
	}
	cli.mtx.Unlock()

	cli.Logger.Error(fmt.Sprintf("Stopping abci.socketClient for error: %v", err.Error()))
	if err := cli.Stop(); err != nil {
		cli.Logger.Error("Error stopping abci.socketClient", "err", err)
	}
}
