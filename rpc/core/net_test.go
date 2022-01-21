package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cfg "github.com/line/ostracon/config"
	"github.com/line/ostracon/libs/log"
	"github.com/line/ostracon/p2p"
	rpctypes "github.com/line/ostracon/rpc/jsonrpc/types"
)

func TestUnsafeDialSeeds(t *testing.T) {
	sw := p2p.MakeSwitch(cfg.DefaultP2PConfig(), 1, "testing", "123.123.123",
		func(n int, sw *p2p.Switch, config *cfg.P2PConfig) *p2p.Switch { return sw })
	err := sw.Start()
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := sw.Stop(); err != nil {
			t.Error(err)
		}
	})

	env.Logger = log.TestingLogger()
	env.P2PPeers = sw

	testCases := []struct {
		seeds []string
		isErr bool
	}{
		{[]string{}, true},
		{[]string{"d51fb70907db1c6c2d5237e78379b25cf1a37ab4@127.0.0.1:41198"}, false},
		{[]string{"127.0.0.1:41198"}, true},
	}

	for _, tc := range testCases {
		res, err := UnsafeDialSeeds(&rpctypes.Context{}, tc.seeds)
		if tc.isErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.NotNil(t, res)
		}
	}
}

func TestUnsafeDialPeers(t *testing.T) {
	sw := p2p.MakeSwitch(cfg.DefaultP2PConfig(), 1, "testing", "123.123.123",
		func(n int, sw *p2p.Switch, config *cfg.P2PConfig) *p2p.Switch { return sw })
	sw.SetAddrBook(&p2p.AddrBookMock{
		Addrs:        make(map[string]struct{}),
		OurAddrs:     make(map[string]struct{}),
		PrivateAddrs: make(map[string]struct{}),
	})
	err := sw.Start()
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := sw.Stop(); err != nil {
			t.Error(err)
		}
	})

	env.Logger = log.TestingLogger()
	env.P2PPeers = sw

	testCases := []struct {
		peers                               []string
		persistence, unconditional, private bool
		isErr                               bool
	}{
		{[]string{}, false, false, false, true},
		{[]string{"d51fb70907db1c6c2d5237e78379b25cf1a37ab4@127.0.0.1:41198"}, true, true, true, false},
		{[]string{"127.0.0.1:41198"}, true, true, false, true},
	}

	for _, tc := range testCases {
		res, err := UnsafeDialPeers(&rpctypes.Context{}, tc.peers, tc.persistence, tc.unconditional, tc.private)
		if tc.isErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.NotNil(t, res)
		}
	}
}

func TestGenesis(t *testing.T) {
	env = &Environment{}

	// success
	env.genChunks = []string{}
	res, err := Genesis(&rpctypes.Context{})
	assert.NoError(t, err)
	assert.NotNil(t, res)

	// error
	env.genChunks = []string{"", ""}
	res, err = Genesis(&rpctypes.Context{})
	assert.Error(t, err)
	assert.Equal(t, "genesis response is large, please use the genesis_chunked API instead", err.Error())
	assert.Nil(t, res)
}

func TestGenesisChunked(t *testing.T) {
	env = &Environment{}

	// success
	env.genChunks = []string{""}
	chunk := uint(0)
	res, err := GenesisChunked(&rpctypes.Context{}, chunk)
	assert.NoError(t, err)
	assert.NotNil(t, res)

	//
	// errors
	//

	env.genChunks = nil
	chunk = uint(0)
	res, err = GenesisChunked(&rpctypes.Context{}, chunk)
	assert.Error(t, err)
	assert.Equal(t, "service configuration error, genesis chunks are not initialized", err.Error())
	assert.Nil(t, res)

	env.genChunks = []string{}
	chunk = uint(0)
	res, err = GenesisChunked(&rpctypes.Context{}, chunk)
	assert.Error(t, err)
	assert.Equal(t, "service configuration error, there are no chunks", err.Error())
	assert.Nil(t, res)

	env.genChunks = []string{""}
	chunk = uint(1)
	res, err = GenesisChunked(&rpctypes.Context{}, chunk)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "there are ")
	assert.Contains(t, err.Error(), " is invalid")
	assert.Nil(t, res)
}
