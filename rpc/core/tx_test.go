package core

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"testing"
	"time"

	dbm "github.com/tendermint/tm-db"

	txidxkv "github.com/Finschia/ostracon/state/txindex/kv"
	txidxnull "github.com/Finschia/ostracon/state/txindex/null"
	"github.com/stretchr/testify/require"

	rpctypes "github.com/Finschia/ostracon/rpc/jsonrpc/types"
	"github.com/Finschia/ostracon/types"
)

func TestTxSearchByTxHashQuery(t *testing.T) {
	height := int64(1)
	txIndex := 0
	tx := []byte{byte(height), byte(txIndex)}
	hash := hex.EncodeToString(types.Tx(tx).Hash())
	ctx := &rpctypes.Context{}

	q := fmt.Sprintf("%s='%s'", types.TxHashKey, hash)
	prove := false
	page := 1
	perPage := 10
	orderBy := TestOrderByDefault

	state, cleanup := makeTestState()
	defer cleanup()

	{
		// Get by tx.hash (not search/range)
		res, err := TxSearch(ctx, q, prove, &page, &perPage, orderBy)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, 0, res.TotalCount) // Don't have tx in db
		require.Equal(t, 0, len(res.Txs))
	}

	numToMakeBlocks := 1
	numToMakeTxs := 1
	numOfGet := 1
	// SaveBlock
	storeTestBlocks(height, int64(numToMakeBlocks), int64(numToMakeTxs), state, time.Now())

	{
		// Get by block.height (not search/range)
		res, err := TxSearch(ctx, q, prove, &page, &perPage, orderBy)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, numToMakeTxs, res.TotalCount) // Get
		require.Equal(t, numOfGet, len(res.Txs))
	}
}

func TestTxSearchByTxHeightQuery(t *testing.T) {
	height := int64(1)
	ctx := &rpctypes.Context{}

	q := fmt.Sprintf("%s>=%d", types.TxHeightKey, height)
	prove := false
	page := 1
	perPage := 10
	orderBy := TestOrderByDefault

	state, cleanup := makeTestState()
	defer cleanup()

	numToMakeBlocks := 5
	numToMakeTxs := 3
	numToGet := perPage
	// SaveBlock
	storeTestBlocks(height, int64(numToMakeBlocks), int64(numToMakeTxs), state, time.Now())

	{
		// Search blocks by range query with asc (default)
		res, err := TxSearch(ctx, q, prove, &page, &perPage, orderBy)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, numToMakeBlocks*numToMakeTxs, res.TotalCount)
		require.Equal(t, numToGet, len(res.Txs))
		// check first tx
		first := res.Txs[0]
		require.Equal(t, height, first.Height)
		require.Equal(t, uint32(0), first.Index)
		// check last tx
		last := res.Txs[numToGet-1]
		require.Equal(t, int64(math.Ceil(float64(numToGet)/float64(numToMakeTxs))), last.Height)
		require.Equal(t, uint32(numToGet%numToMakeTxs-1), last.Index)
	}
	{
		orderBy = TestOrderByDesc
		// Search blocks by range query with desc
		res, err := TxSearch(ctx, q, prove, &page, &perPage, orderBy)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, numToMakeBlocks*numToMakeTxs, res.TotalCount)
		require.Equal(t, numToGet, len(res.Txs))
		// check first tx
		first := res.Txs[0]
		require.Equal(t, int64(numToMakeBlocks), first.Height)
		require.Equal(t, uint32(numToMakeTxs-1), first.Index)
		// check last tx
		last := res.Txs[numToGet-1]
		require.Equal(t, int64(numToMakeBlocks-numToGet/numToMakeTxs), last.Height)
		require.Equal(t, uint32(numToMakeTxs-numToGet%numToMakeTxs), last.Index)
	}
	{
		// Range queries: how to use: see query_test.go
		q = fmt.Sprintf("%s>=%d AND %s<=%d", types.TxHeightKey, height, types.TxHeightKey, height+1)
		orderBy = TestOrderByAsc
		// Search blocks by range query with asc
		res, err := TxSearch(ctx, q, prove, &page, &perPage, orderBy)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, numToMakeTxs*2, res.TotalCount)
		require.Equal(t, numToMakeTxs*2, len(res.Txs))
		// check first tx
		first := res.Txs[0]
		require.Equal(t, height, first.Height)
		require.Equal(t, uint32(0), first.Index)
		// check last tx
		last := res.Txs[len(res.Txs)-1]
		require.Equal(t, height+1, last.Height)
		require.Equal(t, uint32(numToMakeTxs-1), last.Index)
	}
	{
		// Range queries with illegal key
		q = fmt.Sprintf("%s>=%d AND %s<=%d AND test.key>=1",
			types.TxHeightKey, height, types.TxHeightKey, height+1)
		orderBy = TestOrderByAsc
		// Search blocks by range query with asc
		res, err := TxSearch(ctx, q, prove, &page, &perPage, orderBy)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, 0, res.TotalCount) // Cannot Get
		require.Equal(t, 0, len(res.Txs))
	}
}

func TestTxSearch_errors(t *testing.T) {
	ctx := &rpctypes.Context{}

	q := ""
	prove := false
	page := 0
	perPage := 1
	orderBy := "error"

	{
		// error: env.TxIndexer.(*txidxnull.TxIndex)
		env = &Environment{}
		env.TxIndexer = &txidxnull.TxIndex{}

		res, err := TxSearch(ctx, q, prove, &page, &perPage, orderBy)

		require.Error(t, err)
		require.Equal(t, errors.New("transaction indexing is disabled"), err)
		require.Nil(t, res)
	}
	{
		// error: tmquery.New(query)
		env = &Environment{}

		res, err := TxSearch(ctx, q, prove, &page, &perPage, orderBy)

		require.Error(t, err)
		require.Equal(t,
			"\nparse error near Unknown (line 1 symbol 1 - line 1 symbol 1):\n\"\"\n",
			err.Error())
		require.Nil(t, res)
	}
	{
		// error: lookForHash
		env = &Environment{}
		env.TxIndexer = txidxkv.NewTxIndex(dbm.NewMemDB())
		q = fmt.Sprintf("%s=%s", types.TxHashKey, "'1'")

		res, err := TxSearch(ctx, q, prove, &page, &perPage, orderBy)

		require.Error(t, err)
		require.Equal(t,
			"error during searching for a hash in the query: encoding/hex: odd length hex string",
			err.Error())
		require.Nil(t, res)
	}
	{
		// error: switch orderBy
		env = &Environment{}
		env.TxIndexer = txidxkv.NewTxIndex(dbm.NewMemDB())
		q = fmt.Sprintf("%s=%s", types.TxHashKey, "'1234567890abcdef'")

		res, err := TxSearch(ctx, q, prove, &page, &perPage, orderBy)

		require.Error(t, err)
		require.Equal(t,
			"expected order_by to be either `asc` or `desc` or empty",
			err.Error())
		require.Nil(t, res)
	}
	{
		// error: validatePage(pagePtr, perPage, totalCount)
		env = &Environment{}
		env.TxIndexer = txidxkv.NewTxIndex(dbm.NewMemDB())
		q = fmt.Sprintf("%s=%s", types.TxHashKey, "'1234567890abcdef'")
		orderBy = TestOrderByAsc

		res, err := TxSearch(ctx, q, prove, &page, &perPage, orderBy)

		require.Error(t, err)
		require.Equal(t,
			"page should be within [1, 1] range, given 0",
			err.Error())
		require.Nil(t, res)
	}
}
