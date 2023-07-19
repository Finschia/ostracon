package v0

import (
	"github.com/Finschia/ostracon/abci/example/kvstore"
	"github.com/Finschia/ostracon/config"
	mempl "github.com/Finschia/ostracon/mempool"
	mempoolv0 "github.com/Finschia/ostracon/mempool/v0"
	"github.com/Finschia/ostracon/proxy"
)

var mempool mempl.Mempool

func init() {
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	appConnMem, _ := cc.NewABCIClient()
	err := appConnMem.Start()
	if err != nil {
		panic(err)
	}

	cfg := config.DefaultMempoolConfig()
	cfg.Broadcast = false
	mempool = mempoolv0.NewCListMempool(cfg, appConnMem, 0)
}

func Fuzz(data []byte) int {
	err := mempool.CheckTxSync(data, nil, mempl.TxInfo{})
	if err != nil {
		return 0
	}

	return 1
}
