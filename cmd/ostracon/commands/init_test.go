package commands

import (
	"fmt"
	"net"
	"os"
	"path"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	cfg "github.com/line/ostracon/config"
	"github.com/line/ostracon/crypto/ed25519"
	"github.com/line/ostracon/privval"

	"github.com/stretchr/testify/require"
)

func TestInitCmd(t *testing.T) {
	setupEnv(t)
	err := RootCmd.PersistentPreRunE(RootCmd, nil)
	require.NoError(t, err)
	cmd := NewInitCmd()
	err = cmd.RunE(cmd, nil)
	require.NoError(t, err)
}

func TestInitCmdWithKMSOptions(t *testing.T) {
	dir := setupEnv(t)
	config := cfg.TestConfig()
	config.SetRoot(dir)
	cfg.EnsureRoot(dir)
	_ = os.Remove(config.Path()) // config.toml must be generated newly

	WithMockKMS(t, dir, config.ChainID(), func(expected string) {
		config.PrivValidatorListenAddr = expected
		err := RootCmd.PersistentPreRunE(RootCmd, nil)
		require.NoError(t, err)
		require.NoError(t, initFilesWithConfig(config))
		require.NoError(t, err)
		require.NoFileExists(t, config.PrivValidatorKeyFile())
		settingsMustBeContained(t, config.Path(), "priv_validator_laddr", expected)
	})
}

func WithMockKMS(t *testing.T, dir, chainID string, f func(string)) {
	// refer to the source cmd/priv_validator_server/main.go

	// obtain an address using a vacancy port number
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
		return
	}
	addr := listener.Addr().String()
	if err = listener.Close(); err != nil {
		t.Fatal(err)
		return
	}

	// start mock kms server
	shutdown := make(chan string)
	go func() {
		logger.Info(fmt.Sprintf("MockKMS starting: [%s] %s", chainID, addr))
		privKey := ed25519.GenPrivKeyFromSecret([]byte("🏺"))
		pv := privval.NewFilePV(privKey, path.Join(dir, "keyfile"), path.Join(dir, "statefile"))
		connTimeout := 5 * time.Second
		dialer := privval.DialTCPFn(addr, connTimeout, ed25519.GenPrivKeyFromSecret([]byte("🔌")))
		sd := privval.NewSignerDialerEndpoint(logger, dialer)
		ss := privval.NewSignerServer(sd, chainID, pv)
		err := ss.Start()
		if err != nil {
			panic(err)
		}
		logger.Info("MockKMS started")
		<-shutdown
		logger.Info("MockKMS stopping")
		if err = ss.Stop(); err != nil {
			panic(err)
		}
		logger.Info("MockKMS stopped")
	}()
	defer func() {
		shutdown <- "SHUTDOWN"
	}()

	f(addr)
}

func settingsMustBeContained(t *testing.T, file string, key, value string) {
	var config map[string]interface{}
	_, err := toml.DecodeFile(file, &config)
	require.NoError(t, err)
	require.Equal(t, value, config[key])
}

//func TestResetStateCmd(t *testing.T) {
//	setupEnv(t)
//	err := ResetStateCmd.RunE(ResetStateCmd, nil)
//	require.NoError(t, err)
//}
//
//func TestResetPrivValidatorCmd(t *testing.T) {
//	setupEnv(t)
//	err := ResetPrivValidatorCmd.RunE(ResetPrivValidatorCmd, nil)
//	require.NoError(t, err)
//}
//func Test_ResetAll(t *testing.T) {
//	config := cfg.TestConfig()
//	dir := t.TempDir()
//	config.SetRoot(dir)
//	cfg.EnsureRoot(dir)
//	require.NoError(t, initFilesWithConfig(config))
//	pv := privval.LoadFilePV(config.PrivValidatorKeyFile(), config.PrivValidatorStateFile())
//	pv.LastSignState.Height = 10
//	pv.Save()
//	require.NoError(t, resetAll(config.DBDir(), config.P2P.AddrBookFile(), config.PrivValidatorKeyFile(),
//		config.PrivValidatorStateFile(), config.PrivKeyType, logger))
//	require.DirExists(t, config.DBDir())
//	require.NoFileExists(t, filepath.Join(config.DBDir(), "block.db"))
//	require.NoFileExists(t, filepath.Join(config.DBDir(), "state.db"))
//	require.NoFileExists(t, filepath.Join(config.DBDir(), "evidence.db"))
//	require.NoFileExists(t, filepath.Join(config.DBDir(), "tx_index.db"))
//	require.FileExists(t, config.PrivValidatorStateFile())
//	pv = privval.LoadFilePV(config.PrivValidatorKeyFile(), config.PrivValidatorStateFile())
//	require.Equal(t, int64(0), pv.LastSignState.Height)
//}
//
//func Test_ResetState(t *testing.T) {
//	config := cfg.TestConfig()
//	dir := t.TempDir()
//	config.SetRoot(dir)
//	cfg.EnsureRoot(dir)
//	require.NoError(t, initFilesWithConfig(config))
//	pv := privval.LoadFilePV(config.PrivValidatorKeyFile(), config.PrivValidatorStateFile())
//	pv.LastSignState.Height = 10
//	pv.Save()
//	require.NoError(t, resetState(config.DBDir(), logger))
//	require.DirExists(t, config.DBDir())
//	require.NoFileExists(t, filepath.Join(config.DBDir(), "block.db"))
//	require.NoFileExists(t, filepath.Join(config.DBDir(), "state.db"))
//	require.NoFileExists(t, filepath.Join(config.DBDir(), "evidence.db"))
//	require.NoFileExists(t, filepath.Join(config.DBDir(), "tx_index.db"))
//	require.FileExists(t, config.PrivValidatorStateFile())
//	pv = privval.LoadFilePV(config.PrivValidatorKeyFile(), config.PrivValidatorStateFile())
//	// private validator state should still be in tact.
//	require.Equal(t, int64(10), pv.LastSignState.Height)
//}
