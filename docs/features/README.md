# Ostracon: A Fast, Secure Consensus Layer for The Blockchain of New Token Economy

Version 1.0 :: [日本語](README_ja.md)

## Ostracon Overview

Ostracon is a core-component that provides a Byzantine fault-tolerant (BFT) consensus mechanism for the LINE Blockchain ecosystem. This determines the order of transactions that are executed by applications, then generates and verifies blocks which are containers of transactions.

LINE Blockchain sets out a number of principles to be archived in selecting the technology in order to make the consensus mechanism applicable not only to services on the internet, but also to finance and industry.

**Security**: Completeness and soundness sufficient for practical use, based on cryptographic theory.
**Consistency**: A consensus algorithm with strong integrity (finality).
**Fault-Tolerance**: Safety and liveness against system failures, including Byzantine failures.
**Performance and Scalability**: One block every two seconds with a capability of 1000 TPS or above.
**Inter-chain Connectivity**: interoperability with other blockchains besides LINE Blockchain.

P2P consensus algorithms based on BFT are more suitable than Bitcoin-like proof of work (PoW) in terms of functionality and performance. Among them, Tendermint-BFT, with its modern blockchain-optimized design, was the closest implementation in our direction (and even better, it can be connected to Cosmos Hub).

We are introducing two new cryptographic technologies with Tendermint-BFT to further improve our blockchain. One is Verifiable Random Function (VRF), which was introduced to randomly select the Proposer node that will generate blocks and makes future selection unpredictable. This randomness is expected to deter malicious attacks and make it difficult for participants to act in collusion at some point in the future.

Another feature is the  Boneh–Lynn–Shacham (BLS) signature. BLS signature schemes, which are based on bilinear mapping, gives us the ability to aggregate multiple digital signatures into a single one. In many blockchain protocols, large amounts of signatures must be stored to approve a block. Enabling BLS signature aggregation reduces the footprint and can significantly improve communication overhead and storage consumption.

## Layered Structure

Ostracon includes the Consensus and Networking layers of the three layers that construct a LINE BLockchain node: Application, Consensus, and Networking.

![Layered Structure](layered_structure.png)

Transactions that have not yet been incorporated into a block are shared among nodes by an anti-entropy mechanism (gossipping) in the Network layer called mempool. Here, the Network and Consensus layers consider transactions as simple binaries and don't care about the contents of the data.

## Specifications and Technology Stack

| Specifications        | Policy / Algorithms           | Methods / Implementations                                    |
| :-------------------- | :---------------------------- | :----------------------------------------------------------- |
| Participation         | Permissioned                  | Consortium or Private                                        |
| Election              | Proof of Stake                | VRF-based Weighted Sampling without Replacement + SplitMix64 |
| Agreement             | Strong Consistency w/Finality | Tendermint-BFT                                               |
| Signature             | Elliptic Curve Cryptography   | Ed25519, *BLS12-381*<sup>*1</sup>                            |
| Hash                  | SHA2                          | SHA-256, SHA-512                                             |
| HSM                   | *N/A*                         | *No support for VRF or signature aggregation*                |
| Key Auth Protocol     | Station-to-Station            |                                                              |
| Tx Sharing Protocol   | Gossiping                     | mempool                                                      |
| Application Protocol  | ABCI                          |                                                              |
| Interchain Protocol   | IBC (Cosmos Hub)              |                                                              |
| Storage               | Embedded KVS                  | LevelDB                                                      |
| Message Recovery      | WAL                           |                                                              |
| Block Generation Time | 2 seconds                     |                                                              |

<sup>*1</sup> experimental implementation.

## Ostracon Features

* [Extending Tendermint-BFT with VRF-based Election](consensus/index.md)
* [BLS Signature Aggregation](signature-aggregation/index.md)

## Consideration with Other Consensus Schemes

What consensus schemes are used by other blockchain implementations? We went through a lot of comparison and consideration to determine the direction of Ostracon.

The **PoW** used by Bitcoin and Ethereum is the most well-known consensus mechanism for blockchain. It has a proven track record of working as a public chain but has a structural problem of not being able to guarantee consistency until a sufficient amount of time has passed. This would cause significant problems with lost updates in the short term, and the inability to scale performance in the long term. So we eliminated PoW in the early stages of our consideration.

The consensus algorithm of Tendermint, **Tendermint-BFT**, is a well-considered design for blockchains. The ability to guarantee finality in a short period of time was also a good fit for our direction. On the other hand, the weighted round-robin algorithm used as the election algorithm works deterministically, so participants can know the future Proposer, which makes it easy to find the target and prepare an attack. For this reason, Ostracon uses VRF to make the election unpredictable in order to reduce the likelihood of an attack.

**Algorand** also uses VRF, but in a very different way than we do: at the start of an election, each node generates a VRF random number individually and identifies whether it's a winner of the next Validator or not (it's similar to all nodes tossing a coin at the same time). This is a better way to guarantee cryptographic security while saving a large amount of computation time and power consumption compared to the PoW method of identifying the winner by hash calculation. On the other hand, it's difficult to apply this scheme to our blockchain for several reasons: the number of Validators to be selected is non-deterministic and includes random behavior following a binomial distribution, the protocol complexity increases due to mutual recognition among the winning nodes, and it's impossible to find nodes that have been elected but have sabotaged their roles.

We have considered a number of other consensus mechanisms, but we believe that the current choice is the closest realistic choice for role election and agreement algorithms for P2P distributed systems. However, since Ostracon doesn't have a goal of experimental proofs or demonstrations for any particular research theory, we are ready to adopt better algorithms if they are proposed in the future.
