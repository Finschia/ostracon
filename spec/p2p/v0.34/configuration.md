# Ostracon p2p configuration

Please ensure you've first read the spec for [CometBFT p2p configuration](https://github.com/cometbft/cometbft/blob/v0.34.x/spec/p2p/v0.34/configuration.md). Here only defines the difference between CometBFT.


Ostracon allows each reactor to process messages asynchronously in separate threads. Ostracon adds parameters related to async receiving to configurable parameters in addition to parameters defined by CometBFT.

| Parameter| Default| Description |
| --- | --- | --- |
| ... |     | The parameters defined by CometBFT |
|	RecvAsync                   |  true | Set true to enable the async receiving in a reactor                  |
|	PexRecvBufSize              |  1000 | Size of receive buffer used in async receiving of pex reactor        |
|	EvidenceRecvBufSize         |  1000 | Size of receive buffer used in async receiving of evidence reactor   |
|	MempoolRecvBufSize          |  1000 | Size of receive buffer used in async receiving of mempool reactor    |
|	ConsensusRecvBufSize        |  1000 | Size of receive buffer used in async receiving of consensus reactor  |
|	BlockchainRecvBufSize       |  1000 | Size of receive buffer used in async receiving of blockchain reactor |
|	StatesyncRecvBufSize        |  1000 | Size of receive buffer used in async receiving of statesync reactor  |
