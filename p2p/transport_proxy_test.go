package p2p

import (
	"fmt"
	"math/rand"
	"net"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/line/ostracon/crypto/ed25519"
	"github.com/line/ostracon/p2p/conn"
)

// newMultiplexTransportProxy is for newMultiplexTransport
func newMultiplexTransportProxy(
	nodeInfo NodeInfo,
	nodeKey NodeKey,
) *MultiplexTransportProxy {
	return NewMultiplexTransportProxy(
		nodeInfo, nodeKey, conn.DefaultMConnConfig(),
	)
}

func TestTransportMultiplexProxyConnFilter(t *testing.T) {
	mt := newMultiplexTransportProxy(
		emptyNodeInfo(),
		NodeKey{
			PrivKey: ed25519.GenPrivKey(),
		},
	)
	id := mt.defaultTransport.nodeKey.ID()

	MultiplexTransportProxyConnFilters(
		func(_ ConnSet, _ net.Conn, _ []net.IP) error { return nil },
		func(_ ConnSet, _ net.Conn, _ []net.IP) error { return nil },
		func(_ ConnSet, _ net.Conn, _ []net.IP) error {
			return fmt.Errorf("rejected")
		},
	)(mt)

	addr, err := NewNetAddressString(IDAddressString(id, "127.0.0.1:0"))
	if err != nil {
		t.Fatal(err)
	}

	if err := mt.Listen(*addr); err != nil {
		t.Fatal(err)
	}

	errc := make(chan error)

	go func() {
		testDialToTransport(NewNetAddress(id, mt.defaultTransport.listener.Addr()), errc)
		testDialToTransport(NewNetAddress(id, mt.mempoolTransport.listener.Addr()), errc)
		close(errc)
	}()

	if err := <-errc; err != nil {
		t.Errorf("connection failed: %v", err)
	}

	_, err = mt.Accept(peerConfig{})
	if err, ok := err.(ErrRejected); ok {
		if !err.IsFiltered() {
			t.Errorf("expected peer to be filtered, got %v", err)
		}
	} else {
		t.Errorf("expected ErrRejected, got %v", err)
	}
}

func TestTransportMultiplexProxyConnFilterTimeout(t *testing.T) {
	mt := newMultiplexTransportProxy(
		emptyNodeInfo(),
		NodeKey{
			PrivKey: ed25519.GenPrivKey(),
		},
	)
	id := mt.defaultTransport.nodeKey.ID()

	MultiplexTransportProxyFilterTimeout(5 * time.Millisecond)(mt)
	MultiplexTransportProxyConnFilters(
		func(_ ConnSet, _ net.Conn, _ []net.IP) error {
			time.Sleep(1 * time.Second)
			return nil
		},
	)(mt)

	addr, err := NewNetAddressString(IDAddressString(id, "127.0.0.1:0"))
	if err != nil {
		t.Fatal(err)
	}

	if err := mt.Listen(*addr); err != nil {
		t.Fatal(err)
	}

	errc := make(chan error)

	go func() {
		testDialToTransport(NewNetAddress(id, mt.defaultTransport.listener.Addr()), errc)
		testDialToTransport(NewNetAddress(id, mt.mempoolTransport.listener.Addr()), errc)
		close(errc)
	}()

	if err := <-errc; err != nil {
		t.Errorf("connection failed: %v", err)
	}

	_, err = mt.Accept(peerConfig{})
	if _, ok := err.(ErrFilterTimeout); !ok {
		t.Errorf("expected ErrFilterTimeout, got %v", err)
	}
}

func testDialToTransport(addr *NetAddress, errc chan error) net.Conn {
	c, err := addr.Dial()
	if err != nil {
		errc <- err
		return nil
	}
	return c
}

func TestMultiplexTransportProxyResolver(t *testing.T) {
	MultiplexTransportProxyResolver(nil) // FIXME
}

func TestTransportMultiplexProxyMaxIncomingConnections(t *testing.T) {
	pv := ed25519.GenPrivKey()
	id := PubKeyToID(pv.PubKey())
	mt := newMultiplexTransportProxy(
		testNodeInfo(
			id, "transport",
		),
		NodeKey{
			PrivKey: pv,
		},
	)

	MultiplexTransportProxyMaxIncomingConnections(0)(mt)

	addr, err := NewNetAddressString(IDAddressString(id, "127.0.0.1:0"))
	if err != nil {
		t.Fatal(err)
	}
	const maxIncomingConns = 2
	MultiplexTransportProxyMaxIncomingConnections(maxIncomingConns)(mt)
	if err := mt.Listen(*addr); err != nil {
		t.Fatal(err)
	}

	laddr := NewNetAddress(mt.defaultTransport.nodeKey.ID(), mt.defaultTransport.listener.Addr())
	laddrMempool := NewNetAddress(mt.defaultTransport.nodeKey.ID(), mt.mempoolTransport.listener.Addr())

	// Connect more peers than max
	for i := 0; i <= maxIncomingConns; i++ {
		errc := make(chan error)
		go func() {
			var (
				pv     = ed25519.GenPrivKey()
				dialer = newMultiplexTransport(
					testNodeInfo(PubKeyToID(pv.PubKey()), defaultNodeName),
					NodeKey{
						PrivKey: pv,
					},
				)
			)

			testDialFromTransport(dialer, *laddr, errc)
			testDialFromTransport(dialer, *laddrMempool, errc)
			// Signal that the connection was established.
			errc <- nil
		}()

		err = <-errc
		if i < maxIncomingConns {
			if err != nil {
				t.Errorf("dialer connection failed: %v", err)
			}
			_, err = mt.Accept(peerConfig{})
			if err != nil {
				t.Errorf("connection failed: %v", err)
			}
		} else if err == nil || !strings.Contains(err.Error(), "i/o timeout") {
			// mt actually blocks forever on trying to accept a new peer into a full channel so
			// expect the dialer to encounter a timeout error. Calling mt.Accept will block until
			// mt is closed.
			t.Errorf("expected i/o timeout error, got %v", err)
		}
	}
}

func TestTransportMultiplexProxyAcceptMultiple(t *testing.T) {
	mt := testSetupMultiplexTransportProxy(t)
	laddr := NewNetAddress(mt.defaultTransport.nodeKey.ID(), mt.defaultTransport.listener.Addr())
	laddrMempool := NewNetAddress(mt.defaultTransport.nodeKey.ID(), mt.mempoolTransport.listener.Addr())

	var (
		seed     = rand.New(rand.NewSource(time.Now().UnixNano()))
		nDialers = seed.Intn(64) + 64
		errc     = make(chan error, nDialers)
	)

	// Setup dialers.
	for i := 0; i < nDialers; i++ {
		go func() {
			var (
				pv     = ed25519.GenPrivKey()
				dialer = newMultiplexTransport(
					testNodeInfo(PubKeyToID(pv.PubKey()), defaultNodeName),
					NodeKey{
						PrivKey: pv,
					},
				)
			)

			testDialFromTransport(dialer, *laddr, errc)
			testDialFromTransport(dialer, *laddrMempool, errc)
			// Signal that the connection was established.
			errc <- nil
		}()
	}

	// Catch connection errors.
	for i := 0; i < nDialers; i++ {
		if err := <-errc; err != nil {
			t.Fatal(err)
		}
	}

	ps := []Peer{}

	// Accept all peers.
	for i := 0; i < cap(errc); i++ {
		p, err := mt.Accept(peerConfig{})
		if err != nil {
			t.Fatal(err)
		}

		if err := p.Start(); err != nil {
			t.Fatal(err)
		}

		ps = append(ps, p)
	}

	if have, want := len(ps), cap(errc); have != want {
		t.Errorf("have %v, want %v", have, want)
	}

	// Stop all peers.
	for _, p := range ps {
		if err := p.Stop(); err != nil {
			t.Fatal(err)
		}
	}

	if err := mt.Close(); err != nil {
		t.Errorf("close errored: %v", err)
	}
}

func testDialFromTransport(transport *MultiplexTransport, dialAddr NetAddress, errc chan error) {
	_, err := transport.Dial(dialAddr, peerConfig{})
	if err != nil {
		errc <- err
		return
	}
}

func TestTransportMultiplexProxyAcceptNonBlocking(t *testing.T) {
	mt := testSetupMultiplexTransportProxy(t)

	var (
		fastNodePV   = ed25519.GenPrivKey()
		fastNodeInfo = testNodeInfo(PubKeyToID(fastNodePV.PubKey()), "fastnode")
		errc         = make(chan error)
		fastc        = make(chan struct{})
		slowc        = make(chan struct{})
		slowdonec    = make(chan struct{})
	)

	// Simulate slow Peer.
	go func() {
		addr := NewNetAddress(mt.defaultTransport.nodeKey.ID(), mt.defaultTransport.listener.Addr())
		addrMempool := NewNetAddress(mt.mempoolTransport.nodeKey.ID(), mt.mempoolTransport.listener.Addr())

		c := testDialToTransport(addr, errc)
		testDialToTransport(addrMempool, errc)

		close(slowc)
		defer func() {
			close(slowdonec)
		}()

		// Make sure we switch to fast peer goroutine.
		runtime.Gosched()

		select {
		case <-fastc:
			// Fast peer connected.
		case <-time.After(200 * time.Millisecond):
			// We error if the fast peer didn't succeed.
			errc <- fmt.Errorf("fast peer timed out")
		}

		sc, err := upgradeSecretConn(c, 200*time.Millisecond, ed25519.GenPrivKey())
		if err != nil {
			errc <- err
			return
		}

		_, err = handshake(sc, 200*time.Millisecond,
			testNodeInfo(
				PubKeyToID(ed25519.GenPrivKey().PubKey()),
				"slow_peer",
			))
		if err != nil {
			errc <- err
		}
	}()

	// Simulate fast Peer.
	go func() {
		<-slowc

		var (
			dialer = newMultiplexTransport(
				fastNodeInfo,
				NodeKey{
					PrivKey: fastNodePV,
				},
			)
		)
		addr := NewNetAddress(mt.defaultTransport.nodeKey.ID(), mt.defaultTransport.listener.Addr())
		addrMempool := NewNetAddress(mt.mempoolTransport.nodeKey.ID(), mt.mempoolTransport.listener.Addr())

		testDialFromTransport(dialer, *addr, errc)
		testDialFromTransport(dialer, *addrMempool, errc)

		close(fastc)
		<-slowdonec
		close(errc)
	}()

	if err := <-errc; err != nil {
		t.Logf("connection failed: %v", err)
	}

	p, err := mt.Accept(peerConfig{})
	if err != nil {
		t.Fatal(err)
	}

	if have, want := p.NodeInfo(), fastNodeInfo; !reflect.DeepEqual(have, want) {
		t.Errorf("have %v, want %v", have, want)
	}
}

func TestTransportMultiplexProxyValidateNodeInfo(t *testing.T) {
	mt := testSetupMultiplexTransportProxy(t)

	errc := make(chan error)

	go func() {
		var (
			pv     = ed25519.GenPrivKey()
			dialer = newMultiplexTransport(
				testNodeInfo(PubKeyToID(pv.PubKey()), ""), // Should not be empty
				NodeKey{
					PrivKey: pv,
				},
			)
		)

		addr := NewNetAddress(mt.defaultTransport.nodeKey.ID(), mt.defaultTransport.listener.Addr())
		addrMempool := NewNetAddress(mt.mempoolTransport.nodeKey.ID(), mt.mempoolTransport.listener.Addr())

		testDialFromTransport(dialer, *addr, errc)
		testDialFromTransport(dialer, *addrMempool, errc)

		close(errc)
	}()

	if err := <-errc; err != nil {
		t.Errorf("connection failed: %v", err)
	}

	_, err := mt.Accept(peerConfig{})
	if err, ok := err.(ErrRejected); ok {
		if !err.IsNodeInfoInvalid() {
			t.Errorf("expected NodeInfo to be invalid, got %v", err)
		}
	} else {
		t.Errorf("expected ErrRejected, got %v", err)
	}
}

func TestTransportMultiplexProxyRejectMissmatchID(t *testing.T) {
	mt := testSetupMultiplexTransportProxy(t)

	errc := make(chan error)

	go func() {
		dialer := newMultiplexTransport(
			testNodeInfo(
				PubKeyToID(ed25519.GenPrivKey().PubKey()), "dialer",
			),
			NodeKey{
				PrivKey: ed25519.GenPrivKey(),
			},
		)
		addr := NewNetAddress(mt.defaultTransport.nodeKey.ID(), mt.defaultTransport.listener.Addr())
		addrMempool := NewNetAddress(mt.mempoolTransport.nodeKey.ID(), mt.mempoolTransport.listener.Addr())

		testDialFromTransport(dialer, *addr, errc)
		testDialFromTransport(dialer, *addrMempool, errc)

		close(errc)
	}()

	if err := <-errc; err != nil {
		t.Errorf("connection failed: %v", err)
	}

	_, err := mt.Accept(peerConfig{})
	if err, ok := err.(ErrRejected); ok {
		if !err.IsAuthFailure() {
			t.Errorf("expected auth failure, got %v", err)
		}
	} else {
		t.Errorf("expected ErrRejected, got %v", err)
	}
}

func TestTransportMultiplexProxyDialRejectWrongID(t *testing.T) {
	mt := testSetupMultiplexTransportProxy(t)

	var (
		pv     = ed25519.GenPrivKey()
		dialer = newMultiplexTransport(
			testNodeInfo(PubKeyToID(pv.PubKey()), ""), // Should not be empty
			NodeKey{
				PrivKey: pv,
			},
		)
	)

	wrongID := PubKeyToID(ed25519.GenPrivKey().PubKey())
	addr := NewNetAddress(wrongID, mt.defaultTransport.listener.Addr())

	_, err := dialer.Dial(*addr, peerConfig{})
	if err != nil {
		t.Logf("connection failed: %v", err)
		if err, ok := err.(ErrRejected); ok {
			if !err.IsAuthFailure() {
				t.Errorf("expected auth failure, got %v", err)
			}
		} else {
			t.Errorf("expected ErrRejected, got %v", err)
		}
	}
}

func TestTransportMultiplexProxyRejectIncompatible(t *testing.T) {
	mt := testSetupMultiplexTransportProxy(t)

	errc := make(chan error)

	go func() {
		var (
			pv     = ed25519.GenPrivKey()
			dialer = newMultiplexTransport(
				testNodeInfoWithNetwork(PubKeyToID(pv.PubKey()), "dialer", "incompatible-network"),
				NodeKey{
					PrivKey: pv,
				},
			)
		)
		addr := NewNetAddress(mt.defaultTransport.nodeKey.ID(), mt.defaultTransport.listener.Addr())
		addrMempool := NewNetAddress(mt.mempoolTransport.nodeKey.ID(), mt.mempoolTransport.listener.Addr())

		testDialFromTransport(dialer, *addr, errc)
		testDialFromTransport(dialer, *addrMempool, errc)

		close(errc)
	}()
	<-errc // ignored

	_, err := mt.Accept(peerConfig{})
	if err, ok := err.(ErrRejected); ok {
		if !err.IsIncompatible() {
			t.Errorf("expected to reject incompatible, got %v", err)
		}
	} else {
		t.Errorf("expected ErrRejected, got %v", err)
	}
}

func TestTransportMultiplexProxyRejectSelf(t *testing.T) {
	mt := testSetupMultiplexTransportProxy(t)

	errc := make(chan error)

	go func() {
		addr := NewNetAddress(mt.defaultTransport.nodeKey.ID(), mt.defaultTransport.listener.Addr())
		addrMempool := NewNetAddress(mt.mempoolTransport.nodeKey.ID(), mt.mempoolTransport.listener.Addr())

		testDialFromTransport(mt.defaultTransport, *addr, errc)
		testDialFromTransport(mt.mempoolTransport, *addrMempool, errc)

		close(errc)
	}()

	if err := <-errc; err != nil {
		if err, ok := err.(ErrRejected); ok {
			if !err.IsSelf() {
				t.Errorf("expected to reject self, got: %v", err)
			}
		} else {
			t.Errorf("expected ErrRejected, got %v", err)
		}
	} else {
		t.Errorf("expected connection failure")
	}

	_, err := mt.Accept(peerConfig{})
	if err, ok := err.(ErrRejected); ok {
		if !err.IsSelf() {
			t.Errorf("expected to reject self, got: %v", err)
		}
	} else {
		t.Errorf("expected ErrRejected, got %v", nil)
	}
}

func TestTransportProxyConnDuplicateIPFilter(t *testing.T) {
	filter := ConnDuplicateIPFilter()

	if err := filter(nil, &testTransportConn{}, nil); err != nil {
		t.Fatal(err)
	}

	var (
		c  = &testTransportConn{}
		cs = NewConnSet()
	)

	cs.Set(c, []net.IP{
		{10, 0, 10, 1},
		{10, 0, 10, 2},
		{10, 0, 10, 3},
	})

	if err := filter(cs, c, []net.IP{
		{10, 0, 10, 2},
	}); err == nil {
		t.Errorf("expected Peer to be rejected as duplicate")
	}
}

func TestTransportProxyAddChannel(t *testing.T) {
	mt := newMultiplexTransportProxy(
		emptyNodeInfo(),
		NodeKey{
			PrivKey: ed25519.GenPrivKey(),
		},
	)
	testChannel := byte(0x01)

	mt.AddChannel(testChannel)
	if !mt.defaultTransport.nodeInfo.(DefaultNodeInfo).HasChannel(testChannel) {
		t.Errorf("missing added channel %v. Got %v", testChannel, mt.defaultTransport.nodeInfo.(DefaultNodeInfo).Channels)
	}
}

// create listener
func testSetupMultiplexTransportProxy(t *testing.T) *MultiplexTransportProxy {
	var (
		pv = ed25519.GenPrivKey()
		id = PubKeyToID(pv.PubKey())
		mt = newMultiplexTransportProxy(
			testNodeInfo(
				id, "transport",
			),
			NodeKey{
				PrivKey: pv,
			},
		)
	)

	addr, err := NewNetAddressString(IDAddressString(id, "127.0.0.1:0"))
	if err != nil {
		t.Fatal(err)
	}

	if err := mt.Listen(*addr); err != nil {
		t.Fatal(err)
	}

	// give the listener some time to get ready
	time.Sleep(20 * time.Millisecond)

	return mt
}
