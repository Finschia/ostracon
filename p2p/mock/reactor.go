package mock

import (
	"github.com/Finschia/ostracon/libs/log"
	"github.com/Finschia/ostracon/p2p"
	"github.com/Finschia/ostracon/p2p/conn"
)

type Reactor struct {
	p2p.BaseReactor

	Channels []*conn.ChannelDescriptor
}

func NewReactor() *Reactor {
	r := &Reactor{}
	r.BaseReactor = *p2p.NewBaseReactor("Mock-PEX", r, true, 1000)
	r.SetLogger(log.TestingLogger())
	return r
}

func (r *Reactor) GetChannels() []*conn.ChannelDescriptor            { return r.Channels }
func (r *Reactor) AddPeer(peer p2p.Peer)                             {}
func (r *Reactor) RemovePeer(peer p2p.Peer, reason interface{})      {}
func (r *Reactor) Receive(chID byte, peer p2p.Peer, msgBytes []byte) {}
