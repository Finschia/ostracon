package main

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	manifests, err := Generate(rand.New(rand.NewSource(randomSeed)))
	require.NoError(t, err)
	require.NotNil(t, manifests)
}
