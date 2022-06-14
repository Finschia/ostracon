package commands

import (
	"bytes"
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/line/ostracon/types"

	cfg "github.com/line/ostracon/config"
	"github.com/line/ostracon/crypto"
	tmjson "github.com/line/ostracon/libs/json"
	tmos "github.com/line/ostracon/libs/os"
	"github.com/line/ostracon/privval"
	"github.com/stretchr/testify/require"
)

func TestShowValidator(t *testing.T) {
	original := config
	defer func() {
		config = original
	}()

	setupEnv(t)
	config = cfg.DefaultConfig()
	err := RootCmd.PersistentPreRunE(RootCmd, nil)
	require.NoError(t, err)
	init := NewInitCmd()
	err = init.RunE(init, nil)
	require.NoError(t, err)
	output, err := captureStdout(func() {
		err = ShowValidatorCmd.RunE(ShowValidatorCmd, nil)
		require.NoError(t, err)
	})
	require.NoError(t, err)

	// output must match the locally stored priv_validator key
	privKey := loadFilePVKey(t, config.PrivValidatorKeyFile())
	bz, err := tmjson.Marshal(privKey.PubKey)
	require.NoError(t, err)
	require.Equal(t, string(bz), output)
}

func TestShowValidatorWithoutLocalKeyFile(t *testing.T) {
	setupEnv(t)
	config := cfg.DefaultConfig()
	if tmos.FileExists(config.PrivValidatorKeyFile()) {
		err := os.Remove(config.PrivValidatorKeyFile())
		require.NoError(t, err)
	}
	err := showValidator(ShowValidatorCmd, nil, config)
	require.Error(t, err)
}

func TestShowValidatorWithKMS(t *testing.T) {
	dir := setupEnv(t)
	cfg.EnsureRoot(dir)

	original := config
	defer func() {
		config = original
	}()

	config = cfg.DefaultConfig()
	config.SetRoot(dir)
	err := RootCmd.PersistentPreRunE(RootCmd, nil)
	require.NoError(t, err)
	init := NewInitCmd()
	err = init.RunE(init, nil)
	require.NoError(t, err)

	chainID, err := loadChainID(config)
	require.NoError(t, err)

	if tmos.FileExists(config.PrivValidatorKeyFile()) {
		err := os.Remove(config.PrivValidatorKeyFile())
		require.NoError(t, err)
	}
	privval.WithMockKMS(t, dir, chainID, func(addr string, privKey crypto.PrivKey) {
		config.PrivValidatorListenAddr = addr
		require.NoFileExists(t, config.PrivValidatorKeyFile())
		output, err := captureStdout(func() {
			err := showValidator(ShowValidatorCmd, nil, config)
			require.NoError(t, err)
		})
		require.NoError(t, err)

		// output must contains the KMS public key
		bz, err := tmjson.Marshal(privKey.PubKey())
		require.NoError(t, err)
		expected := string(bz)
		require.Contains(t, output, expected)
	})
}

func TestShowValidatorWithInefficientKMSAddress(t *testing.T) {
	dir := setupEnv(t)
	cfg.EnsureRoot(dir)

	original := config
	defer func() {
		config = original
	}()

	config = cfg.DefaultConfig()
	config.SetRoot(dir)
	err := RootCmd.PersistentPreRunE(RootCmd, nil)
	require.NoError(t, err)
	init := NewInitCmd()
	err = init.RunE(init, nil)
	require.NoError(t, err)

	if tmos.FileExists(config.PrivValidatorKeyFile()) {
		err := os.Remove(config.PrivValidatorKeyFile())
		require.NoError(t, err)
	}
	config.PrivValidatorListenAddr = "127.0.0.1:inefficient"
	err = showValidator(ShowValidatorCmd, nil, config)
	require.Error(t, err)
}

func TestLoadChainID(t *testing.T) {
	expected := "c57861"
	config := cfg.ResetTestRootWithChainID("TestLoadChainID", expected)
	defer func() {
		var _ = os.RemoveAll(config.RootDir)
	}()

	require.FileExists(t, config.GenesisFile())
	genDoc, err := types.GenesisDocFromFile(config.GenesisFile())
	require.NoError(t, err)
	require.Equal(t, expected, genDoc.ChainID)

	chainID, err := loadChainID(config)
	require.NoError(t, err)
	require.Equal(t, expected, chainID)
}

func TestLoadChainIDWithoutStateDB(t *testing.T) {
	expected := "c34091"
	config := cfg.ResetTestRootWithChainID("TestLoadChainID", expected)
	defer func() {
		var _ = os.RemoveAll(config.RootDir)
	}()

	config.DBBackend = "goleveldb"
	config.DBPath = "/../path with containing chars that cannot be used\\/:*?\"<>|\x00"

	_, err := loadChainID(config)
	require.Error(t, err)
}

func loadFilePVKey(t *testing.T, file string) privval.FilePVKey {
	// output must match the locally stored priv_validator key
	keyJSONBytes, err := ioutil.ReadFile(file)
	require.NoError(t, err)
	privKey := privval.FilePVKey{}
	err = tmjson.Unmarshal(keyJSONBytes, &privKey)
	require.NoError(t, err)
	return privKey
}

var stdoutMutex sync.Mutex

func captureStdout(f func()) (string, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}

	stdoutMutex.Lock()
	original := os.Stdout
	defer func() {
		stdoutMutex.Lock()
		os.Stdout = original
		stdoutMutex.Unlock()
	}()
	os.Stdout = w
	stdoutMutex.Unlock()

	f()
	_ = w.Close()
	var buffer bytes.Buffer
	if _, err := buffer.ReadFrom(r); err != nil {
		return "", err
	}
	output := buffer.String()
	return output[:len(output)-1], nil
}
