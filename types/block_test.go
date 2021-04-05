package types

import (
	// it is ok to use math/rand here: we do not need a cryptographically secure random
	// number generator here and we can run the tests a bit faster
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/bls"
	"github.com/tendermint/tendermint/crypto/composite"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/crypto/merkle"
	"github.com/tendermint/tendermint/crypto/tmhash"
	"github.com/tendermint/tendermint/libs/bits"
	"github.com/tendermint/tendermint/libs/bytes"
	tmrand "github.com/tendermint/tendermint/libs/rand"
	tmtime "github.com/tendermint/tendermint/types/time"
	"github.com/tendermint/tendermint/version"
)

func TestMain(m *testing.M) {
	RegisterMockEvidences(cdc)

	code := m.Run()
	os.Exit(code)
}

func TestBlockAddEvidence(t *testing.T) {
	txs := []Tx{Tx("foo"), Tx("bar")}
	lastID := makeBlockIDRandom()
	h := int64(3)

	voteSet, _, valSet, vals := randVoteSet(h-1, 1, PrecommitType, 10, 1)
	commit, err := MakeCommit(lastID, h-1, 1, voteSet, vals, time.Now())
	require.NoError(t, err)

	ev := NewMockEvidence(h, time.Now(), 0, valSet.Voters[0].Address)
	evList := []Evidence{ev}

	block := MakeBlock(h, txs, commit, evList)
	require.NotNil(t, block)
	require.Equal(t, 1, len(block.Evidence.Evidence))
	require.NotNil(t, block.EvidenceHash)
}

func TestBlockValidateBasic(t *testing.T) {
	require.Error(t, (*Block)(nil).ValidateBasic())

	txs := []Tx{Tx("foo"), Tx("bar")}
	lastID := makeBlockIDRandom()
	h := int64(3)

	voteSet, valSet, voterSet, vals := randVoteSet(h-1, 1, PrecommitType, 10, 1)
	commit, err := MakeCommit(lastID, h-1, 1, voteSet, vals, time.Now())
	require.NoError(t, err)

	ev := NewMockEvidence(h, time.Now(), 0, voterSet.Voters[0].Address)
	evList := []Evidence{ev}

	testCases := []struct {
		testName      string
		malleateBlock func(*Block)
		expErr        bool
	}{
		{"Make Block", func(blk *Block) {}, false},
		{"Make Block w/ proposer Addr", func(blk *Block) {
			blk.ProposerAddress = valSet.SelectProposer([]byte{}, blk.Height, 0).Address
		}, false},
		{"Negative Height", func(blk *Block) { blk.Height = -1 }, true},
		{"Remove 1/2 the commits", func(blk *Block) {
			blk.LastCommit.Signatures = commit.Signatures[:commit.Size()/2]
			blk.LastCommit.hash = nil // clear hash or change wont be noticed
		}, true},
		{"Remove LastCommitHash", func(blk *Block) { blk.LastCommitHash = []byte("something else") }, true},
		{"Tampered Data", func(blk *Block) {
			blk.Data.Txs[0] = Tx("something else")
			blk.Data.hash = nil // clear hash or change wont be noticed
		}, true},
		{"Tampered DataHash", func(blk *Block) {
			blk.DataHash = tmrand.Bytes(len(blk.DataHash))
		}, true},
		{"Tampered EvidenceHash", func(blk *Block) {
			blk.EvidenceHash = []byte("something else")
		}, true},
	}
	for i, tc := range testCases {
		tc := tc
		i := i
		t.Run(tc.testName, func(t *testing.T) {
			block := MakeBlock(h, txs, commit, evList)
			block.ProposerAddress = valSet.SelectProposer([]byte{}, block.Height, 0).Address
			tc.malleateBlock(block)
			err = block.ValidateBasic()
			assert.Equal(t, tc.expErr, err != nil, "#%d: %v", i, err)
		})
	}
}

func TestBlockHash(t *testing.T) {
	assert.Nil(t, (*Block)(nil).Hash())
	assert.Nil(t, MakeBlock(int64(3), []Tx{Tx("Hello World")}, nil, nil).Hash())
}

func TestBlockMakePartSet(t *testing.T) {
	assert.Nil(t, (*Block)(nil).MakePartSet(2))

	partSet := MakeBlock(int64(3), []Tx{Tx("Hello World")}, nil, nil).MakePartSet(1024)
	assert.NotNil(t, partSet)
	assert.Equal(t, 1, partSet.Total())
}

func TestBlockMakePartSetWithEvidence(t *testing.T) {
	assert.Nil(t, (*Block)(nil).MakePartSet(2))

	lastID := makeBlockIDRandom()
	h := int64(3)

	voteSet, _, voterSet, vals := randVoteSet(h-1, 1, PrecommitType, 10, 1)
	commit, err := MakeCommit(lastID, h-1, 1, voteSet, vals, time.Now())
	require.NoError(t, err)

	ev := NewMockEvidence(h, time.Now(), 0, voterSet.Voters[0].Address)
	evList := []Evidence{ev}

	block := MakeBlock(h, []Tx{Tx("Hello World")}, commit, evList)
	bz, _ := cdc.MarshalBinaryLengthPrefixed(block)
	blockSize := len(bz)
	partSet := block.MakePartSet(512)
	assert.NotNil(t, partSet)
	assert.Equal(t, int(math.Ceil(float64(blockSize)/512.0)), partSet.Total())
}

func TestBlockHashesTo(t *testing.T) {
	assert.False(t, (*Block)(nil).HashesTo(nil))

	lastID := makeBlockIDRandom()
	h := int64(3)
	voteSet, _, voterSet, vals := randVoteSet(h-1, 1, PrecommitType, 10, 1)
	commit, err := MakeCommit(lastID, h-1, 1, voteSet, vals, time.Now())
	require.NoError(t, err)

	ev := NewMockEvidence(h, time.Now(), 0, voterSet.Voters[0].Address)
	evList := []Evidence{ev}

	block := MakeBlock(h, []Tx{Tx("Hello World")}, commit, evList)
	block.VotersHash = voterSet.Hash()
	assert.False(t, block.HashesTo([]byte{}))
	assert.False(t, block.HashesTo([]byte("something else")))
	assert.True(t, block.HashesTo(block.Hash()))
}

func TestBlockSize(t *testing.T) {
	size := MakeBlock(int64(3), []Tx{Tx("Hello World")}, nil, nil).Size()
	if size <= 0 {
		t.Fatal("Size of the block is zero or negative")
	}
}

func TestBlockString(t *testing.T) {
	assert.Equal(t, "nil-Block", (*Block)(nil).String())
	assert.Equal(t, "nil-Block", (*Block)(nil).StringIndented(""))
	assert.Equal(t, "nil-Block", (*Block)(nil).StringShort())

	block := MakeBlock(int64(3), []Tx{Tx("Hello World")}, nil, nil)
	assert.NotEqual(t, "nil-Block", block.String())
	assert.NotEqual(t, "nil-Block", block.StringIndented(""))
	assert.NotEqual(t, "nil-Block", block.StringShort())
}

func makeBlockIDRandom() BlockID {
	var (
		blockHash   = make([]byte, tmhash.Size)
		partSetHash = make([]byte, tmhash.Size)
	)
	rand.Read(blockHash)   //nolint: gosec
	rand.Read(partSetHash) //nolint: gosec
	return BlockID{blockHash, PartSetHeader{123, partSetHash}}
}

func makeBlockID(hash []byte, partSetSize int, partSetHash []byte) BlockID {
	var (
		h   = make([]byte, tmhash.Size)
		psH = make([]byte, tmhash.Size)
	)
	copy(h, hash)
	copy(psH, partSetHash)
	return BlockID{
		Hash: h,
		PartsHeader: PartSetHeader{
			Total: partSetSize,
			Hash:  psH,
		},
	}
}

var nilBytes []byte

func TestNilHeaderHashDoesntCrash(t *testing.T) {
	assert.Equal(t, []byte((*Header)(nil).Hash()), nilBytes)
	assert.Equal(t, []byte((new(Header)).Hash()), nilBytes)
}

func TestNilDataHashDoesntCrash(t *testing.T) {
	assert.Equal(t, []byte((*Data)(nil).Hash()), nilBytes)
	assert.Equal(t, []byte(new(Data).Hash()), nilBytes)
}

func TestNewCommit(t *testing.T) {
	blockID := BlockID{
		Hash: []byte{},
		PartsHeader: PartSetHeader{
			Total: 0,
			Hash:  []byte{},
		},
	}
	privKeys := [...]crypto.PrivKey{
		bls.GenPrivKey(),
		composite.GenPrivKey(),
		ed25519.GenPrivKey(),
		bls.GenPrivKey(),
	}
	msgs := make([][]byte, len(privKeys))
	signs := make([][]byte, len(privKeys))
	pubKeys := make([]crypto.PubKey, len(privKeys))
	commitSigs := make([]CommitSig, len(privKeys))
	for i := 0; i < len(privKeys); i++ {
		msgs[i] = []byte(fmt.Sprintf("hello, world %d", i))
		signs[i], _ = privKeys[i].Sign(msgs[i])
		pubKeys[i] = privKeys[i].PubKey()
		commitSigs[i] = NewCommitSigForBlock(signs[i], pubKeys[i].Address(), time.Now())
		assert.Equal(t, signs[i], commitSigs[i].Signature)
	}
	commit := NewCommit(0, 1, blockID, commitSigs)

	assert.Equal(t, int64(0), commit.Height)
	assert.Equal(t, 1, commit.Round)
	assert.Equal(t, blockID, commit.BlockID)
	assert.Equal(t, len(commitSigs), len(commit.Signatures))
	assert.Nil(t, commit.AggregatedSignature)
	assert.NotNil(t, commit.Signatures[0].Signature)
	assert.NotNil(t, commit.Signatures[1].Signature)
	assert.NotNil(t, commit.Signatures[2].Signature)
	assert.NotNil(t, commit.Signatures[3].Signature)
	assert.True(t, pubKeys[2].VerifyBytes(msgs[2], commit.Signatures[2].Signature))

	blsPubKeys := []bls.PubKeyBLS12{
		*GetSignatureKey(pubKeys[0]),
		*GetSignatureKey(pubKeys[1]),
		*GetSignatureKey(pubKeys[3]),
	}
	blsSigMsgs := [][]byte{msgs[0], msgs[1], msgs[3]}
	func() {
		aggrSig, err := bls.AddSignature(nil, signs[0])
		assert.Nil(t, err)
		aggrSig, err = bls.AddSignature(aggrSig, signs[1])
		assert.Nil(t, err)
		aggrSig, err = bls.AddSignature(aggrSig, signs[3])
		assert.Nil(t, err)
		err = bls.VerifyAggregatedSignature(aggrSig, blsPubKeys, blsSigMsgs)
		assert.Nil(t, err)
		assert.Nil(t, commit.AggregatedSignature)
	}()
}

func TestCommit(t *testing.T) {
	lastID := makeBlockIDRandom()
	h := int64(3)
	voteSet, _, _, vals := randVoteSet(h-1, 1, PrecommitType, 10, 1)
	commit, err := MakeCommit(lastID, h-1, 1, voteSet, vals, time.Now())
	require.NoError(t, err)

	assert.Equal(t, h-1, commit.Height)
	assert.Equal(t, 1, commit.Round)
	assert.Equal(t, PrecommitType, SignedMsgType(commit.Type()))
	if commit.Size() <= 0 {
		t.Fatalf("commit %v has a zero or negative size: %d", commit, commit.Size())
	}

	require.NotNil(t, commit.BitArray())
	assert.Equal(t, bits.NewBitArray(10).Size(), commit.BitArray().Size())

	if len(voteSet.GetByIndex(0).Signature) != bls.SignatureSize {
		assert.Equal(t, voteSet.GetByIndex(0), commit.GetByIndex(0))
	} else {
		assert.NotNil(t, commit.AggregatedSignature)
		isEqualVoteWithoutSignature(t, voteSet.GetByIndex(0), commit.GetByIndex(0))
	}
}

func TestCommitValidateBasic(t *testing.T) {
	testCases := []struct {
		testName       string
		malleateCommit func(*Commit)
		expectErr      bool
	}{
		{"Random Commit", func(com *Commit) {}, false},
		{"Incorrect signature", func(com *Commit) { com.Signatures[0].Signature = []byte{0} }, false},
		{"Incorrect height", func(com *Commit) { com.Height = int64(-100) }, true},
		{"Incorrect round", func(com *Commit) { com.Round = -100 }, true},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			com := randCommit(time.Now())
			tc.malleateCommit(com)
			assert.Equal(t, tc.expectErr, com.ValidateBasic() != nil, "Validate Basic had an unexpected result")
		})
	}
}

func TestCommitHash(t *testing.T) {
	t.Run("receiver is nil", func(t *testing.T) {
		var commit *Commit = nil
		assert.Nil(t, commit.Hash())
	})

	t.Run("without any signatures", func(t *testing.T) {
		commit := &Commit{
			hash:                nil,
			Signatures:          nil,
			AggregatedSignature: nil,
		}
		expected, _ := hex.DecodeString("6E340B9CFFB37A989CA544E6BB780A2C78901D3FB33738768511A30617AFA01D")
		assert.Equal(t, expected, commit.Hash().Bytes())
	})

	t.Run("with out without aggregated signature", func(t *testing.T) {
		signature := []byte{0, 0, 0, 0}
		address := []byte{0, 0, 0, 0}
		tm := time.Unix(0, 0)
		commit := &Commit{
			hash: nil,
			Signatures: []CommitSig{
				NewCommitSigAbsent(),
				NewCommitSigForBlock(signature, address, tm),
			},
			AggregatedSignature: nil,
		}
		expected, _ := hex.DecodeString("82ac742aeeb4266d4e1f659985987c181c494ac17494750cda4dce61e38b0514")
		assert.Equal(t, expected, commit.Hash().Bytes())

		commit.hash = nil
		commit.AggregatedSignature = []byte{0, 0, 0, 0}
		expected, _ = hex.DecodeString("0b3875dd994c60a8781851e5533886f3000203fa2f9587b5c256666dc5fa89ef")
		assert.Equal(t, expected, commit.Hash().Bytes())

		commit.hash = nil
		commit.AggregatedSignature = []byte{0, 1, 2, 3}
		expected, _ = hex.DecodeString("f7d7318af02be9015b6440496844d06ec68684251c2378c41a2c2b4e2f5d76cb")
		assert.Equal(t, expected, commit.Hash().Bytes())
	})
}

func TestHeaderHash(t *testing.T) {
	testCases := []struct {
		desc       string
		header     *Header
		expectHash bytes.HexBytes
	}{
		{"Generates expected hash", &Header{
			Version:            version.Consensus{Block: 1, App: 2},
			ChainID:            "chainId",
			Height:             3,
			Time:               time.Date(2019, 10, 13, 16, 14, 44, 0, time.UTC),
			LastBlockID:        makeBlockID(make([]byte, tmhash.Size), 6, make([]byte, tmhash.Size)),
			LastCommitHash:     tmhash.Sum([]byte("last_commit_hash")),
			DataHash:           tmhash.Sum([]byte("data_hash")),
			VotersHash:         tmhash.Sum([]byte("voters_hash")),
			ValidatorsHash:     tmhash.Sum([]byte("validators_hash")),
			NextValidatorsHash: tmhash.Sum([]byte("next_validators_hash")),
			ConsensusHash:      tmhash.Sum([]byte("consensus_hash")),
			AppHash:            tmhash.Sum([]byte("app_hash")),
			LastResultsHash:    tmhash.Sum([]byte("last_results_hash")),
			EvidenceHash:       tmhash.Sum([]byte("evidence_hash")),
			ProposerAddress:    crypto.AddressHash([]byte("proposer_address")),
			Round:              1,
			Proof:              tmhash.Sum([]byte("proof")),
		}, hexBytesFromString("7A0342C041357246CCE6E9AB81223C3013233A96E185173ED5C43F650A0A8A54")},
		{"nil header yields nil", nil, nil},
		{"nil VotersHash yields nil", &Header{
			Version:            version.Consensus{Block: 1, App: 2},
			ChainID:            "chainId",
			Height:             3,
			Time:               time.Date(2019, 10, 13, 16, 14, 44, 0, time.UTC),
			LastBlockID:        makeBlockID(make([]byte, tmhash.Size), 6, make([]byte, tmhash.Size)),
			LastCommitHash:     tmhash.Sum([]byte("last_commit_hash")),
			DataHash:           tmhash.Sum([]byte("data_hash")),
			VotersHash:         nil,
			ValidatorsHash:     tmhash.Sum([]byte("validators_hash")),
			NextValidatorsHash: tmhash.Sum([]byte("next_validators_hash")),
			ConsensusHash:      tmhash.Sum([]byte("consensus_hash")),
			AppHash:            tmhash.Sum([]byte("app_hash")),
			LastResultsHash:    tmhash.Sum([]byte("last_results_hash")),
			EvidenceHash:       tmhash.Sum([]byte("evidence_hash")),
			ProposerAddress:    crypto.AddressHash([]byte("proposer_address")),
		}, nil},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			assert.Equal(t, tc.expectHash, tc.header.Hash())

			// We also make sure that all fields are hashed in struct order, and that all
			// fields in the test struct are non-zero.
			if tc.header != nil && tc.expectHash != nil {
				byteSlices := [][]byte{}
				s := reflect.ValueOf(*tc.header)
				for i := 0; i < s.NumField(); i++ {
					f := s.Field(i)
					assert.False(t, f.IsZero(), "Found zero-valued field %v",
						s.Type().Field(i).Name)
					byteSlices = append(byteSlices, cdcEncode(f.Interface()))
				}
				assert.Equal(t,
					bytes.HexBytes(merkle.SimpleHashFromByteSlices(byteSlices)), tc.header.Hash())
			}
		})
	}
}

func TestMaxHeaderBytes(t *testing.T) {
	// Construct a UTF-8 string of MaxChainIDLen length using the supplementary
	// characters.
	// Each supplementary character takes 4 bytes.
	// http://www.i18nguy.com/unicode/supplementary-test.html
	maxChainID := ""
	for i := 0; i < MaxChainIDLen; i++ {
		maxChainID += "𠜎"
	}

	// time is varint encoded so need to pick the max.
	// year int, month Month, day, hour, min, sec, nsec int, loc *Location
	timestamp := time.Date(math.MaxInt64, 0, 0, 0, 0, 0, math.MaxInt64, time.UTC)

	h := Header{
		Version:            version.Consensus{Block: math.MaxInt64, App: math.MaxInt64},
		ChainID:            maxChainID,
		Height:             math.MaxInt64,
		Time:               timestamp,
		LastBlockID:        makeBlockID(make([]byte, tmhash.Size), math.MaxInt64, make([]byte, tmhash.Size)),
		LastCommitHash:     tmhash.Sum([]byte("last_commit_hash")),
		DataHash:           tmhash.Sum([]byte("data_hash")),
		VotersHash:         tmhash.Sum([]byte("voters_hash")),
		ValidatorsHash:     tmhash.Sum([]byte("validators_hash")),
		NextValidatorsHash: tmhash.Sum([]byte("next_validators_hash")),
		ConsensusHash:      tmhash.Sum([]byte("consensus_hash")),
		AppHash:            tmhash.Sum([]byte("app_hash")),
		LastResultsHash:    tmhash.Sum([]byte("last_results_hash")),
		EvidenceHash:       tmhash.Sum([]byte("evidence_hash")),
		ProposerAddress:    crypto.AddressHash([]byte("proposer_address")),
	}

	bz, err := cdc.MarshalBinaryLengthPrefixed(h)
	require.NoError(t, err)

	assert.EqualValues(t, MaxHeaderBytes, int64(len(bz)))
}

func randCommit(now time.Time) *Commit {
	lastID := makeBlockIDRandom()
	h := int64(3)
	voteSet, _, _, vals := randVoteSet(h-1, 1, PrecommitType, 10, 1)
	commit, err := MakeCommit(lastID, h-1, 1, voteSet, vals, now)
	if err != nil {
		panic(err)
	}
	return commit
}

func hexBytesFromString(s string) bytes.HexBytes {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return bytes.HexBytes(b)
}

func TestCommitSigNumOfBytes(t *testing.T) {
	pv1 := NewMockPV(PrivKeyEd25519)
	pv2 := NewMockPV(PrivKeyComposite)
	pv3 := NewMockPV(PrivKeyBLS)

	pub1, _ := pv1.GetPubKey()
	pub2, _ := pv2.GetPubKey()
	pub3, _ := pv3.GetPubKey()

	blockID := BlockID{tmrand.Bytes(tmhash.Size),
		PartSetHeader{math.MaxInt32, tmrand.Bytes(tmhash.Size)}}
	chainID := "mychain1"
	timestamp := time.Date(math.MaxInt64, 0, 0, 0, 0, 0, math.MaxInt64, time.UTC)

	vote1 := &Vote{
		ValidatorAddress: pub1.Address(),
		ValidatorIndex:   math.MaxInt64,
		Height:           math.MaxInt64,
		Round:            math.MaxInt64,
		Timestamp:        timestamp,
		Type:             PrecommitType,
		BlockID:          blockID,
	}
	assert.NoError(t, pv1.SignVote(chainID, vote1))

	vote2 := &Vote{
		ValidatorAddress: pub2.Address(),
		ValidatorIndex:   math.MaxInt64,
		Height:           math.MaxInt64,
		Round:            math.MaxInt64,
		Timestamp:        timestamp,
		Type:             PrecommitType,
		BlockID:          blockID,
	}
	assert.NoError(t, pv2.SignVote(chainID, vote2))

	vote3 := &Vote{
		ValidatorAddress: pub2.Address(),
		ValidatorIndex:   math.MaxInt64,
		Height:           math.MaxInt64,
		Round:            math.MaxInt64,
		Timestamp:        timestamp,
		Type:             PrecommitType,
		BlockID:          blockID,
	}
	assert.NoError(t, pv3.SignVote(chainID, vote3))

	commitSig1 := NewCommitSigForBlock(vote1.Signature, pub1.Address(), timestamp)
	commitSig2 := NewCommitSigForBlock(vote2.Signature, pub2.Address(), timestamp)
	commitSig3 := NewCommitSigForBlock(vote3.Signature, pub3.Address(), timestamp)
	aggregatedCommitSig := NewCommitSigForBlock(nil, pub2.Address(), timestamp)

	b1, err1 := cdc.MarshalBinaryLengthPrefixed(commitSig1)
	assert.NoError(t, err1)
	assert.True(t, int64(len(b1)) == commitSig1.MaxCommitSigBytes())

	b2, err2 := cdc.MarshalBinaryLengthPrefixed(commitSig2)
	assert.NoError(t, err2)
	assert.True(t, int64(len(b2)) == commitSig2.MaxCommitSigBytes())

	b3, err3 := cdc.MarshalBinaryLengthPrefixed(commitSig3)
	assert.NoError(t, err3)
	assert.True(t, int64(len(b3)) == commitSig3.MaxCommitSigBytes())

	b4, err4 := cdc.MarshalBinaryLengthPrefixed(aggregatedCommitSig)
	assert.NoError(t, err4)
	assert.True(t, int64(len(b4)) == aggregatedCommitSig.MaxCommitSigBytes())
}

func TestMaxCommitBytes(t *testing.T) {
	pv1 := NewMockPV(PrivKeyEd25519)
	pv2 := NewMockPV(PrivKeyComposite)
	pv3 := NewMockPV(PrivKeyComposite)

	pub1, _ := pv1.GetPubKey()
	pub2, _ := pv2.GetPubKey()
	pub3, _ := pv3.GetPubKey()

	blockID := BlockID{tmrand.Bytes(tmhash.Size),
		PartSetHeader{math.MaxInt32, tmrand.Bytes(tmhash.Size)}}

	chainID := "mychain2"
	timestamp := time.Date(math.MaxInt64, 0, 0, 0, 0, 0, math.MaxInt64, time.UTC)

	vote1 := &Vote{
		ValidatorAddress: pub1.Address(),
		ValidatorIndex:   math.MaxInt64,
		Height:           math.MaxInt64,
		Round:            math.MaxInt64,
		Timestamp:        timestamp,
		Type:             PrecommitType,
		BlockID:          blockID,
	}
	assert.NoError(t, pv1.SignVote(chainID, vote1))

	vote2 := &Vote{
		ValidatorAddress: pub2.Address(),
		ValidatorIndex:   math.MaxInt64,
		Height:           math.MaxInt64,
		Round:            math.MaxInt64,
		Timestamp:        timestamp,
		Type:             PrecommitType,
		BlockID:          blockID,
	}
	assert.NoError(t, pv2.SignVote(chainID, vote2))

	vote3 := &Vote{
		ValidatorAddress: pub2.Address(),
		ValidatorIndex:   math.MaxInt64,
		Height:           math.MaxInt64,
		Round:            math.MaxInt64,
		Timestamp:        timestamp,
		Type:             PrecommitType,
		BlockID:          blockID,
	}
	// does not sign vote3

	commitSig := make([]CommitSig, 3)
	commitSig[0] = NewCommitSigForBlock(vote1.Signature, pub1.Address(), timestamp)
	commitSig[1] = NewCommitSigForBlock(vote2.Signature, pub2.Address(), timestamp)
	commitSig[2] = NewCommitSigForBlock(vote3.Signature, pub3.Address(), timestamp)

	commit := NewCommit(math.MaxInt64, math.MaxInt32, blockID, commitSig)
	commit.AggregatedSignature = tmrand.Bytes(bls.SignatureSize)

	bz1, err1 := cdc.MarshalBinaryLengthPrefixed(blockID)
	assert.NoError(t, err1)
	bz2, err2 := cdc.MarshalBinaryLengthPrefixed(commit)
	assert.NoError(t, err2)
	assert.True(t, CommitBlockIDMaxLen == len(bz1))
	assert.True(t, commit.MaxCommitBytes() == int64(len(bz2)))
}

func TestMaxCommitBytesMany(t *testing.T) {
	commitCount := 100
	commitSig := make([]CommitSig, commitCount)
	blockID := BlockID{tmrand.Bytes(tmhash.Size),
		PartSetHeader{math.MaxInt32, tmrand.Bytes(tmhash.Size)}}

	chainID := "mychain3"
	timestamp := time.Date(math.MaxInt64, 0, 0, 0, 0, 0, math.MaxInt64, time.UTC)

	for i := 0; i < commitCount; i++ {
		pv := NewMockPV(PrivKeyEd25519)
		pub, _ := pv.GetPubKey()
		vote := &Vote{
			ValidatorAddress: pub.Address(),
			ValidatorIndex:   math.MaxInt64,
			Height:           math.MaxInt64,
			Round:            math.MaxInt64,
			Timestamp:        timestamp,
			Type:             PrecommitType,
			BlockID:          blockID,
		}
		assert.NoError(t, pv.SignVote(chainID, vote))
		commitSig[i] = NewCommitSigForBlock(vote.Signature, pub.Address(), timestamp)
	}
	commit := NewCommit(math.MaxInt64, math.MaxInt32, blockID, commitSig)
	bz, err := cdc.MarshalBinaryLengthPrefixed(commit)
	assert.NoError(t, err)
	assert.True(t, commit.MaxCommitBytes() == int64(len(bz)))
}

func TestMaxCommitBytesAggregated(t *testing.T) {
	commitCount := 100
	commitSig := make([]CommitSig, commitCount)
	blockID := BlockID{tmrand.Bytes(tmhash.Size),
		PartSetHeader{math.MaxInt32, tmrand.Bytes(tmhash.Size)}}

	timestamp := time.Date(math.MaxInt64, 0, 0, 0, 0, 0, math.MaxInt64, time.UTC)

	for i := 0; i < commitCount; i++ {
		pv := NewMockPV(PrivKeyComposite)
		pub, _ := pv.GetPubKey()
		vote := &Vote{
			ValidatorAddress: pub.Address(),
			ValidatorIndex:   math.MaxInt64,
			Height:           math.MaxInt64,
			Round:            math.MaxInt64,
			Timestamp:        timestamp,
			Type:             PrecommitType,
			BlockID:          blockID,
		}
		// do not sign
		commitSig[i] = NewCommitSigForBlock(vote.Signature, pub.Address(), timestamp)
	}
	commit := NewCommit(math.MaxInt64, math.MaxInt32, blockID, commitSig)
	commit.AggregatedSignature = tmrand.Bytes(bls.SignatureSize)

	bz, err := cdc.MarshalBinaryLengthPrefixed(commit)
	assert.NoError(t, err)
	assert.True(t, commit.MaxCommitBytes() == int64(len(bz)))
}

func TestMaxCommitBytesMixed(t *testing.T) {
	commitCount := 100
	commitSig := make([]CommitSig, commitCount)
	blockID := BlockID{tmrand.Bytes(tmhash.Size),
		PartSetHeader{math.MaxInt32, tmrand.Bytes(tmhash.Size)}}

	chainID := "mychain4"
	timestamp := time.Date(math.MaxInt64, 0, 0, 0, 0, 0, math.MaxInt64, time.UTC)

	for i := 0; i < commitCount; i++ {
		keyType := RandomKeyType()
		pv := NewMockPV(keyType)
		pub, _ := pv.GetPubKey()
		vote := &Vote{
			ValidatorAddress: pub.Address(),
			ValidatorIndex:   math.MaxInt64,
			Height:           math.MaxInt64,
			Round:            math.MaxInt64,
			Timestamp:        timestamp,
			Type:             PrecommitType,
			BlockID:          blockID,
		}
		// sign only if key type is ed25519
		if keyType == PrivKeyEd25519 {
			assert.NoError(t, pv.SignVote(chainID, vote))
		}
		commitSig[i] = NewCommitSigForBlock(vote.Signature, pub.Address(), timestamp)
	}
	commit := NewCommit(math.MaxInt64, math.MaxInt32, blockID, commitSig)
	commit.AggregatedSignature = tmrand.Bytes(bls.SignatureSize)

	bz, err := cdc.MarshalBinaryLengthPrefixed(commit)
	assert.NoError(t, err)
	assert.True(t, commit.MaxCommitBytes() == int64(len(bz)))
}

func TestBlockMaxDataBytes(t *testing.T) {
	pv1 := NewMockPV(PrivKeyEd25519)
	pv2 := NewMockPV(PrivKeyComposite)
	pv3 := NewMockPV(PrivKeyBLS)

	pub1, _ := pv1.GetPubKey()
	pub2, _ := pv2.GetPubKey()
	pub3, _ := pv3.GetPubKey()

	val := make([]*Validator, 3)
	val[0] = newValidator(pub1.Address(), 100)
	val[1] = newValidator(pub2.Address(), 200)
	val[2] = newValidator(pub3.Address(), 300)
	valSet := NewValidatorSet(val)
	blockID := makeBlockIDRandom()
	chainID := "mychain5"
	vote1, _ := MakeVote(1, blockID, valSet, pv1, chainID, tmtime.Now())
	vote2, _ := MakeVote(1, blockID, valSet, pv2, chainID, tmtime.Now())
	vote3, _ := MakeVote(1, blockID, valSet, pv3, chainID, tmtime.Now())
	vote4, _ := MakeVote(1, makeBlockIDRandom(), valSet, pv3, chainID, tmtime.Now())

	commitSig := make([]CommitSig, 3)
	commitSig[0] = NewCommitSigForBlock(vote1.Signature, pub1.Address(), tmtime.Now())
	commitSig[1] = NewCommitSigForBlock(vote2.Signature, pub2.Address(), tmtime.Now())
	commitSig[2] = NewCommitSigForBlock(vote3.Signature, pub3.Address(), tmtime.Now())

	commit := NewCommit(1, 0, blockID, commitSig)
	dupEv := NewDuplicateVoteEvidence(pub3, vote3, vote4)

	testCases := []struct {
		maxBytes int64
		commit   *Commit
		evidence []Evidence
		panics   bool
		result   int64
	}{
		0: {-10, commit, []Evidence{dupEv}, true, 0},
		1: {10, commit, []Evidence{dupEv}, true, 0},
		2: {1700, commit, []Evidence{dupEv}, true, 0},
		3: {1735, commit, []Evidence{dupEv}, false, 0},
		4: {1736, commit, []Evidence{dupEv}, false, 1},
	}

	for i, tc := range testCases {
		tc := tc
		if tc.panics {
			assert.Panics(t, func() {
				MaxDataBytes(tc.maxBytes, tc.commit, tc.evidence)
			}, "#%v", i)
		} else {
			assert.Equal(t,
				tc.result,
				MaxDataBytes(tc.maxBytes, tc.commit, tc.evidence),
				"#%v", i)
		}
	}
}

func TestBlockMaxDataBytesUnknownEvidence(t *testing.T) {
	testCases := []struct {
		maxBytes  int64
		valsCount int
		panics    bool
		result    int64
	}{
		0: {-10, 1, true, 0},
		1: {10, 1, true, 0},
		2: {961, 1, true, 0},
		3: {1035, 1, false, 0},
		4: {1036, 1, false, 1},
	}

	for i, tc := range testCases {
		tc := tc
		if tc.panics {
			assert.Panics(t, func() {
				MaxDataBytesUnknownEvidence(tc.maxBytes, tc.valsCount)
			}, "#%v", i)
		} else {
			assert.Equal(t,
				tc.result,
				MaxDataBytesUnknownEvidence(tc.maxBytes, tc.valsCount),
				"#%v", i)
		}
	}
}

func isEqualVoteWithoutSignature(t *testing.T, vote1, vote2 *Vote) {
	assert.Equal(t, vote1.Type, vote2.Type)
	assert.Equal(t, vote1.Height, vote2.Height)
	assert.Equal(t, vote1.Round, vote2.Round)
	assert.Equal(t, vote1.BlockID, vote2.BlockID)
	assert.Equal(t, vote1.Timestamp, vote2.Timestamp)
	assert.Equal(t, vote1.ValidatorAddress, vote2.ValidatorAddress)
	assert.Equal(t, vote1.ValidatorIndex, vote2.ValidatorIndex)
}

func TestCommitToVoteSet(t *testing.T) {
	lastID := makeBlockIDRandom()
	h := int64(3)

	voteSet, _, voterSet, vals := randVoteSet(h-1, 1, PrecommitType, 10, 1)
	commit, err := MakeCommit(lastID, h-1, 1, voteSet, vals, time.Now())
	assert.NoError(t, err)

	chainID := voteSet.ChainID()
	voteSet2 := CommitToVoteSet(chainID, commit, voterSet)
	var isAggregate bool
	isAggregate = false
	for i := 0; i < len(vals); i++ {
		// This is the vote before `MakeCommit`.
		vote1 := voteSet.GetByIndex(i)
		// This is the vote created from `CommitToVoteSet`
		vote2 := voteSet2.GetByIndex(i)
		// This is the vote created from `MakeCommit`
		vote3 := commit.GetVote(i)

		if len(vote1.Signature) != bls.SignatureSize {
			vote1bz := cdc.MustMarshalBinaryBare(vote1)
			vote2bz := cdc.MustMarshalBinaryBare(vote2)
			vote3bz := cdc.MustMarshalBinaryBare(vote3)
			assert.Equal(t, vote1bz, vote2bz)
			assert.Equal(t, vote1bz, vote3bz)
		} else {
			isAggregate = true
			vote2bz := cdc.MustMarshalBinaryBare(vote2)
			vote3bz := cdc.MustMarshalBinaryBare(vote3)
			assert.Equal(t, vote2bz, vote3bz)
			assert.NotNil(t, commit.AggregatedSignature)
			assert.Nil(t, vote2.Signature)
			assert.Nil(t, vote3.Signature)
			isEqualVoteWithoutSignature(t, vote1, vote2)
			isEqualVoteWithoutSignature(t, vote1, vote3)
		}
	}
	// panic test
	defer func() {
		err := recover()
		if err != nil {
			wantStr := "This signature of commitSig is already aggregated: commitSig"
			gotStr := fmt.Sprintf("%v", err)
			isPanic := strings.Contains(gotStr, wantStr)
			assert.True(t, isPanic)
		}
	}()
	if isAggregate {
		voteSet2.MakeCommit()
	}
}

func TestCommitToVoteSetWithVotesForNilBlock(t *testing.T) {
	blockID := makeBlockID([]byte("blockhash"), 1000, []byte("partshash"))

	const (
		height = int64(3)
		round  = 0
	)

	type commitVoteTest struct {
		blockIDs      []BlockID
		numVotes      []int // must sum to numValidators
		numValidators int
		valid         bool
	}

	testCases := []commitVoteTest{
		{[]BlockID{blockID, {}}, []int{67, 33}, 100, true},
	}

	for _, tc := range testCases {
		voteSet, _, valSet, vals := randVoteSet(height-1, round, PrecommitType, tc.numValidators, 1)

		vi := 0
		for n := range tc.blockIDs {
			for i := 0; i < tc.numVotes[n]; i++ {
				pubKey, err := vals[vi].GetPubKey()
				require.NoError(t, err)
				vote := &Vote{
					ValidatorAddress: pubKey.Address(),
					ValidatorIndex:   vi,
					Height:           height - 1,
					Round:            round,
					Type:             PrecommitType,
					BlockID:          tc.blockIDs[n],
					Timestamp:        tmtime.Now(),
				}

				added, err := signAddVote(vals[vi], vote, voteSet)
				assert.NoError(t, err)
				assert.True(t, added)

				vi++
			}
		}

		if tc.valid {
			commit := voteSet.MakeCommit() // panics without > 2/3 valid votes
			assert.NotNil(t, commit)
			err := valSet.VerifyCommit(voteSet.ChainID(), blockID, height-1, commit)
			assert.Nil(t, err)
		} else {
			assert.Panics(t, func() { voteSet.MakeCommit() })
		}
	}
}

func TestSignedHeaderValidateBasic(t *testing.T) {
	commit := randCommit(time.Now())
	chainID := "𠜎"
	timestamp := time.Date(math.MaxInt64, 0, 0, 0, 0, 0, math.MaxInt64, time.UTC)
	h := Header{
		Version:            version.Consensus{Block: math.MaxInt64, App: math.MaxInt64},
		ChainID:            chainID,
		Height:             commit.Height,
		Time:               timestamp,
		LastBlockID:        commit.BlockID,
		LastCommitHash:     commit.Hash(),
		DataHash:           commit.Hash(),
		VotersHash:         commit.Hash(),
		ValidatorsHash:     commit.Hash(),
		NextValidatorsHash: commit.Hash(),
		ConsensusHash:      commit.Hash(),
		AppHash:            commit.Hash(),
		LastResultsHash:    commit.Hash(),
		EvidenceHash:       commit.Hash(),
		ProposerAddress:    crypto.AddressHash([]byte("proposer_address")),
	}

	validSignedHeader := SignedHeader{Header: &h, Commit: commit}
	validSignedHeader.Commit.BlockID.Hash = validSignedHeader.Hash()
	invalidSignedHeader := SignedHeader{}

	testCases := []struct {
		testName  string
		shHeader  *Header
		shCommit  *Commit
		expectErr bool
	}{
		{"Valid Signed Header", validSignedHeader.Header, validSignedHeader.Commit, false},
		{"Invalid Signed Header", invalidSignedHeader.Header, validSignedHeader.Commit, true},
		{"Invalid Signed Header", validSignedHeader.Header, invalidSignedHeader.Commit, true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			sh := SignedHeader{
				Header: tc.shHeader,
				Commit: tc.shCommit,
			}
			assert.Equal(
				t,
				tc.expectErr,
				sh.ValidateBasic(validSignedHeader.Header.ChainID) != nil,
				"Validate Basic had an unexpected result",
			)
		})
	}
}

func TestBlockIDValidateBasic(t *testing.T) {
	validBlockID := BlockID{
		Hash: bytes.HexBytes{},
		PartsHeader: PartSetHeader{
			Total: 1,
			Hash:  bytes.HexBytes{},
		},
	}

	invalidBlockID := BlockID{
		Hash: []byte{0},
		PartsHeader: PartSetHeader{
			Total: -1,
			Hash:  bytes.HexBytes{},
		},
	}

	testCases := []struct {
		testName           string
		blockIDHash        bytes.HexBytes
		blockIDPartsHeader PartSetHeader
		expectErr          bool
	}{
		{"Valid BlockID", validBlockID.Hash, validBlockID.PartsHeader, false},
		{"Invalid BlockID", invalidBlockID.Hash, validBlockID.PartsHeader, true},
		{"Invalid BlockID", validBlockID.Hash, invalidBlockID.PartsHeader, true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			blockID := BlockID{
				Hash:        tc.blockIDHash,
				PartsHeader: tc.blockIDPartsHeader,
			}
			assert.Equal(t, tc.expectErr, blockID.ValidateBasic() != nil, "Validate Basic had an unexpected result")
		})
	}
}

func makeRandHeader() Header {
	chainID := "test"
	t := time.Now()
	height := tmrand.Int63()
	randBytes := tmrand.Bytes(tmhash.Size)
	randAddress := tmrand.Bytes(crypto.AddressSize)
	h := Header{
		Version:            version.Consensus{Block: 1, App: 1},
		ChainID:            chainID,
		Height:             height,
		Time:               t,
		LastBlockID:        BlockID{},
		LastCommitHash:     randBytes,
		DataHash:           randBytes,
		VotersHash:         randBytes,
		ValidatorsHash:     randBytes,
		NextValidatorsHash: randBytes,
		ConsensusHash:      randBytes,
		AppHash:            randBytes,

		LastResultsHash: randBytes,

		EvidenceHash:    randBytes,
		ProposerAddress: randAddress,
	}

	return h
}

func TestHeaderProto(t *testing.T) {
	h1 := makeRandHeader()
	tc := []struct {
		msg     string
		h1      *Header
		expPass bool
	}{
		{"success", &h1, true},
		{"failure empty Header", &Header{}, false},
	}

	for _, tt := range tc {
		tt := tt
		t.Run(tt.msg, func(t *testing.T) {
			pb := tt.h1.ToProto()
			h, err := HeaderFromProto(pb)
			if tt.expPass {
				require.NoError(t, err, tt.msg)
				require.Equal(t, tt.h1, &h, tt.msg)
			} else {
				require.Error(t, err, tt.msg)
			}

		})
	}
}

func TestBlockIDProtoBuf(t *testing.T) {
	blockID := makeBlockID([]byte("hash"), 2, []byte("part_set_hash"))
	testCases := []struct {
		msg     string
		bid1    *BlockID
		expPass bool
	}{
		{"success", &blockID, true},
		{"success empty", &BlockID{}, true},
		{"failure BlockID nil", nil, false},
	}
	for _, tc := range testCases {
		protoBlockID := tc.bid1.ToProto()

		bi, err := BlockIDFromProto(&protoBlockID)
		if tc.expPass {
			require.NoError(t, err)
			require.Equal(t, tc.bid1, bi, tc.msg)
		} else {
			require.NotEqual(t, tc.bid1, bi, tc.msg)
		}
	}
}

func TestSignedHeaderProtoBuf(t *testing.T) {
	commit := randCommit(time.Now())
	h := makeRandHeader()

	sh := SignedHeader{Header: &h, Commit: commit}

	testCases := []struct {
		msg     string
		sh1     *SignedHeader
		expPass bool
	}{
		{"empty SignedHeader 2", &SignedHeader{}, true},
		{"success", &sh, true},
		{"failure nil", nil, false},
	}
	for _, tc := range testCases {
		protoSignedHeader := tc.sh1.ToProto()

		sh, err := SignedHeaderFromProto(protoSignedHeader)

		if tc.expPass {
			require.NoError(t, err, tc.msg)
			require.Equal(t, tc.sh1, sh, tc.msg)
		} else {
			require.Error(t, err, tc.msg)
		}
	}
}

func TestCommitProtoBuf(t *testing.T) {
	commit := randCommit(time.Now())

	testCases := []struct {
		msg     string
		c1      *Commit
		expPass bool
	}{
		{"success", commit, true},
		// Empty value sets signatures to nil, signatures should not be nillable
		{"empty commit", &Commit{Signatures: []CommitSig{}}, true},
		{"fail Commit nil", nil, false},
	}
	for _, tc := range testCases {
		tc := tc
		protoCommit := tc.c1.ToProto()

		c, err := CommitFromProto(protoCommit)

		if tc.expPass {
			require.NoError(t, err, tc.msg)
			require.Equal(t, tc.c1, c, tc.msg)
		} else {
			require.Error(t, err, tc.msg)
		}
	}
}
