---
title: Transaction Sharing
---

クライアントアプリケーションはブロックチェーンネットワークを構成している任意の Ostracon ノードにトランザクションを送信することができます。トランザクションは他の Ostracon ノードに伝搬し最終的にすべての Ostracon ノードで共有されます。

## Mempool

未確定のトランザクションはサイズや内容などを検証した上でブロックストレージとは別の [**mempool**](https://github.com/tendermint/tendermint/blob/v0.34.x/spec/abci/apps.md#mempool-connection) と呼ばれる領域に保存されます。

> Tip: mempool のサイズには制限があります。mempool のサイズが制限に達した場合はトランザクションの保存が拒否されることがあります。

ある Ostracon ノードが mempool に保存した未確定のトランザクションは他の Ostracon ノードにもブロードキャストされます。ただし、既に受信済みであったり不正なトランザクションの場合には保存もブロードキャストもされず破棄されます。このような手法は**ゴシッピング** (またはフラッディング) と呼ばれ、$N$ を Ostracon ネットワークのノード数としたとき $O(\log N)$ ホップの速度ですべてのノードに到達します。

[リーダー選出](02-consensus.md)で Proposer に選ばれた Ostracon ノードは mempool に保存されているトランザクションから新しい提案ブロックを生成します。以下の図は Ostracon ノードが未確定のトランザクションを受信し、それを mempool に保存してからブロック生成に使用されるまでの流れを示しています。

![Mempool in Ostracon structure](../static/tx-sharing/mempool.png)

## パフォーマンスと非同期性

ブロックチェーンの性能はブロック生成の速度が注目されがちですが、現実的なシステムではノード間のトランザクション共有効率も全体の性能に大きく影響する重要な要因です。ゴシッピングの高速なネットワーク伝搬のため、Ostracon の mempool は特に短時間で大量のトランザクションを処理する必要があります。このため Ostracon は Tendermint の **Reactor** 実装にいくつかのキューを追加し、トランザクションを含むすべての P2P メッセージの処理を非同期で行うように変更しています。この非同期化により現代的な CPU コアを搭載するノードでのトランザクション共有はより短時間により多くのトランザクションを処理できるようになりネットワークのスループットを改善しています。

この mempool の非同期化により複数のトランザクションが同時に**検証中**の状態を持つようになります。Ostracon は非同期で検証中のトランザクションも容量制限の算出に正しく含まれます。したがって mempool の容量を超過したトランザクションの受信を正しく拒否します。

## ABCI によるトランザクション検証

ABCI (Application Blockchain Interface) はアプリケーションが Ostracon やその他のツールとリモート (gRPC, ABCI-Socket 経由) またはプロセス内 (in-process 経由) で通信するための仕様です。ABCI の詳細については [Tendermint 仕様](https://github.com/tendermint/tendermint/blob/v0.34.x/spec/abci)を参照してください。

未確定トランザクションの検証過程では ABCI 経由でアプリケーションにも問い合わせを行います。この動作により (データの観点では正しいが) 本質的に不要なトランザクションをブロックに含めないようにアプリケーションが判断することができます。Ostracon ではこのための [`CheckTx` リクエスト](https://github.com/tendermint/tendermint/blob/v0.34.x/spec/abci/abci.md#mempool-connection)を非同期化する変更を行い ABCI 側の検証結果を待つことなく次のトランザクションの検証処理を開始できるようにしています。これによりアプリケーションに個別の CPU コアが割り当てられているような環境でのパフォーマンスが向上します。

一方この非同期化の副作用として、アプリケーションが 1 つの ABCI リクエストを処理している間に別の `CheckTx` リクエストを受信するようになります。例えば [Finschia SDK](https://github.com/Finschia/finschia-sdk) の ABCI アプリケーションインターフェース ([BaseApp](https://github.com/Finschia/lbm-sdk/blob/main/baseapp/baseapp.go)) が内部で保持するチェック状態はこの並行実行を適切に排他制御を行う必要があります。このようなロックスコープをアプリケーションレイヤーで適切に設定できるように、Ostracon の ABCI は `RecheckTx` の開始と終了時を通知する API を追加しています。
