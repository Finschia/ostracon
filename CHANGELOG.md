# Changelog

## v0.2
* Changed from the consensus way which the entire validator agrees to a part of the validators is elected as a voter to consensus.
The selected validator is called `voter`
* Base Tendermint version is v0.33.4. please see the [CHANGELOGS](./CHANGELOG_OF_TENDERMINT.md#v0.33.4) of the Tendermint.

### BREAKING CHANGES:

- State
  - [state] [\#92](https://github.com/line/tendermint/pull/92) Add `VoterParams` to Genesis state

- Go API
  - [types] [\#83](https://github.com/line/tendermint/pull/83) Add `StakingPower` to `Validator`
  - [consensus] [\#83](https://github.com/line/tendermint/pull/83) Change calculation of `VotingPower`

### FEATURES:
- [rpc] [\#78](https://github.com/line/tendermint/pull/78) Add `Voters` rpc
- [consensus] [\#83](https://github.com/line/tendermint/pull/83) Selection voters using random sampling without replacement
- [consensus] [\#92](https://github.com/line/tendermint/pull/92) Apply calculation of voter count

### BUG FIXES:
- [circleCI] [\#76](https://github.com/line/tendermint/pull/76) Fix contract test job of circleCI



## v0.1
Base Tendermint v0.33.3. please see the [CHANGELOG](./CHANGELOG_OF_TENDERMINT.md#v0.33.3)

### BREAKING CHANGES:
- Blockchain Protocol
  - [state] [\#7](https://github.com/line/tendermint/issues/7) Add round, proof in block

### FEATURES:
- [types] [\#40](https://github.com/line/tendermint/issues/40) Add vrf interface and add a function generating vrf proof to PrivValidator
- [lib/rand] [\#43](https://github.com/line/tendermint/issues/43) Implementation of selection algorithms using categorical distributions
- [state] [\#44](https://github.com/line/tendermint/issues/44) Add genesis seed for electing proposer of first block
- [types] [\#48](https://github.com/line/tendermint/issues/48) Replace Tendermint's PoS to VRF-based Random Sampling
