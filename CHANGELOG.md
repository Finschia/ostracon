# Changelog

## v1.0.9
*Mar 16, 2023*

* Make a breaking change to the consensus logic for tendermint compatibity
* Define the specification of Ostracon

### BREAKING CHANGE
- [consensus] [\#541](https://github.com/line/ostracon/pull/541) Remove BLS functionality from the mainstream
- [consensus] [\#543](https://github.com/line/ostracon/pull/543) Remove the voter election process
- [consensus] [\#559](https://github.com/line/ostracon/pull/559) Move VRF proof from Header to Entropy for compatibity
- [types] [\#546](https://github.com/line/ostracon/pull/546) feat!: replace a some same proto message to Tendermint

### FEATURES
- [spec] [\#565](https://github.com/line/ostracon/pull/565) Add Ostracon specification skeleton

### IMPROVEMENTS
- [types] [\#563](https://github.com/line/ostracon/pull/563) remove multiple sampling and integrate with Proposer election
- [types] [\#565](https://github.com/line/ostracon/pull/565) Add entropy correctness test
- [types] [\#575](https://github.com/line/ostracon/pull/575) Fix TODOs in [\#546](https://github.com/line/ostracon/pull/546)
- [docs] [\#560](https://github.com/line/ostracon/pull/560) Merge document corrections in LBM
- [github] [\#578](https://github.com/line/ostracon/pull/578) feat: Add codeowner
- [node] [\#576](https://github.com/line/ostracon/pull/576) feat: big genesis file

### BUG FIXES


## v1.0.8
*Dec 27, 2022*

* Update the default value of DefaultMaxTolerableByzantinePercentage
* Fix Validators of RPC implementation in Ostracon
* Add zerolog based rolling log system
* Improve many components
  * `blockchain`, `crypto`, `dependency`, `docs`, `libs`, `lint`, `mempool`, `node`, `p2p`, `privval`, `state`, `statesync`, `test`, `types`
* Fix many bugs
  * `consensus`, `crypto`, `state`, `test`, `types`

### BREAKING CHANGE
- [rpc] [\#478](https://github.com/line/ostracon/pull/478) Fix Validators of RPC implementation in Ostracon
- [types] [\#511](https://github.com/line/ostracon/pull/511) Update the default value of DefaultMaxTolerableByzantinePercentage

### FEATURES
- [libs] [\#535](https://github.com/line/ostracon/pull/535) feat: zerolog based rolling log system

### IMPROVEMENTS
- [blockchain] [\#517](https://github.com/line/ostracon/pull/517) Add `ValidateBlock`
- [crypto] [\#492](https://github.com/line/ostracon/pull/492) Use the value receiver instead of the pointer receiver in Pubkey.Identity()
- [crypto] [\#528](https://github.com/line/ostracon/pull/528) Fix to change expected of test according to build tag
- [dependency] [\#521](https://github.com/line/ostracon/pull/521) fix: replace deprecated package `io/ioutil` with `os`
- [docs] [\#491](https://github.com/line/ostracon/pull/491) fix: Update the dead links
- [libs] [\#494](https://github.com/line/ostracon/pull/494) Remove sort from proposer's selection algorithm
- [libs] [\#496](https://github.com/line/ostracon/pull/496) Add validation at the beginning of func:RandomSamplingWithPriority
- [libs] [\#506](https://github.com/line/ostracon/pull/506) Fix so that HTTP request don't wait for responses indefinitely
- [lint] [\#505](https://github.com/line/ostracon/pull/505) Upgrade golangci-lint to v1.50.1
- [mempool] [\#507](https://github.com/line/ostracon/pull/507) fix: return postCheck error to abci client
- [node] [\#508](https://github.com/line/ostracon/pull/508) Remove unsed the functioin `StateProvider`
- [p2p/conn] [\#485](https://github.com/line/ostracon/pull/485) Optimization of function signChallenge()
- [p2p/pex] [\#484](https://github.com/line/ostracon/pull/484) Avoid panic when addr does not exist in book
- [p2p/pex] [\#487](https://github.com/line/ostracon/pull/487) Add test of IsGood()
- [p2p/pex] [\#509](https://github.com/line/ostracon/pull/509) Fix code duplication
- [p2p/upnp] [\#497](https://github.com/line/ostracon/pull/497) fix: Update the http status code handling on upnp
- [p2p] [\#500](https://github.com/line/ostracon/pull/500) fix: return error when AddChannel fails
- [p2p] [\#527](https://github.com/line/ostracon/pull/527) fix: add support for dns timeout
- [privval] [\#523](https://github.com/line/ostracon/pull/523) fix: remove strange `Ping`
- [state] [\#502](https://github.com/line/ostracon/pull/502) Fix to also remove VoterParams and LastProofHash in PruneStates
- [state] [\#525](https://github.com/line/ostracon/pull/525) Align with ValidatorSet on `PruneStates()`
- [statesync] [\#515](https://github.com/line/ostracon/pull/515) Add unique handling of servers
- [test] [\#512](https://github.com/line/ostracon/pull/512) Update using `GITHUB_OUTPUT` environment
- [test] [\#518](https://github.com/line/ostracon/pull/518) fix: fix to input value to GITHUB_OUTPUT correctly
- [test] [\#522](https://github.com/line/ostracon/pull/522) fix: fix inconsistencies between the validators and voters
- [type] [\#490](https://github.com/line/ostracon/pull/490) fix: Move `types/test_util.go:MakeBlock` into `types/block.go`
- [types] [\#504](https://github.com/line/ostracon/pull/504) Fix typo of the function ValidateBasic
- [types] [\#510](https://github.com/line/ostracon/pull/510) Add validation of the ValidatorsHash, Round and Proof
- [types] [\#530](https://github.com/line/ostracon/pull/530) fix the MaxHeaderSize

### BUG FIXES
- [consensus] [\#514](https://github.com/line/ostracon/pull/514) fix: enable to join existing network with State Sync
- [consensus] [\#520](https://github.com/line/ostracon/pull/520) fix: fix total voters count
- [crypto] [\#493](https://github.com/line/ostracon/pull/493) Validate proof with ECVRF_decode_proof in vrfEd25519r2ishiguro.ProofToHash()
- [state] [\#498](https://github.com/line/ostracon/pull/498) fix: fix overriding tx index of duplicated txs
- [state] [\#526](https://github.com/line/ostracon/pull/526) Fix the bug of Ostracon's changes of [#194](https://github.com/line/ostracon/pull/194)
- [state] [\#533](https://github.com/line/ostracon/pull/533) Fix the mismatch between "State.Version.Consensus.App" and "State.ConsensusParams.Version.AppVersion"
- [test] [\#534](https://github.com/line/ostracon/pull/534) Fix the order of paremeters in require.Equalf()
- [test] [\#536](https://github.com/line/ostracon/pull/536) Backport e2e-test of the latest tendermint main branch
- [types] [\#513](https://github.com/line/ostracon/pull/513) Fix the validation and verification
- [types] [\#531](https://github.com/line/ostracon/pull/531) fix: Set maximum value for SignedMsgType

## v1.0.7

*Oct 27, 2022*

* Revert some to original Tendermint
* Improve docs

### BREAKING CHANGE
- [dependency] [\#446](https://github.com/line/ostracon/pull/446) Use tendermint/tm-db
- [amino] [\#447](https://github.com/line/ostracon/pull/447) Change PubKey/PrivKey prefixes
- [validator] [\#449](https://github.com/line/ostracon/pull/449) Swap StakingPower and VotingPower, and modify from StakingPower to VotingWeight
- [build] [\#450](https://github.com/line/ostracon/pull/450) Upgrade to Golang-1.18

### FEATURES
- Nothing

### IMPROVEMENTS
- [docs] [\#453](https://github.com/line/ostracon/pull/453) Apply docusaurus 2.0 directory structure
- [docs] [\#455](https://github.com/line/ostracon/pull/455) Change doc links to the collect ones within this site
- [docs] [\#456](https://github.com/line/ostracon/pull/456) Add topics for mepool, async behavior, ABCI, KVS, WAL to document

### BUG FIXES
- [state] [\#458](https://github.com/line/ostracon/pull/458) Fix the thread-unsafe of PeerState logging

## v1.0.6

*Jun 17, 2022*

* Improve behavior of KMS in Ostracon
* Improve supporting build environment
  * Support building/running darwin/arm64
  * Support building/running of linux/arm64 on Local/Docker via darwin/arm64
  * Stop supporting linux/arm(32bit)

### BREAKING CHANGE
- [build] [\#431](https://github.com/line/ostracon/pull/431) Stop support for linux/arm(32bit)

### FEATURES
- Nothing

### IMPROVEMENTS
- [kms] [\#417](https://github.com/line/ostracon/pull/417) Add KMS functionality
- [build] [\#426](https://github.com/line/ostracon/pull/426) Remove binary check for localnode
- [build] [\#428](https://github.com/line/ostracon/pull/428) bls-eth-go-binary version update for apple M1 chip
- [security] [\#429](https://github.com/line/ostracon/pull/429) Apply runc version 1.1.2
- [test] [\#408](https://github.com/line/ostracon/pull/408) Use Docker Buildx and Cache in e2e.yml
- [repository/config] [\#432](https://github.com/line/ostracon/pull/432) Clean up unused configuration files

### BUG FIXES
- Nothing

## v1.0.5

*May 9, 2022*

* Improve checking tx with txsMap for fixing the inconsistency between mem.txs and mem.txsMap
* Apply changes up to tendermint v0.34.19

### BREAKING CHANGE
- Nothing

### FEATURES
- Nothing

### IMPROVEMENTS
- [mempool] [\#394](https://github.com/line/ostracon/pull/394) Remove panic for unexpected tx response in resCbRecheck
- [mempool] [\#404](https://github.com/line/ostracon/pull/404) Improve checking tx with txsMap for fixing the inconsistency between mem.txs and mem.txsMap
- [upgrade/tm-db] [\#402](https://github.com/line/ostracon/pull/402) Upgrade to line/tm-db-v2.0.0-init.1.0.20220121012851-61d2bc1d9486
- [backport/tendermint] [\#368](https://github.com/line/ostracon/pull/368) Main patch tm-v0.34.15
- [backport/tendermint] [\#407](https://github.com/line/ostracon/pull/407) Revert: not to use grpc/credentials/insecure for compatibility
- [backport/tendermint] [\#375](https://github.com/line/ostracon/pull/375) Main patch tm-v0.34.16
- [backport/tendermint] [\#389](https://github.com/line/ostracon/pull/389) Main patch tm-v0.34.17
- [backport/tendermint] [\#401](https://github.com/line/ostracon/pull/401) Main patch tm-v0.34.18, tm-v0.34.19
- [github/stale] [\#377](https://github.com/line/ostracon/pull/377) Exclude auto-closing of issues in github actions
- [test] [\#403](https://github.com/line/ostracon/pull/403) Improve vrf test

### BUG FIXES
- Nothing

## v1.0.4

*Feb 25, 2022*

* Apply changes up to tendermint v0.34.14

### BREAKING CHANGE
- Nothing

### FEATURES
- Nothing

### IMPROVEMENTS
- [backport/tendermint] [\#361](https://github.com/line/ostracon/pull/361) Main patch tm-v0.34.12
- [backport/tendermint] [\#364](https://github.com/line/ostracon/pull/364) Main patch tm-v0.34.13
- [backport/tendermint] [\#366](https://github.com/line/ostracon/pull/366) Main patch tm v0.34.14

### BUG FIXES
- Nothing

## v1.0.3

*Jan 20, 2022*

* Improve p2p/peer reactor　so as not to abandon the message
* Apply changes up to tendermint v0.34.11

### BREAKING CHANGE
- Nothing

### FEATURES
- Nothing

### IMPROVEMENTS
- [p2p/peer] [\#341](https://github.com/line/ostracon/pull/341) Remove default case
- [github] [\#346](https://github.com/line/ostracon/pull/346) Add CODEOWNERS
- [backport/tendermint] [\#349](https://github.com/line/ostracon/pull/349) Main patch from tm-v0.34.9
- [lint] [\#356](https://github.com/line/ostracon/pull/356) Upgrade to super-linter-v4 for avoiding broken version
- [backport/tendermint] [\#358](https://github.com/line/ostracon/pull/358) Main patch from tm-v0.34.10
- [backport/tendermint] [\#359](https://github.com/line/ostracon/pull/359) Main patch tm-v0.34.11

### BUG FIXES
- [consensus] [\#345](https://github.com/line/ostracon/pull/345) fix: Modify omission of change to change ValidatorSet to VoterSet for marverick
- [version] [\#348](https://github.com/line/ostracon/pull/348) Fix version.go (Rollback to only use OCCoreSemVer)

## v1.0.2

*Nov 08, 2021*

* Fix bugs
* Improve crypto/composite key

### BREAKING CHANGES
- Nothing

### FEATURES
- Nothing

### IMPROVEMENTS
- [test] [\#327](https://github.com/line/ostracon/pull/327) Add libsodium test on Github Actions
- [crypto/composite] [\#335](https://github.com/line/ostracon/pull/335) Improve composite key Bytes/FromBytes and make tools
- [security] [\#336](https://github.com/line/ostracon/pull/336) Remove unused package-lock.json
- [bot] [\#337](https://github.com/line/ostracon/pull/337) Improve dependabot

### BUG FIXES
- [test] [\#338](https://github.com/line/ostracon/pull/338) bugfix: wrong binary name
- [consensus] [\#340](https://github.com/line/ostracon/pull/340) Modify omission of change to change ValidatorSet to VoterSet

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
