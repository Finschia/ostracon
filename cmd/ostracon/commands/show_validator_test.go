package commands

import (
	"bytes"
	"io/ioutil"
	"os"
	"sync"
	"testing"

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

func TestShowValidatorWithKMS(t *testing.T) {
	original := config
	defer func() {
		config = original
	}()

	dir := setupEnv(t)
	config = cfg.DefaultConfig()
	if tmos.FileExists(config.PrivValidatorKeyFile()) {
		err := os.Remove(config.PrivValidatorKeyFile())
		require.NoError(t, err)
	}
	WithMockKMS(t, dir, config.ChainID(), func(addr string, privKey crypto.PrivKey) {
		config.PrivValidatorListenAddr = addr
		require.NoFileExists(t, config.PrivValidatorKeyFile())
		output, err := captureStdout(func() {
			err := ShowValidatorCmd.RunE(ShowValidatorCmd, nil)
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
		os.Stdout = original
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
