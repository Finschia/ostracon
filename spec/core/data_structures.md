# Data Structure

Please ensure you've first read the spec for [CometBFT Data Structure](https://github.com/tendermint/tendermint/blob/v0.34.x/spec/core/data_structures.md). Here only defines the difference between CometBFT.

## Block

Ostracon extends [CometBFT Block](https://github.com/tendermint/tendermint/blob/v0.34.x/spec/core/data_structures.md#block). In Ostracon, Block consists of Entropy in addition to Header, Data, Evidence, and LastCommit.

| Name | Type | Description | Validation |
|------|------|-------------|------------|
| ...  |      | Header, Data, Evidence, and LastCommit are the same as CometBFT. |            |
| Entropy | [Entropy](#entropy) | Entropy represents height-specific complexity. This field contains infomation used proposer-election. | Must adhere to the validation rules of [entropy](#entropy) |

## Execution

`LastProofHash` is added to the state. Ostracon extends [CometBFT Execute](https://github.com/tendermint/tendermint/blob/v0.34.x/spec/core/data_structures.md#execution) as:

```go
func Execute(s State, app ABCIApp, block Block) State {
 AppHash, ValidatorChanges, ConsensusParamChanges := app.ApplyBlock(block)
 nextConsensusParams := UpdateConsensusParams(state.ConsensusParams, ConsensusParamChanges)
 return State{
  ChainID:         state.ChainID,
  InitialHeight:   state.InitialHeight,
  LastResults:     abciResponses.DeliverTxResults,
  AppHash:         AppHash,
  InitialHeight:   state.InitialHeight,
  LastValidators:  state.Validators,
  Validators:      state.NextValidators,
  NextValidators:  UpdateValidators(state.NextValidators, ValidatorChanges),
  ConsensusParams: nextConsensusParams,
  Version: {
   Consensus: {
    AppVersion: nextConsensusParams.Version.AppVersion,
   },
  },
  LastProofHash: ProofToHash(block.Entropy.Proof),
 }
}
```

Ostracon adds the following steps to the validation of a new block:

- Validate the entropy in the block.
    - Make sure the `Round` is <= current round.
    - Make sure the `ProposerAddress` corresponds to the `Round`.
    - Make sure the `Proof` corresponds to the `Round`.

## Entropy

Ostracon introduces Entropy as a new data structure. This represents height-specific complexity and is used by proposer-election. This consists of a vrf proof and round in which the proposer generates it.

| Name | Type | Description | Validation |
|------|------|-------------|------------|
| Round | int32                     | Round in which proposer generate a vrf proof             | Must be > 0 |
| Proof | slice of bytes (`[]byte`) | Proof is a vrf proof | Length of proof must be == 0, == 81 (r2ishiguro), or == 80 (libsodium) |