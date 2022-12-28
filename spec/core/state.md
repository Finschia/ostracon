# State

Please ensure you've first read the spec for [CometBFT State](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/core/state.md). Here only defines the difference between CometBFT.

Ostracon adds `LastProofHash` to state. `LastProofHash` is used by proposer selection in the current height.

```go
type State struct {
    ChainID        string
    InitialHeight  int64

    LastBlockHeight int64
    LastBlockID     types.BlockID
    LastBlockTime   time.Time

    Version     Version
    LastResults []Result
    AppHash     []byte

    LastValidators ValidatorSet
    Validators     ValidatorSet
    NextValidators ValidatorSet

    ConsensusParams ConsensusParams

    LastProofHash []byte
}
```
