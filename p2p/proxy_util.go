package p2p

import "github.com/line/ostracon/libs/net"

const (
	addMempoolValue = 100
)

func ConvertMempoolAddress(addr NetAddress) *NetAddress {
	port := addr.Port
	if port == 0 {
		// any port number which assign the dynamic and/or private port
	} else if port < 1024 {
		port += 20000 + addMempoolValue // avoid the port number within well known port numbers
	} else if port < 20000 {
		port += 20000 // adjust internal rule roughly: we should use over 20000 port for application
	} else if port < 49052 {
		port += addMempoolValue
	} else if port < 49152 {
		port -= addMempoolValue * 2 // use the port number within registered port numbers
	} else if port > 49151 {
		p, err := net.GetFreePort()
		if err != nil {
			panic(err)
		}
		port = uint16(p)
	}
	nedAddress := NewNetAddressIPPort(addr.IP, port)
	nedAddress.ID = addr.ID
	return nedAddress
}

func ConvertMempoolNodeInfo(nodeInfo DefaultNodeInfo) DefaultNodeInfo {
	if nodeInfo.ID() == "" || nodeInfo.ListenAddr == "" {
		return nodeInfo
	}
	addr, err := NewNetAddressString(IDAddressString(nodeInfo.ID(), nodeInfo.ListenAddr))
	if err != nil {
		panic(err)
	}
	nodeInfo.ListenAddr = ConvertMempoolAddress(*addr).DialString()
	return nodeInfo
}

const (
	peerProxySize = 2
	defaultIndex  = 0
	mempoolIndex  = 1
)

type peerProxyInfo struct {
	size  int
	peers []Peer
	errs  []error
}

func newPeerProxyInfo() peerProxyInfo {
	return peerProxyInfo{
		size:  peerProxySize,
		peers: make([]Peer, peerProxySize),
		errs:  make([]error, peerProxySize),
	}
}

func (p peerProxyInfo) getPeer(index int) Peer {
	return p.peers[index]
}

func (p peerProxyInfo) setPeerAndError(index int, peer Peer, err error) {
	p.peers[index] = peer
	p.errs[index] = err
}

func (p peerProxyInfo) anyError() error {
	for _, e := range p.errs {
		if e != nil {
			return e
		}
	}
	return nil
}
