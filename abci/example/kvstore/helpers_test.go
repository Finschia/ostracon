package kvstore

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Finschia/ostracon/privval"
)

func TestLoadPrivValidatorKeyFile(t *testing.T) {
	tempKeyFile, err := os.CreateTemp("", "priv_validator_key_")
	require.Nil(t, err)
	tempStateFile, err := os.CreateTemp("", "priv_validator_state_")
	require.Nil(t, err)

	{
		// does not exist
		_, err := LoadPrivValidatorKeyFile("DOES_NOT_EXIST")
		require.NotNil(t, err)
		require.Contains(t, err.Error(), "does not exist")
	}

	{
		// error reading since empty
		_, err := LoadPrivValidatorKeyFile(tempKeyFile.Name())
		require.NotNil(t, err)
		require.Contains(t, err.Error(), "error reading")
	}

	expected := privval.GenFilePV(tempKeyFile.Name(), tempStateFile.Name())

	expected.Save()

	// success
	actual, err := LoadPrivValidatorKeyFile(tempKeyFile.Name())
	require.Nil(t, err)
	assert.Equal(t, expected.Key.Address, actual.Address)
	assert.Equal(t, expected.Key.PrivKey, actual.PrivKey)
	assert.Equal(t, expected.Key.PubKey, actual.PubKey)
}
