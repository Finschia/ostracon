package privval

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
)

// WithMockKMS function starts/stops a mock KMS function for testing on an unused local port. The continuation function
// f is passed the address to connect to and the private key that KMS uses for signing. Thus, it is possible to test
// the connection to KMS and verify the signature in the continuation function.
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
	logger := log.NewOCLogger(log.NewSyncWriter(os.Stdout))
	privKey := ed25519.GenPrivKeyFromSecret([]byte("🏺"))
	shutdown := make(chan string)
	go func() {
		logger.Info(fmt.Sprintf("MockKMS starting: [%s] %s", chainID, addr))
		pv := NewFilePV(privKey, path.Join(dir, "keyfile"), path.Join(dir, "statefile"))
		connTimeout := 5 * time.Second
		dialer := DialTCPFn(addr, connTimeout, ed25519.GenPrivKeyFromSecret([]byte("🔌")))
		sd := NewSignerDialerEndpoint(logger, dialer)
		ss := NewSignerServer(sd, chainID, pv)
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
