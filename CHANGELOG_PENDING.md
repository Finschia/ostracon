## v0.4

Special thanks to external contributors on this release:

Friendly reminder, we have a [bug bounty program](https://hackerone.com/tendermint).

### BREAKING CHANGES

- State

- CLI/RPC/Config

- Apps

- P2P Protocol

- Go API
    - [types] [\#83](https://github.com/line/tendermint/pull/83) Add `StakingPower` to `Validator`
    - [consensus] [\#83](https://github.com/line/tendermint/pull/83) Change calculation of `VotingPower`

- Blockchain Protocol
- [consensus] [\#101](https://github.com/line/tendermint/pull/101) Introduce composite key to delegate features to each function key

### FEATURES

  - [types] [\#83](https://github.com/line/tendermint/pull/83) Add `StakingPower` to `Validator`
  - [consensus] [\#83](https://github.com/line/tendermint/pull/83) Change calculation of `VotingPower`

### FEATURES:
- [init command] [\#125](https://github.com/line/tendermint/pull/125) Add an option selecting private key type to init, testnet commands
- [consensus] [\#126](https://github.com/line/tendermint/pull/126) Add some metrics measuring duration of each consensus steps

### IMPROVEMENTS

### BUG FIXES
