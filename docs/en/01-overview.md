---
title: What is Ostracon
---

Ostracon is a fast, secure consensus layer for LINE Blockchain of the new token economy.

## Overview

Ostracon is a core-component that provides a Byzantine fault-tolerant (BFT) consensus mechanism for the LINE Blockchain ecosystem. This determines the order of transactions that are executed by applications, then generates and validates blocks which are containers of transactions.

LINE Blockchain sets out the following principles to be achieved in selecting the technology to make the consensus mechanism applicable not only to services on the internet but also to finance and industry.

1. **Security**: Completeness and soundness sufficient for practical use, based on cryptographic theory.
2. **Consistency**: A consensus algorithm with strong integrity (finality).
3. **Fault Tolerance**: Safety and liveness against system failures, including Byzantine failures.
4. **Performance and Scalability**: One block every two seconds with a capability of 1000 TPS or above.
5. **Inter-chain Connectivity**: interoperability with other blockchains besides LINE Blockchain.

P2P (Peer to peer) consensus algorithms based on BFT are more suitable than Bitcoin-like proof of work (PoW) in terms of finality and performance. Tendermint-BFT, with its modern blockchain-optimized design, was the closest implementation in our direction.

We are improving our Tendermint-BFT-based blockchain by introducing a new cryptographic technology, Verifiable Random Function (VRF), which randomly selects the node that will create new blocks. This randomness helps to prevent malicious attacks and makes it harder for participants to collude, which may happen in the future.

With the improvements, Ostracon supports the following features. Visit each page for more details.

* [Extending Tendermint-BFT with VRF-based election](02-consensus.md)
* [Transaction sharing](03-tx-sharing.md)

## Ostracon in layered structure

Ostracon includes the Consensus and Networking layers of the three layers that construct a LINE Blockchain node: Application, Consensus, and Networking.

![Layered Structure](../static/layered_structure.png)

Transactions that have not yet been incorporated into a block are shared among nodes by an anti-entropy mechanism (gossipping) in the Networking layer called [mempool](03-tx-sharing.md). Here, the Networking and Consensus layers consider transactions as simple binaries and don't care about the contents of the data.

Ostracon's consensus state and generated blocks are stored in the State DB and Block DB, respectively. Ostracon uses an embedded key-value store (KVS) based on LSMT (log-structured merge tree). These storages emphasize fast random access performance keyed by block height; in particular, the Block DB is used frequently for append operations.

> Tip: The actual KVS implementation to be used can be determined at build time from several choices.

## Specifications and technology stack

| Specifications        | Policy/Algorithms              | Methods/Implementations                         |
|:----------------------|:-------------------------------|:------------------------------------------------|
| Participation         | Permissioned                   | Consortium or Private                           |
| Election              | Proof of Stake                 | VRF-based Weighted Sampling without Replacement |
| Agreement             | Strong Consistency w/Finality  | Tendermint-BFT                                  |
| Signature             | Elliptic Curve Cryptography    | Ed25519                                         |
| Hash                  | SHA2                           | SHA-256, SHA-512                                |
| VRF                   | ECVRF-EDWARDS25519-SHA512-ELL2 | Ed25529                                         |
| Key Management        | Local KeyStore, Remote KMS     | *HSM is not support due to VRF*                 |
| Key Auth Protocol     | Station-to-Station             |                                                 |
| Tx Sharing Protocol   | Gossiping                      | mempool                                         |
| Application Protocol  | ABCI                           |                                                 |
| Interchain Protocol   | IBC (Cosmos Hub)               |                                                 |
| Storage               | Embedded KVS                   | LevelDB                                         |
| Message Recovery      | WAL                            |                                                 |
| Block Generation Time | 2 seconds                      |                                                 |

## Consideration of other consensus schemes

What consensus schemes are used by other blockchain implementations? We went through a lot of comparison and consideration to determine the direction of Ostracon.

The followings are the considerations:

- The **PoW** used by Bitcoin and Ethereum (v1.10.22 and below) is the most well-known consensus mechanism for blockchain. It has a proven track record of working as a public chain but has a structural problem of not being able to guarantee consistency until a sufficient amount of time has passed. This would cause significant problems with lost updates in the short term and the inability to scale performance in the long term. So we eliminated PoW in the early stages of our consideration.

- The consensus algorithm of Tendermint, **Tendermint-BFT**, is a well-considered design for blockchains. The ability to guarantee finality in a short period was also a good fit for our direction. On the other hand, the weighted round-robin algorithm used as the election algorithm works deterministically, so participants can know the future proposer, which makes it easy to find the target and prepare an attack. For this reason, Ostracon uses VRF to make the election unpredictable to reduce the likelihood of an attack.

- **Algorand** also uses VRF, but in a very different way than we do. At the start of an election, each node generates a VRF random number individually, and the number is used to identify the next proposal. (It's similar to all nodes tossing a coin at the same time.) This is a better way to guarantee cryptographic security while saving a large amount of computation time and power consumption compared to the PoW method of identifying the winner by hash calculation. On the other hand, applying this scheme to LINE Blockchain is difficult for several reasons. First, the number of validators to be selected is non-deterministic. And the random behavior occurs following a binomial distribution. The protocol complexity increases due to mutual recognition among the winning nodes, and it's impossible to find nodes that have been elected but have sabotaged their roles.

We have considered a number of other consensus mechanisms, but we believe that the current choice is the closest realistic choice for role election and agreement algorithms for P2P distributed systems. However, since Ostracon doesn't have a goal of experimental proofs or demonstrations for any particular research theory, we are ready to adopt it if better algorithms are proposed in the future.
