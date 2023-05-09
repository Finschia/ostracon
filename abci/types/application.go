package types

import (
	context "golang.org/x/net/context"

	"github.com/tendermint/tendermint/abci/types"
)

//go:generate ../../scripts/mockery_generate.sh Application

type CheckTxCallback func(ResponseCheckTx)

// Application is an interface that enables any finite, deterministic state machine
// to be driven by a blockchain-based replication engine via the ABCI.
// All methods take a RequestXxx argument and return a ResponseXxx argument,
// except CheckTx/DeliverTx, which take `tx []byte`, and `Commit`, which takes nothing.
type Application interface {
	// Info/Query Connection
	Info(types.RequestInfo) types.ResponseInfo                // Return application info
	SetOption(types.RequestSetOption) types.ResponseSetOption // Set application option
	Query(types.RequestQuery) types.ResponseQuery             // Query for state

	// Mempool Connection
	CheckTxSync(types.RequestCheckTx) ResponseCheckTx            // Validate a tx for the mempool
	CheckTxAsync(types.RequestCheckTx, CheckTxCallback)          // Asynchronously validate a tx for the mempool
	BeginRecheckTx(RequestBeginRecheckTx) ResponseBeginRecheckTx // Signals the beginning of rechecking
	EndRecheckTx(RequestEndRecheckTx) ResponseEndRecheckTx       // Signals the end of rechecking

	// Consensus Connection
	InitChain(types.RequestInitChain) types.ResponseInitChain // Initialize blockchain w validators/other info from OstraconCore
	BeginBlock(RequestBeginBlock) types.ResponseBeginBlock    // Signals the beginning of a block
	DeliverTx(types.RequestDeliverTx) types.ResponseDeliverTx // Deliver a tx for full processing
	EndBlock(types.RequestEndBlock) types.ResponseEndBlock    // Signals the end of a block, returns changes to the validator set
	Commit() types.ResponseCommit                             // Commit the state and return the application Merkle root hash

	// State Sync Connection
	ListSnapshots(types.RequestListSnapshots) types.ResponseListSnapshots                // List available snapshots
	OfferSnapshot(types.RequestOfferSnapshot) types.ResponseOfferSnapshot                // Offer a snapshot to the application
	LoadSnapshotChunk(types.RequestLoadSnapshotChunk) types.ResponseLoadSnapshotChunk    // Load a snapshot chunk
	ApplySnapshotChunk(types.RequestApplySnapshotChunk) types.ResponseApplySnapshotChunk // Apply a shapshot chunk
}

//-------------------------------------------------------
// BaseApplication is a base form of Application

var _ Application = (*BaseApplication)(nil)

type BaseApplication struct {
}

func NewBaseApplication() *BaseApplication {
	return &BaseApplication{}
}

func (BaseApplication) Info(req types.RequestInfo) types.ResponseInfo {
	return types.ResponseInfo{}
}

func (BaseApplication) SetOption(req types.RequestSetOption) types.ResponseSetOption {
	return types.ResponseSetOption{}
}

func (BaseApplication) DeliverTx(req types.RequestDeliverTx) types.ResponseDeliverTx {
	return types.ResponseDeliverTx{Code: CodeTypeOK}
}

func (BaseApplication) CheckTxSync(req types.RequestCheckTx) ResponseCheckTx {
	return ResponseCheckTx{Code: CodeTypeOK}
}

func (BaseApplication) CheckTxAsync(req types.RequestCheckTx, callback CheckTxCallback) {
	callback(ResponseCheckTx{Code: CodeTypeOK})
}

func (BaseApplication) BeginRecheckTx(req RequestBeginRecheckTx) ResponseBeginRecheckTx {
	return ResponseBeginRecheckTx{Code: CodeTypeOK}
}

func (BaseApplication) EndRecheckTx(req RequestEndRecheckTx) ResponseEndRecheckTx {
	return ResponseEndRecheckTx{Code: CodeTypeOK}
}

func (BaseApplication) Commit() types.ResponseCommit {
	return types.ResponseCommit{}
}

func (BaseApplication) Query(req types.RequestQuery) types.ResponseQuery {
	return types.ResponseQuery{Code: CodeTypeOK}
}

func (BaseApplication) InitChain(req types.RequestInitChain) types.ResponseInitChain {
	return types.ResponseInitChain{}
}

func (BaseApplication) BeginBlock(req RequestBeginBlock) types.ResponseBeginBlock {
	return types.ResponseBeginBlock{}
}

func (BaseApplication) EndBlock(req types.RequestEndBlock) types.ResponseEndBlock {
	return types.ResponseEndBlock{}
}

func (BaseApplication) ListSnapshots(req types.RequestListSnapshots) types.ResponseListSnapshots {
	return types.ResponseListSnapshots{}
}

func (BaseApplication) OfferSnapshot(req types.RequestOfferSnapshot) types.ResponseOfferSnapshot {
	return types.ResponseOfferSnapshot{}
}

func (BaseApplication) LoadSnapshotChunk(req types.RequestLoadSnapshotChunk) types.ResponseLoadSnapshotChunk {
	return types.ResponseLoadSnapshotChunk{}
}

func (BaseApplication) ApplySnapshotChunk(req types.RequestApplySnapshotChunk) types.ResponseApplySnapshotChunk {
	return types.ResponseApplySnapshotChunk{}
}

//-------------------------------------------------------

// GRPCApplication is a GRPC wrapper for Application
type GRPCApplication struct {
	app Application
}

func NewGRPCApplication(app Application) *GRPCApplication {
	return &GRPCApplication{app}
}

func (app *GRPCApplication) Echo(ctx context.Context, req *types.RequestEcho) (*types.ResponseEcho, error) {
	return &types.ResponseEcho{Message: req.Message}, nil
}

func (app *GRPCApplication) Flush(ctx context.Context, req *types.RequestFlush) (*types.ResponseFlush, error) {
	return &types.ResponseFlush{}, nil
}

func (app *GRPCApplication) Info(ctx context.Context, req *types.RequestInfo) (*types.ResponseInfo, error) {
	res := app.app.Info(*req)
	return &res, nil
}

func (app *GRPCApplication) SetOption(ctx context.Context, req *types.RequestSetOption) (*types.ResponseSetOption, error) {
	res := app.app.SetOption(*req)
	return &res, nil
}

func (app *GRPCApplication) DeliverTx(ctx context.Context, req *types.RequestDeliverTx) (*types.ResponseDeliverTx, error) {
	res := app.app.DeliverTx(*req)
	return &res, nil
}

func (app *GRPCApplication) CheckTx(ctx context.Context, req *types.RequestCheckTx) (*ResponseCheckTx, error) {
	res := app.app.CheckTxSync(*req)
	return &res, nil
}

func (app *GRPCApplication) BeginRecheckTx(ctx context.Context, req *RequestBeginRecheckTx) (
	*ResponseBeginRecheckTx, error) {
	res := app.app.BeginRecheckTx(*req)
	return &res, nil
}

func (app *GRPCApplication) EndRecheckTx(ctx context.Context, req *RequestEndRecheckTx) (*ResponseEndRecheckTx, error) {
	res := app.app.EndRecheckTx(*req)
	return &res, nil
}

func (app *GRPCApplication) Query(ctx context.Context, req *types.RequestQuery) (*types.ResponseQuery, error) {
	res := app.app.Query(*req)
	return &res, nil
}

func (app *GRPCApplication) Commit(ctx context.Context, req *types.RequestCommit) (*types.ResponseCommit, error) {
	res := app.app.Commit()
	return &res, nil
}

func (app *GRPCApplication) InitChain(ctx context.Context, req *types.RequestInitChain) (*types.ResponseInitChain, error) {
	res := app.app.InitChain(*req)
	return &res, nil
}

func (app *GRPCApplication) BeginBlock(ctx context.Context, req *RequestBeginBlock) (*types.ResponseBeginBlock, error) {
	res := app.app.BeginBlock(*req)
	return &res, nil
}

func (app *GRPCApplication) EndBlock(ctx context.Context, req *types.RequestEndBlock) (*types.ResponseEndBlock, error) {
	res := app.app.EndBlock(*req)
	return &res, nil
}

func (app *GRPCApplication) ListSnapshots(
	ctx context.Context, req *types.RequestListSnapshots) (*types.ResponseListSnapshots, error) {
	res := app.app.ListSnapshots(*req)
	return &res, nil
}

func (app *GRPCApplication) OfferSnapshot(
	ctx context.Context, req *types.RequestOfferSnapshot) (*types.ResponseOfferSnapshot, error) {
	res := app.app.OfferSnapshot(*req)
	return &res, nil
}

func (app *GRPCApplication) LoadSnapshotChunk(
	ctx context.Context, req *types.RequestLoadSnapshotChunk) (*types.ResponseLoadSnapshotChunk, error) {
	res := app.app.LoadSnapshotChunk(*req)
	return &res, nil
}

func (app *GRPCApplication) ApplySnapshotChunk(
	ctx context.Context, req *types.RequestApplySnapshotChunk) (*types.ResponseApplySnapshotChunk, error) {
	res := app.app.ApplySnapshotChunk(*req)
	return &res, nil
}
