package kvstore

import (
	"io/ioutil"
	"testing"

	"github.com/line/ostracon/privval"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadPrivValidatorKeyFile(t *testing.T) {
	tempKeyFile, err := ioutil.TempFile("", "priv_validator_key_")
	require.Nil(t, err)
	tempStateFile, err := ioutil.TempFile("", "priv_validator_state_")
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

	expected, err := privval.GenFilePV(tempKeyFile.Name(), tempStateFile.Name(), privval.PrivKeyTypeEd25519)
	require.Nil(t, err)

	expected.Save()

	// success
	actual, err := LoadPrivValidatorKeyFile(tempKeyFile.Name())
	require.Nil(t, err)
	assert.Equal(t, expected.Key.Address, actual.Address)
	assert.Equal(t, expected.Key.PrivKey, actual.PrivKey)
	assert.Equal(t, expected.Key.PubKey, actual.PubKey)
}
