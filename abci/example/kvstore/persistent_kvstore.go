package kvstore

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	abci "github.com/tendermint/tendermint/abci/types"
	pc "github.com/tendermint/tendermint/proto/tendermint/crypto"
	dbm "github.com/tendermint/tm-db"

	"github.com/line/ostracon/abci/example/code"
	ocabci "github.com/line/ostracon/abci/types"
	cryptoenc "github.com/line/ostracon/crypto/encoding"
	"github.com/line/ostracon/libs/log"
)

const (
	ValidatorSetChangePrefix string = "val:"
)

//-----------------------------------------

var _ ocabci.Application = (*PersistentKVStoreApplication)(nil)

type PersistentKVStoreApplication struct {
	app *Application

	// validator set
	ValUpdates []abci.ValidatorUpdate

	valAddrToPubKeyMap map[string]pc.PublicKey

	logger log.Logger
}

func NewPersistentKVStoreApplication(dbDir string) *PersistentKVStoreApplication {
	name := "kvstore"
	db, err := dbm.NewGoLevelDB(name, dbDir)
	if err != nil {
		panic(err)
	}

	state := loadState(db)

	return &PersistentKVStoreApplication{
		app:                &Application{state: state},
		valAddrToPubKeyMap: make(map[string]pc.PublicKey),
		logger:             log.NewNopLogger(),
	}
}

func (app *PersistentKVStoreApplication) SetLogger(l log.Logger) {
	app.logger = l
}

func (app *PersistentKVStoreApplication) Info(req abci.RequestInfo) abci.ResponseInfo {
	res := app.app.Info(req)
	res.LastBlockHeight = app.app.state.Height
	res.LastBlockAppHash = app.app.state.AppHash
	return res
}

func (app *PersistentKVStoreApplication) SetOption(req abci.RequestSetOption) abci.ResponseSetOption {
	return app.app.SetOption(req)
}

// tx is either "val:pubkey!power" or "key=value" or just arbitrary bytes
func (app *PersistentKVStoreApplication) DeliverTx(req abci.RequestDeliverTx) abci.ResponseDeliverTx {
	// if it starts with "val:", update the validator set
	// format is "val:pubkey!power"
	if isValidatorTx(req.Tx) {
		// update validators in the merkle tree
		// and in app.ValUpdates
		return app.execValidatorTx(req.Tx)
	}

	// otherwise, update the key-value store
	return app.app.DeliverTx(req)
}

func (app *PersistentKVStoreApplication) CheckTxSync(req abci.RequestCheckTx) ocabci.ResponseCheckTx {
	return app.app.CheckTxSync(req)
}

func (app *PersistentKVStoreApplication) CheckTxAsync(req abci.RequestCheckTx, callback ocabci.CheckTxCallback) {
	app.app.CheckTxAsync(req, callback)
}

func (app *PersistentKVStoreApplication) BeginRecheckTx(req ocabci.RequestBeginRecheckTx) ocabci.ResponseBeginRecheckTx {
	return app.app.BeginRecheckTx(req)
}

func (app *PersistentKVStoreApplication) EndRecheckTx(req ocabci.RequestEndRecheckTx) ocabci.ResponseEndRecheckTx {
	return app.app.EndRecheckTx(req)
}

// Commit will panic if InitChain was not called
func (app *PersistentKVStoreApplication) Commit() abci.ResponseCommit {
	return app.app.Commit()
}

// When path=/val and data={validator address}, returns the validator update (abci.ValidatorUpdate) varint encoded.
// For any other path, returns an associated value or nil if missing.
func (app *PersistentKVStoreApplication) Query(reqQuery abci.RequestQuery) (resQuery abci.ResponseQuery) {
	switch reqQuery.Path {
	case "/val":
		key := []byte("val:" + string(reqQuery.Data))
		value, err := app.app.state.db.Get(key)
		if err != nil {
			panic(err)
		}

		resQuery.Key = reqQuery.Data
		resQuery.Value = value
		return
	default:
		return app.app.Query(reqQuery)
	}
}

// Save the validators in the merkle tree
func (app *PersistentKVStoreApplication) InitChain(req abci.RequestInitChain) abci.ResponseInitChain {
	for _, v := range req.Validators {
		r := app.updateValidator(v)
		if r.IsErr() {
			app.logger.Error("Error updating validators", "r", r)
		}
	}
	return abci.ResponseInitChain{}
}

// Track the block hash and header information
func (app *PersistentKVStoreApplication) BeginBlock(req ocabci.RequestBeginBlock) abci.ResponseBeginBlock {
	// reset valset changes
	app.ValUpdates = make([]abci.ValidatorUpdate, 0)

	// Punish validators who committed equivocation.
	for _, ev := range req.ByzantineValidators {
		if ev.Type == abci.EvidenceType_DUPLICATE_VOTE {
			addr := string(ev.Validator.Address)
			if pubKey, ok := app.valAddrToPubKeyMap[addr]; ok {
				app.updateValidator(abci.ValidatorUpdate{
					PubKey: pubKey,
					Power:  ev.Validator.Power - 1,
				})
				app.logger.Info("Decreased val power by 1 because of the equivocation",
					"val", addr)
			} else {
				app.logger.Error("Wanted to punish val, but can't find it",
					"val", addr)
			}
		}
	}

	return abci.ResponseBeginBlock{}
}

// Update the validator set
func (app *PersistentKVStoreApplication) EndBlock(req abci.RequestEndBlock) abci.ResponseEndBlock {
	return abci.ResponseEndBlock{ValidatorUpdates: app.ValUpdates}
}

func (app *PersistentKVStoreApplication) ListSnapshots(
	req abci.RequestListSnapshots) abci.ResponseListSnapshots {
	return abci.ResponseListSnapshots{}
}

func (app *PersistentKVStoreApplication) LoadSnapshotChunk(
	req abci.RequestLoadSnapshotChunk) abci.ResponseLoadSnapshotChunk {
	return abci.ResponseLoadSnapshotChunk{}
}

func (app *PersistentKVStoreApplication) OfferSnapshot(
	req abci.RequestOfferSnapshot) abci.ResponseOfferSnapshot {
	return abci.ResponseOfferSnapshot{Result: abci.ResponseOfferSnapshot_ABORT}
}

func (app *PersistentKVStoreApplication) ApplySnapshotChunk(
	req abci.RequestApplySnapshotChunk) abci.ResponseApplySnapshotChunk {
	return abci.ResponseApplySnapshotChunk{Result: abci.ResponseApplySnapshotChunk_ABORT}
}

//---------------------------------------------
// update validators

func (app *PersistentKVStoreApplication) Validators() (validators []abci.ValidatorUpdate) {
	itr, err := app.app.state.db.Iterator(nil, nil)
	if err != nil {
		panic(err)
	}
	for ; itr.Valid(); itr.Next() {
		if isValidatorTx(itr.Key()) {
			validator := new(abci.ValidatorUpdate)
			err := ocabci.ReadMessage(bytes.NewBuffer(itr.Value()), validator)
			if err != nil {
				panic(err)
			}
			validators = append(validators, *validator)
		}
	}
	if err = itr.Error(); err != nil {
		panic(err)
	}
	return
}

func MakeValSetChangeTx(pubkey pc.PublicKey, power int64) []byte {
	_, tx := MakeValSetChangeTxAndMore(pubkey, power)
	return []byte(tx)
}

func MakeValSetChangeTxAndMore(pubkey pc.PublicKey, power int64) (string, string) {
	pkBytes, err := pubkey.Marshal()
	if err != nil {
		panic(err)
	}
	pubStr := base64.StdEncoding.EncodeToString(pkBytes)
	return pubStr, fmt.Sprintf("val:%s!%d", pubStr, power)
}

func isValidatorTx(tx []byte) bool {
	return strings.HasPrefix(string(tx), ValidatorSetChangePrefix)
}

// format is "val:pubkey!power"
// pubkey is a base64-encoded proto.ostracon.crypto.PublicKey bytes
// See MakeValSetChangeTx
func (app *PersistentKVStoreApplication) execValidatorTx(tx []byte) abci.ResponseDeliverTx {
	tx = tx[len(ValidatorSetChangePrefix):]

	// get the pubkey and power
	pubKeyAndPower := strings.Split(string(tx), "!")
	if len(pubKeyAndPower) != 2 {
		return abci.ResponseDeliverTx{
			Code: code.CodeTypeEncodingError,
			Log:  fmt.Sprintf("Expected 'pubkey!power'. Got %v", pubKeyAndPower)}
	}
	pubkeyS, powerS := pubKeyAndPower[0], pubKeyAndPower[1]

	// decode the pubkey
	pkBytes, err := base64.StdEncoding.DecodeString(pubkeyS)
	if err != nil {
		return abci.ResponseDeliverTx{
			Code: code.CodeTypeEncodingError,
			Log:  fmt.Sprintf("pubkeyS (%s) is invalid base64", pubkeyS)}
	}
	var pkProto pc.PublicKey
	err = pkProto.Unmarshal(pkBytes)
	if err != nil {
		return abci.ResponseDeliverTx{
			Code: code.CodeTypeEncodingError,
			Log:  fmt.Sprintf("pkBytes (%x) is invalid binary", pkBytes)}
	}
	pubkey, err := cryptoenc.PubKeyFromProto(&pkProto)
	if err != nil {
		return abci.ResponseDeliverTx{
			Code: code.CodeTypeEncodingError,
			Log:  fmt.Sprintf("pkProto (%s) is invalid binary", pkProto)}
	}

	// decode the power
	power, err := strconv.ParseInt(powerS, 10, 64)
	if err != nil {
		return abci.ResponseDeliverTx{
			Code: code.CodeTypeEncodingError,
			Log:  fmt.Sprintf("Power (%s) is not an int", powerS)}
	}

	// update
	return app.updateValidator(ocabci.NewValidatorUpdate(pubkey, power))
}

// add, update, or remove a validator
// See MakeValSetChangeTx
func (app *PersistentKVStoreApplication) updateValidator(v abci.ValidatorUpdate) abci.ResponseDeliverTx {
	pubkey, err := cryptoenc.PubKeyFromProto(&v.PubKey)
	if err != nil {
		return abci.ResponseDeliverTx{
			Code: code.CodeTypeEncodingError,
			Log:  fmt.Sprintf("Error encoding Public Key: %s", err)}
	}
	pubStr, _ := MakeValSetChangeTxAndMore(v.PubKey, v.Power)
	key := []byte("val:" + pubStr)

	if v.Power == 0 {
		// remove validator
		hasKey, err := app.app.state.db.Has(key)
		if err != nil {
			panic(err)
		}
		if !hasKey {
			return abci.ResponseDeliverTx{
				Code: code.CodeTypeUnauthorized,
				Log:  fmt.Sprintf("Cannot remove non-existent validator %s", pubStr)}
		}
		if err = app.app.state.db.Delete(key); err != nil {
			panic(err)
		}
		delete(app.valAddrToPubKeyMap, string(pubkey.Address()))
	} else {
		// add or update validator
		value := bytes.NewBuffer(make([]byte, 0))
		if err := abci.WriteMessage(&v, value); err != nil {
			return abci.ResponseDeliverTx{
				Code: code.CodeTypeEncodingError,
				Log:  fmt.Sprintf("Error encoding validator: %v", err)}
		}
		if err = app.app.state.db.Set(key, value.Bytes()); err != nil {
			panic(err)
		}
		app.valAddrToPubKeyMap[string(pubkey.Address())] = v.PubKey
	}

	// we only update the changes array if we successfully updated the tree
	app.ValUpdates = append(app.ValUpdates, v)

	return abci.ResponseDeliverTx{Code: code.CodeTypeOK}
}
