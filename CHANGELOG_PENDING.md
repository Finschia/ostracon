## v0.4

\*\*

### BREAKING CHANGES:

- State

- CLI/RPC/Config

- Apps

- P2P Protocol

- Go API

- Blockchain Protocol
- [consensus] [\#101](https://github.com/line/tendermint/pull/101) Introduce composite key to delegate features to each function key
- [consensus] [\#117](https://github.com/line/tendermint/pull/117) BLS Signature Aggregation and Verification

### FEATURES:
- [init command] [\#125](https://github.com/line/tendermint/pull/125) Add an option selecting private key type to init, testnet commands
- [consensus] [\#126](https://github.com/line/tendermint/pull/126) Add some metrics measuring duration of each consensus steps

### IMPROVEMENTS:
- [p2p] [\#135](https://github.com/line/tendermint/pull/135) Add async mode for reactors
- [encoding/decoding] [\#159](https://github.com/line/tendermint/pull/159) Extend the maximum number of characters that can be decoded to 200 characters

### BUG FIXES:

- [consensus] [\#4895](https://github.com/tendermint/tendermint/pull/4895) Cache the address of the validator to reduce querying a remote KMS (@joe-bowman)
- [privval] \#5638 Increase read/write timeout to 5s and calculate ping interval based on it (@JoeKash)
