package types

import (
	context "golang.org/x/net/context"

	tmabci "github.com/tendermint/tendermint/abci/types"
)

//go:generate mockery --case underscore --name Application

type CheckTxCallback func(ResponseCheckTx)

// Application is an interface that enables any finite, deterministic state machine
// to be driven by a blockchain-based replication engine via the ABCI.
// All methods take a RequestXxx argument and return a ResponseXxx argument,
// except CheckTx/DeliverTx, which take `tx []byte`, and `Commit`, which takes nothing.
type Application interface {
	// Info/Query Connection
	Info(tmabci.RequestInfo) tmabci.ResponseInfo                // Return application info
	SetOption(tmabci.RequestSetOption) tmabci.ResponseSetOption // Set application option
	Query(tmabci.RequestQuery) tmabci.ResponseQuery             // Query for state

	// Mempool Connection
	CheckTxSync(tmabci.RequestCheckTx) ResponseCheckTx           // Validate a tx for the mempool
	CheckTxAsync(tmabci.RequestCheckTx, CheckTxCallback)         // Asynchronously validate a tx for the mempool
	BeginRecheckTx(RequestBeginRecheckTx) ResponseBeginRecheckTx // Signals the beginning of rechecking
	EndRecheckTx(RequestEndRecheckTx) ResponseEndRecheckTx       // Signals the end of rechecking

	// Consensus Connection
	InitChain(RequestInitChain) ResponseInitChain               // Initialize blockchain w validators/other info from OstraconCore
	BeginBlock(RequestBeginBlock) tmabci.ResponseBeginBlock     // Signals the beginning of a block
	DeliverTx(tmabci.RequestDeliverTx) tmabci.ResponseDeliverTx // Deliver a tx for full processing
	EndBlock(tmabci.RequestEndBlock) ResponseEndBlock           // Signals the end of a block, returns changes to the validator set
	Commit() tmabci.ResponseCommit                              // Commit the state and return the application Merkle root hash

	// State Sync Connection
	ListSnapshots(tmabci.RequestListSnapshots) tmabci.ResponseListSnapshots                // List available snapshots
	OfferSnapshot(tmabci.RequestOfferSnapshot) tmabci.ResponseOfferSnapshot                // Offer a snapshot to the application
	LoadSnapshotChunk(tmabci.RequestLoadSnapshotChunk) tmabci.ResponseLoadSnapshotChunk    // Load a snapshot chunk
	ApplySnapshotChunk(tmabci.RequestApplySnapshotChunk) tmabci.ResponseApplySnapshotChunk // Apply a shapshot chunk
}

//-------------------------------------------------------
// BaseApplication is a base form of Application

var _ Application = (*BaseApplication)(nil)

type BaseApplication struct {
}

func NewBaseApplication() *BaseApplication {
	return &BaseApplication{}
}

func (BaseApplication) Info(req tmabci.RequestInfo) tmabci.ResponseInfo {
	return tmabci.ResponseInfo{}
}

func (BaseApplication) SetOption(req tmabci.RequestSetOption) tmabci.ResponseSetOption {
	return tmabci.ResponseSetOption{}
}

func (BaseApplication) DeliverTx(req tmabci.RequestDeliverTx) tmabci.ResponseDeliverTx {
	return tmabci.ResponseDeliverTx{Code: CodeTypeOK}
}

func (BaseApplication) CheckTxSync(req tmabci.RequestCheckTx) ResponseCheckTx {
	return ResponseCheckTx{Code: CodeTypeOK}
}

func (BaseApplication) CheckTxAsync(req tmabci.RequestCheckTx, callback CheckTxCallback) {
	callback(ResponseCheckTx{Code: CodeTypeOK})
}

func (BaseApplication) BeginRecheckTx(req RequestBeginRecheckTx) ResponseBeginRecheckTx {
	return ResponseBeginRecheckTx{Code: CodeTypeOK}
}

func (BaseApplication) EndRecheckTx(req RequestEndRecheckTx) ResponseEndRecheckTx {
	return ResponseEndRecheckTx{Code: CodeTypeOK}
}

func (BaseApplication) Commit() tmabci.ResponseCommit {
	return tmabci.ResponseCommit{}
}

func (BaseApplication) Query(req tmabci.RequestQuery) tmabci.ResponseQuery {
	return tmabci.ResponseQuery{Code: CodeTypeOK}
}

func (BaseApplication) InitChain(req RequestInitChain) ResponseInitChain {
	return ResponseInitChain{}
}

func (BaseApplication) BeginBlock(req RequestBeginBlock) tmabci.ResponseBeginBlock {
	return tmabci.ResponseBeginBlock{}
}

func (BaseApplication) EndBlock(req tmabci.RequestEndBlock) ResponseEndBlock {
	return ResponseEndBlock{}
}

func (BaseApplication) ListSnapshots(req tmabci.RequestListSnapshots) tmabci.ResponseListSnapshots {
	return tmabci.ResponseListSnapshots{}
}

func (BaseApplication) OfferSnapshot(req tmabci.RequestOfferSnapshot) tmabci.ResponseOfferSnapshot {
	return tmabci.ResponseOfferSnapshot{}
}

func (BaseApplication) LoadSnapshotChunk(req tmabci.RequestLoadSnapshotChunk) tmabci.ResponseLoadSnapshotChunk {
	return tmabci.ResponseLoadSnapshotChunk{}
}

func (BaseApplication) ApplySnapshotChunk(req tmabci.RequestApplySnapshotChunk) tmabci.ResponseApplySnapshotChunk {
	return tmabci.ResponseApplySnapshotChunk{}
}

//-------------------------------------------------------

// GRPCApplication is a GRPC wrapper for Application
type GRPCApplication struct {
	app Application
}

func NewGRPCApplication(app Application) *GRPCApplication {
	return &GRPCApplication{app}
}

func (app *GRPCApplication) Echo(ctx context.Context, req *tmabci.RequestEcho) (*tmabci.ResponseEcho, error) {
	return &tmabci.ResponseEcho{Message: req.Message}, nil
}

func (app *GRPCApplication) Flush(ctx context.Context, req *tmabci.RequestFlush) (*tmabci.ResponseFlush, error) {
	return &tmabci.ResponseFlush{}, nil
}

func (app *GRPCApplication) Info(ctx context.Context, req *tmabci.RequestInfo) (*tmabci.ResponseInfo, error) {
	res := app.app.Info(*req)
	return &res, nil
}

func (app *GRPCApplication) SetOption(ctx context.Context, req *tmabci.RequestSetOption) (*tmabci.ResponseSetOption, error) {
	res := app.app.SetOption(*req)
	return &res, nil
}

func (app *GRPCApplication) DeliverTx(ctx context.Context, req *tmabci.RequestDeliverTx) (*tmabci.ResponseDeliverTx, error) {
	res := app.app.DeliverTx(*req)
	return &res, nil
}

func (app *GRPCApplication) CheckTx(ctx context.Context, req *tmabci.RequestCheckTx) (*ResponseCheckTx, error) {
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

func (app *GRPCApplication) Query(ctx context.Context, req *tmabci.RequestQuery) (*tmabci.ResponseQuery, error) {
	res := app.app.Query(*req)
	return &res, nil
}

func (app *GRPCApplication) Commit(ctx context.Context, req *tmabci.RequestCommit) (*tmabci.ResponseCommit, error) {
	res := app.app.Commit()
	return &res, nil
}

func (app *GRPCApplication) InitChain(ctx context.Context, req *RequestInitChain) (*ResponseInitChain, error) {
	res := app.app.InitChain(*req)
	return &res, nil
}

func (app *GRPCApplication) BeginBlock(ctx context.Context, req *RequestBeginBlock) (*tmabci.ResponseBeginBlock, error) {
	res := app.app.BeginBlock(*req)
	return &res, nil
}

func (app *GRPCApplication) EndBlock(ctx context.Context, req *tmabci.RequestEndBlock) (*ResponseEndBlock, error) {
	res := app.app.EndBlock(*req)
	return &res, nil
}

func (app *GRPCApplication) ListSnapshots(
	ctx context.Context, req *tmabci.RequestListSnapshots) (*tmabci.ResponseListSnapshots, error) {
	res := app.app.ListSnapshots(*req)
	return &res, nil
}

func (app *GRPCApplication) OfferSnapshot(
	ctx context.Context, req *tmabci.RequestOfferSnapshot) (*tmabci.ResponseOfferSnapshot, error) {
	res := app.app.OfferSnapshot(*req)
	return &res, nil
}

func (app *GRPCApplication) LoadSnapshotChunk(
	ctx context.Context, req *tmabci.RequestLoadSnapshotChunk) (*tmabci.ResponseLoadSnapshotChunk, error) {
	res := app.app.LoadSnapshotChunk(*req)
	return &res, nil
}

func (app *GRPCApplication) ApplySnapshotChunk(
	ctx context.Context, req *tmabci.RequestApplySnapshotChunk) (*tmabci.ResponseApplySnapshotChunk, error) {
	res := app.app.ApplySnapshotChunk(*req)
	return &res, nil
}
