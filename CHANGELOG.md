# Changelog

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
