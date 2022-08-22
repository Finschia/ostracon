package p2p

import (
	"sync"
	"time"

	"github.com/line/ostracon/p2p/conn"
)

type MultiplexTransportProxyOption func(*MultiplexTransportProxy)

func MultiplexTransportProxyConnFilters(filters ...ConnFilterFunc) MultiplexTransportProxyOption {
	return func(mt *MultiplexTransportProxy) {
		mt.defaultTransport.connFilters = filters
		mt.mempoolTransport.connFilters = filters
	}
}

func MultiplexTransportProxyFilterTimeout(
	timeout time.Duration,
) MultiplexTransportProxyOption {
	return func(mt *MultiplexTransportProxy) {
		mt.defaultTransport.filterTimeout = timeout
		mt.mempoolTransport.filterTimeout = timeout
	}
}

func MultiplexTransportProxyResolver(resolver IPResolver) MultiplexTransportProxyOption {
	return func(mt *MultiplexTransportProxy) {
		mt.defaultTransport.resolver = resolver
		mt.mempoolTransport.resolver = resolver
	}
}

func MultiplexTransportProxyMaxIncomingConnections(n int) MultiplexTransportProxyOption {
	return func(mt *MultiplexTransportProxy) {
		mt.defaultTransport.maxIncomingConnections = n
		mt.mempoolTransport.maxIncomingConnections = n
	}
}

// MultiplexTransportProxy proxy MultiplexTransport
type MultiplexTransportProxy struct {
	defaultTransport *MultiplexTransport
	mempoolTransport *MultiplexTransport
}

var _ Transport = (*MultiplexTransportProxy)(nil)
var _ transportLifecycle = (*MultiplexTransportProxy)(nil)

func NewMultiplexTransportProxy(
	nodeInfo NodeInfo,
	nodeKey NodeKey,
	mConfig conn.MConnConfig) *MultiplexTransportProxy {
	return &MultiplexTransportProxy{
		defaultTransport: NewMultiplexTransport(nodeInfo, nodeKey, mConfig),
		mempoolTransport: NewMultiplexTransport(ConvertMempoolNodeInfo(nodeInfo.(DefaultNodeInfo)), nodeKey, mConfig),
	}
}

// Close implement transportLifecycle
func (mt *MultiplexTransportProxy) Close() error {
	if err := mt.defaultTransport.Close(); err != nil {
		return err
	}
	return mt.mempoolTransport.Close()
}

// Listen implement transportLifecycle
func (mt *MultiplexTransportProxy) Listen(addr NetAddress) error {
	if err := mt.defaultTransport.Listen(addr); err != nil {
		return err
	}
	return mt.mempoolTransport.Listen(*ConvertMempoolAddress(addr))
}

// NetAddress overrides MultiplexTransport.NetAddress
func (mt *MultiplexTransportProxy) NetAddress() NetAddress {
	return mt.defaultTransport.NetAddress()
}

// Accept overrides MultiplexTransport.Accept
func (mt *MultiplexTransportProxy) Accept(cfg peerConfig) (Peer, error) {
	proxyInfo := newPeerProxyInfo()
	var wg sync.WaitGroup
	acceptFunc := func(cfg peerConfig, index int, transport *MultiplexTransport) {
		peer, err := transport.Accept(cfg)
		proxyInfo.setPeerAndError(index, peer, err)
		wg.Done()
	}
	wg.Add(peerProxySize)
	go acceptFunc(cfg, defaultIndex, mt.defaultTransport)
	go acceptFunc(cfg, mempoolIndex, mt.mempoolTransport)
	wg.Wait()
	if err := proxyInfo.anyError(); err != nil {
		return nil, err
	}
	return newPeerProxy(proxyInfo), nil
}

// Dial overrides MultiplexTransport.Dial
func (mt *MultiplexTransportProxy) Dial(addr NetAddress, cfg peerConfig) (Peer, error) {
	mempoolAddr := *ConvertMempoolAddress(addr)
	proxyInfo := newPeerProxyInfo()
	var wg sync.WaitGroup
	dialFunc := func(cfg peerConfig, index int, transport *MultiplexTransport, addr *NetAddress) {
		peer, err := transport.Dial(*addr, cfg)
		proxyInfo.setPeerAndError(index, peer, err)
		wg.Done()
	}
	wg.Add(peerProxySize)
	go dialFunc(cfg, defaultIndex, mt.defaultTransport, &addr)
	go dialFunc(cfg, mempoolIndex, mt.mempoolTransport, &mempoolAddr)
	wg.Wait()
	if err := proxyInfo.anyError(); err != nil {
		return nil, err
	}
	return newPeerProxy(proxyInfo), nil
}

// Cleanup overrides MultiplexTransport.Cleanup
func (mt *MultiplexTransportProxy) Cleanup(p Peer) {
	mt.defaultTransport.conns.RemoveAddr(p.(*peerProxy).defaultPeer.RemoteAddr())
	_ = p.(*peerProxy).defaultPeer.CloseConn()
	mt.mempoolTransport.conns.RemoveAddr(p.(*peerProxy).mempoolPeer.RemoteAddr())
	_ = p.(*peerProxy).mempoolPeer.CloseConn()
}

func (mt *MultiplexTransportProxy) AddChannel(chID byte) {
	mt.defaultTransport.AddChannel(chID)
	mt.mempoolTransport.AddChannel(chID)
}
