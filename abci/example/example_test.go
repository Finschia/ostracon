package example

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"google.golang.org/grpc"

	"golang.org/x/net/context"

	"github.com/tendermint/tendermint/abci/types"

	abcicli "github.com/Finschia/ostracon/abci/client"
	"github.com/Finschia/ostracon/abci/example/code"
	"github.com/Finschia/ostracon/abci/example/kvstore"
	abciserver "github.com/Finschia/ostracon/abci/server"
	ocabci "github.com/Finschia/ostracon/abci/types"
	"github.com/Finschia/ostracon/libs/log"
	tmnet "github.com/Finschia/ostracon/libs/net"
)

var globalRand = rand.New(rand.NewSource(time.Now().UnixNano()))

func TestKVStore(t *testing.T) {
	fmt.Println("### Testing KVStore")
	testStream(t, kvstore.NewApplication())
}

func TestBaseApp(t *testing.T) {
	fmt.Println("### Testing BaseApp")
	testStream(t, ocabci.NewBaseApplication())
}

func TestGRPC(t *testing.T) {
	fmt.Println("### Testing GRPC")
	testGRPCSync(t, ocabci.NewGRPCApplication(ocabci.NewBaseApplication()))
}

func testStream(t *testing.T, app ocabci.Application) {
	numDeliverTxs := 20000
	socketFile := fmt.Sprintf("test-%08x.sock", globalRand.Int31n(1<<30))
	defer os.Remove(socketFile)
	socket := fmt.Sprintf("unix://%v", socketFile)

	// Start the listener
	server := abciserver.NewSocketServer(socket, app)
	server.SetLogger(log.TestingLogger().With("module", "abci-server"))
	if err := server.Start(); err != nil {
		require.NoError(t, err, "Error starting socket server")
	}
	t.Cleanup(func() {
		if err := server.Stop(); err != nil {
			t.Error(err)
		}
	})

	// Connect to the socket
	client := abcicli.NewSocketClient(socket, false)
	client.SetLogger(log.TestingLogger().With("module", "abci-client"))
	if err := client.Start(); err != nil {
		t.Fatalf("Error starting socket client: %v", err.Error())
	}
	t.Cleanup(func() {
		if err := client.Stop(); err != nil {
			t.Error(err)
		}
	})

	done := make(chan struct{})
	counter := 0
	client.SetGlobalCallback(func(req *ocabci.Request, res *ocabci.Response) {
		// Process response
		switch r := res.Value.(type) {
		case *ocabci.Response_DeliverTx:
			counter++
			if r.DeliverTx.Code != code.CodeTypeOK {
				t.Error("DeliverTx failed with ret_code", r.DeliverTx.Code)
			}
			if counter > numDeliverTxs {
				t.Fatalf("Too many DeliverTx responses. Got %d, expected %d", counter, numDeliverTxs)
			}
			if counter == numDeliverTxs {
				go func() {
					time.Sleep(time.Second * 1) // Wait for a bit to allow counter overflow
					close(done)
				}()
				return
			}
		case *ocabci.Response_Flush:
			// ignore
		default:
			t.Error("Unexpected response type", reflect.TypeOf(res.Value))
		}
	})

	// Write requests
	for counter := 0; counter < numDeliverTxs; counter++ {
		// Send request
		reqRes := client.DeliverTxAsync(types.RequestDeliverTx{Tx: []byte("test")}, nil)
		_ = reqRes
		// check err ?

		// Sometimes send flush messages
		if counter%123 == 0 {
			client.FlushAsync(nil)
			// check err ?
		}
	}

	// Send final flush message
	client.FlushAsync(nil)

	<-done
}

//-------------------------
// test grpc

func dialerFunc(ctx context.Context, addr string) (net.Conn, error) {
	return tmnet.Connect(addr)
}

func testGRPCSync(t *testing.T, app ocabci.ABCIApplicationServer) {
	numDeliverTxs := 2000
	socketFile := fmt.Sprintf("/tmp/test-%08x.sock", globalRand.Int31n(1<<30))
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
	//nolint:staticcheck // SA1019 Existing use of deprecated but supported dial option.
	conn, err := grpc.Dial(socket, grpc.WithInsecure(), grpc.WithContextDialer(dialerFunc))
	if err != nil {
		t.Fatalf("Error dialing GRPC server: %v", err.Error())
	}

	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	})

	client := ocabci.NewABCIApplicationClient(conn)

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
