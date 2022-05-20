package commands

import (
	"testing"

	cfg "github.com/line/ostracon/config"

	"github.com/line/ostracon/crypto"
	nm "github.com/line/ostracon/node"
	"github.com/line/ostracon/types"
	"github.com/stretchr/testify/require"
)

// RunCmd.RunE can only be stopped by a signal for process. For this reason, NewOstraconNode() is called directly
// to test.
func TestNewOstraconNode(t *testing.T) {
	original := config
	defer func() {
		config = original
	}()

	// setup environment
	dir := setupEnv(t)
	config = cfg.TestConfig()
	err := RootCmd.PersistentPreRunE(RootCmd, nil)
	require.NoError(t, err)
	config.SetRoot(dir)
	cfg.EnsureRoot(dir)
	init := NewInitCmd()
	err = init.RunE(init, nil)
	require.NoError(t, err)

	// start node
	config.ProxyApp = "noop"
	node, err := nm.NewOstraconNode(config, logger)
	require.NoError(t, err)

	// verify the retrieved public key matches the local public key
	expected := loadFilePVKey(t, config.PrivValidatorKeyFile()).PubKey
	actual, err := node.PrivValidator().GetPubKey()
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestNewOstraconNodeWithKMS(t *testing.T) {
	original := config
	defer func() {
		config = original
	}()

	// setup environment
	dir := setupEnv(t)
	config = cfg.TestConfig()
	err := RootCmd.PersistentPreRunE(RootCmd, nil)
	require.NoError(t, err)
	config.SetRoot(dir)
	cfg.EnsureRoot(dir)
	err = initFilesWithConfig(config)
	require.NoError(t, err)

	// retrieve chainID
	genDoc, err := types.GenesisDocFromFile(config.GenesisFile())
	require.NoError(t, err)
	chainID := genDoc.ChainID

	WithMockKMS(t, dir, chainID, func(addr string, privKey crypto.PrivKey) {

		// start node
		config.ProxyApp = "noop"
		config.PrivValidatorListenAddr = addr
		node, err := nm.NewOstraconNode(config, logger)
		require.NoError(t, err, addr)

		// verify the retrieved public key matches the remote public key
		expected := privKey.PubKey()
		actual, err := node.PrivValidator().GetPubKey()
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})
}
