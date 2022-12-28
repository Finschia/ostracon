# Applications

Please ensure you've first read the spec for [CometBFT Applications](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/abci/apps.md). Here only defines the difference between CometBFT.

## Connection State

### Commit

Ostracon fixes [CometBFT Commit](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/abci/apps.md#commit).

In CometBFT、`Commit` assumes mempool is locked. It needs this assumption only for connection states sync. However `Commit` usually takes long time (about 500ms~1s), mempool is locked too long. Additionally, connection state sync only needs to be performed between `Commit` and rechecks. `BeginRecheckTx` and `EndRecheckTx` are added to notify the application of the start and end of the recheck so that connection states sync can be performed at the appropriate time.

The PR [#160](https://github.com/line/ostracon/pull/160) contains this change.

### BeginRecheckTx

Before `BeginRecheckTx` is called, Ostracon locks and flushes the mempool so that no new messages will be received on the mempool connection. This provides an opportunity to safely update all four connection states to the latest committed state at once.

After `EndRecheckTx` is called, it unlocks the mempool.

### Mempool Connection

Ostracon fixes [CometBFT Mempool Connection](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/abci/apps.md#mempool-connection).

Ostracon calls `Commit` without locking the mempool connection. After `Commit`, Ostracon locks the mempool.

After `BeginRecheckTx`, CheckTx is run again on all transactions that remain in the node's local mempool after filtering those included in the block.

Finally, after `EndRecheckTx`, Ostracon will unlock
the mempool connection. New transactions are once again able to be processed through CheckTx.
