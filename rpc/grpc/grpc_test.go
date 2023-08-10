package coregrpc_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	core_grpc "github.com/tendermint/tendermint/rpc/grpc"

	"github.com/Finschia/ostracon/abci/example/kvstore"
	rpctest "github.com/Finschia/ostracon/rpc/test"
)

func TestMain(m *testing.M) {
	// start an ostracon node in the background to test against
	app := kvstore.NewApplication()
	node := rpctest.StartOstracon(app)

	code := m.Run()

	// and shut down proper at the end
	rpctest.StopOstracon(node)
	os.Exit(code)
}

func TestBroadcastTx(t *testing.T) {
	res, err := rpctest.GetGRPCClient().BroadcastTx(
		context.Background(),
		&core_grpc.RequestBroadcastTx{Tx: []byte("this is a tx")},
	)
	require.NoError(t, err)
	require.EqualValues(t, 0, res.CheckTx.Code)
	require.EqualValues(t, 0, res.DeliverTx.Code)
}
