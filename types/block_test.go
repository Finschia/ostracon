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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gogotypes "github.com/gogo/protobuf/types"

	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/bls"
	"github.com/tendermint/tendermint/crypto/composite"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/crypto/merkle"
	"github.com/tendermint/tendermint/crypto/tmhash"
	"github.com/tendermint/tendermint/crypto/vrf"
	"github.com/tendermint/tendermint/libs/bits"
	"github.com/tendermint/tendermint/libs/bytes"
	tmrand "github.com/tendermint/tendermint/libs/rand"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmversion "github.com/tendermint/tendermint/proto/tendermint/version"
	tmtime "github.com/tendermint/tendermint/types/time"
	"github.com/tendermint/tendermint/version"
)

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

func TestBlockAddEvidence(t *testing.T) {
	txs := []Tx{Tx("foo"), Tx("bar")}
	lastID := makeBlockIDRandom()
	h := int64(3)

	voteSet, _, _, vals := randVoteSet(h-1, 1, tmproto.PrecommitType, 10, 1)
	commit, err := MakeCommit(lastID, h-1, 1, voteSet, vals, time.Now())
	require.NoError(t, err)

	ev := NewMockDuplicateVoteEvidenceWithValidator(h, time.Now(), vals[0], "block-test-chain")
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

	voteSet, valSet, _, vals := randVoteSet(h-1, 1, tmproto.PrecommitType, 10, 1)
	commit, err := MakeCommit(lastID, h-1, 1, voteSet, vals, time.Now())
	require.NoError(t, err)

	ev := NewMockDuplicateVoteEvidenceWithValidator(h, time.Now(), vals[0], "block-test-chain")
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
		{"Incorrect block protocol version", func(blk *Block) {
			blk.Version.Block = 1
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
	assert.EqualValues(t, 1, partSet.Total())
}

func TestBlockMakePartSetWithEvidence(t *testing.T) {
	assert.Nil(t, (*Block)(nil).MakePartSet(2))

	lastID := makeBlockIDRandom()
	h := int64(3)

	voteSet, _, _, vals := randVoteSet(h-1, 1, tmproto.PrecommitType, 10, 1)
	commit, err := MakeCommit(lastID, h-1, 1, voteSet, vals, time.Now())
	require.NoError(t, err)

	ev := NewMockDuplicateVoteEvidenceWithValidator(h, time.Now(), vals[0], "block-test-chain")
	evList := []Evidence{ev}

	block := MakeBlock(h, []Tx{Tx("Hello World")}, commit, evList)
	blockProto, err := block.ToProto()
	assert.NoError(t, err)
	bz, err := blockProto.Marshal()
	assert.NoError(t, err)
	blockSize := len(bz)
	partSet := block.MakePartSet(512)
	assert.NotNil(t, partSet)
	assert.Equal(t, uint32(math.Ceil(float64(blockSize)/512.0)), partSet.Total())
}

func TestBlockHashesTo(t *testing.T) {
	assert.False(t, (*Block)(nil).HashesTo(nil))

	lastID := makeBlockIDRandom()
	h := int64(3)
	voteSet, _, voterSet, vals := randVoteSet(h-1, 1, tmproto.PrecommitType, 10, 1)
	commit, err := MakeCommit(lastID, h-1, 1, voteSet, vals, time.Now())
	require.NoError(t, err)

	ev := NewMockDuplicateVoteEvidenceWithValidator(h, time.Now(), vals[0], "block-test-chain")
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
	rand.Read(blockHash)   //nolint: errcheck // ignore errcheck for read
	rand.Read(partSetHash) //nolint: errcheck // ignore errcheck for read
	return BlockID{blockHash, PartSetHeader{123, partSetHash}}
}

func makeBlockID(hash []byte, partSetSize uint32, partSetHash []byte) BlockID {
	var (
		h   = make([]byte, tmhash.Size)
		psH = make([]byte, tmhash.Size)
	)
	copy(h, hash)
	copy(psH, partSetHash)
	return BlockID{
		Hash: h,
		PartSetHeader: PartSetHeader{
			Total: partSetSize,
			Hash:  psH,
		},
	}
}

var nilBytes []byte

// This follows RFC-6962, i.e. `echo -n '' | sha256sum`
var emptyBytes = []byte{0xe3, 0xb0, 0xc4, 0x42, 0x98, 0xfc, 0x1c, 0x14, 0x9a, 0xfb, 0xf4, 0xc8,
	0x99, 0x6f, 0xb9, 0x24, 0x27, 0xae, 0x41, 0xe4, 0x64, 0x9b, 0x93, 0x4c, 0xa4, 0x95, 0x99, 0x1b,
	0x78, 0x52, 0xb8, 0x55}

func TestNilHeaderHashDoesntCrash(t *testing.T) {
	assert.Equal(t, nilBytes, []byte((*Header)(nil).Hash()))
	assert.Equal(t, nilBytes, []byte((new(Header)).Hash()))
}

func TestNilDataHashDoesntCrash(t *testing.T) {
	assert.Equal(t, emptyBytes, []byte((*Data)(nil).Hash()))
	assert.Equal(t, emptyBytes, []byte(new(Data).Hash()))
}

func TestNewCommit(t *testing.T) {
	blockID := BlockID{
		Hash: []byte{},
		PartSetHeader: PartSetHeader{
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
	assert.Equal(t, int32(1), commit.Round)
	assert.Equal(t, blockID, commit.BlockID)
	assert.Equal(t, len(commitSigs), len(commit.Signatures))
	assert.Nil(t, commit.AggregatedSignature)
	assert.NotNil(t, commit.Signatures[0].Signature)
	assert.NotNil(t, commit.Signatures[1].Signature)
	assert.NotNil(t, commit.Signatures[2].Signature)
	assert.NotNil(t, commit.Signatures[3].Signature)
	assert.True(t, pubKeys[2].VerifySignature(msgs[2], commit.Signatures[2].Signature))

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
	voteSet, _, _, vals := randVoteSet(h-1, 1, tmproto.PrecommitType, 10, 1)
	commit, err := MakeCommit(lastID, h-1, 1, voteSet, vals, time.Now())
	require.NoError(t, err)

	assert.Equal(t, h-1, commit.Height)
	assert.EqualValues(t, 1, commit.Round)
	assert.Equal(t, tmproto.PrecommitType, tmproto.SignedMsgType(commit.Type()))
	if commit.Size() <= 0 {
		t.Fatalf("commit %v has a zero or negative size: %d", commit, commit.Size())
	}

	require.NotNil(t, commit.BitArray())
	assert.Equal(t, bits.NewBitArray(10).Size(), commit.BitArray().Size())

	vote1, vote2 := voteSet.GetByIndex(0), commit.GetByIndex(0)
	assert.Equal(t, vote1.BlockID, vote2.BlockID)
	assert.Equal(t, vote1.Height, vote2.Height)
	assert.Equal(t, vote1.Round, vote2.Round)
	assert.Equal(t, vote1.Timestamp, vote2.Timestamp)
	assert.Equal(t, vote1.Type, vote2.Type)
	assert.Equal(t, vote1.ValidatorAddress, vote2.ValidatorAddress)
	assert.Equal(t, vote1.ValidatorIndex, vote2.ValidatorIndex)
	assert.NotNil(t, vote1.Signature)
	if len(vote1.Signature) == bls.SignatureSize { // RandVoterSet() generates private key type randomly ;(
		assert.Nil(t, vote2.Signature)
	}
	assert.True(t, commit.IsCommit())
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

func TestMaxCommitBytes(t *testing.T) {
	// time is varint encoded so need to pick the max.
	// year int, month Month, day, hour, min, sec, nsec int, loc *Location
	timestamp := time.Date(math.MaxInt64, 0, 0, 0, 0, 0, math.MaxInt64, time.UTC)

	cs := CommitSig{
		BlockIDFlag:      BlockIDFlagNil,
		ValidatorAddress: crypto.AddressHash([]byte("validator_address")),
		Timestamp:        timestamp,
		Signature:        crypto.CRandBytes(MaxSignatureSize),
	}

	pbSig := cs.ToProto()
	// test that a single commit sig doesn't exceed max commit sig bytes
	assert.EqualValues(t, MaxCommitSigBytes(len(cs.Signature)), int64(pbSig.Size()))

	// check size with a single commit
	commit := &Commit{
		Height: math.MaxInt64,
		Round:  math.MaxInt32,
		BlockID: BlockID{
			Hash: tmhash.Sum([]byte("blockID_hash")),
			PartSetHeader: PartSetHeader{
				Total: math.MaxUint32,
				Hash:  tmhash.Sum([]byte("blockID_part_set_header_hash")),
			},
		},
		Signatures: []CommitSig{cs},
	}

	pb := commit.ToProto()

	assert.EqualValues(t, MaxCommitBytes([]int{len(commit.Signatures[0].Signature)}, 0), int64(pb.Size()))
	assert.EqualValues(t, commit.MaxCommitBytes(), int64(pb.Size()))

	// check the upper bound of the commit size
	sigsBytes := make([]int, MaxVotesCount)
	sigsBytes[0] = len(commit.Signatures[0].Signature)
	for i := 1; i < MaxVotesCount; i++ {
		commit.Signatures = append(commit.Signatures, cs)
		sigsBytes[i] = len(commit.Signatures[i].Signature)
	}

	pb = commit.ToProto()

	assert.EqualValues(t, MaxCommitBytes(sigsBytes, 0), int64(pb.Size()))
	assert.EqualValues(t, commit.MaxCommitBytes(), int64(pb.Size()))

	pv1 := NewMockPV(PrivKeyEd25519)
	pv2 := NewMockPV(PrivKeyComposite)
	pv3 := NewMockPV(PrivKeyComposite)

	pub1, _ := pv1.GetPubKey()
	pub2, _ := pv2.GetPubKey()
	pub3, _ := pv3.GetPubKey()

	blockID := BlockID{tmrand.Bytes(tmhash.Size),
		PartSetHeader{math.MaxUint32, tmrand.Bytes(tmhash.Size)}}

	chainID := "mychain2"

	vote1 := &Vote{
		ValidatorAddress: pub1.Address(),
		ValidatorIndex:   math.MaxInt32,
		Height:           math.MaxInt64,
		Round:            math.MaxInt32,
		Timestamp:        timestamp,
		Type:             tmproto.PrecommitType,
		BlockID:          blockID,
		Signature:        tmrand.Bytes(MaxSignatureSize),
	}
	pbVote1 := vote1.ToProto()
	assert.NoError(t, pv1.SignVote(chainID, pbVote1))

	vote2 := &Vote{
		ValidatorAddress: pub2.Address(),
		ValidatorIndex:   math.MaxInt32,
		Height:           math.MaxInt64,
		Round:            math.MaxInt32,
		Timestamp:        timestamp,
		Type:             tmproto.PrecommitType,
		BlockID:          blockID,
		Signature:        tmrand.Bytes(MaxSignatureSize),
	}
	pbVote2 := vote2.ToProto()
	assert.NoError(t, pv2.SignVote(chainID, pbVote2))

	vote3 := &Vote{
		ValidatorAddress: pub2.Address(),
		ValidatorIndex:   math.MaxInt32,
		Height:           math.MaxInt64,
		Round:            math.MaxInt32,
		Timestamp:        timestamp,
		Type:             tmproto.PrecommitType,
		BlockID:          blockID,
		Signature:        tmrand.Bytes(MaxSignatureSize),
	}
	pbVote3 := vote3.ToProto()
	// does not sign vote3

	commitSig := make([]CommitSig, 3)
	commitSig[0] = NewCommitSigForBlock(pbVote1.Signature, pub1.Address(), timestamp)
	commitSig[1] = NewCommitSigForBlock(pbVote2.Signature, pub2.Address(), timestamp)
	commitSig[2] = NewCommitSigForBlock(pbVote3.Signature, pub3.Address(), timestamp)

	commit = NewCommit(math.MaxInt64, math.MaxInt32, blockID, commitSig)
	commit.AggregatedSignature = tmrand.Bytes(bls.SignatureSize)

	protoBlockID := blockID.ToProto()
	bz1, err1 := protoBlockID.Marshal()
	assert.NoError(t, err1)
	bz2, err2 := commit.ToProto().Marshal()
	assert.NoError(t, err2)
	assert.Equal(t, len(bz1), protoBlockID.Size())
	assert.Equal(t, len(bz2), commit.ToProto().Size())
	assert.Equal(t, CommitBlockIDMaxLen, protoBlockID.Size())
	assert.Equal(t, commit.MaxCommitBytes(), int64(commit.ToProto().Size()))
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
		expected, _ := hex.DecodeString("5EF6A2B65D0585177BD139364D23D9680A7387C2BFEE845F3F4AAF043FEBC555")
		assert.Equal(t, expected, commit.Hash().Bytes())

		commit.hash = nil
		commit.AggregatedSignature = []byte{0, 0, 0, 0}
		expected, _ = hex.DecodeString("3D9CD0C08000318B48DB77B1E2F974AD8F6AF32B922F571BB5BAB922D4443B65")
		assert.Equal(t, expected, commit.Hash().Bytes())

		commit.hash = nil
		commit.AggregatedSignature = []byte{0, 1, 2, 3}
		expected, _ = hex.DecodeString("F8960CFC0FFE82E323FD0484BA715589618AFCCF852731BDE6C9964F95DFC800")
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
			Version:            tmversion.Consensus{Block: 1, App: 2},
			ChainID:            "chainId",
			Height:             3,
			Time:               time.Date(2019, 10, 13, 16, 14, 44, 0, time.UTC),
			LastBlockID:        makeBlockID(make([]byte, tmhash.Size), 6, make([]byte, tmhash.Size)),
			LastCommitHash:     tmhash.Sum([]byte("last_commit_hash")),
			DataHash:           tmhash.Sum([]byte("data_hash")),
			ValidatorsHash:     tmhash.Sum([]byte("validators_hash")),
			VotersHash:         tmhash.Sum([]byte("voters_hash")),
			NextValidatorsHash: tmhash.Sum([]byte("next_validators_hash")),
			ConsensusHash:      tmhash.Sum([]byte("consensus_hash")),
			AppHash:            tmhash.Sum([]byte("app_hash")),
			LastResultsHash:    tmhash.Sum([]byte("last_results_hash")),
			EvidenceHash:       tmhash.Sum([]byte("evidence_hash")),
			ProposerAddress:    crypto.AddressHash([]byte("proposer_address")),
			Round:              1,
			Proof:              tmhash.Sum([]byte("proof")),
		}, hexBytesFromString("0368E6F15B6B7BC9DC5B10F36F37D6F867E132A22333F083A11290324274E183")},
		{"nil header yields nil", nil, nil},
		{"nil VotersHash yields nil", &Header{
			Version:            tmversion.Consensus{Block: 1, App: 2},
			ChainID:            "chainId",
			Height:             3,
			Time:               time.Date(2019, 10, 13, 16, 14, 44, 0, time.UTC),
			LastBlockID:        makeBlockID(make([]byte, tmhash.Size), 6, make([]byte, tmhash.Size)),
			LastCommitHash:     tmhash.Sum([]byte("last_commit_hash")),
			DataHash:           tmhash.Sum([]byte("data_hash")),
			VotersHash:         nil,
			NextValidatorsHash: tmhash.Sum([]byte("next_validators_hash")),
			ConsensusHash:      tmhash.Sum([]byte("consensus_hash")),
			AppHash:            tmhash.Sum([]byte("app_hash")),
			LastResultsHash:    tmhash.Sum([]byte("last_results_hash")),
			EvidenceHash:       tmhash.Sum([]byte("evidence_hash")),
			ProposerAddress:    crypto.AddressHash([]byte("proposer_address")),
			Round:              1,
			Proof:              tmhash.Sum([]byte("proof")),
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

					switch f := f.Interface().(type) {
					case int32, int64, bytes.HexBytes, vrf.Proof, string:
						byteSlices = append(byteSlices, cdcEncode(f))
					case time.Time:
						bz, err := gogotypes.StdTimeMarshal(f)
						require.NoError(t, err)
						byteSlices = append(byteSlices, bz)
					case tmversion.Consensus:
						bz, err := f.Marshal()
						require.NoError(t, err)
						byteSlices = append(byteSlices, bz)
					case BlockID:
						pbbi := f.ToProto()
						bz, err := pbbi.Marshal()
						require.NoError(t, err)
						byteSlices = append(byteSlices, bz)
					default:
						t.Errorf("unknown type %T", f)
					}
				}
				assert.Equal(t,
					bytes.HexBytes(merkle.HashFromByteSlices(byteSlices)), tc.header.Hash())
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
		Version:            tmversion.Consensus{Block: math.MaxInt64, App: math.MaxInt64},
		ChainID:            maxChainID,
		Height:             math.MaxInt64,
		Time:               timestamp,
		LastBlockID:        makeBlockID(make([]byte, tmhash.Size), math.MaxInt32, make([]byte, tmhash.Size)),
		LastCommitHash:     tmhash.Sum([]byte("last_commit_hash")),
		DataHash:           tmhash.Sum([]byte("data_hash")),
		ValidatorsHash:     tmhash.Sum([]byte("validators_hash")),
		VotersHash:         tmhash.Sum([]byte("voters_hash")),
		NextValidatorsHash: tmhash.Sum([]byte("next_validators_hash")),
		ConsensusHash:      tmhash.Sum([]byte("consensus_hash")),
		AppHash:            tmhash.Sum([]byte("app_hash")),
		LastResultsHash:    tmhash.Sum([]byte("last_results_hash")),
		EvidenceHash:       tmhash.Sum([]byte("evidence_hash")),
		ProposerAddress:    crypto.AddressHash([]byte("proposer_address")),
	}

	bz, err := h.ToProto().Marshal()
	require.NoError(t, err)

	assert.EqualValues(t, MaxHeaderBytes, int64(len(bz)))
}

func randCommit(now time.Time) *Commit {
	lastID := makeBlockIDRandom()
	h := int64(3)
	voteSet, _, _, vals := randVoteSet(h-1, 1, tmproto.PrecommitType, 10, 1)
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
		ValidatorIndex:   math.MaxInt32,
		Height:           math.MaxInt64,
		Round:            math.MaxInt32,
		Timestamp:        timestamp,
		Type:             tmproto.PrecommitType,
		BlockID:          blockID,
	}
	pbVote1 := vote1.ToProto()
	assert.NoError(t, pv1.SignVote(chainID, pbVote1))

	vote2 := &Vote{
		ValidatorAddress: pub2.Address(),
		ValidatorIndex:   math.MaxInt32,
		Height:           math.MaxInt64,
		Round:            math.MaxInt32,
		Timestamp:        timestamp,
		Type:             tmproto.PrecommitType,
		BlockID:          blockID,
	}
	pbVote2 := vote2.ToProto()
	assert.NoError(t, pv2.SignVote(chainID, pbVote2))

	vote3 := &Vote{
		ValidatorAddress: pub3.Address(),
		ValidatorIndex:   math.MaxInt32,
		Height:           math.MaxInt64,
		Round:            math.MaxInt32,
		Timestamp:        timestamp,
		Type:             tmproto.PrecommitType,
		BlockID:          blockID,
	}
	pbVote3 := vote3.ToProto()
	assert.NoError(t, pv3.SignVote(chainID, pbVote3))

	commitSig1 := NewCommitSigForBlock(pbVote1.Signature, pub1.Address(), timestamp)
	commitSig2 := NewCommitSigForBlock(pbVote2.Signature, pub2.Address(), timestamp)
	commitSig3 := NewCommitSigForBlock(pbVote3.Signature, pub3.Address(), timestamp)
	aggregatedCommitSig := NewCommitSigForBlock(nil, pub2.Address(), timestamp)

	b1, err1 := commitSig1.ToProto().Marshal()
	assert.NoError(t, err1)
	assert.Equal(t, int64(len(b1)), commitSig1.MaxCommitSigBytes())

	b2, err2 := commitSig2.ToProto().Marshal()
	assert.NoError(t, err2)
	assert.Equal(t, int64(len(b2)), commitSig2.MaxCommitSigBytes())

	b3, err3 := commitSig3.ToProto().Marshal()
	assert.NoError(t, err3)
	assert.Equal(t, int64(len(b3)), commitSig3.MaxCommitSigBytes())

	b4, err4 := aggregatedCommitSig.ToProto().Marshal()
	assert.NoError(t, err4)
	assert.Equal(t, int64(len(b4)), aggregatedCommitSig.MaxCommitSigBytes())
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
			ValidatorIndex:   math.MaxInt32,
			Height:           math.MaxInt64,
			Round:            math.MaxInt32,
			Timestamp:        timestamp,
			Type:             tmproto.PrecommitType,
			BlockID:          blockID,
		}
		assert.NoError(t, pv.SignVote(chainID, vote.ToProto()))
		commitSig[i] = NewCommitSigForBlock(vote.Signature, pub.Address(), timestamp)
	}
	commit := NewCommit(math.MaxInt64, math.MaxInt32, blockID, commitSig)
	bz, err := commit.ToProto().Marshal()
	assert.NoError(t, err)
	assert.Equal(t, commit.MaxCommitBytes(), int64(len(bz)))
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
			ValidatorIndex:   math.MaxInt32,
			Height:           math.MaxInt64,
			Round:            math.MaxInt32,
			Timestamp:        timestamp,
			Type:             tmproto.PrecommitType,
			BlockID:          blockID,
		}
		// do not sign
		commitSig[i] = NewCommitSigForBlock(vote.Signature, pub.Address(), timestamp)
	}
	commit := NewCommit(math.MaxInt64, math.MaxInt32, blockID, commitSig)
	commit.AggregatedSignature = tmrand.Bytes(bls.SignatureSize)

	bz, err := commit.ToProto().Marshal()
	assert.NoError(t, err)
	assert.Equal(t, commit.MaxCommitBytes(), int64(len(bz)))
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
			ValidatorIndex:   math.MaxInt32,
			Height:           math.MaxInt64,
			Round:            math.MaxInt32,
			Timestamp:        timestamp,
			Type:             tmproto.PrecommitType,
			BlockID:          blockID,
		}
		// sign only if key type is ed25519
		if keyType == PrivKeyEd25519 {
			assert.NoError(t, pv.SignVote(chainID, vote.ToProto()))
		}
		commitSig[i] = NewCommitSigForBlock(vote.Signature, pub.Address(), timestamp)
	}
	commit := NewCommit(math.MaxInt64, math.MaxInt32, blockID, commitSig)
	commit.AggregatedSignature = tmrand.Bytes(bls.SignatureSize)

	bz, err := commit.ToProto().Marshal()
	assert.NoError(t, err)
	assert.Equal(t, commit.MaxCommitBytes(), int64(len(bz)))
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
	dupEv := NewDuplicateVoteEvidence(vote3, vote4, defaultVoteTime, ToVoterAll(val))

	testCases := []struct {
		maxBytes int64
		commit   *Commit
		evidence []Evidence
		panics   bool
		result   int64
	}{
		0: {-10, commit, []Evidence{dupEv}, true, 0},
		1: {10, commit, []Evidence{dupEv}, true, 0},
		2: {1600, commit, []Evidence{dupEv}, true, 0},
		3: {1692, commit, []Evidence{dupEv}, false, 0},
		4: {1693, commit, []Evidence{dupEv}, false, 1},
	}

	for i, tc := range testCases {
		tc := tc
		if tc.panics {
			assert.Panics(t, func() {
				MaxDataBytes(tc.maxBytes, tc.commit, tc.evidence)
			}, "#%v", i)
		} else {
			assert.NotPanics(t, func() {
				MaxDataBytes(tc.maxBytes, tc.commit, tc.evidence)
			}, "#%v", i)
			assert.Equal(t,
				tc.result,
				MaxDataBytes(tc.maxBytes, tc.commit, tc.evidence),
				"#%v", i)
		}
	}
}

func TestBlockMaxDataBytesNoEvidence(t *testing.T) {
	testCases := []struct {
		maxBytes  int64
		valsCount int
		panics    bool
		result    int64
	}{
		0: {-10, 1, true, 0},
		1: {10, 1, true, 0},
		2: {909, 1, true, 0},
		3: {910, 1, false, 0},
		4: {911, 1, false, 1},
	}

	for i, tc := range testCases {
		tc := tc
		if tc.panics {
			assert.Panics(t, func() {
				MaxDataBytesNoEvidence(tc.maxBytes, tc.valsCount)
			}, "#%v", i)
		} else {
			assert.NotPanics(t, func() {
				MaxDataBytesNoEvidence(tc.maxBytes, tc.valsCount)
			}, "#%v", i)
			assert.Equal(t,
				tc.result,
				MaxDataBytesNoEvidence(tc.maxBytes, tc.valsCount),
				"#%v", i)
		}
	}
}

func TestCommitToVoteSet(t *testing.T) {
	lastID := makeBlockIDRandom()
	h := int64(3)

	voteSet, _, voterSet, vals := randVoteSet(h-1, 1, tmproto.PrecommitType, 10, 1)
	commit, err := MakeCommit(lastID, h-1, 1, voteSet, vals, time.Now())
	assert.NoError(t, err)

	chainID := voteSet.ChainID()
	voteSet2 := CommitToVoteSet(chainID, commit, voterSet)

	assert.Nil(t, voteSet.aggregatedSignature)
	assert.NotNil(t, commit.AggregatedSignature)
	assert.Equal(t, commit.AggregatedSignature, voteSet2.aggregatedSignature)

	for i := int32(0); int(i) < len(vals); i++ {
		vote1 := voteSet2.GetByIndex(i)
		vote2 := commit.GetVote(i)

		vote1bz, err := vote1.ToProto().Marshal()
		require.NoError(t, err)
		vote2bz, err := vote2.ToProto().Marshal()
		require.NoError(t, err)
		assert.Equal(t, vote1bz, vote2bz)
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
		voteSet, _, voterSet, vals := randVoteSet(height-1, round, tmproto.PrecommitType, tc.numValidators, 1)

		vi := int32(0)
		for n := range tc.blockIDs {
			for i := 0; i < tc.numVotes[n]; i++ {
				pubKey, err := vals[vi].GetPubKey()
				require.NoError(t, err)
				vote := &Vote{
					ValidatorAddress: pubKey.Address(),
					ValidatorIndex:   vi,
					Height:           height - 1,
					Round:            round,
					Type:             tmproto.PrecommitType,
					BlockID:          tc.blockIDs[n],
					Timestamp:        tmtime.Now(),
					Signature:        []byte{},
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
			err := voterSet.VerifyCommit(voteSet.ChainID(), blockID, height-1, commit)
			assert.Nil(t, err)
		} else {
			assert.Panics(t, func() { voteSet.MakeCommit() })
		}
	}
}

func TestBlockIDValidateBasic(t *testing.T) {
	validBlockID := BlockID{
		Hash: bytes.HexBytes{},
		PartSetHeader: PartSetHeader{
			Total: 1,
			Hash:  bytes.HexBytes{},
		},
	}

	invalidBlockID := BlockID{
		Hash: []byte{0},
		PartSetHeader: PartSetHeader{
			Total: 1,
			Hash:  []byte{0},
		},
	}

	testCases := []struct {
		testName             string
		blockIDHash          bytes.HexBytes
		blockIDPartSetHeader PartSetHeader
		expectErr            bool
	}{
		{"Valid BlockID", validBlockID.Hash, validBlockID.PartSetHeader, false},
		{"Invalid BlockID", invalidBlockID.Hash, validBlockID.PartSetHeader, true},
		{"Invalid BlockID", validBlockID.Hash, invalidBlockID.PartSetHeader, true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			blockID := BlockID{
				Hash:          tc.blockIDHash,
				PartSetHeader: tc.blockIDPartSetHeader,
			}
			assert.Equal(t, tc.expectErr, blockID.ValidateBasic() != nil, "Validate Basic had an unexpected result")
		})
	}
}

func TestBlockProtoBuf(t *testing.T) {
	h := tmrand.Int63()
	c1 := randCommit(time.Now())
	b1 := MakeBlock(h, []Tx{Tx([]byte{1})}, &Commit{Signatures: []CommitSig{}}, []Evidence{})
	b1.ProposerAddress = tmrand.Bytes(crypto.AddressSize)

	b2 := MakeBlock(h, []Tx{Tx([]byte{1})}, c1, []Evidence{})
	b2.ProposerAddress = tmrand.Bytes(crypto.AddressSize)
	evidenceTime := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
	evi := NewMockDuplicateVoteEvidence(h, evidenceTime, "block-test-chain")
	b2.Evidence = EvidenceData{Evidence: EvidenceList{evi}}
	b2.EvidenceHash = b2.Evidence.Hash()

	b3 := MakeBlock(h, []Tx{}, c1, []Evidence{})
	b3.ProposerAddress = tmrand.Bytes(crypto.AddressSize)
	testCases := []struct {
		msg      string
		b1       *Block
		expPass  bool
		expPass2 bool
	}{
		{"nil block", nil, false, false},
		{"b1", b1, true, true},
		{"b2", b2, true, true},
		{"b3", b3, true, true},
	}
	for _, tc := range testCases {
		pb, err := tc.b1.ToProto()
		if tc.expPass {
			require.NoError(t, err, tc.msg)
		} else {
			require.Error(t, err, tc.msg)
		}

		block, err := BlockFromProto(pb)
		if tc.expPass2 {
			require.NoError(t, err, tc.msg)
			require.EqualValues(t, tc.b1.Header, block.Header, tc.msg)
			require.EqualValues(t, tc.b1.Data, block.Data, tc.msg)
			require.EqualValues(t, tc.b1.Evidence.Evidence, block.Evidence.Evidence, tc.msg)
			require.EqualValues(t, *tc.b1.LastCommit, *block.LastCommit, tc.msg)
		} else {
			require.Error(t, err, tc.msg)
		}
	}
}

func TestDataProtoBuf(t *testing.T) {
	data := &Data{Txs: Txs{Tx([]byte{1}), Tx([]byte{2}), Tx([]byte{3})}}
	data2 := &Data{Txs: Txs{}}
	testCases := []struct {
		msg     string
		data1   *Data
		expPass bool
	}{
		{"success", data, true},
		{"success data2", data2, true},
	}
	for _, tc := range testCases {
		protoData := tc.data1.ToProto()
		d, err := DataFromProto(&protoData)
		if tc.expPass {
			require.NoError(t, err, tc.msg)
			require.EqualValues(t, tc.data1, &d, tc.msg)
		} else {
			require.Error(t, err, tc.msg)
		}
	}
}

// TestEvidenceDataProtoBuf ensures parity in converting to and from proto.
func TestEvidenceDataProtoBuf(t *testing.T) {
	const chainID = "mychain6"
	ev := NewMockDuplicateVoteEvidence(math.MaxInt64, time.Now(), chainID)
	data := &EvidenceData{Evidence: EvidenceList{ev}}
	_ = data.ByteSize()
	testCases := []struct {
		msg      string
		data1    *EvidenceData
		expPass1 bool
		expPass2 bool
	}{
		{"success", data, true, true},
		{"empty evidenceData", &EvidenceData{Evidence: EvidenceList{}}, true, true},
		{"fail nil Data", nil, false, false},
	}

	for _, tc := range testCases {
		protoData, err := tc.data1.ToProto()
		if tc.expPass1 {
			require.NoError(t, err, tc.msg)
		} else {
			require.Error(t, err, tc.msg)
		}

		eviD := new(EvidenceData)
		err = eviD.FromProto(protoData)
		if tc.expPass2 {
			require.NoError(t, err, tc.msg)
			require.Equal(t, tc.data1, eviD, tc.msg)
		} else {
			require.Error(t, err, tc.msg)
		}
	}
}

func makeRandHeader() Header {
	chainID := "test"
	t := time.Now()
	height := tmrand.Int63()
	randBytes := tmrand.Bytes(tmhash.Size)
	randAddress := tmrand.Bytes(crypto.AddressSize)
	h := Header{
		Version:            tmversion.Consensus{Block: version.BlockProtocol, App: 1},
		ChainID:            chainID,
		Height:             height,
		Time:               t,
		LastBlockID:        BlockID{},
		LastCommitHash:     randBytes,
		DataHash:           randBytes,
		VotersHash:         randBytes,
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

func TestBlockIDEquals(t *testing.T) {
	var (
		blockID          = makeBlockID([]byte("hash"), 2, []byte("part_set_hash"))
		blockIDDuplicate = makeBlockID([]byte("hash"), 2, []byte("part_set_hash"))
		blockIDDifferent = makeBlockID([]byte("different_hash"), 2, []byte("part_set_hash"))
		blockIDEmpty     = BlockID{}
	)

	assert.True(t, blockID.Equals(blockIDDuplicate))
	assert.False(t, blockID.Equals(blockIDDifferent))
	assert.False(t, blockID.Equals(blockIDEmpty))
	assert.True(t, blockIDEmpty.Equals(blockIDEmpty))
	assert.False(t, blockIDEmpty.Equals(blockIDDifferent))
}
