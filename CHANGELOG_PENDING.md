## v0.3

Special thanks to external contributors on this release:

Friendly reminder, we have a [bug bounty program](https://hackerone.com/tendermint).

### BREAKING CHANGES

- State
  - [state] [\#100](https://github.com/line/tendermint/pull/100) Remove `NextVoters` from state

- CLI/RPC/Config

- Apps

- P2P Protocol
  - [abci] [\#100](https://github.com/line/tendermint/pull/100) Add `voters_hash` field, which is needed for verification of a block header
  - [abci] [\#102](https://github.com/line/tendermint/pull/102) Add voting power in `VoterInfo` of abci

- Go API
    - [types] [\#83](https://github.com/line/tendermint/pull/83) Add `StakingPower` to `Validator`
    - [consensus] [\#83](https://github.com/line/tendermint/pull/83) Change calculation of `VotingPower`
- Blockchain Protocol

### FEATURES

  - [types] [\#83](https://github.com/line/tendermint/pull/83) Add `StakingPower` to `Validator`
  - [consensus] [\#83](https://github.com/line/tendermint/pull/83) Change calculation of `VotingPower`

### FEATURES:
- [BLS] [\#81](https://github.com/line/tendermint/issues/81) Modify to generate at the same time as Ed25519 key generation
- [lite] [\#100](https://github.com/line/tendermint/pull/100) Lite calls `Genesis()` rpc when it starts up

### IMPROVEMENTS

### BUG FIXES
