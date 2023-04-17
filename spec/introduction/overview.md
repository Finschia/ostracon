# Overview

See below for an overview of Ostracon.

- [What is Ostracon](https://github.com/Finschia/ostracon/blob/main/docs/en/01-overview.md)
- [Consensus](https://github.com/Finschia/ostracon/blob/main/docs/en/02-consensus.md)
- [Transaction Sharing](https://github.com/Finschia/ostracon/blob/main/docs/en/03-tx-sharing.md)

## Optimization

Ostracon has the following optimizations to improve performance:

- Fixed each reactor to process messages asynchronously in separate threads.
    - https://github.com/Finschia/ostracon/issues/128
    - https://github.com/Finschia/ostracon/pull/135
- Fixed some ABCI methdos to be executed concurrently.
    - https://github.com/Finschia/ostracon/pull/160
    - https://github.com/Finschia/ostracon/pull/163
