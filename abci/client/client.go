package abcicli

import (
	"fmt"
	"sync"

	tmabci "github.com/tendermint/tendermint/abci/types"

	types "github.com/line/ostracon/abci/types"
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
	InfoAsync(tmabci.RequestInfo, ResponseCallback) *ReqRes
	SetOptionAsync(tmabci.RequestSetOption, ResponseCallback) *ReqRes
	DeliverTxAsync(tmabci.RequestDeliverTx, ResponseCallback) *ReqRes
	CheckTxAsync(tmabci.RequestCheckTx, ResponseCallback) *ReqRes
	QueryAsync(tmabci.RequestQuery, ResponseCallback) *ReqRes
	CommitAsync(ResponseCallback) *ReqRes
	InitChainAsync(types.RequestInitChain, ResponseCallback) *ReqRes
	BeginBlockAsync(types.RequestBeginBlock, ResponseCallback) *ReqRes
	EndBlockAsync(tmabci.RequestEndBlock, ResponseCallback) *ReqRes
	BeginRecheckTxAsync(types.RequestBeginRecheckTx, ResponseCallback) *ReqRes
	EndRecheckTxAsync(types.RequestEndRecheckTx, ResponseCallback) *ReqRes
	ListSnapshotsAsync(tmabci.RequestListSnapshots, ResponseCallback) *ReqRes
	OfferSnapshotAsync(tmabci.RequestOfferSnapshot, ResponseCallback) *ReqRes
	LoadSnapshotChunkAsync(tmabci.RequestLoadSnapshotChunk, ResponseCallback) *ReqRes
	ApplySnapshotChunkAsync(tmabci.RequestApplySnapshotChunk, ResponseCallback) *ReqRes

	FlushSync() (*tmabci.ResponseFlush, error)
	EchoSync(string) (*tmabci.ResponseEcho, error)
	InfoSync(tmabci.RequestInfo) (*tmabci.ResponseInfo, error)
	SetOptionSync(tmabci.RequestSetOption) (*tmabci.ResponseSetOption, error)
	DeliverTxSync(tmabci.RequestDeliverTx) (*tmabci.ResponseDeliverTx, error)
	CheckTxSync(tmabci.RequestCheckTx) (*types.ResponseCheckTx, error)
	QuerySync(tmabci.RequestQuery) (*tmabci.ResponseQuery, error)
	CommitSync() (*tmabci.ResponseCommit, error)
	InitChainSync(types.RequestInitChain) (*types.ResponseInitChain, error)
	BeginBlockSync(types.RequestBeginBlock) (*tmabci.ResponseBeginBlock, error)
	EndBlockSync(tmabci.RequestEndBlock) (*types.ResponseEndBlock, error)
	BeginRecheckTxSync(types.RequestBeginRecheckTx) (*types.ResponseBeginRecheckTx, error)
	EndRecheckTxSync(types.RequestEndRecheckTx) (*types.ResponseEndRecheckTx, error)
	ListSnapshotsSync(tmabci.RequestListSnapshots) (*tmabci.ResponseListSnapshots, error)
	OfferSnapshotSync(tmabci.RequestOfferSnapshot) (*tmabci.ResponseOfferSnapshot, error)
	LoadSnapshotChunkSync(tmabci.RequestLoadSnapshotChunk) (*tmabci.ResponseLoadSnapshotChunk, error)
	ApplySnapshotChunkSync(tmabci.RequestApplySnapshotChunk) (*tmabci.ResponseApplySnapshotChunk, error)
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
