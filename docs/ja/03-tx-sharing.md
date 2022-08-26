---
title: Transaction Sharing
---

クライアントはブロックチェーンネットワークを構成している Ostracon ノードのいずれかにトランザクションを送信することができます。トランザクションは
他の Ostracon ノードに伝搬し最終的にすべての Ostracon ノードで共有されます。

## Mempool

あるブロックが Ostracon のコンセンサス機構によって受理されたとき、そのブロックに含まれているトランザクションは *確定した* とみなされます。
未確定のトランザクションを受信した Ostracon ノードは署名などの検証を行ってブロックとは別の **mempool** という領域に保存します。

ある Ostracon ノードが mempool に保存した未確定のトランザクションは他の Ostracon ノードにもブロードキャストされます。ただし、既に受信済み
であったり不正なトランザクションの場合には保存やブロードキャストは行われずに破棄されます。このような手法は *gossipping* (または flooding)
と呼ばれ、$N$ は Ostracon ネットワークのノード数として $O(\log N)$ ホップの速度ですべてのノードに到達します。

[リーダー選出](02-consensus.md)によって Proposer に選ばれた Ostracon ノードは mempool に保存されているトランザクションから
新しい提案ブロックを生成します。以下の図は Ostracon ノードがトランザクションを受信し mempool に保存してブロック生成に使用されるまでの
流れを示しています。

![Mempool in Ostracon structure](../static/tx-sharing/mempool.png)

## Performance and Asynchronization

ブロックチェーンの性能はブロックの生成速度が注目されがちですが、現実的なブロックチェーンシステムではノード間のトランザクション共有効率も全体の
性能に大きく影響する要因です。特に Ostarcon の mempool はネットワーク浸透速度の速い gossipping を使用している対価に短時間で大量の
トランザクションを処理する必要があります。

Ostracon は mempool に関して Tendermint の実装にいくつかのキューを追加して非同期化を行っています。この改善によって短時間に大量のトランザクションを
mempool に格納することができるようになり、より現代的な CPU コア数を搭載するノード環境でブロックチェーンネットワークのスループットを改善して
います。

## Tx Validation via ABCI

ABCI (Application Blockchain Interface) はアプリケーションが Ostracon やその他のツールとリモート (gRPC, ABCI-Socket 経由) または
プロセス内 (in-process 経由) で通信するための仕様です。ABCI の詳細については [Tendermint のリポジトリ](https://github.com/tendermint/tendermint/tree/main/abci)
を参照してください。

Ostracon のトランザクション検証過程では ABCI を経由してアプリケーションレイヤーにも問い合わせを行います。この動作により (データの観点では正しいが)
本質的に不要なトランザクションをブロックに含めないようにアプリケーションが判断することができます。
