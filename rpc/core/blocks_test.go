package core

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	abci "github.com/tendermint/tendermint/abci/types"
	dbm "github.com/tendermint/tm-db"

	cfg "github.com/Finschia/ostracon/config"
	"github.com/Finschia/ostracon/crypto"
	tmrand "github.com/Finschia/ostracon/libs/rand"
	tmstate "github.com/Finschia/ostracon/proto/ostracon/state"
	ctypes "github.com/Finschia/ostracon/rpc/core/types"
	rpctypes "github.com/Finschia/ostracon/rpc/jsonrpc/types"
	sm "github.com/Finschia/ostracon/state"
	blockidxkv "github.com/Finschia/ostracon/state/indexer/block/kv"
	blockidxnull "github.com/Finschia/ostracon/state/indexer/block/null"
	txidxkv "github.com/Finschia/ostracon/state/txindex/kv"
	"github.com/Finschia/ostracon/store"
	"github.com/Finschia/ostracon/types"
)

func TestBlockchainInfo(t *testing.T) {
	cases := []struct {
		min, max     int64
		base, height int64
		limit        int64
		resultLength int64
		wantErr      bool
	}{

		// min > max
		{0, 0, 0, 0, 10, 0, true},  // min set to 1
		{0, 1, 0, 0, 10, 0, true},  // max set to height (0)
		{0, 0, 0, 1, 10, 1, false}, // max set to height (1)
		{2, 0, 0, 1, 10, 0, true},  // max set to height (1)
		{2, 1, 0, 5, 10, 0, true},

		// negative
		{1, 10, 0, 14, 10, 10, false}, // control
		{-1, 10, 0, 14, 10, 0, true},
		{1, -10, 0, 14, 10, 0, true},
		{-9223372036854775808, -9223372036854775788, 0, 100, 20, 0, true},

		// check base
		{1, 1, 1, 1, 1, 1, false},
		{2, 5, 3, 5, 5, 3, false},

		// check limit and height
		{1, 1, 0, 1, 10, 1, false},
		{1, 1, 0, 5, 10, 1, false},
		{2, 2, 0, 5, 10, 1, false},
		{1, 2, 0, 5, 10, 2, false},
		{1, 5, 0, 1, 10, 1, false},
		{1, 5, 0, 10, 10, 5, false},
		{1, 15, 0, 10, 10, 10, false},
		{1, 15, 0, 15, 10, 10, false},
		{1, 15, 0, 15, 20, 15, false},
		{1, 20, 0, 15, 20, 15, false},
		{1, 20, 0, 20, 20, 20, false},
	}

	for i, c := range cases {
		caseString := fmt.Sprintf("test %d failed", i)
		min, max, err := filterMinMax(c.base, c.height, c.min, c.max, c.limit)
		if c.wantErr {
			require.Error(t, err, caseString)
		} else {
			require.NoError(t, err, caseString)
			require.Equal(t, 1+max-min, c.resultLength, caseString)
		}
	}
}

func TestBlockResults(t *testing.T) {
	results := &tmstate.ABCIResponses{
		DeliverTxs: []*abci.ResponseDeliverTx{
			{Code: 0, Data: []byte{0x01}, Log: "ok"},
			{Code: 0, Data: []byte{0x02}, Log: "ok"},
			{Code: 1, Log: "not ok"},
		},
		EndBlock:   &abci.ResponseEndBlock{},
		BeginBlock: &abci.ResponseBeginBlock{},
	}

	env = &Environment{}
	env.StateStore = sm.NewStore(dbm.NewMemDB())
	err := env.StateStore.SaveABCIResponses(100, results)
	require.NoError(t, err)
	env.BlockStore = mockBlockStore{height: 100}

	testCases := []struct {
		height  int64
		wantErr bool
		wantRes *ctypes.ResultBlockResults
	}{
		{-1, true, nil},
		{0, true, nil},
		{101, true, nil},
		{100, false, &ctypes.ResultBlockResults{
			Height:                100,
			TxsResults:            results.DeliverTxs,
			BeginBlockEvents:      results.BeginBlock.Events,
			EndBlockEvents:        results.EndBlock.Events,
			ValidatorUpdates:      results.EndBlock.ValidatorUpdates,
			ConsensusParamUpdates: results.EndBlock.ConsensusParamUpdates,
		}},
	}

	for _, tc := range testCases {
		res, err := BlockResults(&rpctypes.Context{}, &tc.height)
		if tc.wantErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tc.wantRes, res)
		}
	}
}

func TestBlockSearchByBlockHeightQuery(t *testing.T) {
	height := int64(1)
	ctx := &rpctypes.Context{}

	q := fmt.Sprintf("%s=%d", types.BlockHeightKey, height)
	page := 1
	perPage := 10
	orderBy := TestOrderByDefault

	state, cleanup := makeTestState()
	defer cleanup()

	{
		// Get by block.height (not search/range)
		res, err := BlockSearch(ctx, q, &page, &perPage, orderBy)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, 0, res.TotalCount) // Don't have height in db
		require.Equal(t, 0, len(res.Blocks))
	}

	numToMakeBlocks := 1
	numToGet := 1
	// Save blocks
	storeTestBlocks(height, int64(numToMakeBlocks), 0, state, time.Now())

	{
		// Get by block.height (not search/range)
		res, err := BlockSearch(ctx, q, &page, &perPage, orderBy)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, numToMakeBlocks, res.TotalCount) // Get
		require.Equal(t, numToGet, len(res.Blocks))
	}
}

func TestBlockSearchByRangeQuery(t *testing.T) {
	height := int64(1)
	ctx := &rpctypes.Context{}

	q := fmt.Sprintf("%s>=%d", types.BlockHeightKey, height)
	page := 1
	perPage := 10
	orderBy := TestOrderByDefault

	state, cleanup := makeTestState()
	defer cleanup()

	numToMakeBlocks := 15
	numToGet := perPage
	// Save blocks
	storeTestBlocks(height, int64(numToMakeBlocks), 0, state, time.Now())

	{
		// Search blocks by range query with desc (default)
		res, err := BlockSearch(ctx, q, &page, &perPage, orderBy)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, numToMakeBlocks, res.TotalCount)
		require.Equal(t, numToGet, len(res.Blocks))
		require.Equal(t, int64(numToMakeBlocks), res.Blocks[0].Block.Height)
		require.Equal(t, height+int64(numToMakeBlocks-numToGet), res.Blocks[numToGet-1].Block.Height)
	}
	{
		orderBy = TestOrderByAsc
		// Search blocks by range query with asc
		res, err := BlockSearch(ctx, q, &page, &perPage, orderBy)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, numToMakeBlocks, res.TotalCount)
		require.Equal(t, numToGet, len(res.Blocks))
		require.Equal(t, height, res.Blocks[0].Block.Height)
		require.Equal(t, int64(numToGet), res.Blocks[numToGet-1].Block.Height)
	}
}

func TestBlockSearch_errors(t *testing.T) {
	ctx := &rpctypes.Context{}

	q := ""
	page := 0
	perPage := 1
	orderBy := "error"

	{
		// error: env.BlockIndexer.(*blockidxnull.BlockerIndexer)
		env = &Environment{}
		env.BlockIndexer = &blockidxnull.BlockerIndexer{}

		res, err := BlockSearch(ctx, q, &page, &perPage, orderBy)

		require.Error(t, err)
		require.Equal(t, errors.New("block indexing is disabled"), err)
		require.Nil(t, res)
	}
	{
		// error: tmquery.New(query)
		env = &Environment{}

		res, err := BlockSearch(ctx, q, &page, &perPage, orderBy)

		require.Error(t, err)
		require.Equal(t,
			"\nparse error near Unknown (line 1 symbol 1 - line 1 symbol 1):\n\"\"\n",
			err.Error())
		require.Nil(t, res)
	}
	{
		// error: switch orderBy
		env = &Environment{}
		env.BlockIndexer = blockidxkv.New(dbm.NewMemDB())
		q = fmt.Sprintf("%s>%d", types.BlockHeightKey, 1)

		res, err := BlockSearch(ctx, q, &page, &perPage, orderBy)

		require.Error(t, err)
		require.Equal(t,
			"expected order_by to be either `asc` or `desc` or empty",
			err.Error())
		require.Nil(t, res)
	}
	{
		// error: validatePage(pagePtr, perPage, totalCount)
		env = &Environment{}
		env.BlockIndexer = blockidxkv.New(dbm.NewMemDB())
		q = fmt.Sprintf("%s>%d", types.BlockHeightKey, 1)
		orderBy = TestOrderByDesc

		res, err := BlockSearch(ctx, q, &page, &perPage, orderBy)

		require.Error(t, err)
		require.Equal(t,
			"page should be within [1, 1] range, given 0",
			err.Error())
		require.Nil(t, res)
	}
}

func makeTestState() (sm.State, func()) {
	config := cfg.ResetTestRoot("rpc_core_test")
	env = &Environment{}
	env.StateStore = sm.NewStore(dbm.NewMemDB())
	env.BlockStore = store.NewBlockStore(dbm.NewMemDB())
	env.BlockIndexer = blockidxkv.New(dbm.NewMemDB())
	env.TxIndexer = txidxkv.NewTxIndex(dbm.NewMemDB())

	state, _ := env.StateStore.LoadFromDBOrGenesisFile(config.GenesisFile())
	return state, func() { os.RemoveAll(config.RootDir) }
}

func storeTestBlocks(startHeight, numToMakeBlocks, numToMakeTxs int64, state sm.State, timestamp time.Time) {
	for i := int64(0); i < numToMakeBlocks; i++ {
		commitSigs := []types.CommitSig{{
			BlockIDFlag:      types.BlockIDFlagCommit,
			ValidatorAddress: tmrand.Bytes(crypto.AddressSize),
			Timestamp:        timestamp,
			Signature:        []byte("Signature"),
		}}
		height := startHeight + i
		lastHeight := startHeight - 1
		round := int32(0)
		hash := []byte("")
		partSize := uint32(2)
		blockID := types.BlockID{Hash: hash, PartSetHeader: types.PartSetHeader{Hash: hash, Total: partSize}}
		proposer := state.Validators.SelectProposer(state.LastProofHash, startHeight, round)
		txs := make([]types.Tx, numToMakeTxs)
		for txIndex := int64(0); txIndex < numToMakeTxs; txIndex++ {
			tx := []byte{byte(height), byte(txIndex)}
			txs[txIndex] = tx
			// Indexing
			env.TxIndexer.Index(&abci.TxResult{Height: height, Index: uint32(txIndex), Tx: tx}) // nolint:errcheck
		}
		lastCommit := types.NewCommit(lastHeight, round, blockID, commitSigs)
		block, _ := state.MakeBlock(height, txs, lastCommit, nil, proposer.Address, round, nil)
		blockPart := block.MakePartSet(partSize)
		// Indexing
		env.BlockIndexer.Index(types.EventDataNewBlockHeader{Header: block.Header}) // nolint:errcheck
		// Save
		env.BlockStore.SaveBlock(block, blockPart, lastCommit)
	}
}

const (
	TestOrderByDefault = ""
	TestOrderByDesc    = "desc"
	TestOrderByAsc     = "asc"
)

type mockBlockStore struct {
	height int64
}

func (mockBlockStore) Base() int64                                       { return 1 }
func (store mockBlockStore) Height() int64                               { return store.height }
func (store mockBlockStore) Size() int64                                 { return store.height }
func (mockBlockStore) LoadBaseMeta() *types.BlockMeta                    { return nil }
func (mockBlockStore) LoadBlockMeta(height int64) *types.BlockMeta       { return nil }
func (mockBlockStore) LoadBlock(height int64) *types.Block               { return nil }
func (mockBlockStore) LoadBlockByHash(hash []byte) *types.Block          { return nil }
func (mockBlockStore) LoadBlockPart(height int64, index int) *types.Part { return nil }
func (mockBlockStore) LoadBlockCommit(height int64) *types.Commit        { return nil }
func (mockBlockStore) LoadSeenCommit(height int64) *types.Commit         { return nil }
func (mockBlockStore) PruneBlocks(height int64) (uint64, error)          { return 0, nil }
func (mockBlockStore) SaveBlock(block *types.Block, blockParts *types.PartSet, seenCommit *types.Commit) {
}
