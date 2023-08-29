package internal

import (
	"github.com/stretchr/testify/assert"
	"net"
	"strings"
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
		expected   net.Addr
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
				expected   net.Addr
			}{"127.0.0.1", addrStub{"127.0.0.1:45678"}, addrStub{"127.0.0.1:45678"}},
		},
		{
			"should not allow different ip",
			struct {
				allowIP    string
				remoteAddr net.Addr
				expected   net.Addr
			}{"127.0.0.1", addrStub{"10.0.0.2:45678"}, nil},
		},
		{
			"should works for IPv6 with correct ip",
			struct {
				allowIP    string
				remoteAddr net.Addr
				expected   net.Addr
			}{"2001:db8::1", addrStub{"[2001:db8::1]:80"}, addrStub{"[2001:db8::1]:80"}},
		},
		{
			"should works for IPv6 with incorrect ip",
			struct {
				allowIP    string
				remoteAddr net.Addr
				expected   net.Addr
			}{"2001:db8::2", addrStub{"[2001:db8::1]:80"}, nil},
		},
		{
			"empty allowIP should deny all",
			struct {
				allowIP    string
				remoteAddr net.Addr
				expected   net.Addr
			}{"", addrStub{"127.0.0.1:45678"}, nil},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cut := NewIpFilter([]string{tt.fields.allowIP}, nil)
			assert.Equalf(t, tt.fields.expected, cut.Filter(tt.fields.remoteAddr), tt.name)
		})
	}
}

func TestFilterRemoteConnectionByIPWithMultipleAllowIPs(t *testing.T) {
	type fields struct {
		allowList  []string
		remoteAddr net.Addr
		expected   net.Addr
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			"should allow the one in the allow list",
			struct {
				allowList  []string
				remoteAddr net.Addr
				expected   net.Addr
			}{[]string{"127.0.0.1", "192.168.1.1"}, addrStub{"192.168.1.1:45678"}, addrStub{"192.168.1.1:45678"}},
		},
		{
			"should not allow any ip which is not in the allow list",
			struct {
				allowList  []string
				remoteAddr net.Addr
				expected   net.Addr
			}{[]string{"127.0.0.1", "192.168.1.1"}, addrStub{"10.0.0.2:45678"}, nil},
		},
		{
			"should works for IPv6 with one of correct ip in the allow list",
			struct {
				allowList  []string
				remoteAddr net.Addr
				expected   net.Addr
			}{[]string{"2001:db8::1", "2001:db8::2"}, addrStub{"[2001:db8::1]:80"}, addrStub{"[2001:db8::1]:80"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cut := NewIpFilter(tt.fields.allowList, nil)
			assert.Equalf(t, tt.fields.expected, cut.Filter(tt.fields.remoteAddr), tt.name)
		})
	}
}

func TestIpFilterShouldSetAllowAddress(t *testing.T) {
	expected := []string{"192.168.0.1"}

	cut := NewIpFilter(expected, nil)

	assert.Equal(t, expected, cut.allowList)
}

func TestIpFilterStringShouldReturnsIP(t *testing.T) {
	expected := []string{"127.0.0.1", "192.168.1.10"}
	assert.Equal(t, strings.Join(expected, ","), NewIpFilter(expected, nil).String())
}
