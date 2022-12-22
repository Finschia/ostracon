package main

import (
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
)

func TestNewCLI(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "generator")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir) //nolint:staticcheck
	cmd := NewCLI()
	testcases := []struct {
		name    string
		wantErr bool
		args    []string
	}{
		{
			name:    "default",
			wantErr: true,
			args: []string{
				"-d", tempDir,
			},
		},
		{
			name:    "specify groups",
			wantErr: true,
			args: []string{
				"-d", tempDir,
				"-g", "1",
			},
		},
		{
			name:    "specify version",
			wantErr: true,
			args: []string{
				"-d", tempDir,
				"-m", "1",
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			err := cmd.root.ParseFlags(tc.args)
			require.NoError(t, err)
			cmd.Run()
		})
	}
}
