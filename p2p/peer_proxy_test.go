package p2p

import (
	"testing"
	"time"

	"github.com/line/ostracon/libs/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/line/ostracon/config"
	"github.com/line/ostracon/crypto/ed25519"
	tmconn "github.com/line/ostracon/p2p/conn"
)

func TestPeerProxyBasic(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	// simulate remote peer
	rp1 := &remotePeer{PrivKey: ed25519.GenPrivKey(), Config: cfg}
	rp1.Start()
	t.Cleanup(rp1.Stop)
	rp2 := &remotePeer{PrivKey: ed25519.GenPrivKey(), Config: cfg}
	rp2.Start()
	t.Cleanup(rp2.Stop)

	p, err := createOutboundPeerProxyAndPerformHandshake(rp1.Addr(), rp2.Addr(), cfg, tmconn.DefaultMConnConfig())
	require.Nil(err)

	err = p.Start()
	require.Nil(err)
	t.Cleanup(func() {
		if err := p.Stop(); err != nil {
			t.Error(err)
		}
	})

	assert.True(p.IsRunning())
	assert.True(p.IsOutbound())
	assert.Equal(rp1.Addr().DialString(), p.RemoteAddr().String())
	assert.Equal(rp1.ID(), p.ID())
}

func TestPeerProxySend(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	config := cfg

	// simulate remote peer
	rp1 := &remotePeer{PrivKey: ed25519.GenPrivKey(), Config: config}
	rp1.channels = []byte{testCh, MempoolChannel}
	rp1.Start()
	t.Cleanup(rp1.Stop)
	rp2 := &remotePeer{PrivKey: ed25519.GenPrivKey(), Config: config}
	rp2.channels = []byte{testCh, MempoolChannel}
	rp2.Start()
	t.Cleanup(rp2.Stop)

	p, err := createOutboundPeerProxyAndPerformHandshake(rp1.Addr(), rp2.Addr(), config, tmconn.DefaultMConnConfig())
	require.Nil(err)

	err = p.Start()
	require.Nil(err)

	t.Cleanup(func() {
		if err := p.Stop(); err != nil {
			t.Error(err)
		}
	})

	assert.True(p.CanSend(testCh))
	assert.True(p.Send(testCh, []byte("Asylum")))
	assert.True(p.CanSend(MempoolChannel))
	assert.True(p.Send(MempoolChannel, []byte("Asylum")))
}

func createOutboundPeerProxyAndPerformHandshake(
	addr *NetAddress,
	addrMempool *NetAddress,
	config *config.P2PConfig,
	mConfig tmconn.MConnConfig,
) (*peerProxy, error) {
	proxyInfo := newPeerProxyInfo()
	{
		defaultPeer, err := createOutboundPeerAndPerformHandshakeForPeerProxy(addr, config, mConfig)
		proxyInfo.setPeerAndError(defaultIndex, defaultPeer, err)
	}
	{
		mempoolPeer, err := createOutboundPeerAndPerformHandshakeForPeerProxy(addrMempool, config, mConfig)
		proxyInfo.setPeerAndError(mempoolIndex, mempoolPeer, err)
	}
	peerProxy := newPeerProxy(proxyInfo)
	peerProxy.SetLogger(log.TestingLogger().With("peerProxy", addr))
	return peerProxy, nil
}

func createOutboundPeerAndPerformHandshakeForPeerProxy(
	addr *NetAddress,
	config *config.P2PConfig,
	mConfig tmconn.MConnConfig,
) (*peer, error) {
	chDescs := []*tmconn.ChannelDescriptor{
		{ID: testCh, Priority: 1},
		{ID: MempoolChannel, Priority: 2},
	}
	reactorsByCh := map[byte]Reactor{
		testCh:         NewTestReactor(chDescs, true, 1000, true),
		MempoolChannel: NewTestReactor(chDescs, true, 1000, true),
	}
	pk := ed25519.GenPrivKey()
	pc, err := testOutboundPeerConn(addr, config, false, pk)
	if err != nil {
		return nil, err
	}
	timeout := 1 * time.Second
	ourNodeInfo := testNodeInfo(addr.ID, "host_peer")
	ourNodeInfoCasted := ourNodeInfo.(DefaultNodeInfo)
	ourNodeInfoCasted.Channels = []byte{testCh, MempoolChannel}
	peerNodeInfo, err := handshake(pc.conn, timeout, ourNodeInfoCasted)
	if err != nil {
		return nil, err
	}

	p := newPeer(pc, mConfig, peerNodeInfo, reactorsByCh, chDescs, func(p Peer, r interface{}) {})
	p.SetLogger(log.TestingLogger().With("peer", addr))
	return p, nil
}
