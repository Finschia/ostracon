package checktx

import (
	"github.com/Finschia/ostracon/abci/example/kvstore"
	"github.com/Finschia/ostracon/config"
	mempl "github.com/Finschia/ostracon/mempool"
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

	mempool = mempl.NewCListMempool(cfg, appConnMem, 0)
}

func Fuzz(data []byte) int {
	err := mempool.CheckTxSync(data, nil, mempl.TxInfo{})
	if err != nil {
		return 0
	}

	return 1
}
