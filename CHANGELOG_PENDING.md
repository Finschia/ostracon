## v0.2

\*\*

### BREAKING CHANGES:

- CLI/RPC/Config

- Apps

- P2P Protocol

- Go API

  - [types] [\#83](https://github.com/line/tendermint/pull/83) Add `StakingPower` to `Validator`
  - [consensus] [\#83](https://github.com/line/tendermint/pull/83) Change calculation of `VotingPower`

### FEATURES:
- [rpc] [\#78](https://github.com/line/tendermint/pull/78) Add `Voters` rpc
- [consensus] [\#83](https://github.com/line/tendermint/pull/83) Selection voters using random sampling without replacement
- [BLS] [\#81](https://github.com/line/tendermint/issues/81) Modify to generate at the same time as Ed25519 key generation

### IMPROVEMENTS:

### BUG FIXES:
- [circleCI] [\#76](https://github.com/line/tendermint/pull/76) Fix contract test job of circleCI
