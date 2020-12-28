package example

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"reflect"
	"testing"
	"time"

	"google.golang.org/grpc"

	"golang.org/x/net/context"

	"github.com/line/ostracon/libs/log"
	tmnet "github.com/line/ostracon/libs/net"

	abcicli "github.com/line/ostracon/abci/client"
	"github.com/line/ostracon/abci/example/code"
	"github.com/line/ostracon/abci/example/kvstore"
	abciserver "github.com/line/ostracon/abci/server"
	"github.com/line/ostracon/abci/types"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func TestKVStore(t *testing.T) {
	fmt.Println("### Testing KVStore")
	testGRPCSync(t, types.NewGRPCApplication(kvstore.NewApplication()))
}

func TestBaseApp(t *testing.T) {
	fmt.Println("### Testing BaseApp")
	testGRPCSync(t, types.NewGRPCApplication(types.NewBaseApplication()))
}

//-------------------------
// test grpc

func dialerFunc(ctx context.Context, addr string) (net.Conn, error) {
	return tmnet.Connect(addr)
}

func testGRPCSync(t *testing.T, app types.ABCIApplicationServer) {
	numDeliverTxs := 2000
	socketFile := fmt.Sprintf("test-%08x.sock", rand.Int31n(1<<30))
	defer os.Remove(socketFile)
	socket := fmt.Sprintf("unix://%v", socketFile)

	// Start the listener
	server := abciserver.NewGRPCServer(socket, app)
	server.SetLogger(log.TestingLogger().With("module", "abci-server"))
	if err := server.Start(); err != nil {
		t.Fatalf("Error starting GRPC server: %v", err.Error())
	}

	t.Cleanup(func() {
		if err := server.Stop(); err != nil {
			t.Error(err)
		}
	})

	// Connect to the socket
	conn, err := grpc.Dial(socket, grpc.WithInsecure(), grpc.WithContextDialer(dialerFunc))
	if err != nil {
		t.Fatalf("Error dialing GRPC server: %v", err.Error())
	}

	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	})

	client := types.NewABCIApplicationClient(conn)

	// Write requests
	for counter := 0; counter < numDeliverTxs; counter++ {
		// Send request
		response, err := client.DeliverTx(context.Background(), &types.RequestDeliverTx{Tx: []byte("test")})
		if err != nil {
			t.Fatalf("Error in GRPC DeliverTx: %v", err.Error())
		}
		counter++
		if response.Code != code.CodeTypeOK {
			t.Error("DeliverTx failed with ret_code", response.Code)
		}
		if counter > numDeliverTxs {
			t.Fatal("Too many DeliverTx responses")
		}
		t.Log("response", counter)
		if counter == numDeliverTxs {
			go func() {
				time.Sleep(time.Second * 1) // Wait for a bit to allow counter overflow
			}()
		}
	}
}
