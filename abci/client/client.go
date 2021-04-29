package abcicli

import (
	"fmt"
	"sync"

	"github.com/line/ostracon/abci/types"
	"github.com/line/ostracon/libs/service"
	tmsync "github.com/line/ostracon/libs/sync"
)

//go:generate mockery --case underscore --name Client
const (
	dialRetryIntervalSeconds = 3
	echoRetryIntervalSeconds = 1
)

// Client defines an interface for an ABCI client.
// All `Async` methods return a `ReqRes` object.
// All `Sync` methods return the appropriate protobuf ResponseXxx struct and an error.
// Note these are client errors, eg. ABCI socket connectivity issues.
// Application-related errors are reflected in response via ABCI error codes and logs.
type Client interface {
	service.Service

	SetGlobalCallback(GlobalCallback)
	GetGlobalCallback() GlobalCallback
	Error() error

	FlushAsync(ResponseCallback) *ReqRes
	EchoAsync(string, ResponseCallback) *ReqRes
	InfoAsync(types.RequestInfo, ResponseCallback) *ReqRes
	SetOptionAsync(types.RequestSetOption, ResponseCallback) *ReqRes
	DeliverTxAsync(types.RequestDeliverTx, ResponseCallback) *ReqRes
	CheckTxAsync(types.RequestCheckTx, ResponseCallback) *ReqRes
	QueryAsync(types.RequestQuery, ResponseCallback) *ReqRes
	CommitAsync(ResponseCallback) *ReqRes
	InitChainAsync(types.RequestInitChain, ResponseCallback) *ReqRes
	BeginBlockAsync(types.RequestBeginBlock, ResponseCallback) *ReqRes
	EndBlockAsync(types.RequestEndBlock, ResponseCallback) *ReqRes
	BeginRecheckTxAsync(types.RequestBeginRecheckTx, ResponseCallback) *ReqRes
	EndRecheckTxAsync(types.RequestEndRecheckTx, ResponseCallback) *ReqRes
	ListSnapshotsAsync(types.RequestListSnapshots, ResponseCallback) *ReqRes
	OfferSnapshotAsync(types.RequestOfferSnapshot, ResponseCallback) *ReqRes
	LoadSnapshotChunkAsync(types.RequestLoadSnapshotChunk, ResponseCallback) *ReqRes
	ApplySnapshotChunkAsync(types.RequestApplySnapshotChunk, ResponseCallback) *ReqRes

	FlushSync() (*types.ResponseFlush, error)
	EchoSync(string) (*types.ResponseEcho, error)
	InfoSync(types.RequestInfo) (*types.ResponseInfo, error)
	SetOptionSync(types.RequestSetOption) (*types.ResponseSetOption, error)
	DeliverTxSync(types.RequestDeliverTx) (*types.ResponseDeliverTx, error)
	CheckTxSync(types.RequestCheckTx) (*types.ResponseCheckTx, error)
	QuerySync(types.RequestQuery) (*types.ResponseQuery, error)
	CommitSync() (*types.ResponseCommit, error)
	InitChainSync(types.RequestInitChain) (*types.ResponseInitChain, error)
	BeginBlockSync(types.RequestBeginBlock) (*types.ResponseBeginBlock, error)
	EndBlockSync(types.RequestEndBlock) (*types.ResponseEndBlock, error)
	BeginRecheckTxSync(types.RequestBeginRecheckTx) (*types.ResponseBeginRecheckTx, error)
	EndRecheckTxSync(types.RequestEndRecheckTx) (*types.ResponseEndRecheckTx, error)
	ListSnapshotsSync(types.RequestListSnapshots) (*types.ResponseListSnapshots, error)
	OfferSnapshotSync(types.RequestOfferSnapshot) (*types.ResponseOfferSnapshot, error)
	LoadSnapshotChunkSync(types.RequestLoadSnapshotChunk) (*types.ResponseLoadSnapshotChunk, error)
	ApplySnapshotChunkSync(types.RequestApplySnapshotChunk) (*types.ResponseApplySnapshotChunk, error)
}

//----------------------------------------

// NewClient returns a new ABCI client of the specified transport type.
// It returns an error if the transport is not "socket" or "grpc"
func NewClient(addr, transport string, mustConnect bool) (client Client, err error) {
	switch transport {
	case "socket":
		client = NewSocketClient(addr, mustConnect)
	case "grpc":
		client = NewGRPCClient(addr, mustConnect)
	default:
		err = fmt.Errorf("unknown abci transport %s", transport)
	}
	return
}

type GlobalCallback func(*types.Request, *types.Response)
type ResponseCallback func(*types.Response)

type ReqRes struct {
	*types.Request
	*types.Response // Not set atomically, so be sure to use WaitGroup.

	mtx  tmsync.Mutex
	wg   *sync.WaitGroup
	done bool             // Gets set to true once *after* WaitGroup.Done().
	cb   ResponseCallback // A single callback that may be set.
}

func NewReqRes(req *types.Request, cb ResponseCallback) *ReqRes {
	return &ReqRes{
		Request:  req,
		Response: nil,

		wg:   waitGroup1(),
		done: false,
		cb:   cb,
	}
}

// InvokeCallback invokes a thread-safe execution of the configured callback
// if non-nil.
func (reqRes *ReqRes) InvokeCallback() {
	reqRes.mtx.Lock()
	defer reqRes.mtx.Unlock()

	if reqRes.cb != nil {
		reqRes.cb(reqRes.Response)
	}
}

func (reqRes *ReqRes) SetDone(res *types.Response) (set bool) {
	reqRes.mtx.Lock()
	// TODO should we panic if it's already done?
	set = !reqRes.done
	if set {
		reqRes.Response = res
		reqRes.done = true
		reqRes.wg.Done()
	}
	reqRes.mtx.Unlock()

	// NOTE `reqRes.cb` is immutable so we're safe to access it at here without `mtx`
	if set && reqRes.cb != nil {
		reqRes.cb(res)
	}

	return set
}

func (reqRes *ReqRes) Wait() {
	reqRes.wg.Wait()
}

func waitGroup1() (wg *sync.WaitGroup) {
	wg = &sync.WaitGroup{}
	wg.Add(1)
	return
}
