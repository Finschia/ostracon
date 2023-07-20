package client_test

import (
	"os"
	"testing"

	"github.com/Finschia/ostracon/abci/example/kvstore"
	nm "github.com/Finschia/ostracon/node"
	rpctest "github.com/Finschia/ostracon/rpc/test"
)

var node *nm.Node

func TestMain(m *testing.M) {
	// start an ostracon node (and kvstore) in the background to test against
	dir, err := os.MkdirTemp("/tmp", "rpc-client-test")
	if err != nil {
		panic(err)
	}

	app := kvstore.NewPersistentKVStoreApplication(dir)
	node = rpctest.StartOstracon(app)

	code := m.Run()

	// and shut down proper at the end
	rpctest.StopOstracon(node)
	_ = os.RemoveAll(dir)
	os.Exit(code)
}
