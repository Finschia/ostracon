package psql

import (
	"github.com/line/ostracon/state/indexer"
	"github.com/line/ostracon/state/txindex"
)

var (
	_ indexer.BlockIndexer = BackportBlockIndexer{}
	_ txindex.TxIndexer    = BackportTxIndexer{}
)
