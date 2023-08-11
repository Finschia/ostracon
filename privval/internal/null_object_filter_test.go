package internal

import (
	"net"
	"reflect"
	"testing"
)

func TestNullObject_filter(t *testing.T) {
	stubInput := addrStub{}
	tests := []struct {
		name string
		addr net.Addr
		want net.Addr
	}{
		{
			name: "null object does nothing, returns what it receives",
			addr: stubInput,
			want: stubInput,
		},
		{
			name: "null object does nothing, returns nil it receives nil",
			addr: nil,
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &NullObject{}
			if got := n.Filter(tt.addr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Filter() = %v, want %v", got, tt.want)
			}
		})
	}
}
