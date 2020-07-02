## v0.3

Special thanks to external contributors on this release:

Friendly reminder, we have a [bug bounty program](https://hackerone.com/tendermint).

### BREAKING CHANGES

- State
  - [state] [\#83](https://github.com/line/tendermint/pull/92) Add `VoterParams` to state
  - [state] [\#100](https://github.com/line/tendermint/pull/100) Remove `NextVoters` from state

- CLI/RPC/Config

- Apps

- P2P Protocol
  - [abci] [\#100](https://github.com/line/tendermint/pull/100) Add `voters_hash` field, which is needed for verification of a block header

- Go API
    - [types] [\#83](https://github.com/line/tendermint/pull/83) Add `StakingPower` to `Validator`
    - [consensus] [\#83](https://github.com/line/tendermint/pull/83) Change calculation of `VotingPower`
- Blockchain Protocol

### FEATURES

  - [types] [\#83](https://github.com/line/tendermint/pull/83) Add `StakingPower` to `Validator`
  - [consensus] [\#83](https://github.com/line/tendermint/pull/83) Change calculation of `VotingPower`

### FEATURES:
- [rpc] [\#78](https://github.com/line/tendermint/pull/78) Add `Voters` rpc
- [types] [\#83](https://github.com/line/tendermint/pull/83) Add `StakingPower` to `Validator`
- [consensus] [\#83](https://github.com/line/tendermint/pull/83) Change calculation of `VotingPower`
- [consensus] [\#92](https://github.com/line/tendermint/pull/92) Apply calculation of voter count
- [lite] [\#100](https://github.com/line/tendermint/pull/100) Lite calls `Genesis()` rpc when it starts up

### IMPROVEMENTS

### BUG FIXES
