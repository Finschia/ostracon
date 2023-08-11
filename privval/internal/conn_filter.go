package internal

import "net"

type ConnectionFilter interface {
	Filter(addr net.Addr) net.Addr
	String() string
}
