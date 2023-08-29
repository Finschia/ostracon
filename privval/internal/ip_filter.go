package internal

import (
	"fmt"
	"github.com/Finschia/ostracon/libs/log"
	"net"
	"strings"
)

type IpFilter struct {
	allowList []string
	log       log.Logger
}

func NewIpFilter(allowAddresses []string, l log.Logger) *IpFilter {
	return &IpFilter{
		allowList: allowAddresses,
		log:       l,
	}
}

func (f *IpFilter) Filter(addr net.Addr) net.Addr {
	if f.isAllowedAddr(addr) {
		return addr
	}
	return nil
}

func (f *IpFilter) String() string {
	return strings.Join(f.allowList, ",")
}

func (f *IpFilter) isAllowedAddr(addr net.Addr) bool {
	if len(f.allowList) == 0 {
		return false
	}
	hostAddr, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		if f.log != nil {
			f.log.Error(fmt.Sprintf("IpFilter: can't split host and port from addr.String()=%s", addr.String()))
		}
		return false
	}
	for _, address := range f.allowList {
		if address == hostAddr {
			return true
		}
	}
	return false
}
