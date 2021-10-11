# Changelog

## v1.0.1

*Sep 30, 2021*

* Improved performances
* Improved interfaces for abci/light client
* Add max txs per block
* Make documents for VRF/BLS
* Fixed test environments

### BREAKING CHANGES
- Nothing

### FEATURES
- [performance] [\#287](https://github.com/line/ostracon/pull/287) perf: improve performance and modify some abci
- [abci] [\#312](https://github.com/line/ostracon/pull/312) Add VotingPower to abci.Evidence
- [light] [\#313](https://github.com/line/ostracon/pull/313) fix: modify verifying interface for integrating lfb
- [mempool] [\#317](https://github.com/line/ostracon/pull/317) feat: added max txs per block to config.toml
- [logging] [\#324](https://github.com/line/ostracon/pull/324) chore: added extra timing info regarding block generation
- [docs] [\#294](https://github.com/line/ostracon/pull/294) doc: [ja] Add ostracon-specific VRF+BLS feature documents
- [docs] [\#304](https://github.com/line/ostracon/pull/304) doc: [en] Add ostracon-specific VRF+BLS feature documents

### BUG FIXES
- [test] [\#290](https://github.com/line/ostracon/pull/290) Fix broken Github Actions environments of main branch
- [test] [\#301](https://github.com/line/ostracon/pull/301) Enable maverick node for e2e test
- [test] [\#297](https://github.com/line/ostracon/pull/297) Support for VRF implementation with libsodium
- [test] [\#303](https://github.com/line/ostracon/pull/303) Update libsodium impl and add benchmark test
- [test] [\#307](https://github.com/line/ostracon/pull/307) Remove t.Skip in testcases
- [test] [\#315](https://github.com/line/ostracon/pull/315) Support arm64 and arm
- [test] [\#319](https://github.com/line/ostracon/pull/319) Fix the test case that often fails

## v1.0.0

*Jun 29, 2021*

* Changed from the consensus way which the entire validator agrees to a part of the validators is elected as a voter to
  consensus. The selected validator is called `voter`
* The voter to be elected has been changed so that it can be determined in the n-1 block from the one determined by
  the n-2 block.
* A BLS signature library was added. The ability to use the BLS signature library will be added in the next version.
* When distributing rewards in Cosmos-SDK, some ABCIs have been modified to distribute the voting power of elected
  voters.

### BREAKING CHANGES
- State
  - [state] [\#92](https://github.com/line/ostracon/pull/92) Add `VoterParams` to Genesis state
  - [state] [\#100](https://github.com/line/ostracon/pull/100) Remove `NextVoters` from state
- Go API
  - [types] [\#83](https://github.com/line/ostracon/pull/83) Add `StakingPower` to `Validator`
  - [consensus] [\#83](https://github.com/line/ostracon/pull/83) Change calculation of `VotingPower`
- Blockchain Protocol
  - [state] [\#7](https://github.com/line/ostracon/issues/7) Add round, proof in block
- P2P Protocol
  - [abci] [\#100](https://github.com/line/ostracon/pull/100) Add `voters_hash` field, which is needed for verification of a block header
  - [abci] [\#102](https://github.com/line/ostracon/pull/102) Add voting power in `VoterInfo` of abci

### FEATURES
- [types] [\#40](https://github.com/line/ostracon/issues/40) Add vrf interface and add a function generating vrf proof to PrivValidator
- [lib/rand] [\#43](https://github.com/line/ostracon/issues/43) Implementation of selection algorithms using categorical distributions
- [state] [\#44](https://github.com/line/ostracon/issues/44) Add genesis seed for electing proposer of first block
- [types] [\#48](https://github.com/line/ostracon/issues/48) Replace tendermint's PoS to VRF-based Random Sampling
- [rpc] [\#78](https://github.com/line/ostracon/pull/78) Add `Voters` rpc
- [consensus] [\#83](https://github.com/line/ostracon/pull/83) Selection voters using random sampling without replacement
- [consensus] [\#92](https://github.com/line/ostracon/pull/92) Apply calculation of voter count
- [BLS] [\#81](https://github.com/line/ostracon/issues/81) Modify to generate at the same time as Ed25519 key generation
- [lite] [\#100](https://github.com/line/ostracon/pull/100) Lite calls `Genesis()` rpc when it starts up

### BUG FIXES
- [circleCI] [\#76](https://github.com/line/ostracon/pull/76) Fix contract test job of circleCI

## v0.0.0

*Mar 15, 2021*

This release rewrite to ostracon.

## PreHistory
Initial ostracon is based on the tendermint v0.34.8

## [tendermint v0.34.8] - 2021-02-25

* (tendermint) [v0.34.8](https://github.com/tendermint/tendermint/releases/tag/v0.34.8).

Please refer [CHANGELOG_OF_TENDERMINT_v0.34.8](https://github.com/tendermint/tendermint/blob/v0.34.8/CHANGELOG.md)
<!-- Release links -->
