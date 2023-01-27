package proxy

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tendermint/tendermint/abci/types"

	abcicli "github.com/line/ostracon/abci/client"
	"github.com/line/ostracon/abci/example/kvstore"
	"github.com/line/ostracon/abci/server"
	"github.com/line/ostracon/libs/log"
	tmrand "github.com/line/ostracon/libs/rand"
)

//----------------------------------------

type AppConnTest interface {
	FlushSync() (*types.ResponseFlush, error)
	EchoAsync(string, abcicli.ResponseCallback) *abcicli.ReqRes
	InfoSync(types.RequestInfo) (*types.ResponseInfo, error)
}

type appConnTest struct {
	appConn abcicli.Client
}

func NewAppConnTest(appConn abcicli.Client) AppConnTest {
	return &appConnTest{appConn}
}

func (app *appConnTest) EchoAsync(msg string, cb abcicli.ResponseCallback) *abcicli.ReqRes {
	return app.appConn.EchoAsync(msg, cb)
}

func (app *appConnTest) FlushSync() (*types.ResponseFlush, error) {
	return app.appConn.FlushSync()
}

func (app *appConnTest) InfoSync(req types.RequestInfo) (*types.ResponseInfo, error) {
	return app.appConn.InfoSync(req)
}

//----------------------------------------

var SOCKET = "socket"

func TestEcho(t *testing.T) {
	sockPath := fmt.Sprintf("unix:///tmp/echo_%v.sock", tmrand.Str(6))
	clientCreator := NewRemoteClientCreator(sockPath, SOCKET, true)

	// Start server
	s := server.NewSocketServer(sockPath, kvstore.NewApplication())
	s.SetLogger(log.TestingLogger().With("module", "abci-server"))
	if err := s.Start(); err != nil {
		t.Fatalf("Error starting socket server: %v", err.Error())
	}
	t.Cleanup(func() {
		if err := s.Stop(); err != nil {
			t.Error(err)
		}
	})

	// Start client
	cli, err := clientCreator.NewABCIClient()
	if err != nil {
		t.Fatalf("Error creating ABCI client: %v", err.Error())
	}
	cli.SetLogger(log.TestingLogger().With("module", "abci-client"))
	if err := cli.Start(); err != nil {
		t.Fatalf("Error starting ABCI client: %v", err.Error())
	}

	proxy := NewAppConnTest(cli)
	t.Log("Connected")

	for i := 0; i < 1000; i++ {
		proxy.EchoAsync(fmt.Sprintf("echo-%v", i), nil)
	}
	_, err2 := proxy.FlushSync()
	if err2 != nil {
		t.Error(err2)
	}
}

func BenchmarkEcho(b *testing.B) {
	b.StopTimer() // Initialize
	sockPath := fmt.Sprintf("unix:///tmp/echo_%v.sock", tmrand.Str(6))
	clientCreator := NewRemoteClientCreator(sockPath, SOCKET, true)

	// Start server
	s := server.NewSocketServer(sockPath, kvstore.NewApplication())
	s.SetLogger(log.TestingLogger().With("module", "abci-server"))
	if err := s.Start(); err != nil {
		b.Fatalf("Error starting socket server: %v", err.Error())
	}
	b.Cleanup(func() {
		if err := s.Stop(); err != nil {
			b.Error(err)
		}
	})

	// Start client
	cli, err := clientCreator.NewABCIClient()
	if err != nil {
		b.Fatalf("Error creating ABCI client: %v", err.Error())
	}
	cli.SetLogger(log.TestingLogger().With("module", "abci-client"))
	if err := cli.Start(); err != nil {
		b.Fatalf("Error starting ABCI client: %v", err.Error())
	}

	proxy := NewAppConnTest(cli)
	b.Log("Connected")
	echoString := strings.Repeat(" ", 200)
	b.StartTimer() // Start benchmarking tests

	for i := 0; i < b.N; i++ {
		proxy.EchoAsync(echoString, nil)
	}
	_, err2 := proxy.FlushSync()
	if err2 != nil {
		b.Error(err2)
	}

	b.StopTimer()
	// info := proxy.InfoSync(abci.RequestInfo{""})
	// b.Log("N: ", b.N, info)
}

func TestInfo(t *testing.T) {
	sockPath := fmt.Sprintf("unix:///tmp/echo_%v.sock", tmrand.Str(6))
	clientCreator := NewRemoteClientCreator(sockPath, SOCKET, true)

	// Start server
	s := server.NewSocketServer(sockPath, kvstore.NewApplication())
	s.SetLogger(log.TestingLogger().With("module", "abci-server"))
	if err := s.Start(); err != nil {
		t.Fatalf("Error starting socket server: %v", err.Error())
	}
	t.Cleanup(func() {
		if err := s.Stop(); err != nil {
			t.Error(err)
		}
	})

	// Start client
	cli, err := clientCreator.NewABCIClient()
	if err != nil {
		t.Fatalf("Error creating ABCI client: %v", err.Error())
	}
	cli.SetLogger(log.TestingLogger().With("module", "abci-client"))
	if err := cli.Start(); err != nil {
		t.Fatalf("Error starting ABCI client: %v", err.Error())
	}

	proxy := NewAppConnTest(cli)
	t.Log("Connected")

	resInfo, err := proxy.InfoSync(RequestInfo)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resInfo.Data != "{\"size\":0}" {
		t.Error("Expected ResponseInfo with one element '{\"size\":0}' but got something else")
	}
}
