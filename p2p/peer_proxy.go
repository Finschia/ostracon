package p2p

import (
	"fmt"
	"net"

	"github.com/line/ostracon/libs/log"
	"github.com/line/ostracon/libs/service"
	tmconn "github.com/line/ostracon/p2p/conn"
)

// MempoolChannel is copy from mempool.(reactor.go) for avoiding import cycle
const MempoolChannel = byte(0x30)

// peerProxy proxy peer
type peerProxy struct {
	service.BaseService
	defaultPeer peer
	mempoolPeer peer
}

var _ Peer = (*peerProxy)(nil)
var _ service.Service = (*peerProxy)(nil)

func newPeerProxy(
	proxyInfo peerProxyInfo,
	options ...PeerOption,
) *peerProxy {

	p := &peerProxy{
		defaultPeer: *proxyInfo.getPeer(defaultIndex).(*peer),
		mempoolPeer: *proxyInfo.getPeer(mempoolIndex).(*peer),
	}

	p.BaseService = *service.NewBaseService(nil, "PeerProxy", p)
	for _, option := range options {
		option(&p.defaultPeer)
		option(&p.mempoolPeer)
	}

	return p
}

// String overrides service.Service
func (p *peerProxy) String() string {
	return fmt.Sprintf("default: %s, mempool: %s", p.defaultPeer.String(), p.mempoolPeer.String())
}

// SetLogger overrides service.Service
func (p *peerProxy) SetLogger(l log.Logger) {
	p.Logger = l
	p.defaultPeer.SetLogger(l)
	p.mempoolPeer.SetLogger(l)
}

// OnStart overrides service.Service
func (p *peerProxy) OnStart() error {
	if err := p.defaultPeer.Start(); err != nil {
		return err
	}
	return p.mempoolPeer.Start()
}

// OnStop overrides service.Service
func (p *peerProxy) OnStop() {
	p.defaultPeer.Stop() //nolint:errcheck
	p.mempoolPeer.Stop() //nolint:errcheck
}

// FlushStop implements Peer
func (p *peerProxy) FlushStop() {
	p.defaultPeer.FlushStop()
	p.mempoolPeer.FlushStop()
}

// CloseConn implements Peer
func (p *peerProxy) CloseConn() error {
	if err := p.defaultPeer.CloseConn(); err != nil {
		return err
	}
	return p.defaultPeer.CloseConn()
}

// ID implements Peer
func (p *peerProxy) ID() ID {
	return p.defaultPeer.ID()
}

// RemoteIP implements Peer
func (p *peerProxy) RemoteIP() net.IP {
	return p.defaultPeer.RemoteIP()
}

// RemoteAddr implements Peer
func (p *peerProxy) RemoteAddr() net.Addr {
	return p.defaultPeer.RemoteAddr()
}

// IsOutbound implements Peer
func (p *peerProxy) IsOutbound() bool {
	return p.defaultPeer.IsOutbound()
}

// IsPersistent implements Peer
func (p *peerProxy) IsPersistent() bool {
	return p.defaultPeer.IsPersistent()
}

// NodeInfo implements Peer
func (p *peerProxy) NodeInfo() NodeInfo {
	return p.defaultPeer.NodeInfo()
}

// SocketAddr implements Peer
func (p *peerProxy) SocketAddr() *NetAddress {
	return p.defaultPeer.SocketAddr()
}

// Get implements Peer
func (p *peerProxy) Get(key string) interface{} {
	return p.defaultPeer.Get(key)
}

// Set implements Peer
func (p *peerProxy) Set(key string, data interface{}) {
	p.defaultPeer.Set(key, data)
}

// Status implements Peer
func (p *peerProxy) Status() tmconn.ConnectionStatus {
	status := p.defaultPeer.Status()
	for i, channel := range status.Channels {
		if channel.ID == MempoolChannel {
			status.Channels[i] = p.mempoolPeer.Status().Channels[i]
			break
		}
	}
	return p.defaultPeer.Status()
}

// Send implements Peer
func (p *peerProxy) Send(chID byte, msgBytes []byte) bool {
	if chID == MempoolChannel {
		return p.mempoolPeer.Send(chID, msgBytes)
	}
	return p.defaultPeer.Send(chID, msgBytes)
}

// TrySend implements Peer
func (p *peerProxy) TrySend(chID byte, msgBytes []byte) bool {
	if chID == MempoolChannel {
		return p.mempoolPeer.TrySend(chID, msgBytes)
	}
	return p.defaultPeer.TrySend(chID, msgBytes)
}

// CanSend overrides peer.CanSend which is for test(TestPeerSend)
func (p *peerProxy) CanSend(chID byte) bool {
	if chID == MempoolChannel {
		return p.mempoolPeer.CanSend(chID)
	}
	return p.defaultPeer.CanSend(chID)
}
