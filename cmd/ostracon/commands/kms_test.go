package commands

import (
	"fmt"
	"net"
	"os"
	"path"
	"testing"
	"time"

	"github.com/line/ostracon/crypto"
	"github.com/line/ostracon/crypto/ed25519"
	"github.com/line/ostracon/libs/log"
	"github.com/line/ostracon/privval"
)

func WithMockKMS(t *testing.T, dir, chainID string, f func(string, crypto.PrivKey)) {
	// This process is based on cmd/priv_validator_server/main.go

	// obtain an address using a vacancy port number
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
		return
	}
	addr := listener.Addr().String()
	if err = listener.Close(); err != nil {
		t.Fatal(err)
		return
	}

	// start mock kms server
	privKey := ed25519.GenPrivKeyFromSecret([]byte("🏺"))
	shutdown := make(chan string)
	go func() {
		stdoutMutex.Lock()
		stdout := os.Stdout
		stdoutMutex.Unlock()

		logger := log.NewOCLogger(log.NewSyncWriter(stdout))
		logger.Info(fmt.Sprintf("MockKMS starting: [%s] %s", chainID, addr))
		pv := privval.NewFilePV(privKey, path.Join(dir, "keyfile"), path.Join(dir, "statefile"))
		connTimeout := 5 * time.Second
		dialer := privval.DialTCPFn(addr, connTimeout, ed25519.GenPrivKeyFromSecret([]byte("🔌")))
		sd := privval.NewSignerDialerEndpoint(logger, dialer)
		ss := privval.NewSignerServer(sd, chainID, pv)
		err := ss.Start()
		if err != nil {
			panic(err)
		}
		logger.Info("MockKMS started")
		<-shutdown
		logger.Info("MockKMS stopping")
		if err = ss.Stop(); err != nil {
			panic(err)
		}
		logger.Info("MockKMS stopped")
	}()
	defer func() {
		shutdown <- "SHUTDOWN"
	}()

	f(addr, privKey)
}
