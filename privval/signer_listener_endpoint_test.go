package privval

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Finschia/ostracon/crypto/ed25519"
	"github.com/Finschia/ostracon/libs/log"
	tmnet "github.com/Finschia/ostracon/libs/net"
	tmrand "github.com/Finschia/ostracon/libs/rand"
	"github.com/Finschia/ostracon/types"
)

var (
	testTimeoutAccept = defaultTimeoutAcceptSeconds * time.Second

	testTimeoutReadWrite    = 1000 * time.Millisecond // increase timeout for slow test env
	testTimeoutReadWrite2o3 = 60 * time.Millisecond   // 2/3 of the other one
)

type dialerTestCase struct {
	addr   string
	dialer SocketDialer
}

// TestSignerRemoteRetryTCPOnly will test connection retry attempts over TCP. We
// don't need this for Unix sockets because the OS instantly knows the state of
// both ends of the socket connection. This basically causes the
// SignerDialerEndpoint.dialer() call inside SignerDialerEndpoint.acceptNewConnection() to return
// successfully immediately, putting an instant stop to any retry attempts.
func TestSignerRemoteRetryTCPOnly(t *testing.T) {
	var (
		attemptCh = make(chan int)
		retries   = 10
	)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	// Continuously Accept connection and close {attempts} times
	go func(ln net.Listener, attemptCh chan<- int) {
		attempts := 0
		for {
			conn, err := ln.Accept()
			require.NoError(t, err)

			err = conn.Close()
			require.NoError(t, err)

			attempts++

			if attempts == retries {
				attemptCh <- attempts
				break
			}
		}
	}(ln, attemptCh)

	dialerEndpoint := NewSignerDialerEndpoint(
		log.TestingLogger(),
		DialTCPFn(ln.Addr().String(), testTimeoutReadWrite, ed25519.GenPrivKey()),
	)
	SignerDialerEndpointTimeoutReadWrite(time.Millisecond)(dialerEndpoint)
	SignerDialerEndpointConnRetries(retries)(dialerEndpoint)

	chainID := tmrand.Str(12)
	mockPV := types.NewMockPV()
	signerServer := NewSignerServer(dialerEndpoint, chainID, mockPV)

	err = signerServer.Start()
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := signerServer.Stop(); err != nil {
			t.Error(err)
		}
	})

	select {
	case attempts := <-attemptCh:
		assert.Equal(t, retries, attempts)
	case <-time.After(1500 * time.Millisecond):
		t.Error("expected remote to observe connection attempts")
	}
}

func TestRetryConnToRemoteSigner(t *testing.T) {
	for _, tc := range getDialerTestCases(t) {
		var (
			logger           = log.TestingLogger()
			chainID          = tmrand.Str(12)
			mockPV           = types.NewMockPV()
			endpointIsOpenCh = make(chan struct{})
			thisConnTimeout  = testTimeoutReadWrite
			listenerEndpoint = newSignerListenerEndpoint(logger, tc.addr, thisConnTimeout)
		)

		dialerEndpoint := NewSignerDialerEndpoint(
			logger,
			tc.dialer,
		)
		SignerDialerEndpointTimeoutReadWrite(testTimeoutReadWrite)(dialerEndpoint)
		SignerDialerEndpointConnRetries(10)(dialerEndpoint)

		signerServer := NewSignerServer(dialerEndpoint, chainID, mockPV)

		startListenerEndpointAsync(t, listenerEndpoint, endpointIsOpenCh)
		t.Cleanup(func() {
			if err := listenerEndpoint.Stop(); err != nil {
				t.Error(err)
			}
		})

		require.NoError(t, signerServer.Start())
		assert.True(t, signerServer.IsRunning())
		<-endpointIsOpenCh
		if err := signerServer.Stop(); err != nil {
			t.Error(err)
		}

		dialerEndpoint2 := NewSignerDialerEndpoint(
			logger,
			tc.dialer,
		)
		signerServer2 := NewSignerServer(dialerEndpoint2, chainID, mockPV)

		// let some pings pass
		require.NoError(t, signerServer2.Start())
		assert.True(t, signerServer2.IsRunning())
		t.Cleanup(func() {
			if err := signerServer2.Stop(); err != nil {
				t.Error(err)
			}
		})

		// give the client some time to re-establish the conn to the remote signer
		// should see sth like this in the logs:
		//
		// E[10016-01-10|17:12:46.128] Ping                                         err="remote signer timed out"
		// I[10016-01-10|17:16:42.447] Re-created connection to remote signer       impl=SocketVal
		time.Sleep(testTimeoutReadWrite * 2)
	}
}

type addrStub struct {
	address string
}

func (a addrStub) Network() string {
	return ""
}

func (a addrStub) String() string {
	return a.address
}

func TestFilterRemoteConnectionByIP(t *testing.T) {
	type fields struct {
		allowIP    string
		remoteAddr net.Addr
		expected   bool
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			"should allow correct ip",
			struct {
				allowIP    string
				remoteAddr net.Addr
				expected   bool
			}{"127.0.0.1", addrStub{"127.0.0.1:45678"}, true},
		}, {
			"should allow correct ip without port",
			struct {
				allowIP    string
				remoteAddr net.Addr
				expected   bool
			}{"127.0.0.1", addrStub{"127.0.0.1"}, true},
		},
		{
			"should not allow different ip",
			struct {
				allowIP    string
				remoteAddr net.Addr
				expected   bool
			}{"127.0.0.1", addrStub{"10.0.0.2:45678"}, false},
		},
		{
			"empty allowIP should allow all",
			struct {
				allowIP    string
				remoteAddr net.Addr
				expected   bool
			}{"", addrStub{"127.0.0.1:45678"}, true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sl := &SignerListenerEndpoint{allowAddr: tt.fields.allowIP}
			assert.Equalf(t, tt.fields.expected, sl.isAllowedAddr(tt.fields.remoteAddr), tt.name)
		})
	}
}

func TestSignerListenerEndpointAllowAddress(t *testing.T) {
	expected := "192.168.0.1"

	cut := NewSignerListenerEndpoint(nil, nil, SignerListenerEndpointAllowAddress(expected))

	assert.Equal(t, expected, cut.allowAddr)
}

func newSignerListenerEndpoint(logger log.Logger, addr string, timeoutReadWrite time.Duration) *SignerListenerEndpoint {
	proto, address := tmnet.ProtocolAndAddress(addr)

	ln, err := net.Listen(proto, address)
	logger.Info("SignerListener: Listening", "proto", proto, "address", address)
	if err != nil {
		panic(err)
	}

	var listener net.Listener

	if proto == "unix" {
		unixLn := NewUnixListener(ln)
		UnixListenerTimeoutAccept(testTimeoutAccept)(unixLn)
		UnixListenerTimeoutReadWrite(timeoutReadWrite)(unixLn)
		listener = unixLn
	} else {
		tcpLn := NewTCPListener(ln, ed25519.GenPrivKey())
		TCPListenerTimeoutAccept(testTimeoutAccept)(tcpLn)
		TCPListenerTimeoutReadWrite(timeoutReadWrite)(tcpLn)
		listener = tcpLn
	}

	return NewSignerListenerEndpoint(
		logger,
		listener,
		SignerListenerEndpointTimeoutReadWrite(testTimeoutReadWrite),
	)
}

func startListenerEndpointAsync(t *testing.T, sle *SignerListenerEndpoint, endpointIsOpenCh chan struct{}) {
	go func(sle *SignerListenerEndpoint) {
		require.NoError(t, sle.Start())
		assert.True(t, sle.IsRunning())
		close(endpointIsOpenCh)
	}(sle)
}

func getMockEndpoints(
	t *testing.T,
	addr string,
	socketDialer SocketDialer,
) (*SignerListenerEndpoint, *SignerDialerEndpoint) {

	var (
		logger           = log.TestingLogger()
		endpointIsOpenCh = make(chan struct{})

		dialerEndpoint = NewSignerDialerEndpoint(
			logger,
			socketDialer,
		)

		listenerEndpoint = newSignerListenerEndpoint(logger, addr, testTimeoutReadWrite)
	)

	SignerDialerEndpointTimeoutReadWrite(testTimeoutReadWrite)(dialerEndpoint)
	SignerDialerEndpointConnRetries(1e6)(dialerEndpoint)

	startListenerEndpointAsync(t, listenerEndpoint, endpointIsOpenCh)

	require.NoError(t, dialerEndpoint.Start())
	assert.True(t, dialerEndpoint.IsRunning())

	<-endpointIsOpenCh

	return listenerEndpoint, dialerEndpoint
}
