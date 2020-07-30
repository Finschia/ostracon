package consensus

import (
	"crypto/ed25519"
	"testing"

	"github.com/tendermint/tendermint/crypto/bls"

	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/libs/log"
)

func startConsensusAndMakeBlocks(t *testing.T, nPeers, nVals, nValsWithComposite int) []*State {
	css, _, _, cleanup := consensusNetWithPeers(
		nVals,
		nPeers,
		t.Name(),
		newMockTickerFunc(true),
		newPersistentKVStoreWithPath,
		nValsWithComposite)

	defer cleanup()
	logger := log.TestingLogger()

	reactors, blocksSubs, eventBuses := startConsensusNet(t, css, nPeers)
	defer stopConsensusNet(logger, reactors, eventBuses)

	// map of active validators
	activeVals := make(map[string]struct{})
	for i := 0; i < nVals; i++ {
		pubKey, err := css[i].privValidator.GetPubKey()
		require.NoError(t, err)
		activeVals[string(pubKey.Address())] = struct{}{}
	}

	// wait till everyone makes block 1
	timeoutWaitGroup(t, nPeers, func(j int) {
		<-blocksSubs[j].Out()
	}, css)

	// wait till everyone makes block 2
	waitForAndValidateBlock(t, nPeers, activeVals, blocksSubs, css)

	return css
}

func TestAggregateSignature(t *testing.T) {
	const (
		nPeers             = 4
		nVals              = 4
		nValsWithComposite = 0
	)
	css := startConsensusAndMakeBlocks(t, nPeers, nVals, nValsWithComposite)
	for _, state := range css {
		block := state.blockStore.LoadBlock(2)

		// validators are ed25519 only
		for _, comsig := range block.LastCommit.Signatures {
			require.EqualValues(t, ed25519.PrivateKeySize, len(comsig.Signature))
		}
		require.EqualValues(t, nVals, len(block.LastCommit.Signatures))
		require.Nil(t, block.LastCommit.AggregatedSignature)
	}
}

func TestAggregateSignatureWithComposite(t *testing.T) {
	const (
		nPeers             = 4
		nVals              = 4
		nValsWithComposite = 4
	)
	css := startConsensusAndMakeBlocks(t, nPeers, nVals, nValsWithComposite)

	for _, state := range css {
		block := state.blockStore.LoadBlock(2)
		// validators are composite only
		for _, comsig := range block.LastCommit.Signatures {
			require.Nil(t, comsig.Signature)
		}
		require.EqualValues(t, nVals, len(block.LastCommit.Signatures))
		require.EqualValues(t, bls.SignatureSize, len(block.LastCommit.AggregatedSignature))
	}
}

func TestAggregateSignatureWithMix(t *testing.T) {
	const (
		nPeers               = 4
		nVals                = 4
		nValsWithComposite   = 2
		expectedCntNotNilSig = nVals - nValsWithComposite
	)
	css := startConsensusAndMakeBlocks(t, nPeers, nVals, nValsWithComposite)

	for _, state := range css {
		block := state.blockStore.LoadBlock(2)
		// composite and ed25519 validators
		cnt := 0
		for _, comsig := range block.LastCommit.Signatures {
			if comsig.Signature != nil {
				cnt++
			}
		}
		// count the unaggregated sig
		require.EqualValues(t, expectedCntNotNilSig, cnt)
		// count the aggregated sig
		require.EqualValues(t, nValsWithComposite, len(block.LastCommit.Signatures)-cnt)
	}
}
