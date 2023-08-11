package internal

import "net"

// NullObject is null object pattern. It does nothing
type NullObject struct {
}

func NewNullObject() NullObject {
	return NullObject{}
}

func (n NullObject) Filter(addr net.Addr) net.Addr {
	return addr
}

func (n NullObject) String() string {
	return "NullObject"
}
