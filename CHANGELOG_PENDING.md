## v0.3

\*\*

### BREAKING CHANGES:

- State
  - [state] [\#83](https://github.com/line/tendermint/pull/92) Add `VoterParams` to state
  - [state] [\#100](https://github.com/line/tendermint/pull/100) Remove `NextVoters` from state

- CLI/RPC/Config

- Apps

- P2P Protocol
  - [abci] [\#100](https://github.com/line/tendermint/pull/100) Add `voters_hash` field, which is needed for verification of a block header
   
- Go API

### FEATURES:
- [rpc] [\#78](https://github.com/line/tendermint/pull/78) Add `Voters` rpc
- [consensus] [\#83](https://github.com/line/tendermint/pull/83) Selection voters using random sampling without replacement
- [consensus] [\#92](https://github.com/line/tendermint/pull/92) Apply calculation of voter count
- [BLS] [\#81](https://github.com/line/tendermint/issues/81) Modify to generate at the same time as Ed25519 key generation
- [lite] [\#100](https://github.com/line/tendermint/pull/100) Lite calls `Genesis()` rpc when it starts up

### IMPROVEMENTS:

### BUG FIXES:
