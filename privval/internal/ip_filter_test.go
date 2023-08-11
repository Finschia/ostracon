package internal

import (
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

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
			"should works for IPv6 with correct ip",
			struct {
				allowIP    string
				remoteAddr net.Addr
				expected   bool
			}{"2001:db8::1", addrStub{"[2001:db8::1]:80"}, true},
		},
		{
			"should works for IPv6 with incorrect ip",
			struct {
				allowIP    string
				remoteAddr net.Addr
				expected   bool
			}{"2001:db8::2", addrStub{"[2001:db8::1]:80"}, false},
		},
		{
			"empty allowIP should deny all",
			struct {
				allowIP    string
				remoteAddr net.Addr
				expected   bool
			}{"", addrStub{"127.0.0.1:45678"}, false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cut := NewIpFilter(tt.fields.allowIP, nil)
			assert.Equalf(t, tt.fields.expected, cut.isAllowedAddr(tt.fields.remoteAddr), tt.name)
		})
	}
}

func TestIpFilterShouldSetAllowAddress(t *testing.T) {
	expected := "192.168.0.1"

	cut := NewIpFilter(expected, nil)

	assert.Equal(t, expected, cut.allowAddr)
}
