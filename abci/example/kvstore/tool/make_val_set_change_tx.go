package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"

	"github.com/line/ostracon/abci/example/kvstore"
	"github.com/line/ostracon/config"
	"github.com/line/ostracon/crypto/encoding"
)

func main() {
	c := config.DefaultConfig()
	c.SetRoot(os.Getenv("HOME") + "/" + config.DefaultOstraconDir)
	keyFilePath := c.PrivValidatorKeyFile()
	var flagKeyFilePath = flag.String("priv-key", keyFilePath, "priv val key file path")
	var flagStakingPower = flag.Int64("staking", 10, "staking power for priv valedator")
	flag.Parse()
	keyFile, err := kvstore.LoadPrivValidatorKeyFile(*flagKeyFilePath)
	if err != nil {
		panic(err)
	}
	publicKey, err := encoding.PubKeyToProto(keyFile.PubKey)
	if err != nil {
		panic(err)
	}
	pubStr, tx := kvstore.MakeValSetChangeTxAndMore(publicKey, *flagStakingPower)
	{
		fmt.Println("\n# Send tx of ValSetChangeTx for persist_kvstore")
		fmt.Println("# See: persist_kvstore.go#DeliveredTx")
		broadcastTxCommit := fmt.Sprintf("curl -s 'localhost:26657/broadcast_tx_commit?tx=\"%s\"'",
			url.QueryEscape(tx))
		fmt.Println(broadcastTxCommit)
	}
	{
		fmt.Println("\n# Query tx of ValSetChangeTx for persist_kvstore")
		fmt.Println("# See: persist_kvstore.go#Query")
		query := fmt.Sprintf("curl -s 'localhost:26657/abci_query?path=\"%s\"&data=\"%s\"'",
			url.QueryEscape("/val"),
			url.QueryEscape(pubStr))
		fmt.Println(query)
	}
}
