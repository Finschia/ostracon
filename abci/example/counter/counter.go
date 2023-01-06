package counter

import (
	"encoding/binary"
	"fmt"

	tmabci "github.com/tendermint/tendermint/abci/types"

	"github.com/line/ostracon/abci/example/code"
	abci "github.com/line/ostracon/abci/types"
)

type Application struct {
	abci.BaseApplication

	hashCount int
	txCount   int
	serial    bool
}

func NewApplication(serial bool) *Application {
	return &Application{serial: serial}
}

func (app *Application) Info(req tmabci.RequestInfo) tmabci.ResponseInfo {
	return tmabci.ResponseInfo{Data: fmt.Sprintf("{\"hashes\":%v,\"txs\":%v}", app.hashCount, app.txCount)}
}

func (app *Application) SetOption(req tmabci.RequestSetOption) tmabci.ResponseSetOption {
	key, value := req.Key, req.Value
	if key == "serial" && value == "on" {
		app.serial = true
	} else {
		/*
			TODO Panic and have the ABCI server pass an exception.
			The client can call SetOptionSync() and get an `error`.
			return abci.ResponseSetOption{
				Error: fmt.Sprintf("Unknown key (%s) or value (%s)", key, value),
			}
		*/
		return tmabci.ResponseSetOption{}
	}

	return tmabci.ResponseSetOption{}
}

func (app *Application) DeliverTx(req tmabci.RequestDeliverTx) tmabci.ResponseDeliverTx {
	if app.serial {
		if len(req.Tx) > 8 {
			return tmabci.ResponseDeliverTx{
				Code: code.CodeTypeEncodingError,
				Log:  fmt.Sprintf("Max tx size is 8 bytes, got %d", len(req.Tx))}
		}
		tx8 := make([]byte, 8)
		copy(tx8[len(tx8)-len(req.Tx):], req.Tx)
		txValue := binary.BigEndian.Uint64(tx8)
		if txValue != uint64(app.txCount) {
			return tmabci.ResponseDeliverTx{
				Code: code.CodeTypeBadNonce,
				Log:  fmt.Sprintf("Invalid nonce. Expected %v, got %v", app.txCount, txValue)}
		}
	}
	app.txCount++
	return tmabci.ResponseDeliverTx{Code: code.CodeTypeOK}
}

func (app *Application) CheckTxSync(req tmabci.RequestCheckTx) abci.ResponseCheckTx {
	return app.checkTx(req)
}

func (app *Application) CheckTxAsync(req tmabci.RequestCheckTx, callback abci.CheckTxCallback) {
	callback(app.checkTx(req))
}

func (app *Application) checkTx(req tmabci.RequestCheckTx) abci.ResponseCheckTx {
	if app.serial {
		if len(req.Tx) > 8 {
			return abci.ResponseCheckTx{
				Code: code.CodeTypeEncodingError,
				Log:  fmt.Sprintf("Max tx size is 8 bytes, got %d", len(req.Tx))}
		}
		tx8 := make([]byte, 8)
		copy(tx8[len(tx8)-len(req.Tx):], req.Tx)
		txValue := binary.BigEndian.Uint64(tx8)
		if txValue < uint64(app.txCount) {
			return abci.ResponseCheckTx{
				Code: code.CodeTypeBadNonce,
				Log:  fmt.Sprintf("Invalid nonce. Expected >= %v, got %v", app.txCount, txValue)}
		}
	}
	return abci.ResponseCheckTx{Code: code.CodeTypeOK}
}

func (app *Application) Commit() (resp tmabci.ResponseCommit) {
	app.hashCount++
	if app.txCount == 0 {
		return tmabci.ResponseCommit{}
	}
	hash := make([]byte, 8)
	binary.BigEndian.PutUint64(hash, uint64(app.txCount))
	return tmabci.ResponseCommit{Data: hash}
}

func (app *Application) Query(reqQuery tmabci.RequestQuery) tmabci.ResponseQuery {
	switch reqQuery.Path {
	case "hash":
		return tmabci.ResponseQuery{Value: []byte(fmt.Sprintf("%v", app.hashCount))}
	case "tx":
		return tmabci.ResponseQuery{Value: []byte(fmt.Sprintf("%v", app.txCount))}
	default:
		return tmabci.ResponseQuery{Log: fmt.Sprintf("Invalid query path. Expected hash or tx, got %v", reqQuery.Path)}
	}
}
