package http_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/line/ostracon/abci/example/counter"

	coretypes "github.com/line/ostracon/rpc/core/types"
	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/line/ostracon/abci/example/kvstore"
	"github.com/line/ostracon/light/provider"
	lighthttp "github.com/line/ostracon/light/provider/http"
	rpcclient "github.com/line/ostracon/rpc/client"
	rpchttp "github.com/line/ostracon/rpc/client/http"
	rpcmocks "github.com/line/ostracon/rpc/client/mocks"
	rpcjson "github.com/line/ostracon/rpc/jsonrpc/client"
	rpctest "github.com/line/ostracon/rpc/test"
	"github.com/line/ostracon/types"
)

func TestNewProvider(t *testing.T) {
	c, err := lighthttp.New("chain-test", "192.168.0.1:26657")
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("%s", c), "http{http://192.168.0.1:26657}")

	c, err = lighthttp.New("chain-test", "http://153.200.0.1:26657")
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("%s", c), "http{http://153.200.0.1:26657}")

	c, err = lighthttp.New("chain-test", "153.200.0.1")
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("%s", c), "http{http://153.200.0.1}")
}

func TestProvider(t *testing.T) {
	app := kvstore.NewApplication()
	app.RetainBlocks = 10
	node := rpctest.StartOstracon(app)

	cfg := rpctest.GetConfig()
	defer os.RemoveAll(cfg.RootDir)
	rpcAddr := cfg.RPC.ListenAddress
	genDoc, err := types.GenesisDocFromFile(cfg.GenesisFile())
	require.NoError(t, err)
	chainID := genDoc.ChainID

	jsonClient, err := rpcjson.HTTPClientForTest(rpcAddr)
	require.Nil(t, err)
	c, err := rpchttp.NewWithClient(rpcAddr, "/websocket", jsonClient)
	require.Nil(t, err)

	p := lighthttp.NewWithClient(chainID, c)
	require.NoError(t, err)
	require.NotNil(t, p)

	// let it produce some blocks
	err = rpcclient.WaitForHeight(c, 10, nil)
	require.NoError(t, err)

	// let's get the highest block
	lb, err := p.LightBlock(context.Background(), 0)
	require.NoError(t, err)
	require.NotNil(t, lb)
	assert.True(t, lb.Height < 1000)

	// let's check this is valid somehow
	assert.Nil(t, lb.ValidateBasic(chainID))

	// historical queries now work :)
	lower := lb.Height - 3
	lb, err = p.LightBlock(context.Background(), lower)
	require.NoError(t, err)
	assert.Equal(t, lower, lb.Height)

	// fetching missing heights (both future and pruned) should return appropriate errors
	lb, err = p.LightBlock(context.Background(), 1000)
	require.Error(t, err)
	require.Nil(t, lb)
	assert.Equal(t, provider.ErrHeightTooHigh, err)

	_, err = p.LightBlock(context.Background(), 1)
	require.Error(t, err)
	require.Nil(t, lb)
	assert.Equal(t, provider.ErrLightBlockNotFound, err)

	// stop the full node and check that a no response error is returned
	rpctest.StopOstracon(node)
	time.Sleep(1 * time.Second) // depends on Transport.IdleConnTimeout
	lb, err = p.LightBlock(context.Background(), lower+2)
	// we should see a connection refused
	require.Error(t, err)
	require.Contains(t, err.Error(), "connection refused")
	require.Nil(t, lb)
}

func TestProviderWithErrors(t *testing.T) {
	height := int64(1)
	c := makeMockRemoteClientForTestProviderErrors(height)
	p := makeTestProvider(t, c, height)

	// via provider.LightBlock.voterSet
	c.On("ValidatorsWithVoters", mock.Anything, &height, mock.Anything, mock.Anything).Once().Return(
		nil, fmt.Errorf("mock: no match errors"))

	lb, err := p.LightBlock(context.Background(), height)
	require.Error(t, err)
	require.Equal(t, "mock: no match errors", err.Error())
	require.Nil(t, lb)

	// via provider.LightBlock.voterSet
	c.On("ValidatorsWithVoters", mock.Anything, &height, mock.Anything, mock.Anything).Once().Return(
		nil, fmt.Errorf("mock: is not available"))

	lb, err = p.LightBlock(context.Background(), height)
	require.Error(t, err)
	require.Contains(t, err.Error(), "is not available")
	require.Nil(t, lb)

	// via provider.LightBlock.voterSet
	c.On("ValidatorsWithVoters", mock.Anything, &height, mock.Anything, mock.Anything).Once().Return(
		nil, fmt.Errorf("mock: must be less than or equal to"))

	lb, err = p.LightBlock(context.Background(), height)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be less than or equal to")
	require.Nil(t, lb)

	// via provider.LightBlock.voterSet
	c.On("ValidatorsWithVoters", mock.Anything, &height, mock.Anything, mock.Anything).Times(5).Return(
		nil, fmt.Errorf("mock: Timeout exceeded"))

	lb, err = p.LightBlock(context.Background(), height)
	require.Error(t, err)
	require.Contains(t, err.Error(), "client failed to respond")
	require.Nil(t, lb)

	// via provider.LightBlock.voterSet
	var validators []*types.Validator
	c.On("ValidatorsWithVoters", mock.Anything, &height, mock.Anything, mock.Anything).Once().Return(
		&coretypes.ResultValidatorsWithVoters{Validators: validators}, nil)

	lb, err = p.LightBlock(context.Background(), height)
	require.Error(t, err)
	require.Contains(t, err.Error(), "client provided bad signed header: validator set is empty")
	require.Nil(t, lb)

	// via provider.LightBlock.voterSet
	validatorSet, _, _ := types.RandVoterSet(4, 10)
	validators = validatorSet.Validators
	c.On("ValidatorsWithVoters", mock.Anything, &height, mock.Anything, mock.Anything).Once().Return(
		&coretypes.ResultValidatorsWithVoters{Validators: validators, Total: 0}, nil)

	lb, err = p.LightBlock(context.Background(), height)
	require.Error(t, err)
	require.Contains(t, err.Error(), "client provided bad signed header: total number of vals is <=")
	require.Nil(t, lb)
}

func makeMockRemoteClientForTestProviderErrors(height int64) *rpcmocks.RemoteClient {
	c := &rpcmocks.RemoteClient{}

	// rpcclient.WaitForHeight
	c.On("Status", mock.Anything).Return(
		&coretypes.ResultStatus{SyncInfo: coretypes.SyncInfo{LatestBlockHeight: height}}, nil)

	// via provider.LightBlock.signedHeader
	c.On("Commit", mock.Anything, &height).Once().Return(
		nil, fmt.Errorf("mock: Timeout exceeded")) // one error for code coverage
	c.On("Commit", mock.Anything, &height).Return(
		&coretypes.ResultCommit{SignedHeader: types.SignedHeader{Header: &types.Header{Height: height}}}, nil)

	return c
}

func makeTestProvider(t *testing.T, c *rpcmocks.RemoteClient, height int64) provider.Provider {
	cfg := rpctest.GetConfig(true)
	defer os.RemoveAll(cfg.RootDir)

	app := counter.NewApplication(false)
	node := rpctest.StartOstracon(app)
	defer rpctest.StopOstracon(node)

	genDoc, err := types.GenesisDocFromFile(cfg.GenesisFile())
	require.NoError(t, err)
	chainID := genDoc.ChainID

	p := lighthttp.NewWithClient(chainID, c)
	require.NoError(t, err)
	require.NotNil(t, p)

	// let it produce some blocks
	err = rpcclient.WaitForHeight(c, height, nil)
	require.NoError(t, err)

	return p
}
