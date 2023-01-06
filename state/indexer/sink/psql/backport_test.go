package psql

import (
	"context"
	"testing"

	tmabci "github.com/tendermint/tendermint/abci/types"

	"github.com/line/ostracon/libs/pubsub/query"
	"github.com/line/ostracon/state/indexer"
	"github.com/line/ostracon/state/txindex"
	"github.com/line/ostracon/types"
	"github.com/stretchr/testify/require"
)

var (
	_ indexer.BlockIndexer = BackportBlockIndexer{}
	_ txindex.TxIndexer    = BackportTxIndexer{}
)

func TestBackportTxIndexer_AddBatch(t *testing.T) {
	indexer := &EventSink{store: testDB(), chainID: chainID}
	txIndexer := indexer.TxIndexer()
	err := txIndexer.AddBatch(&txindex.Batch{})
	require.NoError(t, err)
}

func TestBackportTxIndexer_Index(t *testing.T) {
	indexer := &EventSink{store: testDB(), chainID: chainID}
	txIndexer := indexer.TxIndexer()
	err := txIndexer.Index(&tmabci.TxResult{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "finding block ID: ")

	blockIndexer := indexer.BlockIndexer()
	err = blockIndexer.Index(types.EventDataNewBlockHeader{})
	require.NoError(t, err)
	err = txIndexer.Index(&tmabci.TxResult{})
	require.NoError(t, err)
}

func TestBackportTxIndexer_Get(t *testing.T) {
	indexer := &EventSink{store: testDB(), chainID: chainID}
	txIndexer := indexer.TxIndexer()
	result, err := txIndexer.Get([]byte{1})
	require.Error(t, err)
	require.Equal(t, "the TxIndexer.Get method is not supported", err.Error())
	require.Nil(t, result)
}

func TestBackportTxIndexer_Search(t *testing.T) {
	indexer := &EventSink{store: testDB(), chainID: chainID}
	txIndexer := indexer.TxIndexer()
	result, err := txIndexer.Get([]byte{1})
	require.Error(t, err)
	require.Equal(t, "the TxIndexer.Get method is not supported", err.Error())
	require.Nil(t, result)
}

func TestBackportBlockIndexer_Has(t *testing.T) {
	indexer := &EventSink{store: testDB(), chainID: chainID}
	blockIndexer := indexer.BlockIndexer()
	result, err := blockIndexer.Has(0)
	require.Error(t, err)
	require.Equal(t, "the BlockIndexer.Has method is not supported", err.Error())
	require.False(t, result)
}

func TestBackportBlockIndexer_Index(t *testing.T) {
	indexer := &EventSink{store: testDB(), chainID: chainID}
	blockIndexer := indexer.BlockIndexer()
	err := blockIndexer.Index(types.EventDataNewBlockHeader{})
	require.NoError(t, err)
}

func TestBackportBlockIndexer_Search(t *testing.T) {
	indexer := &EventSink{store: testDB(), chainID: chainID}
	blockIndexer := indexer.BlockIndexer()
	result, err := blockIndexer.Search(context.Background(), &query.Query{})
	require.Error(t, err)
	require.Equal(t, "the BlockIndexer.Search method is not supported", err.Error())
	require.Nil(t, result)
}
