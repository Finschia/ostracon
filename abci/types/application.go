package types

import (
	context "golang.org/x/net/context"

	abci "github.com/tendermint/tendermint/abci/types"
)

//go:generate mockery --case underscore --name Application

type CheckTxCallback func(ResponseCheckTx)

// Application is an interface that enables any finite, deterministic state machine
// to be driven by a blockchain-based replication engine via the ABCI.
// All methods take a RequestXxx argument and return a ResponseXxx argument,
// except CheckTx/DeliverTx, which take `tx []byte`, and `Commit`, which takes nothing.
type Application interface {
	// Info/Query Connection
	Info(abci.RequestInfo) abci.ResponseInfo                // Return application info
	SetOption(abci.RequestSetOption) abci.ResponseSetOption // Set application option
	Query(abci.RequestQuery) abci.ResponseQuery             // Query for state

	// Mempool Connection
	CheckTxSync(abci.RequestCheckTx) ResponseCheckTx             // Validate a tx for the mempool
	CheckTxAsync(abci.RequestCheckTx, CheckTxCallback)           // Asynchronously validate a tx for the mempool
	BeginRecheckTx(RequestBeginRecheckTx) ResponseBeginRecheckTx // Signals the beginning of rechecking
	EndRecheckTx(RequestEndRecheckTx) ResponseEndRecheckTx       // Signals the end of rechecking

	// Consensus Connection
	InitChain(abci.RequestInitChain) abci.ResponseInitChain // Initialize blockchain w validators/other info from OstraconCore
	BeginBlock(RequestBeginBlock) abci.ResponseBeginBlock   // Signals the beginning of a block
	DeliverTx(abci.RequestDeliverTx) abci.ResponseDeliverTx // Deliver a tx for full processing
	EndBlock(abci.RequestEndBlock) abci.ResponseEndBlock    // Signals the end of a block, returns changes to the validator set
	Commit() abci.ResponseCommit                            // Commit the state and return the application Merkle root hash

	// State Sync Connection
	ListSnapshots(abci.RequestListSnapshots) abci.ResponseListSnapshots                // List available snapshots
	OfferSnapshot(abci.RequestOfferSnapshot) abci.ResponseOfferSnapshot                // Offer a snapshot to the application
	LoadSnapshotChunk(abci.RequestLoadSnapshotChunk) abci.ResponseLoadSnapshotChunk    // Load a snapshot chunk
	ApplySnapshotChunk(abci.RequestApplySnapshotChunk) abci.ResponseApplySnapshotChunk // Apply a shapshot chunk
}

//-------------------------------------------------------
// BaseApplication is a base form of Application

var _ Application = (*BaseApplication)(nil)

type BaseApplication struct {
}

func NewBaseApplication() *BaseApplication {
	return &BaseApplication{}
}

func (BaseApplication) Info(req abci.RequestInfo) abci.ResponseInfo {
	return abci.ResponseInfo{}
}

func (BaseApplication) SetOption(req abci.RequestSetOption) abci.ResponseSetOption {
	return abci.ResponseSetOption{}
}

func (BaseApplication) DeliverTx(req abci.RequestDeliverTx) abci.ResponseDeliverTx {
	return abci.ResponseDeliverTx{Code: CodeTypeOK}
}

func (BaseApplication) CheckTxSync(req abci.RequestCheckTx) ResponseCheckTx {
	return ResponseCheckTx{Code: CodeTypeOK}
}

func (BaseApplication) CheckTxAsync(req abci.RequestCheckTx, callback CheckTxCallback) {
	callback(ResponseCheckTx{Code: CodeTypeOK})
}

func (BaseApplication) BeginRecheckTx(req RequestBeginRecheckTx) ResponseBeginRecheckTx {
	return ResponseBeginRecheckTx{Code: CodeTypeOK}
}

func (BaseApplication) EndRecheckTx(req RequestEndRecheckTx) ResponseEndRecheckTx {
	return ResponseEndRecheckTx{Code: CodeTypeOK}
}

func (BaseApplication) Commit() abci.ResponseCommit {
	return abci.ResponseCommit{}
}

func (BaseApplication) Query(req abci.RequestQuery) abci.ResponseQuery {
	return abci.ResponseQuery{Code: CodeTypeOK}
}

func (BaseApplication) InitChain(req abci.RequestInitChain) abci.ResponseInitChain {
	return abci.ResponseInitChain{}
}

func (BaseApplication) BeginBlock(req RequestBeginBlock) abci.ResponseBeginBlock {
	return abci.ResponseBeginBlock{}
}

func (BaseApplication) EndBlock(req abci.RequestEndBlock) abci.ResponseEndBlock {
	return abci.ResponseEndBlock{}
}

func (BaseApplication) ListSnapshots(req abci.RequestListSnapshots) abci.ResponseListSnapshots {
	return abci.ResponseListSnapshots{}
}

func (BaseApplication) OfferSnapshot(req abci.RequestOfferSnapshot) abci.ResponseOfferSnapshot {
	return abci.ResponseOfferSnapshot{}
}

func (BaseApplication) LoadSnapshotChunk(req abci.RequestLoadSnapshotChunk) abci.ResponseLoadSnapshotChunk {
	return abci.ResponseLoadSnapshotChunk{}
}

func (BaseApplication) ApplySnapshotChunk(req abci.RequestApplySnapshotChunk) abci.ResponseApplySnapshotChunk {
	return abci.ResponseApplySnapshotChunk{}
}

//-------------------------------------------------------

// GRPCApplication is a GRPC wrapper for Application
type GRPCApplication struct {
	app Application
}

func NewGRPCApplication(app Application) *GRPCApplication {
	return &GRPCApplication{app}
}

func (app *GRPCApplication) Echo(ctx context.Context, req *abci.RequestEcho) (*abci.ResponseEcho, error) {
	return &abci.ResponseEcho{Message: req.Message}, nil
}

func (app *GRPCApplication) Flush(ctx context.Context, req *abci.RequestFlush) (*abci.ResponseFlush, error) {
	return &abci.ResponseFlush{}, nil
}

func (app *GRPCApplication) Info(ctx context.Context, req *abci.RequestInfo) (*abci.ResponseInfo, error) {
	res := app.app.Info(*req)
	return &res, nil
}

func (app *GRPCApplication) SetOption(ctx context.Context, req *abci.RequestSetOption) (*abci.ResponseSetOption, error) {
	res := app.app.SetOption(*req)
	return &res, nil
}

func (app *GRPCApplication) DeliverTx(ctx context.Context, req *abci.RequestDeliverTx) (*abci.ResponseDeliverTx, error) {
	res := app.app.DeliverTx(*req)
	return &res, nil
}

func (app *GRPCApplication) CheckTx(ctx context.Context, req *abci.RequestCheckTx) (*ResponseCheckTx, error) {
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

func (app *GRPCApplication) Query(ctx context.Context, req *abci.RequestQuery) (*abci.ResponseQuery, error) {
	res := app.app.Query(*req)
	return &res, nil
}

func (app *GRPCApplication) Commit(ctx context.Context, req *abci.RequestCommit) (*abci.ResponseCommit, error) {
	res := app.app.Commit()
	return &res, nil
}

func (app *GRPCApplication) InitChain(ctx context.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	res := app.app.InitChain(*req)
	return &res, nil
}

func (app *GRPCApplication) BeginBlock(ctx context.Context, req *RequestBeginBlock) (*abci.ResponseBeginBlock, error) {
	res := app.app.BeginBlock(*req)
	return &res, nil
}

func (app *GRPCApplication) EndBlock(ctx context.Context, req *abci.RequestEndBlock) (*abci.ResponseEndBlock, error) {
	res := app.app.EndBlock(*req)
	return &res, nil
}

func (app *GRPCApplication) ListSnapshots(
	ctx context.Context, req *abci.RequestListSnapshots) (*abci.ResponseListSnapshots, error) {
	res := app.app.ListSnapshots(*req)
	return &res, nil
}

func (app *GRPCApplication) OfferSnapshot(
	ctx context.Context, req *abci.RequestOfferSnapshot) (*abci.ResponseOfferSnapshot, error) {
	res := app.app.OfferSnapshot(*req)
	return &res, nil
}

func (app *GRPCApplication) LoadSnapshotChunk(
	ctx context.Context, req *abci.RequestLoadSnapshotChunk) (*abci.ResponseLoadSnapshotChunk, error) {
	res := app.app.LoadSnapshotChunk(*req)
	return &res, nil
}

func (app *GRPCApplication) ApplySnapshotChunk(
	ctx context.Context, req *abci.RequestApplySnapshotChunk) (*abci.ResponseApplySnapshotChunk, error) {
	res := app.app.ApplySnapshotChunk(*req)
	return &res, nil
}
