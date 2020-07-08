## v0.3

\*\*

### BREAKING CHANGES:

- State
  - [state] [\#100](https://github.com/line/tendermint/pull/100) Remove `NextVoters` from state

- CLI/RPC/Config

- Apps

- P2P Protocol
  - [abci] [\#100](https://github.com/line/tendermint/pull/100) Add `voters_hash` field, which is needed for verification of a block header
   
- Go API

- Blockchain Protocol

### FEATURES:
- [BLS] [\#81](https://github.com/line/tendermint/issues/81) Modify to generate at the same time as Ed25519 key generation
- [lite] [\#100](https://github.com/line/tendermint/pull/100) Lite calls `Genesis()` rpc when it starts up

### IMPROVEMENTS:

### BUG FIXES:
