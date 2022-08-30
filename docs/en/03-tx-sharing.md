---
title: Transaction Sharing
---

A client can send a transaction to any of the Ostracon nodes that joining the blockchain network. The transaction propagates to other Ostracon nodes and is ultimately shared by all Ostracon nodes.

## Mempool

Once a block is accepted by the Ostracon consensus mechanism, the transactions contained in that block are considered *confirmed*. The unconfirmed transactions are validated stored in an area called **mempool**, which is separate from the block storage, after validation such as signatures.

Unconfirmed transactions stored in the mempool by an Ostracon node are broadcast to other Ostracon nodes.
However, if the transaction has already been received or is invalid, it's neither saved nor broadcast, but discarded.
Such a method is called *gossipping* (or flooding) and a transaction will reach all nodes at a rate of $O(\log N)$ hops,
where $N$ is the number of nodes in the Ostracon network.

The Ostracon node selected as a Proposer by [leader election](02-consensus.md) generates new proposal blocks from transactions stored in the mempool.
The following figure shows the flow of an Ostracon node from receiving an unconfirmed transaction and storing it in the mempool until it's used to generate a block.

![Mempool in Ostracon structure](../static/tx-sharing/mempool.png)

## Performance and Asynchronization

Blockchain performance tends to focus on the speed of block generation, but in a practical system, the efficiency of sharing transactions among nodes is also an important factor that significantly affects overall performance.
In particular, Ostarcon's mempool must process a large number of transactions in a short period in exchange for Gossipping's network propagation speed.

Ostracon has added several queues to the Tendermint implementation for the mempool to make them asynchronous.
This change allows large numbers of transactions to be stored in the mempool in a short period of time, improving the throughput of the blockchain network in more modern CPU core-equipped node environments.

With this asynchronization of the mempool, multiple transactions will have a *validation-processing* state at the same time; Ostracon will refuse to receive transactions when the mempool capacity is exceeded, but asynchronous validation-processing transactions are also correctly included in the calculation of this capacity limit.

## Tx Validation via ABCI

ABCI (Application Blockchain Interface) is a specification for applications to communicate with Ostracon and other tools remotely (via gRPC, ABCI-Socket) or in-process (via in-process).
For more information, see [Tendermint repository](https://github.com/tendermint/tendermint/tree/main/abci).

The process of validating unconfirmed transactions also queries the application layer via ABCI. This behavior allows the application to avoid including transactions in the block that are essentially unnecessary (although correct from a data point of view). Here, Ostracon replaces the Tendermint implementation with an asynchronous API that can start the validation process for the next transaction without waiting for ABCI-side validation results. This improvement improves performance in environments where applications are allocated separate CPU cores.

