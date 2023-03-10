# Ostracon Spec

This document is a specification of the Ostracon. Ostracon is designed to have many compatibilities with CometBFT. As such, the Ostracon spec is based on the [CometBFT v0.34.x Spec](https://github.com/cometbft/cometbft/tree/v0.34.x/spec). The Ostracon also has many unique improvements to improve security and performance. This specification clarifies the differences between CometBFT and defines the base data structures, how they are validated, and how they are communicated over the network.

## Contents

> **Note**
> ![oc](./static/oc.svg) represents items with Ostracon-specific improvements added. It also describes the major changes.

> **Note**
> ![tm](./static/tm.svg) represents items with specifications equivalent to CometBFT. As such, the article links directly to CometBFT's repository.

### Introduction

- [![oc](./static/oc.svg)Overview](./introduction/overview.md)

### Core

- [![oc](./static/oc.svg)Data Structure](./core/data_structures.md): Ostacon uses a more secure proposer election algorithm. Add definitions for the data structures required for this election algorithm.
- [![tm](./static/tm.svg)Encoding](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/core/encoding.md)
- [![tm](./static/tm.svg)Genesis](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/core/genesis.md)
- [![oc](./static/oc.svg)State](./core/state.md): Ostacon uses a more secure proposer election algorithm. Add the data required for this election algorithm to the state.

### Consensus Protocol

- [![tm](./static/tm.svg)Consensus Algorithm](https://github.com/cometbft/cometbft/blob/v0.34.x/spec//consensus/consensus.md)
- [![tm](./static/tm.svg)BFT Time](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/consensus/bft-time.md)
- [![oc](./static/oc.svg)Proposer Selection](./consensus/proposer-selection.md): Ostacon adopts a proposer selection algorithm using VRF. Using VRF makes the election unpredictable and makes the Proposer election more secure.
- [![tm](./static/tm.svg)Creating a proposal](https://github.com/cometbft/cometbft/blob/v0.34.x/spec//consensus/creating-proposal.md)
- [![tm](./static/tm.svg)Siging](https://github.com/cometbft/cometbft/blob/v0.34.x/spec//consensus/signing.md)
- [![tm](./static/tm.svg)Write-Ahead Log](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/consensus/wal.md)

#### Proposer-Based Timestamps

- [![tm](./static/tm.svg)Overview](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/consensus/proposer-based-timestamp/pbts_001_draft.md)
- [![tm](./static/tm.svg)System Model and Properties](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/consensus/proposer-based-timestamp/pbts-sysmodel_001_draft.md)
- [![tm](./static/tm.svg)Protocol Specification](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/consensus/proposer-based-timestamp/pbts-algorithm_001_draft.md)

### Light-Client

- [![tm](./static/tm.svg)Spec](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/light-client/README.md)

### P2P and Network Protocols

- [![tm](./static/tm.svg)Node](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/p2p/node.md)
- [![tm](./static/tm.svg)Peer](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/p2p/peer.md)
- [![tm](./static/tm.svg)Connection](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/p2p/connection.md)
- [![tm](./static/tm.svg)Config](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/p2p/config.md)
#### Messages Type

- [![tm](./static/tm.svg)Block Sync](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/p2p/messages/block-sync.md)
- [![tm](./static/tm.svg)Mempool](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/p2p/messages/mempool.md)
- [![tm](./static/tm.svg)Evidence](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/p2p/messages/evidence.md)
- [![tm](./static/tm.svg)State Sync](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/p2p/messages/state-sync.md)
- [![tm](./static/tm.svg)Peer Exchange (PEX)](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/p2p/messages/pex.md)
- [![tm](./static/tm.svg)Consensus](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/p2p/messages/consensus.md)

#### v0.34

- [![tm](./static/tm.svg)Transport](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/p2p/v0.34/transport.md)
- [![tm](./static/tm.svg)Switch](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/p2p/v0.34/switch.md)
- [![tm](./static/tm.svg)PEX Reactor](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/p2p/v0.34/pex.md)
- [![tm](./static/tm.svg)Peer Exchange protocol](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/p2p/v0.34/pex-protocol.md)
- [![tm](./static/tm.svg)Address Book](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/p2p/v0.34/addressbook.md)
- [![tm](./static/tm.svg)Peer Manager](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/p2p/v0.34/peer_manager.md)
- [![tm](./static/tm.svg)Type](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/p2p/v0.34/types.md)
- [![oc](./static/oc.svg)Configuration](./p2p/v0.34/configuration.md): Ostracon allows each reactor to process messages asynchronously in separate threads. Ostracon adds parameters related to that.

### RPC

- [![oc](./static/oc.svg)Spec](./rpc/README.md): Add a entropy to the Block and BlockByHash API response.

### ABCI

- [![tm](./static/tm.svg)Overview](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/abci/README.md)
- [![oc](./static/oc.svg)Methods and Types](./abci/abci.md): Ostracon adds new ABCI Methods.
- [![oc](./static/oc.svg)Applications](./abci/apps.md): Ostracon improves the mempool connection cycle to reduce the time the mempool is locked.
- [![tm](./static/tm.svg)Client and Server](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/abci/client-server.md)
