package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func setupResetCmd(t *testing.T) {
	clearConfig(defaultRoot)
	config.SetRoot(defaultRoot)
	require.NoError(t, os.MkdirAll(filepath.Dir(config.PrivValidatorKeyFile()), 0755))
	require.NoError(t, os.MkdirAll(filepath.Dir(config.PrivValidatorStateFile()), 0755))
}

func TestResetAllCmd(t *testing.T) {
	setupResetCmd(t)
	err := ResetAllCmd.RunE(ResetAllCmd, nil)
	require.NoError(t, err)
}

func TestResetStateCmd(t *testing.T) {
	setupResetCmd(t)
	err := ResetStateCmd.RunE(ResetStateCmd, nil)
	require.NoError(t, err)
}

func TestResetPrivValidatorCmd(t *testing.T) {
	setupResetCmd(t)
	ResetPrivValidatorCmd.Run(ResetPrivValidatorCmd, nil)
}
