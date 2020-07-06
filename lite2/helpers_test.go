package lite_test

import (
	"time"

	"github.com/tendermint/tendermint/crypto/vrf"
	"github.com/tendermint/tendermint/libs/rand"

	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/crypto/tmhash"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"
	"github.com/tendermint/tendermint/types"
	tmtime "github.com/tendermint/tendermint/types/time"
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

func (pkz privKeys) ToVoters(init, inc int64) *types.VoterSet {
	res := make([]*types.Validator, len(pkz))
	for i, k := range pkz {
		res[i] = types.NewValidator(k.PubKey(), init+int64(i)*inc)
	}
	return types.ToVoterAll(res)
}

// signHeader properly signs the header with all keys from first to last exclusive.
func (pkz privKeys) signHeader(header *types.Header, voterSet *types.VoterSet, first, last int) *types.Commit {
	commitSigs := make([]types.CommitSig, voterSet.Size())
	for i := 0; i < len(commitSigs); i++ {
		commitSigs[i] = types.NewCommitSigAbsent()
	}

	blockID := types.BlockID{
		Hash:        header.Hash(),
		PartsHeader: types.PartSetHeader{Total: 1, Hash: crypto.CRandBytes(32)},
	}

	// Fill in the votes we want.
	for i := first; i < last && i < len(pkz); i++ {
		idx, voter := voterSet.GetByAddress(pkz[i].PubKey().Address())
		if voter == nil {
			continue
		}
		vote := makeVote(header, idx, pkz[i], blockID)
		commitSigs[vote.ValidatorIndex] = vote.CommitSig()
	}

	return types.NewCommit(header.Height, 1, blockID, commitSigs)
}

func (pkz privKeys) signHeaderByRate(header *types.Header, voterSet *types.VoterSet, rate float64) *types.Commit {
	commitSigs := make([]types.CommitSig, voterSet.Size())
	for i := 0; i < len(commitSigs); i++ {
		commitSigs[i] = types.NewCommitSigAbsent()
	}

	blockID := types.BlockID{
		Hash:        header.Hash(),
		PartsHeader: types.PartSetHeader{Total: 1, Hash: crypto.CRandBytes(32)},
	}

	// Fill in the votes we want.
	until := int64(float64(voterSet.TotalVotingPower()) * rate)
	sum := int64(0)
	for i := 0; i < len(pkz); i++ {
		idx, voter := voterSet.GetByAddress(pkz[i].PubKey().Address())
		if voter == nil {
			continue
		}
		vote := makeVote(header, idx, pkz[i], blockID)
		commitSigs[vote.ValidatorIndex] = vote.CommitSig()

		sum += voter.VotingPower
		if sum > until {
			break
		}
	}

	return types.NewCommit(header.Height, 1, blockID, commitSigs)
}

func makeVote(header *types.Header, voterIdx int, key crypto.PrivKey, blockID types.BlockID) *types.Vote {

	addr := key.PubKey().Address()
	vote := &types.Vote{
		ValidatorAddress: addr,
		ValidatorIndex:   voterIdx,
		Height:           header.Height,
		Round:            1,
		Timestamp:        tmtime.Now(),
		Type:             types.PrecommitType,
		BlockID:          blockID,
	}
	// Sign it
	signBytes := vote.SignBytes(header.ChainID)
	// TODO Consider reworking makeVote API to return an error
	sig, err := key.Sign(signBytes)
	if err != nil {
		panic(err)
	}
	vote.Signature = sig

	return vote
}

func genHeader(chainID string, height int64, bTime time.Time, txs types.Txs,
	voterSet *types.VoterSet, valset, nextValset *types.ValidatorSet, appHash, consHash, resHash []byte,
	proof tmbytes.HexBytes) *types.Header {

	return &types.Header{
		ChainID: chainID,
		Height:  height,
		Time:    bTime,
		// LastBlockID
		// LastCommitHash
		VotersHash:         voterSet.Hash(),
		ValidatorsHash:     valset.Hash(),
		NextValidatorsHash: nextValset.Hash(),
		DataHash:           txs.Hash(),
		AppHash:            appHash,
		Proof:              proof,
		ConsensusHash:      consHash,
		LastResultsHash:    resHash,
		ProposerAddress:    voterSet.Voters[0].Address,
	}
}

// GenSignedHeader calls genHeader and signHeader and combines them into a SignedHeader.
func (pkz privKeys) GenSignedHeader(chainID string, height int64, bTime time.Time, txs types.Txs,
	valset, nextValset *types.ValidatorSet, appHash, consHash, resHash []byte,
	first, last int, voterParams *types.VoterParams) *types.SignedHeader {

	secret := [64]byte{}
	privateKey := ed25519.GenPrivKeyFromSecret(secret[:])
	message := rand.Bytes(10)
	proof, _ := vrf.Prove(privateKey, message)
	proofHash, _ := vrf.ProofToHash(proof)
	voterSet := types.SelectVoter(valset, proofHash, voterParams)

	header := genHeader(chainID, height, bTime, txs, voterSet, valset, nextValset, appHash, consHash, resHash,
		tmbytes.HexBytes(proof))
	return &types.SignedHeader{
		Header: header,
		Commit: pkz.signHeader(header, voterSet, first, last),
	}
}

func (pkz privKeys) GenSignedHeaderByRate(chainID string, height int64, bTime time.Time, txs types.Txs,
	valset, nextValset *types.ValidatorSet, appHash, consHash, resHash []byte,
	rate float64, voterParams *types.VoterParams) *types.SignedHeader {

	secret := [64]byte{}
	privateKey := ed25519.GenPrivKeyFromSecret(secret[:])
	message := rand.Bytes(10)
	proof, _ := vrf.Prove(privateKey, message)
	proofHash, _ := vrf.ProofToHash(proof)
	voterSet := types.SelectVoter(valset, proofHash, voterParams)

	header := genHeader(chainID, height, bTime, txs, voterSet, valset, nextValset, appHash, consHash, resHash,
		tmbytes.HexBytes(proof))
	return &types.SignedHeader{
		Header: header,
		Commit: pkz.signHeaderByRate(header, voterSet, rate),
	}
}

// GenSignedHeaderLastBlockID calls genHeader and signHeader and combines them into a SignedHeader.
func (pkz privKeys) GenSignedHeaderLastBlockID(chainID string, height int64, bTime time.Time, txs types.Txs,
	valset, nextValset *types.ValidatorSet, appHash, consHash, resHash []byte, first, last int,
	lastBlockID types.BlockID, voterParams *types.VoterParams) *types.SignedHeader {

	secret := [64]byte{}
	privateKey := ed25519.GenPrivKeyFromSecret(secret[:])
	message := rand.Bytes(10)
	proof, _ := vrf.Prove(privateKey, message)
	proofHash, _ := vrf.ProofToHash(proof)
	voterSet := types.SelectVoter(valset, proofHash, voterParams)

	header := genHeader(chainID, height, bTime, txs, voterSet, valset, nextValset, appHash, consHash, resHash,
		tmbytes.HexBytes(proof))
	header.LastBlockID = lastBlockID
	return &types.SignedHeader{
		Header: header,
		Commit: pkz.signHeader(header, voterSet, first, last),
	}
}

func (pkz privKeys) ChangeKeys(delta int) privKeys {
	newKeys := pkz[delta:]
	return newKeys.Extend(delta)
}

// Generates the header and validator set to create a full entire mock node with blocks to height (
// blockSize) and with variation in validator sets. BlockIntervals are in per minute.
// NOTE: Expected to have a large validator set size ~ 100 validators.
func GenMockNode(
	chainID string,
	blockSize int64,
	valSize int,
	valVariation float32,
	bTime time.Time) (
	string,
	map[int64]*types.SignedHeader,
	map[int64]*types.ValidatorSet) {

	var (
		headers         = make(map[int64]*types.SignedHeader, blockSize)
		valSet          = make(map[int64]*types.ValidatorSet, blockSize)
		keys            = genPrivKeys(valSize)
		totalVariation  = valVariation
		valVariationInt int
		newKeys         privKeys
	)

	valVariationInt = int(totalVariation)
	totalVariation = -float32(valVariationInt)
	newKeys = keys.ChangeKeys(valVariationInt)

	// genesis header and vals
	vals := keys.ToValidators(2, 2)
	lastHeader := keys.GenSignedHeader(chainID, 1, bTime.Add(1*time.Minute), nil,
		vals, newKeys.ToValidators(2, 2), hash("app_hash"), hash("cons_hash"),
		hash("results_hash"), 0, len(keys), types.DefaultVoterParams())
	currentHeader := lastHeader
	headers[1] = currentHeader
	valSet[1] = keys.ToValidators(2, 2)
	keys = newKeys

	for height := int64(2); height <= blockSize; height++ {
		totalVariation += valVariation
		valVariationInt = int(totalVariation)
		totalVariation = -float32(valVariationInt)
		newKeys = keys.ChangeKeys(valVariationInt)
		vals = keys.ToValidators(2, 2)
		currentHeader = keys.GenSignedHeaderLastBlockID(chainID, height, bTime.Add(time.Duration(height)*time.Minute),
			nil,
			vals, newKeys.ToValidators(2, 2), hash("app_hash"), hash("cons_hash"),
			hash("results_hash"), 0, len(keys), types.BlockID{Hash: lastHeader.Hash()}, types.DefaultVoterParams())
		headers[height] = currentHeader
		valSet[height] = keys.ToValidators(2, 2)
		lastHeader = currentHeader
		keys = newKeys
	}

	return chainID, headers, valSet
}

func hash(s string) []byte {
	return tmhash.Sum([]byte(s))
}
