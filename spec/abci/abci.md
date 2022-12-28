# Methods and Types

Please ensure you've first read the spec for [CometBFT Methods and Types](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/abci/abci.md). Here only defines the difference between CometBFT.

## Connections

#### **Mempool** connection

Ostracon handles the `BeginRecheckTx` and `EndRecheckTx` calls in addition to `CheckTx`.

## Messages

### BeginBlock

Ostracon adds a entropy to the [BeginBlock](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/abci/abci.md#BeginBlock) response.

* **Request**:

    | Name                 | Type                                          | Description                                                                                                       | Field Number |
    |----------------------|-----------------------------------------------|-------------------------------------------------------------------------------------------------------------------|--------------|
    | hash                 | bytes                                         | The block's hash. This can be derived from the block header.                                                      | 1            |
    | header               | [Header](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/core/data_structures.md#header)   | The block header.                                                                                                 | 2            |
    | last_commit_info     | [LastCommitInfo](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/abci/abci.md#lastcommitinfo)             | Info about the last commit, including the round, and the list of validators and which ones signed the last block. | 3            |
    | byzantine_validators | repeated [Evidence](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/abci/abci.md#evidence)                | List of evidence of validators that acted maliciously.                                                            | 4            |
    | entropy              | [Entropy](../core/data_structures.md#entropy) | The block's entropy.                                                                                              | 1000         |

* **Response**:

    | Name   | Type                      | Description                         | Field Number |
    |--------|---------------------------|-------------------------------------|--------------|
    | events | repeated [Event](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/abci/abci.md#events) | type & Key-Value events for indexing | 1           |

* **Usage**:
    * Signals the beginning of a new block.
    * Called prior to any `DeliverTx` method calls.
    * The header contains the height, timestamp, and more - it exactly matches the
    CometBFT block header. We may seek to generalize this in the future.
    * The `LastCommitInfo` and `ByzantineValidators` can be used to determine
    rewards and punishments for the validators.
    * The `entropy` can be used to determine the next validators set.

### BeginRecheckTx

* **Request**:

    | Name   | Type                                                                                            | Description                    | Field Number |
    |--------|-------------------------------------------------------------------------------------------------|--------------------------------|--------------|
    | header | [Header](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/core/data_structures.md#header) | The block header.              | 1            |

* **Response**:

    | Name       | Type   | Description      | Field Number |
    |------------|--------|------------------|--------------|
    | code       | uint32 | Response code.   | 1            |

* **Usage**:
    * Signals the beginning of re-checking transactions.

### EndRecheckTx

* **Request**:

    | Name   | Type  | Description                    | Field Number |
    |--------|-------|--------------------------------|--------------|
    | height | int64 | Height of the executing block. | 1            |

* **Response**:

    | Name       | Type   | Description    | Field Number |
    |------------|--------|----------------|--------------|
    | code       | uint32 | Response code. | 1            |

* **Usage**:
    * Signals the end of re-checking transactions.
