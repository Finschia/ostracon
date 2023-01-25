package abcicli

import (
	"fmt"
	"sync"

	abci "github.com/tendermint/tendermint/abci/types"

	ocabci "github.com/line/ostracon/abci/types"
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
	InfoAsync(abci.RequestInfo, ResponseCallback) *ReqRes
	SetOptionAsync(abci.RequestSetOption, ResponseCallback) *ReqRes
	DeliverTxAsync(abci.RequestDeliverTx, ResponseCallback) *ReqRes
	CheckTxAsync(abci.RequestCheckTx, ResponseCallback) *ReqRes
	QueryAsync(abci.RequestQuery, ResponseCallback) *ReqRes
	CommitAsync(ResponseCallback) *ReqRes
	InitChainAsync(abci.RequestInitChain, ResponseCallback) *ReqRes
	BeginBlockAsync(ocabci.RequestBeginBlock, ResponseCallback) *ReqRes
	EndBlockAsync(abci.RequestEndBlock, ResponseCallback) *ReqRes
	BeginRecheckTxAsync(ocabci.RequestBeginRecheckTx, ResponseCallback) *ReqRes
	EndRecheckTxAsync(ocabci.RequestEndRecheckTx, ResponseCallback) *ReqRes
	ListSnapshotsAsync(abci.RequestListSnapshots, ResponseCallback) *ReqRes
	OfferSnapshotAsync(abci.RequestOfferSnapshot, ResponseCallback) *ReqRes
	LoadSnapshotChunkAsync(abci.RequestLoadSnapshotChunk, ResponseCallback) *ReqRes
	ApplySnapshotChunkAsync(abci.RequestApplySnapshotChunk, ResponseCallback) *ReqRes

	FlushSync() (*abci.ResponseFlush, error)
	EchoSync(string) (*abci.ResponseEcho, error)
	InfoSync(abci.RequestInfo) (*abci.ResponseInfo, error)
	SetOptionSync(abci.RequestSetOption) (*abci.ResponseSetOption, error)
	DeliverTxSync(abci.RequestDeliverTx) (*abci.ResponseDeliverTx, error)
	CheckTxSync(abci.RequestCheckTx) (*ocabci.ResponseCheckTx, error)
	QuerySync(abci.RequestQuery) (*abci.ResponseQuery, error)
	CommitSync() (*abci.ResponseCommit, error)
	InitChainSync(abci.RequestInitChain) (*abci.ResponseInitChain, error)
	BeginBlockSync(ocabci.RequestBeginBlock) (*abci.ResponseBeginBlock, error)
	EndBlockSync(abci.RequestEndBlock) (*abci.ResponseEndBlock, error)
	BeginRecheckTxSync(ocabci.RequestBeginRecheckTx) (*ocabci.ResponseBeginRecheckTx, error)
	EndRecheckTxSync(ocabci.RequestEndRecheckTx) (*ocabci.ResponseEndRecheckTx, error)
	ListSnapshotsSync(abci.RequestListSnapshots) (*abci.ResponseListSnapshots, error)
	OfferSnapshotSync(abci.RequestOfferSnapshot) (*abci.ResponseOfferSnapshot, error)
	LoadSnapshotChunkSync(abci.RequestLoadSnapshotChunk) (*abci.ResponseLoadSnapshotChunk, error)
	ApplySnapshotChunkSync(abci.RequestApplySnapshotChunk) (*abci.ResponseApplySnapshotChunk, error)
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

type GlobalCallback func(*ocabci.Request, *ocabci.Response)
type ResponseCallback func(*ocabci.Response)

type ReqRes struct {
	*ocabci.Request
	*ocabci.Response // Not set atomically, so be sure to use WaitGroup.

	mtx  tmsync.Mutex
	wg   *sync.WaitGroup
	done bool             // Gets set to true once *after* WaitGroup.Done().
	cb   ResponseCallback // A single callback that may be set.
}

func NewReqRes(req *ocabci.Request, cb ResponseCallback) *ReqRes {
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

func (reqRes *ReqRes) SetDone(res *ocabci.Response) (set bool) {
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
