package types

import (
	"testing"

	tmproto "github.com/line/ostracon/proto/ostracon/types"

	"github.com/line/ostracon/crypto/secp256k1"

	"github.com/golang/protobuf/proto" // nolint: staticcheck // still used by gogoproto
	"github.com/tendermint/go-amino"

	"github.com/line/ostracon/proto/ostracon/version"
	"github.com/line/ostracon/types/time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	abci "github.com/line/ostracon/abci/types"
	"github.com/line/ostracon/crypto"
	"github.com/line/ostracon/crypto/bls"
	"github.com/line/ostracon/crypto/composite"
	"github.com/line/ostracon/crypto/ed25519"
	cryptoenc "github.com/line/ostracon/crypto/encoding"
)

func TestABCIPubKey(t *testing.T) {
	pkEd := ed25519.GenPrivKey().PubKey()
	pkSecp := secp256k1.GenPrivKey().PubKey()
	pkComposite := composite.NewPrivKeyComposite(bls.GenPrivKey(), ed25519.GenPrivKey()).PubKey()
	err := testABCIPubKey(t, pkEd, ABCIPubKeyTypeEd25519)
	assert.NoError(t, err)
	err = testABCIPubKey(t, pkSecp, ABCIPubKeyTypeSecp256k1)
	assert.NoError(t, err)
	err = testABCIPubKey(t, pkComposite, ABCIPubKeyTypeBls12WithEd25519)
	assert.NoError(t, err)
}

func testABCIPubKey(t *testing.T, pk crypto.PubKey, typeStr string) error {
	abciPubKey, err := cryptoenc.PubKeyToProto(pk)
	require.NoError(t, err)
	pk2, err := cryptoenc.PubKeyFromProto(&abciPubKey)
	require.NoError(t, err)
	require.Equal(t, pk, pk2)
	return nil
}

func TestABCIValidators(t *testing.T) {
	pkEd := ed25519.GenPrivKey().PubKey()

	// correct validator
	tmValExpected := NewValidator(pkEd, 10)

	tmVal := NewValidator(pkEd, 10)

	abciVal := TM2PB.ValidatorUpdate(tmVal)
	tmVals, err := PB2TM.ValidatorUpdates([]abci.ValidatorUpdate{abciVal})
	assert.Nil(t, err)
	assert.Equal(t, tmValExpected, tmVals[0])

	abciVals := TM2PB.ValidatorUpdates(NewValidatorSet(tmVals))
	assert.Equal(t, []abci.ValidatorUpdate{abciVal}, abciVals)

	// val with address
	tmVal.Address = pkEd.Address()

	abciVal = TM2PB.ValidatorUpdate(tmVal)
	tmVals, err = PB2TM.ValidatorUpdates([]abci.ValidatorUpdate{abciVal})
	assert.Nil(t, err)
	assert.Equal(t, tmValExpected, tmVals[0])
}

func TestABCIConsensusParams(t *testing.T) {
	cp := DefaultConsensusParams()
	abciCP := TM2PB.ConsensusParams(cp)
	cp2 := UpdateConsensusParams(*cp, abciCP)

	assert.Equal(t, *cp, cp2)
}

func newHeader(
	height int64, commitHash, dataHash, evidenceHash []byte,
) *Header {
	return &Header{
		Height:         height,
		LastCommitHash: commitHash,
		DataHash:       dataHash,
		EvidenceHash:   evidenceHash,
	}
}

func TestABCIHeader(t *testing.T) {
	// build a full header
	var height int64 = 5
	header := newHeader(height, []byte("lastCommitHash"), []byte("dataHash"), []byte("evidenceHash"))
	protocolVersion := version.Consensus{Block: 7, App: 8}
	timestamp := time.Now()
	lastBlockID := BlockID{
		Hash: []byte("hash"),
		PartSetHeader: PartSetHeader{
			Total: 10,
			Hash:  []byte("hash"),
		},
	}
	header.Populate(
		protocolVersion, "chainID", timestamp, lastBlockID,
		[]byte("votersHash"), []byte("valHash"), []byte("nextValHash"),
		[]byte("consHash"), []byte("appHash"), []byte("lastResultsHash"),
		[]byte("proposerAddress"), 0, []byte("lastProof"),
	)

	cdc := amino.NewCodec()
	headerBz := cdc.MustMarshalBinaryBare(header)

	pbHeader := TM2PB.Header(header)
	pbHeaderBz, err := proto.Marshal(&pbHeader)
	assert.NoError(t, err)

	// assert some fields match
	assert.EqualValues(t, protocolVersion.Block, pbHeader.Version.Block)
	assert.EqualValues(t, protocolVersion.App, pbHeader.Version.App)
	assert.EqualValues(t, "chainID", pbHeader.ChainID)
	assert.EqualValues(t, height, pbHeader.Height)
	assert.EqualValues(t, timestamp, pbHeader.Time)
	assert.EqualValues(t, lastBlockID.Hash, pbHeader.LastBlockId.Hash)
	assert.EqualValues(t, []byte("lastCommitHash"), pbHeader.LastCommitHash)
	assert.Equal(t, []byte("proposerAddress"), pbHeader.ProposerAddress)

	// assert the encodings match
	// NOTE: they don't yet because Amino encodes
	// int64 as zig-zag and we're using non-zigzag in the protobuf.
	// See https://github.com/line/ostracon/issues/2682
	_, _ = headerBz, pbHeaderBz
	// assert.EqualValues(t, headerBz, pbHeaderBz)

}

func TestABCIEvidence(t *testing.T) {
	forAllPrivKeyTypes(t, func(t *testing.T, name string, kt PrivKeyType) {
		val := NewMockPV(kt)
		blockID := makeBlockID([]byte("blockhash"), 1000, []byte("partshash"))
		blockID2 := makeBlockID([]byte("blockhash2"), 1000, []byte("partshash"))
		const chainID = "mychain"
		now := time.Now()
		ev := &DuplicateVoteEvidence{
			VoteA:            makeVote(t, val, chainID, 0, 10, 2, tmproto.PrevoteType, blockID, now),
			VoteB:            makeVote(t, val, chainID, 0, 10, 2, tmproto.PrevoteType, blockID2, now),
			TotalVotingPower: int64(100),
			ValidatorPower:   int64(10),
			Timestamp:        now,
		}
		for _, abciEv := range ev.ABCI() {
			assert.Equal(t, "DUPLICATE_VOTE", abciEv.Type.String())
		}
	})
}

type pubKeyEddie struct{}

func (pubKeyEddie) Address() Address                                                { return []byte{} }
func (pubKeyEddie) Bytes() []byte                                                   { return []byte{} }
func (pubKeyEddie) VerifySignature(msg []byte, sig []byte) bool                     { return false }
func (pubKeyEddie) VRFVerify(proof crypto.Proof, msg []byte) (crypto.Output, error) { return nil, nil }
func (pubKeyEddie) Equals(crypto.PubKey) bool                                       { return false }
func (pubKeyEddie) String() string                                                  { return "" }
func (pubKeyEddie) Type() string                                                    { return "pubKeyEddie" }

func TestABCIValidatorFromPubKeyAndPower(t *testing.T) {
	pubkey := ed25519.GenPrivKey().PubKey()

	abciVal := TM2PB.NewValidatorUpdate(pubkey, 10)
	assert.Equal(t, int64(10), abciVal.Power)

	assert.Panics(t, func() { TM2PB.NewValidatorUpdate(nil, 10) })
	assert.Panics(t, func() { TM2PB.NewValidatorUpdate(pubKeyEddie{}, 10) })
}

func TestABCIValidatorWithoutPubKey(t *testing.T) {
	pkEd := ed25519.GenPrivKey().PubKey()

	abciVal := TM2PB.Validator(NewValidator(pkEd, 10))

	// pubkey must be nil
	tmValExpected := abci.Validator{
		Address: pkEd.Address(),
		Power:   10,
	}

	assert.Equal(t, tmValExpected, abciVal)
}
