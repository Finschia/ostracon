package light_test

import (
	"bytes"
	"fmt"
	"time"

	tmversion "github.com/tendermint/tendermint/proto/tendermint/version"

	"github.com/line/ostracon/crypto"
	"github.com/line/ostracon/crypto/ed25519"
	"github.com/line/ostracon/crypto/tmhash"
	tmbytes "github.com/line/ostracon/libs/bytes"
	"github.com/line/ostracon/libs/rand"
	tmproto "github.com/line/ostracon/proto/ostracon/types"
	"github.com/line/ostracon/types"
	tmtime "github.com/line/ostracon/types/time"
	"github.com/line/ostracon/version"
)

// privKeys is a helper type for testing.
//
// It lets us simulate signing with many keys.  The main use case is to create
// a set, and call GenSignedHeader to get properly signed header for testing.
//
// You can set different weights of validators each time you call ToValidators,
// and can optionally extend the validator set later with Extend.
type privKeys []crypto.PrivKey

// genPrivKeys produces an array of private keys to generate commits.
func genPrivKeys(n int) privKeys {
	res := make(privKeys, n)
	for i := range res {
		res[i] = ed25519.GenPrivKey()
	}
	return res
}

// // Change replaces the key at index i.
// func (pkz privKeys) Change(i int) privKeys {
// 	res := make(privKeys, len(pkz))
// 	copy(res, pkz)
// 	res[i] = ed25519.GenPrivKey()
// 	return res
// }

// Extend adds n more keys (to remove, just take a slice).
func (pkz privKeys) Extend(n int) privKeys {
	extra := genPrivKeys(n)
	return append(pkz, extra...)
}

// // GenSecpPrivKeys produces an array of secp256k1 private keys to generate commits.
// func GenSecpPrivKeys(n int) privKeys {
// 	res := make(privKeys, n)
// 	for i := range res {
// 		res[i] = secp256k1.GenPrivKey()
// 	}
// 	return res
// }

// // ExtendSecp adds n more secp256k1 keys (to remove, just take a slice).
// func (pkz privKeys) ExtendSecp(n int) privKeys {
// 	extra := GenSecpPrivKeys(n)
// 	return append(pkz, extra...)
// }

// ToValidators produces a valset from the set of keys.
// The first key has weight `init` and it increases by `inc` every step
// so we can have all the same weight, or a simple linear distribution
// (should be enough for testing).
func (pkz privKeys) ToValidators(init, inc int64) *types.ValidatorSet {
	res := make([]*types.Validator, len(pkz))
	for i, k := range pkz {
		res[i] = types.NewValidator(k.PubKey(), init+int64(i)*inc)
	}
	return types.NewValidatorSet(res)
}

// signHeader properly signs the header with all keys from first to last exclusive.
func (pkz privKeys) signHeader(header *types.Header, valSet *types.ValidatorSet, first, last int) *types.Commit {
	commitSigs := make([]types.CommitSig, len(pkz))
	for i := 0; i < len(pkz); i++ {
		commitSigs[i] = types.NewCommitSigAbsent()
	}

	blockID := types.BlockID{
		Hash:          header.Hash(),
		PartSetHeader: types.PartSetHeader{Total: 1, Hash: crypto.CRandBytes(32)},
	}

	// Fill in the votes we want.
	for i := first; i < last && i < len(pkz); i++ {
		vote := makeVote(header, valSet, pkz[i], blockID)
		commitSigs[vote.ValidatorIndex] = vote.CommitSig()
	}

	return types.NewCommit(header.Height, 1, blockID, commitSigs)
}

func (pkz privKeys) signHeaderByRate(header *types.Header, valSet *types.ValidatorSet, rate float64) *types.Commit {
	commitSigs := make([]types.CommitSig, len(pkz))
	for i := 0; i < len(pkz); i++ {
		commitSigs[i] = types.NewCommitSigAbsent()
	}

	blockID := types.BlockID{
		Hash:          header.Hash(),
		PartSetHeader: types.PartSetHeader{Total: 1, Hash: crypto.CRandBytes(32)},
	}

	// Fill in the votes we want.
	until := int64(float64(valSet.TotalVotingPower()) * rate)
	sum := int64(0)
	for i := 0; i < len(pkz); i++ {
		_, val := valSet.GetByAddress(pkz[i].PubKey().Address())
		if val == nil {
			continue
		}
		vote := makeVote(header, valSet, pkz[i], blockID)
		commitSigs[vote.ValidatorIndex] = vote.CommitSig()

		sum += val.VotingPower
		if sum > until {
			break
		}
	}

	return types.NewCommit(header.Height, 1, blockID, commitSigs)
}

func makeVote(header *types.Header, valset *types.ValidatorSet,
	key crypto.PrivKey, blockID types.BlockID) *types.Vote {

	addr := key.PubKey().Address()
	idx, _ := valset.GetByAddress(addr)
	if idx < 0 {
		panic(fmt.Sprintf("address %X is not contained in ValSet: %+v", addr, valset))
	}
	vote := &types.Vote{
		ValidatorAddress: addr,
		ValidatorIndex:   idx,
		Height:           header.Height,
		Round:            1,
		Timestamp:        tmtime.Now(),
		Type:             tmproto.PrecommitType,
		BlockID:          blockID,
	}

	v := vote.ToProto()
	// Sign it
	signBytes := types.VoteSignBytes(header.ChainID, v)
	sig, err := key.Sign(signBytes)
	if err != nil {
		panic(err)
	}

	vote.Signature = sig

	return vote
}

func genHeader(chainID string, height int64, bTime time.Time, txs types.Txs,
	valset, nextValset *types.ValidatorSet, appHash, consHash, resHash []byte, proof tmbytes.HexBytes) *types.Header {

	return &types.Header{
		Version: tmversion.Consensus{Block: version.BlockProtocol, App: version.AppProtocol},
		ChainID: chainID,
		Height:  height,
		Time:    bTime,
		// LastBlockID
		// LastCommitHash
		ValidatorsHash:     valset.Hash(),
		NextValidatorsHash: nextValset.Hash(),
		DataHash:           txs.Hash(),
		AppHash:            appHash,
		Proof:              proof,
		ConsensusHash:      consHash,
		LastResultsHash:    resHash,
		ProposerAddress:    valset.SelectProposer(proof, height, 0).Address,
	}
}

// GenSignedHeader calls genHeader and signHeader and combines them into a SignedHeader.
func (pkz privKeys) GenSignedHeader(chainID string, height int64, bTime time.Time, txs types.Txs,
	valset, nextValset *types.ValidatorSet, appHash, consHash, resHash []byte,
	first, last int) *types.SignedHeader {

	secret := [64]byte{}
	privateKey := ed25519.GenPrivKeyFromSecret(secret[:])
	message := rand.Bytes(10)
	proof, _ := privateKey.VRFProve(message)

	header := genHeader(chainID, height, bTime, txs, valset, nextValset, appHash, consHash, resHash,
		tmbytes.HexBytes(proof))
	return &types.SignedHeader{
		Header: header,
		Commit: pkz.signHeader(header, valset, first, last),
	}
}

func (pkz privKeys) GenSignedHeaderByRate(chainID string, height int64, bTime time.Time, txs types.Txs,
	valset, nextValset *types.ValidatorSet, appHash, consHash, resHash []byte,
	rate float64) *types.SignedHeader {

	secret := [64]byte{}
	privateKey := ed25519.GenPrivKeyFromSecret(secret[:])
	message := rand.Bytes(10)
	proof, _ := privateKey.VRFProve(message)

	header := genHeader(chainID, height, bTime, txs, valset, nextValset, appHash, consHash, resHash,
		tmbytes.HexBytes(proof))
	return &types.SignedHeader{
		Header: header,
		Commit: pkz.signHeaderByRate(header, valset, rate),
	}
}

// GenSignedHeaderLastBlockID calls genHeader and signHeader and combines them into a SignedHeader.
func (pkz privKeys) GenSignedHeaderLastBlockID(chainID string, height int64, bTime time.Time, txs types.Txs,
	valset, nextValset *types.ValidatorSet, appHash, consHash, resHash []byte, first, last int,
	lastBlockID types.BlockID) *types.SignedHeader {

	secret := [64]byte{}
	privateKey := ed25519.GenPrivKeyFromSecret(secret[:])
	message := rand.Bytes(10)
	proof, _ := privateKey.VRFProve(message)

	header := genHeader(chainID, height, bTime, txs, valset, nextValset, appHash, consHash, resHash,
		tmbytes.HexBytes(proof))
	header.LastBlockID = lastBlockID
	return &types.SignedHeader{
		Header: header,
		Commit: pkz.signHeader(header, valset, first, last),
	}
}

func (pkz privKeys) ChangeKeys(delta int) privKeys {
	newKeys := pkz[delta:]
	return newKeys.Extend(delta)
}

func genCalcValVariationFunc() func(valVariation float32) int {
	totalVariation := float32(0)
	valVariationInt := int(totalVariation)
	return func(valVariation float32) int {
		totalVariation += valVariation
		valVariationInt = int(totalVariation)
		totalVariation = -float32(valVariationInt)
		return valVariationInt
	}
}

func genMockNodeWithKey(
	chainID string,
	height int64,
	txs types.Txs,
	keys, newKeys privKeys,
	valSet, newValSet *types.ValidatorSet,
	lastHeader *types.SignedHeader,
	bTime time.Time,
	first, last int,
	valVariation float32,
	calcValVariation func(valVariation float32) int) (
	*types.SignedHeader, *types.ValidatorSet, privKeys) {

	if newKeys == nil {
		newKeys = keys.ChangeKeys(calcValVariation(valVariation))
	}
	if valSet == nil {
		valSet = keys.ToValidators(2, 2)
	}
	if newValSet == nil {
		newValSet = newKeys.ToValidators(2, 2)
	}
	if lastHeader == nil {
		header := keys.GenSignedHeader(
			chainID, height, bTime,
			txs, valSet, newValSet,
			hash("app_hash"), hash("cons_hash"), hash("results_hash"),
			first, last,
		)
		return header, valSet, newKeys
	}
	header := keys.GenSignedHeaderLastBlockID(
		chainID, height, bTime,
		nil, valSet, newValSet,
		hash("app_hash"), hash("cons_hash"), hash("results_hash"),
		0, len(keys),
		types.BlockID{Hash: lastHeader.Hash()},
	)
	return header, valSet, newKeys
}

// Generates the header and validator set to create a full entire mock node with blocks to height (
// blockSize) and with variation in validator sets. BlockIntervals are in per minute.
// NOTE: Expected to have a large validator set size ~ 100 validators.
func genMockNodeWithKeys(chainID string, blockSize int64, valSize int, valVariation float32, bTime time.Time) (
	map[int64]*types.SignedHeader,
	map[int64]*types.ValidatorSet,
	map[int64]privKeys) {

	var (
		headers = make(map[int64]*types.SignedHeader, blockSize)
		valSet  = make(map[int64]*types.ValidatorSet, blockSize+1)
		keymap  = make(map[int64]privKeys, blockSize+1)
	)

	setter := func(height int64,
		header *types.SignedHeader, vals *types.ValidatorSet, newKeys privKeys) {
		headers[height] = header
		valSet[height] = vals
		keymap[height+1] = newKeys
	}

	height := int64(1)
	keys := genPrivKeys(valSize)
	keymap[height] = keys
	calcValVariationFunc := genCalcValVariationFunc()

	header, vals, newKeys := genMockNodeWithKey(chainID, height, nil,
		keys, nil,
		nil, nil, nil,
		bTime.Add(time.Duration(height)*time.Minute),
		0, len(keys),
		valVariation, calcValVariationFunc)

	// genesis header and vals
	setter(height, header, vals, newKeys)

	keys = newKeys
	lastHeader := header

	for height := int64(2); height <= blockSize; height++ {
		header, vals, newKeys := genMockNodeWithKey(chainID, height, nil,
			keys, nil,
			nil, nil, lastHeader,
			bTime.Add(time.Duration(height)*time.Minute),
			0, len(keys),
			valVariation, calcValVariationFunc)
		if !bytes.Equal(header.Hash(), header.Commit.BlockID.Hash) {
			panic(fmt.Sprintf("commit hash didn't match: %X != %X", header.Hash(), header.Commit.BlockID.Hash))
		}
		setter(height, header, vals, newKeys)

		keys = newKeys
		lastHeader = header
	}

	return headers, valSet, keymap
}

func genMockNode(
	chainID string,
	blockSize int64,
	valSize int,
	valVariation float32,
	bTime time.Time) (
	string,
	map[int64]*types.SignedHeader,
	map[int64]*types.ValidatorSet) {
	headers, valset, _ := genMockNodeWithKeys(chainID, blockSize, valSize, valVariation, bTime)
	return chainID, headers, valset
}

func hash(s string) []byte {
	return tmhash.Sum([]byte(s))
}
