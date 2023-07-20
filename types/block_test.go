package types

import (
	// it is ok to use math/rand here: we do not need a cryptographically secure random
	// number generator here and we can run the tests a bit faster
	"crypto/rand"
	"encoding/hex"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	gogotypes "github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmversion "github.com/tendermint/tendermint/proto/tendermint/version"

	"github.com/Finschia/ostracon/crypto"
	"github.com/Finschia/ostracon/crypto/merkle"
	"github.com/Finschia/ostracon/crypto/tmhash"
	"github.com/Finschia/ostracon/libs/bits"
	"github.com/Finschia/ostracon/libs/bytes"
	tmrand "github.com/Finschia/ostracon/libs/rand"
	tmtime "github.com/Finschia/ostracon/types/time"
	"github.com/Finschia/ostracon/version"
	vrf "github.com/oasisprotocol/curve25519-voi/primitives/ed25519/extra/ecvrf"
)

var TestConsensusVersion = tmversion.Consensus{
	Block: version.BlockProtocol,
	App:   version.AppProtocol,
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

func TestBlockAddEvidence(t *testing.T) {
	txs := []Tx{Tx("foo"), Tx("bar")}
	lastID := makeBlockIDRandom()
	h := int64(3)

	voteSet, _, vals := randVoteSet(h-1, 1, tmproto.PrecommitType, 10, 1)
	commit, err := MakeCommit(lastID, h-1, 1, voteSet, vals, time.Now())
	require.NoError(t, err)

	ev := NewMockDuplicateVoteEvidenceWithValidator(h, time.Now(), vals[0], "block-test-chain")
	evList := []Evidence{ev}

	block := MakeBlock(h, txs, commit, evList, TestConsensusVersion)
	require.NotNil(t, block)
	require.Equal(t, 1, len(block.Evidence.Evidence))
	require.NotNil(t, block.EvidenceHash)
}

func TestBlockValidateBasic(t *testing.T) {
	require.Error(t, (*Block)(nil).ValidateBasic())

	txs := []Tx{Tx("foo"), Tx("bar")}
	lastID := makeBlockIDRandom()
	h := int64(3)

	voteSet, valSet, vals := randVoteSet(h-1, 1, tmproto.PrecommitType, 10, 1)
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
		{"Tampered ProposerAddress", func(blk *Block) {
			blk.ProposerAddress = []byte("something else")
		}, true},
		{"Incorrect chain id", func(blk *Block) {
			blk.ChainID = "123456789012345678901234567890123456789012345678901"
		}, true},
		{"Negative Height", func(blk *Block) { blk.Height = -1 }, true},
		{"Height is zero", func(blk *Block) { blk.Height = 0 }, true},
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
		{"Tampered DataHash", func(blk *Block) {
			blk.DataHash = []byte("something else")
		}, true},
		{"Tampered EvidenceHash", func(blk *Block) {
			blk.EvidenceHash = []byte("something else")
		}, true},
		{"Incorrect block protocol version", func(blk *Block) {
			blk.Version.Block = 1
		}, true},
		{"Tampered ValidatorsHash", func(blk *Block) {
			blk.ValidatorsHash = []byte("something else")
		}, true},
		{"Tampered NextValidatorsHash", func(blk *Block) {
			blk.NextValidatorsHash = []byte("something else")
		}, true},
		{"Tampered ConsensusHash", func(blk *Block) {
			blk.ConsensusHash = []byte("something else")
		}, true},
		{"Tampered LastResultsHash", func(blk *Block) {
			blk.LastResultsHash = []byte("something else")
		}, true},
		{"Tampered LastBlockID", func(blk *Block) {
			blk.LastBlockID = BlockID{Hash: []byte("something else")}
		}, true},
		{"Negative Round", func(blk *Block) {
			blk.Round = -1
		}, true},
		{"Incorrect Proof Size", func(blk *Block) {
			blk.Proof = []byte("wrong proof size")
		}, true},
	}
	for i, tc := range testCases {
		tc := tc
		i := i
		t.Run(tc.testName, func(t *testing.T) {
			block := MakeBlock(h, txs, commit, evList, TestConsensusVersion)
			block.ProposerAddress = valSet.SelectProposer([]byte{}, block.Height, 0).Address
			tc.malleateBlock(block)
			err = block.ValidateBasic()
			assert.Equal(t, tc.expErr, err != nil, "#%d: %v", i, err)
		})
	}
}

func TestBlockHash(t *testing.T) {
	assert.Nil(t, (*Block)(nil).Hash())
	assert.Nil(t, MakeBlock(int64(3), []Tx{Tx("Hello World")}, nil, nil, TestConsensusVersion).Hash())
}

func TestBlockMakePartSet(t *testing.T) {
	assert.Nil(t, (*Block)(nil).MakePartSet(2))

	partSet := MakeBlock(int64(3), []Tx{Tx("Hello World")}, nil, nil, TestConsensusVersion).MakePartSet(1024)
	assert.NotNil(t, partSet)
	assert.EqualValues(t, 1, partSet.Total())
}

func TestBlockMakePartSetWithEvidence(t *testing.T) {
	assert.Nil(t, (*Block)(nil).MakePartSet(2))

	lastID := makeBlockIDRandom()
	h := int64(3)

	voteSet, _, vals := randVoteSet(h-1, 1, tmproto.PrecommitType, 10, 1)
	commit, err := MakeCommit(lastID, h-1, 1, voteSet, vals, time.Now())
	require.NoError(t, err)

	ev := NewMockDuplicateVoteEvidenceWithValidator(h, time.Now(), vals[0], "block-test-chain")
	evList := []Evidence{ev}

	block := MakeBlock(h, []Tx{Tx("Hello World")}, commit, evList, TestConsensusVersion)
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
	voteSet, valSet, vals := randVoteSet(h-1, 1, tmproto.PrecommitType, 10, 1)
	commit, err := MakeCommit(lastID, h-1, 1, voteSet, vals, time.Now())
	require.NoError(t, err)

	ev := NewMockDuplicateVoteEvidenceWithValidator(h, time.Now(), vals[0], "block-test-chain")
	evList := []Evidence{ev}

	block := MakeBlock(h, []Tx{Tx("Hello World")}, commit, evList, TestConsensusVersion)
	block.ValidatorsHash = valSet.Hash()
	assert.False(t, block.HashesTo([]byte{}))
	assert.False(t, block.HashesTo([]byte("something else")))
	assert.True(t, block.HashesTo(block.Hash()))
}

func TestBlockSize(t *testing.T) {
	size := MakeBlock(int64(3), []Tx{Tx("Hello World")}, nil, nil, TestConsensusVersion).Size()
	if size <= 0 {
		t.Fatal("Size of the block is zero or negative")
	}
}

func TestBlockString(t *testing.T) {
	assert.Equal(t, "nil-Block", (*Block)(nil).String())
	assert.Equal(t, "nil-Block", (*Block)(nil).StringIndented(""))
	assert.Equal(t, "nil-Block", (*Block)(nil).StringShort())

	block := MakeBlock(int64(3), []Tx{Tx("Hello World")}, nil, nil, TestConsensusVersion)
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

// This follows RFC-6962, i.e. `echo -n ” | sha256sum`
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

func TestCommit(t *testing.T) {
	lastID := makeBlockIDRandom()
	h := int64(3)
	voteSet, _, vals := randVoteSet(h-1, 1, tmproto.PrecommitType, 10, 1)
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

	assert.Equal(t, voteSet.GetByIndex(0), commit.GetByIndex(0))
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
	assert.EqualValues(t, MaxCommitSigBytes, int64(pbSig.Size()))

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

	assert.EqualValues(t, MaxCommitBytes(1), int64(pb.Size()))

	// check the upper bound of the commit size
	for i := 1; i < MaxVotesCount; i++ {
		commit.Signatures = append(commit.Signatures, cs)
	}

	pb = commit.ToProto()

	assert.EqualValues(t, MaxCommitBytes(MaxVotesCount), int64(pb.Size()))
}

func TestCommitHash(t *testing.T) {
	t.Run("receiver is nil", func(t *testing.T) {
		var commit *Commit
		assert.Nil(t, commit.Hash())
	})

	t.Run("without any signatures", func(t *testing.T) {
		commit := &Commit{
			hash:       nil,
			Signatures: nil,
		}
		expected := []byte{0xe3, 0xb0, 0xc4, 0x42, 0x98, 0xfc, 0x1c, 0x14, 0x9a, 0xfb, 0xf4, 0xc8, 0x99, 0x6f, 0xb9,
			0x24, 0x27, 0xae, 0x41, 0xe4, 0x64, 0x9b, 0x93, 0x4c, 0xa4, 0x95, 0x99, 0x1b, 0x78, 0x52, 0xb8, 0x55}
		assert.Equal(t, expected, commit.Hash().Bytes())
	})

	t.Run("with signatures", func(t *testing.T) {
		signature := []byte{0, 0, 0, 0}
		address := []byte{0, 0, 0, 0}
		tm := time.Unix(0, 0)
		commit := &Commit{
			hash: nil,
			Signatures: []CommitSig{
				NewCommitSigAbsent(),
				NewCommitSigForBlock(signature, address, tm),
			},
		}
		expected := []byte{0xf9, 0x3c, 0x17, 0x4b, 0x5c, 0x27, 0x56, 0xef, 0x81, 0x7a, 0x43, 0x83, 0x63, 0x15, 0x60,
			0x84, 0xc1, 0x3d, 0x6, 0x10, 0xfd, 0x94, 0xb9, 0x5d, 0xb0, 0x46, 0xbb, 0x11, 0x1d, 0x6c, 0x65, 0x2a}
		assert.Equal(t, expected, commit.Hash().Bytes())

		commit.hash = nil
		expected = []byte{0xf9, 0x3c, 0x17, 0x4b, 0x5c, 0x27, 0x56, 0xef, 0x81, 0x7a, 0x43, 0x83, 0x63, 0x15, 0x60,
			0x84, 0xc1, 0x3d, 0x6, 0x10, 0xfd, 0x94, 0xb9, 0x5d, 0xb0, 0x46, 0xbb, 0x11, 0x1d, 0x6c, 0x65, 0x2a}
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
			NextValidatorsHash: tmhash.Sum([]byte("next_validators_hash")),
			ConsensusHash:      tmhash.Sum([]byte("consensus_hash")),
			AppHash:            tmhash.Sum([]byte("app_hash")),
			LastResultsHash:    tmhash.Sum([]byte("last_results_hash")),
			EvidenceHash:       tmhash.Sum([]byte("evidence_hash")),
			ProposerAddress:    crypto.AddressHash([]byte("proposer_address")),
		}, hexBytesFromString("F740121F553B5418C3EFBD343C2DBFE9E007BB67B0D020A0741374BAB65242A4")},
		{"nil header yields nil", nil, nil},
		{"nil ValidatorsHash yields nil", &Header{
			Version:            tmversion.Consensus{Block: 1, App: 2},
			ChainID:            "chainId",
			Height:             3,
			Time:               time.Date(2019, 10, 13, 16, 14, 44, 0, time.UTC),
			LastBlockID:        makeBlockID(make([]byte, tmhash.Size), 6, make([]byte, tmhash.Size)),
			LastCommitHash:     tmhash.Sum([]byte("last_commit_hash")),
			DataHash:           tmhash.Sum([]byte("data_hash")),
			ValidatorsHash:     nil,
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

					switch f := f.Interface().(type) {
					case int32, int64, bytes.HexBytes, []byte, string:
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

func TestHeaderValidateBasic(t *testing.T) {
	invalidHashLength := tmhash.Size - 1

	testCases := []struct {
		testName       string
		malleateHeader func(*Header)
		expErr         bool
	}{
		{"Make Header", func(header *Header) {}, false},
		{"Incorrect block protocol version", func(header *Header) {
			header.Version.Block = uint64(1)
		}, true},
		{"Too long chainID", func(header *Header) {
			header.ChainID = "long chainID" + strings.Repeat("-", MaxChainIDLen)
		}, true},
		{"Negative Height", func(header *Header) {
			header.Height = -1
		}, true},
		{"Zero Height", func(header *Header) {
			header.Height = 0
		}, true},
		{"Invalid Last Block ID", func(header *Header) {
			header.LastBlockID = BlockID{
				Hash: make([]byte, invalidHashLength),
				PartSetHeader: PartSetHeader{
					Total: 6,
					Hash:  make([]byte, invalidHashLength),
				},
			}
		}, true},
		{"Invalid Last Commit Hash", func(header *Header) {
			header.LastCommitHash = []byte(strings.Repeat("h", invalidHashLength))
		}, true},
		{"Invalid Data Hash", func(header *Header) {
			header.DataHash = []byte(strings.Repeat("h", invalidHashLength))
		}, true},
		{"Invalid Evidence Hash", func(header *Header) {
			header.EvidenceHash = []byte(strings.Repeat("h", invalidHashLength))
		}, true},
		{"Invalid Proposer Address length", func(header *Header) {
			header.ProposerAddress = make([]byte, crypto.AddressSize-1)
		}, true},
		{"Invalid Next Validators Hash", func(header *Header) {
			header.NextValidatorsHash = []byte(strings.Repeat("h", invalidHashLength))
		}, true},
		{"Invalid Consensus Hash", func(header *Header) {
			header.ConsensusHash = []byte(strings.Repeat("h", invalidHashLength))
		}, true},
		{"Invalid Results Hash", func(header *Header) {
			header.LastResultsHash = []byte(strings.Repeat("h", invalidHashLength))
		}, true},
		{"Invalid Validators Hash", func(header *Header) {
			header.ValidatorsHash = []byte(strings.Repeat("h", invalidHashLength))
		}, true},
	}
	for i, tc := range testCases {
		tc := tc
		i := i
		t.Run(tc.testName, func(t *testing.T) {
			header := &Header{
				Version:            tmversion.Consensus{Block: version.BlockProtocol, App: version.AppProtocol},
				ChainID:            "chainId",
				Height:             3,
				Time:               time.Date(2019, 10, 13, 16, 14, 44, 0, time.UTC),
				LastBlockID:        makeBlockID(make([]byte, tmhash.Size), 6, make([]byte, tmhash.Size)),
				LastCommitHash:     tmhash.Sum([]byte("last_commit_hash")),
				DataHash:           tmhash.Sum([]byte("data_hash")),
				ValidatorsHash:     tmhash.Sum([]byte("validators_hash")),
				NextValidatorsHash: tmhash.Sum([]byte("next_validators_hash")),
				ConsensusHash:      tmhash.Sum([]byte("consensus_hash")),
				AppHash:            tmhash.Sum([]byte("app_hash")),
				LastResultsHash:    tmhash.Sum([]byte("last_results_hash")),
				EvidenceHash:       tmhash.Sum([]byte("evidence_hash")),
				ProposerAddress:    crypto.AddressHash([]byte("proposer_address")),
			}
			tc.malleateHeader(header)
			err := header.ValidateBasic()
			assert.Equal(t, tc.expErr, err != nil, "#%d: %v", i, err)
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

	proof := make([]byte, vrf.ProofSize)
	for i := 0; i < len(proof); i++ {
		proof[i] = 0xFF
	}

	h := Header{
		Version:            tmversion.Consensus{Block: math.MaxInt64, App: math.MaxInt64},
		ChainID:            maxChainID,
		Height:             math.MaxInt64,
		Time:               timestamp,
		LastBlockID:        makeBlockID(make([]byte, tmhash.Size), math.MaxInt32, make([]byte, tmhash.Size)),
		LastCommitHash:     tmhash.Sum([]byte("last_commit_hash")),
		DataHash:           tmhash.Sum([]byte("data_hash")),
		ValidatorsHash:     tmhash.Sum([]byte("validators_hash")),
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
	voteSet, _, vals := randVoteSet(h-1, 1, tmproto.PrecommitType, 10, 1)
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

func TestBlockMaxDataBytes(t *testing.T) {
	testCases := []struct {
		maxBytes      int64
		valsCount     int
		evidenceBytes int64
		panics        bool
		result        int64
	}{
		0:  {-10, 1, 0, true, 0},
		1:  {10, 1, 0, true, 0},
		2:  {849 + int64(vrf.ProofSize), 1, 0, true, 0},
		3:  {850 + int64(vrf.ProofSize), 1, 0, false, 0},
		4:  {851 + int64(vrf.ProofSize), 1, 0, false, 1},
		5:  {960 + int64(vrf.ProofSize), 2, 0, true, 0},
		6:  {961 + int64(vrf.ProofSize), 2, 0, false, 0},
		7:  {962 + int64(vrf.ProofSize), 2, 0, false, 1},
		8:  {1060 + int64(vrf.ProofSize), 2, 100, true, 0},
		9:  {1061 + int64(vrf.ProofSize), 2, 100, false, 0},
		10: {1062 + int64(vrf.ProofSize), 2, 100, false, 1},
	}

	for i, tc := range testCases {
		tc := tc
		if tc.panics {
			assert.Panics(t, func() {
				MaxDataBytes(tc.maxBytes, tc.evidenceBytes, tc.valsCount)
			}, "#%v", i)
		} else {
			assert.Equal(t,
				tc.result,
				MaxDataBytes(tc.maxBytes, tc.evidenceBytes, tc.valsCount),
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
		2: {849 + int64(vrf.ProofSize), 1, true, 0},
		3: {850 + int64(vrf.ProofSize), 1, false, 0},
		4: {851 + int64(vrf.ProofSize), 1, false, 1},
		5: {960 + int64(vrf.ProofSize), 2, true, 0},
		6: {961 + int64(vrf.ProofSize), 2, false, 0},
		7: {962 + int64(vrf.ProofSize), 2, false, 1},
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

	voteSet, valSet, vals := randVoteSet(h-1, 1, tmproto.PrecommitType, 10, 1)
	commit, err := MakeCommit(lastID, h-1, 1, voteSet, vals, time.Now())
	assert.NoError(t, err)

	chainID := voteSet.ChainID()
	voteSet2 := CommitToVoteSet(chainID, commit, valSet)

	for i := int32(0); int(i) < len(vals); i++ {
		// This is the vote before `MakeCommit`.
		vote1 := voteSet.GetByIndex(i)
		// This is the vote created from `CommitToVoteSet`
		vote2 := voteSet2.GetByIndex(i)
		// This is the vote created from `MakeCommit`
		vote3 := commit.GetVote(i)

		vote1bz, err := vote1.ToProto().Marshal()
		require.NoError(t, err)
		vote2bz, err := vote2.ToProto().Marshal()
		require.NoError(t, err)
		vote3bz, err := vote3.ToProto().Marshal()
		require.NoError(t, err)
		assert.Equal(t, vote1bz, vote2bz)
		assert.Equal(t, vote1bz, vote3bz)
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
		voteSet, valSet, vals := randVoteSet(height-1, round, tmproto.PrecommitType, tc.numValidators, 1)

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
	b1 := MakeBlock(h, []Tx{Tx([]byte{1})}, &Commit{Signatures: []CommitSig{}}, []Evidence{}, TestConsensusVersion)
	b1.ProposerAddress = tmrand.Bytes(crypto.AddressSize)

	b2 := MakeBlock(h, []Tx{Tx([]byte{1})}, c1, []Evidence{}, TestConsensusVersion)
	b2.ProposerAddress = tmrand.Bytes(crypto.AddressSize)
	evidenceTime := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
	evi := NewMockDuplicateVoteEvidence(h, evidenceTime, "block-test-chain")
	b2.Evidence = EvidenceData{Evidence: EvidenceList{evi}}
	b2.EvidenceHash = b2.Evidence.Hash()

	b3 := MakeBlock(h, []Tx{}, c1, []Evidence{}, TestConsensusVersion)
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
			require.EqualValues(t, tc.b1.Entropy, block.Entropy, tc.msg)
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
	const chainID = "mychain"
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
		Version:            tmversion.Consensus{Block: version.BlockProtocol, App: version.AppProtocol},
		ChainID:            chainID,
		Height:             height,
		Time:               t,
		LastBlockID:        BlockID{},
		LastCommitHash:     randBytes,
		DataHash:           randBytes,
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

func TestEntropyHash(t *testing.T) {
	testCases := []struct {
		desc       string
		entropy    *Entropy
		expectHash bytes.HexBytes
	}{
		{"Generates expected hash", &Entropy{
			Round: 1,
			// The Proof defined here does not depend on the vrf ProofLength,
			// but it is a fixed value for the purpose of calculating the Hash value.
			Proof: tmhash.Sum([]byte("proof")),
		}, hexBytesFromString("3EEC62453202DEF45126D758F5DF58962147B358E7B135E19D4CDB79B0CDA5C7")},
		{"nil entropy yields nil", nil, nil},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			assert.Equal(t, tc.expectHash, tc.entropy.Hash())

			// We also make sure that all fields are hashed in struct order, and that all
			// fields in the test struct are non-zero.
			if tc.entropy != nil && tc.expectHash != nil {
				byteSlices := [][]byte{}

				s := reflect.ValueOf(*tc.entropy)
				for i := 0; i < s.NumField(); i++ {
					f := s.Field(i)

					assert.False(t, f.IsZero(), "Found zero-valued field %v",
						s.Type().Field(i).Name)

					switch f := f.Interface().(type) {
					case int32, int64, bytes.HexBytes, []byte, string:
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
					bytes.HexBytes(merkle.HashFromByteSlices(byteSlices)), tc.entropy.Hash())
			}
		})
	}
}

func TestEntropyValidateBasic(t *testing.T) {
	testCases := []struct {
		testName        string
		malleateEntropy func(*Entropy)
		expErr          bool
	}{
		{"Make Entropy", func(entropy *Entropy) {}, false},
		{"Negative Round", func(entropy *Entropy) {
			entropy.Round = -1
		}, true},
		{"Invalid Proof", func(entropy *Entropy) {
			entropy.Proof = make([]byte, vrf.ProofSize-1)
		}, true},
	}
	for i, tc := range testCases {
		tc := tc
		i := i
		t.Run(tc.testName, func(t *testing.T) {
			header := &Entropy{
				Round: 1,
				Proof: make([]byte, vrf.ProofSize),
			}
			tc.malleateEntropy(header)
			err := header.ValidateBasic()
			assert.Equal(t, tc.expErr, err != nil, "#%d: %v", i, err)
		})
	}
}

func TestMaxEntropyBytes(t *testing.T) {
	proof := make([]byte, vrf.ProofSize)
	for i := 0; i < len(proof); i++ {
		proof[i] = 0xFF
	}

	h := Entropy{
		Round: math.MaxInt32,
		Proof: proof,
	}

	bz, err := h.ToProto().Marshal()
	require.NoError(t, err)

	assert.EqualValues(t, MaxEntropyBytes, int64(len(bz)))
}

func makeEntropyHeader() Entropy {
	round := tmrand.Int31()
	randProof := tmrand.Bytes(vrf.ProofSize)
	vp := Entropy{
		Round: round,
		Proof: randProof,
	}

	return vp
}

func TestEntropyProto(t *testing.T) {
	vp1 := makeEntropyHeader()
	tc := []struct {
		msg     string
		vp1     *Entropy
		expPass bool
	}{
		{"success", &vp1, true},
		{"success empty Entropy", &Entropy{}, true},
	}

	for _, tt := range tc {
		tt := tt
		t.Run(tt.msg, func(t *testing.T) {
			pb := tt.vp1.ToProto()
			h, err := EntropyFromProto(pb)
			if tt.expPass {
				require.NoError(t, err, tt.msg)
				require.Equal(t, tt.vp1, &h, tt.msg)
			} else {
				require.Error(t, err, tt.msg)
			}

		})
	}
}

func TestEntropyCorrectness(t *testing.T) {
	chainID := "test"
	height := int64(1)
	tx := []Tx{}
	evList := []Evidence{}
	commit := &Commit{}

	round := int32(0)
	proof := []byte("proof")
	differentRound := round + 1
	differentProof := []byte("different proof")

	testBlock := MakeBlock(height, tx, commit, evList, TestConsensusVersion)
	testBlock.Entropy.Populate(round, proof)
	testBlockId := BlockID{Hash: testBlock.Hash(), PartSetHeader: testBlock.MakePartSet(1024).Header()}

	sameBlock := MakeBlock(height, tx, commit, evList, TestConsensusVersion)
	sameBlock.Entropy.Populate(round, proof)
	sameBlockId := BlockID{Hash: sameBlock.Hash(), PartSetHeader: sameBlock.MakePartSet(1024).Header()}

	roundDiffBlock := MakeBlock(height, tx, commit, evList, TestConsensusVersion)
	roundDiffBlock.Entropy.Populate(differentRound, proof)
	roundDiffBlockId := BlockID{Hash: roundDiffBlock.Hash(), PartSetHeader: roundDiffBlock.MakePartSet(1024).Header()}

	proofDiffBlock := MakeBlock(height, tx, commit, evList, TestConsensusVersion)
	proofDiffBlock.Entropy.Populate(round, differentProof)
	proofDiffBlockId := BlockID{Hash: proofDiffBlock.Hash(), PartSetHeader: proofDiffBlock.MakePartSet(1024).Header()}

	entropyDiffBlock := MakeBlock(height, tx, commit, evList, TestConsensusVersion)
	entropyDiffBlock.Entropy.Populate(differentRound, differentProof)
	entropyDiffBlockId := BlockID{Hash: entropyDiffBlock.Hash(), PartSetHeader: entropyDiffBlock.MakePartSet(1024).Header()}

	t.Run("test block id equality with different entropy", func(t *testing.T) {
		assert.Equal(t, testBlockId, sameBlockId)
		assert.NotEqual(t, testBlockId, roundDiffBlockId)
		assert.NotEqual(t, testBlockId, proofDiffBlockId)
		assert.NotEqual(t, testBlockId, entropyDiffBlockId)
	})

	t.Run("test vote signature verification with different entropy", func(t *testing.T) {
		_, privVals := RandValidatorSet(1, 1)
		privVal := privVals[0]
		pubKey, err := privVal.GetPubKey()
		assert.NoError(t, err)

		testVote := &Vote{
			ValidatorAddress: pubKey.Address(),
			ValidatorIndex:   0,
			Height:           height,
			Round:            0,
			Timestamp:        tmtime.Now(),
			Type:             tmproto.PrecommitType,
			BlockID:          testBlockId,
		}
		tv := testVote.ToProto()
		err = privVal.SignVote(chainID, tv)
		assert.NoError(t, err)
		testVote.Signature = tv.Signature

		sameVote := testVote.Copy()
		sameVote.BlockID = sameBlockId

		roundDiffVote := testVote.Copy()
		roundDiffVote.BlockID = roundDiffBlockId

		proofDiffVote := testVote.Copy()
		proofDiffVote.BlockID = proofDiffBlockId

		entropyDiffVote := testVote.Copy()
		entropyDiffVote.BlockID = entropyDiffBlockId

		err = testVote.Verify(chainID, pubKey)
		assert.NoError(t, err)
		err = sameVote.Verify(chainID, pubKey)
		assert.NoError(t, err)
		err = roundDiffVote.Verify(chainID, pubKey)
		assert.Error(t, err)
		err = proofDiffVote.Verify(chainID, pubKey)
		assert.Error(t, err)
		err = entropyDiffVote.Verify(chainID, pubKey)
		assert.Error(t, err)
	})
}
