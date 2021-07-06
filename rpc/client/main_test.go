package client_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/line/ostracon/abci/example/kvstore"
	nm "github.com/line/ostracon/node"
	rpctest "github.com/line/ostracon/rpc/test"
)

var node *nm.Node

func TestMain(m *testing.M) {
	// start an ostracon node (and kvstore) in the background to test against
	dir, err := ioutil.TempDir("/tmp", "rpc-client-test")
	if err != nil {
		panic(err)
	}

	app := kvstore.NewPersistentKVStoreApplication(dir)
	node = rpctest.StartTendermint(app)

	code := m.Run()

	// and shut down proper at the end
	rpctest.StopTendermint(node)
	_ = os.RemoveAll(dir)
	os.Exit(code)
}
