package main

import (
	"github.com/stretchr/testify/require"
	"math/rand"
	"testing"
)

func TestGenerate(t *testing.T) {
	testcases := []struct {
		name    string
		version string
	}{
		{
			name:    "empty version",
			version: "",
		},
		{
			name:    "specify version",
			version: "2",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			manifests, err := Generate(rand.New(rand.NewSource(randomSeed)), tc.version)
			require.NoError(t, err)
			require.NotNil(t, manifests)
		})
	}
}
