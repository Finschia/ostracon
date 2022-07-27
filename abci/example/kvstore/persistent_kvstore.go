package kvstore

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	dbm "github.com/tendermint/tm-db"

	"github.com/line/ostracon/abci/example/code"
	"github.com/line/ostracon/abci/types"
	cryptoenc "github.com/line/ostracon/crypto/encoding"
	"github.com/line/ostracon/libs/log"
	pc "github.com/line/ostracon/proto/ostracon/crypto"
)

const (
	ValidatorSetChangePrefix string = "val:"
)

//-----------------------------------------

var _ types.Application = (*PersistentKVStoreApplication)(nil)

type PersistentKVStoreApplication struct {
	app *Application

	// validator set
	ValUpdates []types.ValidatorUpdate

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

func (app *PersistentKVStoreApplication) Info(req types.RequestInfo) types.ResponseInfo {
	res := app.app.Info(req)
	res.LastBlockHeight = app.app.state.Height
	res.LastBlockAppHash = app.app.state.AppHash
	return res
}

func (app *PersistentKVStoreApplication) SetOption(req types.RequestSetOption) types.ResponseSetOption {
	return app.app.SetOption(req)
}

// tx is either "val:pubkey!power" or "key=value" or just arbitrary bytes
func (app *PersistentKVStoreApplication) DeliverTx(req types.RequestDeliverTx) types.ResponseDeliverTx {
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

func (app *PersistentKVStoreApplication) CheckTxSync(req types.RequestCheckTx) types.ResponseCheckTx {
	return app.app.CheckTxSync(req)
}

func (app *PersistentKVStoreApplication) CheckTxAsync(req types.RequestCheckTx, callback types.CheckTxCallback) {
	app.app.CheckTxAsync(req, callback)
}

func (app *PersistentKVStoreApplication) BeginRecheckTx(req types.RequestBeginRecheckTx) types.ResponseBeginRecheckTx {
	return app.app.BeginRecheckTx(req)
}

func (app *PersistentKVStoreApplication) EndRecheckTx(req types.RequestEndRecheckTx) types.ResponseEndRecheckTx {
	return app.app.EndRecheckTx(req)
}

// Commit will panic if InitChain was not called
func (app *PersistentKVStoreApplication) Commit() types.ResponseCommit {
	return app.app.Commit()
}

// When path=/val and data={validator address}, returns the validator update (types.ValidatorUpdate) varint encoded.
// For any other path, returns an associated value or nil if missing.
func (app *PersistentKVStoreApplication) Query(reqQuery types.RequestQuery) (resQuery types.ResponseQuery) {
	switch reqQuery.Path {
	case "/val":
		key := []byte("val:" + string(reqQuery.Data))
		value, err := app.app.state.db.Get(key)
		if err != nil {
			panic(err)
		}

		resQuery.Key = reqQuery.Data
		resQuery.Value = value

		if value == nil {
			resQuery.Log = "cannot get"
			return
		}
		validatorUpdate := types.ValidatorUpdate{}
		err = types.ReadMessage(bytes.NewReader(resQuery.Value), &validatorUpdate)
		if err != nil {
			panic(err)
		}
		pubKey, _ := cryptoenc.PubKeyFromProto(&validatorUpdate.PubKey)
		resQuery.Log = fmt.Sprintf("key=%s, validatorUpdate.PubKey=%v, validatorUpdate.Power=%d",
			resQuery.Key, pubKey, validatorUpdate.Power)
		return
	default:
		return app.app.Query(reqQuery)
	}
}

// Save the validators in the merkle tree
func (app *PersistentKVStoreApplication) InitChain(req types.RequestInitChain) types.ResponseInitChain {
	for _, v := range req.Validators {
		r := app.updateValidator(v)
		if r.IsErr() {
			app.logger.Error("Error updating validators", "r", r)
		}
	}
	return types.ResponseInitChain{}
}

// Track the block hash and header information
func (app *PersistentKVStoreApplication) BeginBlock(req types.RequestBeginBlock) types.ResponseBeginBlock {
	// reset valset changes
	app.ValUpdates = make([]types.ValidatorUpdate, 0)

	// Punish validators who committed equivocation.
	for _, ev := range req.ByzantineValidators {
		if ev.Type == types.EvidenceType_DUPLICATE_VOTE {
			addr := string(ev.Validator.Address)
			if pubKey, ok := app.valAddrToPubKeyMap[addr]; ok {
				app.updateValidator(types.ValidatorUpdate{
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

	return types.ResponseBeginBlock{}
}

// Update the validator set
func (app *PersistentKVStoreApplication) EndBlock(req types.RequestEndBlock) types.ResponseEndBlock {
	return types.ResponseEndBlock{ValidatorUpdates: app.ValUpdates}
}

func (app *PersistentKVStoreApplication) ListSnapshots(
	req types.RequestListSnapshots) types.ResponseListSnapshots {
	return types.ResponseListSnapshots{}
}

func (app *PersistentKVStoreApplication) LoadSnapshotChunk(
	req types.RequestLoadSnapshotChunk) types.ResponseLoadSnapshotChunk {
	return types.ResponseLoadSnapshotChunk{}
}

func (app *PersistentKVStoreApplication) OfferSnapshot(
	req types.RequestOfferSnapshot) types.ResponseOfferSnapshot {
	return types.ResponseOfferSnapshot{Result: types.ResponseOfferSnapshot_ABORT}
}

func (app *PersistentKVStoreApplication) ApplySnapshotChunk(
	req types.RequestApplySnapshotChunk) types.ResponseApplySnapshotChunk {
	return types.ResponseApplySnapshotChunk{Result: types.ResponseApplySnapshotChunk_ABORT}
}

//---------------------------------------------
// update validators

func (app *PersistentKVStoreApplication) Validators() (validators []types.ValidatorUpdate) {
	itr, err := app.app.state.db.Iterator(nil, nil)
	if err != nil {
		panic(err)
	}
	for ; itr.Valid(); itr.Next() {
		if isValidatorTx(itr.Key()) {
			validator := new(types.ValidatorUpdate)
			err := types.ReadMessage(bytes.NewBuffer(itr.Value()), validator)
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
func (app *PersistentKVStoreApplication) execValidatorTx(tx []byte) types.ResponseDeliverTx {
	tx = tx[len(ValidatorSetChangePrefix):]

	// get the pubkey and power
	pubKeyAndPower := strings.Split(string(tx), "!")
	if len(pubKeyAndPower) != 2 {
		return types.ResponseDeliverTx{
			Code: code.CodeTypeEncodingError,
			Log:  fmt.Sprintf("Expected 'pubkey!power'. Got %v", pubKeyAndPower)}
	}
	pubkeyS, powerS := pubKeyAndPower[0], pubKeyAndPower[1]

	// decode the pubkey
	pkBytes, err := base64.StdEncoding.DecodeString(pubkeyS)
	if err != nil {
		return types.ResponseDeliverTx{
			Code: code.CodeTypeEncodingError,
			Log:  fmt.Sprintf("pubkeyS (%s) is invalid base64", pubkeyS)}
	}
	var pkProto pc.PublicKey
	err = pkProto.Unmarshal(pkBytes)
	if err != nil {
		return types.ResponseDeliverTx{
			Code: code.CodeTypeEncodingError,
			Log:  fmt.Sprintf("pkBytes (%x) is invalid binary", pkBytes)}
	}
	pubkey, err := cryptoenc.PubKeyFromProto(&pkProto)
	if err != nil {
		return types.ResponseDeliverTx{
			Code: code.CodeTypeEncodingError,
			Log:  fmt.Sprintf("pkProto (%s) is invalid binary", pkProto)}
	}

	// decode the power
	power, err := strconv.ParseInt(powerS, 10, 64)
	if err != nil {
		return types.ResponseDeliverTx{
			Code: code.CodeTypeEncodingError,
			Log:  fmt.Sprintf("Power (%s) is not an int", powerS)}
	}

	// update
	return app.updateValidator(types.NewValidatorUpdate(pubkey, power))
}

// add, update, or remove a validator
// See MakeValSetChangeTx
func (app *PersistentKVStoreApplication) updateValidator(v types.ValidatorUpdate) types.ResponseDeliverTx {
	pubkey, err := cryptoenc.PubKeyFromProto(&v.PubKey)
	if err != nil {
		return types.ResponseDeliverTx{
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
			return types.ResponseDeliverTx{
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
		if err := types.WriteMessage(&v, value); err != nil {
			return types.ResponseDeliverTx{
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

	return types.ResponseDeliverTx{Code: code.CodeTypeOK}
}
