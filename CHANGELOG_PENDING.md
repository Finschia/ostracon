# Unreleased Changes

## v0.2

Special thanks to external contributors on this release:

Friendly reminder, we have a [bug bounty program](https://hackerone.com/tendermint).

### BREAKING CHANGES

- State
  - [state] [\#92](https://github.com/line/tendermint/pull/92) Genesis state
  
- CLI/RPC/Config

- Apps

- P2P Protocol

- Go API
    - [types] [\#83](https://github.com/line/tendermint/pull/83) Add `StakingPower` to `Validator`
    - [consensus] [\#83](https://github.com/line/tendermint/pull/83) Change calculation of `VotingPower`
- Blockchain Protocol

### FEATURES
- [rpc] [\#78](https://github.com/line/tendermint/pull/78) Add `Voters` rpc
- [consensus] [\#83](https://github.com/line/tendermint/pull/83) Selection voters using random sampling without replacement
- [consensus] [\#92](https://github.com/line/tendermint/pull/92) Apply calculation of voter count

### IMPROVEMENTS

### BUG FIXES

### BUG FIXES

- [circleCI] [\#76](https://github.com/line/tendermint/pull/76) Fix contract test job of circleCI
