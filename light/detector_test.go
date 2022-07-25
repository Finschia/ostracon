package light_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dbm "github.com/tendermint/tm-db"

	"github.com/line/ostracon/libs/log"
	"github.com/line/ostracon/light"
	"github.com/line/ostracon/light/provider"
	mockp "github.com/line/ostracon/light/provider/mock"
	dbs "github.com/line/ostracon/light/store/db"
	"github.com/line/ostracon/types"
)

func TestLightClientAttackEvidence_Lunatic(t *testing.T) {
	// primary performs a lunatic attack
	var (
		latestHeight      = int64(10)
		valSize           = 5
		divergenceHeight  = int64(6)
		primaryHeaders    = make(map[int64]*types.SignedHeader, latestHeight)
		primaryValidators = make(map[int64]*types.ValidatorSet, latestHeight)
		primaryVoters     = make(map[int64]*types.VoterSet, latestHeight)
	)

	witnessHeaders, witnessValidators, witnessVoters, chainKeys :=
		genMockNodeWithKeys(chainID, latestHeight, valSize, 2, bTime)
	witness := mockp.New(chainID, witnessHeaders, witnessValidators, witnessVoters)

	preDivergenceHeight := divergenceHeight - 1
	for height := int64(1); height < preDivergenceHeight; height++ {
		primaryHeaders[height] = witnessHeaders[height]
		primaryValidators[height] = witnessValidators[height]
		primaryVoters[height] = witnessVoters[height]
	}
	// previous divergence height
	curKeys := chainKeys[preDivergenceHeight]
	forgedKeys := curKeys.ChangeKeys(3) // we change 3 out of the 5 validators (still 2/5 remain)
	forgedVals := forgedKeys.ToValidators(2, 0)
	header, vals, voters, _ := genMockNodeWithKey(chainID, preDivergenceHeight, nil,
		curKeys, forgedKeys, // Should specify the correct current/next keys
		nil, forgedVals, primaryHeaders[preDivergenceHeight-1],
		bTime.Add(time.Duration(preDivergenceHeight)*time.Minute),
		0, len(curKeys),
		0, nil)
	primaryHeaders[preDivergenceHeight] = header
	primaryValidators[preDivergenceHeight] = vals
	primaryVoters[preDivergenceHeight] = voters
	// after divergence height
	for height := divergenceHeight; height <= latestHeight; height++ {
		header, vals, voters, _ := genMockNodeWithKey(chainID, height, nil,
			forgedKeys, forgedKeys,
			forgedVals, forgedVals, primaryHeaders[height-1],
			bTime.Add(time.Duration(height)*time.Minute),
			0, len(forgedKeys),
			0, nil)
		primaryHeaders[height] = header
		primaryValidators[height] = vals
		primaryVoters[height] = voters
	}

	primary := mockp.New(chainID, primaryHeaders, primaryValidators, primaryVoters)

	c, err := light.NewClient(
		ctx,
		chainID,
		light.TrustOptions{
			Period: 4 * time.Hour,
			Height: 1,
			Hash:   primaryHeaders[1].Hash(),
		},
		primary,
		[]provider.Provider{witness},
		dbs.New(dbm.NewMemDB(), chainID),
		types.DefaultVoterParams(),
		light.Logger(log.TestingLogger()),
		light.MaxRetryAttempts(1),
	)
	require.NoError(t, err)

	// Check verification returns an error.
	_, err = c.VerifyLightBlockAtHeight(ctx, 10, bTime.Add(1*time.Hour))
	if assert.Error(t, err) {
		assert.Equal(t, light.ErrLightClientAttack, err)
	}

	// Check evidence was sent to both full nodes.
	evAgainstPrimary := &types.LightClientAttackEvidence{
		// after the divergence height the valset doesn't change, so we expect the evidence to be for height 10
		ConflictingBlock: &types.LightBlock{
			SignedHeader: primaryHeaders[preDivergenceHeight], // Need to specify the exact header
			ValidatorSet: primaryValidators[10],
			VoterSet:     primaryVoters[10],
		},
		CommonHeight: 4,
	}
	assert.True(t, witness.HasEvidence(evAgainstPrimary))

	evAgainstWitness := &types.LightClientAttackEvidence{
		// when forming evidence against witness we learn that the canonical chain continued to change validator sets
		// hence the conflicting block is at 7
		ConflictingBlock: &types.LightBlock{
			SignedHeader: witnessHeaders[preDivergenceHeight], // Need to specify the exact header
			ValidatorSet: witnessValidators[7],
			VoterSet:     witnessVoters[7],
		},
		CommonHeight: 4,
	}
	assert.True(t, primary.HasEvidence(evAgainstWitness))
}

func TestLightClientAttackEvidence_Equivocation(t *testing.T) {
	verificationOptions := map[string]light.Option{
		"sequential": light.SequentialVerification(),
		// "skipping":   light.SkippingVerification(light.DefaultTrustLevel), // Cannot do skipping-test
	}

	for s, verificationOption := range verificationOptions {
		t.Log("==> verification", s)

		// primary performs an equivocation attack
		var (
			latestHeight      = int64(10)
			valSize           = 5
			divergenceHeight  = int64(6)
			primaryHeaders    = make(map[int64]*types.SignedHeader, latestHeight)
			primaryValidators = make(map[int64]*types.ValidatorSet, latestHeight)
			primaryVoters     = make(map[int64]*types.VoterSet, latestHeight)
		)
		// validators don't change in this network (however we still use a map just for convenience)
		witnessHeaders, witnessValidators, witnessVoters, chainKeys :=
			genMockNodeWithKeys(chainID, latestHeight+2, valSize, 2, bTime)
		witness := mockp.New(chainID, witnessHeaders, witnessValidators, witnessVoters)

		for height := int64(1); height <= latestHeight; height++ {
			if height < divergenceHeight {
				primaryHeaders[height] = witnessHeaders[height]
				primaryValidators[height] = witnessValidators[height]
				primaryVoters[height] = witnessVoters[height]
				continue
			}
			// we don't have a network partition, so we will make 4/5 (greater than 2/3) malicious and vote again for
			// a different block (which we do by adding txs)
			header, vals, voters, _ := genMockNodeWithKey(chainID, height, []types.Tx{[]byte("abcd")},
				chainKeys[height], chainKeys[height+1],
				witnessValidators[height], witnessValidators[height+1], primaryHeaders[height-1],
				bTime.Add(time.Duration(height)*time.Minute),
				0, len(chainKeys[height])-1, // make 4/5
				0, nil)
			primaryHeaders[height] = header
			primaryValidators[height] = vals
			primaryVoters[height] = voters
		}
		primary := mockp.New(chainID, primaryHeaders, primaryValidators, primaryVoters)

		c, err := light.NewClient(
			ctx,
			chainID,
			light.TrustOptions{
				Period: 4 * time.Hour,
				Height: 1,
				Hash:   primaryHeaders[1].Hash(),
			},
			primary,
			[]provider.Provider{witness},
			dbs.New(dbm.NewMemDB(), chainID),
			types.DefaultVoterParams(),
			light.Logger(log.TestingLogger()),
			light.MaxRetryAttempts(1),
			verificationOption,
		)
		require.NoError(t, err)

		// Check verification returns an error.
		_, err = c.VerifyLightBlockAtHeight(ctx, 10, bTime.Add(1*time.Hour))
		if assert.Error(t, err) {
			assert.Equal(t, light.ErrLightClientAttack, err)
		}

		// Check evidence was sent to both full nodes.
		// Common height should be set to the height of the divergent header in the instance
		// of an equivocation attack and the validator sets are the same as what the witness has
		evAgainstPrimary := &types.LightClientAttackEvidence{
			ConflictingBlock: &types.LightBlock{
				SignedHeader: primaryHeaders[divergenceHeight],
				ValidatorSet: primaryValidators[divergenceHeight],
				VoterSet:     primaryVoters[divergenceHeight],
			},
			CommonHeight: divergenceHeight,
		}
		assert.True(t, witness.HasEvidence(evAgainstPrimary))

		evAgainstWitness := &types.LightClientAttackEvidence{
			ConflictingBlock: &types.LightBlock{
				SignedHeader: witnessHeaders[divergenceHeight],
				ValidatorSet: witnessValidators[divergenceHeight],
				VoterSet:     witnessVoters[divergenceHeight],
			},
			CommonHeight: divergenceHeight,
		}
		assert.True(t, primary.HasEvidence(evAgainstWitness))
	}
}

func TestLightClientAttackEvidence_ForwardLunatic(t *testing.T) {
	// primary performs a lunatic attack but changes the time of the header to
	// something in the future relative to the blockchain
	var (
		latestHeight      = int64(10)
		valSize           = 5
		forgedHeight      = int64(12)
		proofHeight       = int64(11)
		primaryHeaders    = make(map[int64]*types.SignedHeader, forgedHeight)
		primaryValidators = make(map[int64]*types.ValidatorSet, forgedHeight)
		primaryVoters     = make(map[int64]*types.VoterSet, forgedHeight)
	)

	witnessHeaders, witnessValidators, witnessVoters, chainKeys :=
		genMockNodeWithKeys(chainID, latestHeight, valSize, 2, bTime)

	// primary has the exact same headers except it forges one extra header in the future using keys from 2/5ths of
	// the validators
	for h := range witnessHeaders {
		primaryHeaders[h] = witnessHeaders[h]
		primaryValidators[h] = witnessValidators[h]
		primaryVoters[h] = witnessVoters[h]
	}

	// Make height=proofHeight for forgedHeight
	proofKeys := chainKeys[proofHeight]
	forgedKeys := proofKeys.ChangeKeys(3) // we change 3 out of the 5 validators (still 2/5 remain)
	forgedVals := forgedKeys.ToValidators(2, 0)
	header, vals, voters, _ := genMockNodeWithKey(chainID, proofHeight, nil,
		proofKeys, forgedKeys,
		nil, forgedVals, primaryHeaders[proofHeight-1],
		bTime.Add(time.Duration(proofHeight)*time.Minute), // 11 mins
		0, len(proofKeys),
		0, nil)
	primaryHeaders[proofHeight] = header
	primaryValidators[proofHeight] = vals
	primaryVoters[proofHeight] = voters

	// Make height=forgedHeight for forward lunatic
	header, vals, voters, _ = genMockNodeWithKey(chainID, forgedHeight, nil,
		forgedKeys, forgedKeys,
		forgedVals, forgedVals, primaryHeaders[forgedHeight-1],
		bTime.Add(time.Duration(forgedHeight)*time.Minute),
		0, len(forgedKeys),
		0, nil)
	primaryHeaders[forgedHeight] = header
	primaryValidators[forgedHeight] = vals
	primaryVoters[forgedHeight] = voters

	witness := mockp.New(chainID, witnessHeaders, witnessValidators, witnessVoters)
	primary := mockp.New(chainID, primaryHeaders, primaryValidators, primaryVoters)

	laggingWitness := witness.Copy(chainID)

	// In order to perform the attack, the primary needs at least one accomplice as a witness to also
	// send the forged block
	accomplice := primary

	c, err := light.NewClient(
		ctx,
		chainID,
		light.TrustOptions{
			Period: 4 * time.Hour,
			Height: 1,
			Hash:   primaryHeaders[1].Hash(),
		},
		primary,
		[]provider.Provider{witness, accomplice},
		dbs.New(dbm.NewMemDB(), chainID),
		types.DefaultVoterParams(),
		light.Logger(log.TestingLogger()),
		light.MaxClockDrift(1*time.Second),
		light.MaxBlockLag(1*time.Second),
	)
	require.NoError(t, err)

	// two seconds later, the supporting witness should receive the header that can be used
	// to prove that there was an attack
	header, vals, voters, _ = genMockNodeWithKey(chainID, proofHeight, nil,
		proofKeys, proofKeys,
		nil, nil, primaryHeaders[proofHeight-1],
		bTime.Add(time.Duration(proofHeight+1)*time.Minute), // 12 mins
		0, len(proofKeys),
		0, nil)
	newLb := &types.LightBlock{
		SignedHeader: header,
		ValidatorSet: vals,
		VoterSet:     voters,
	}
	go func() {
		time.Sleep(2 * time.Second)
		witness.AddLightBlock(newLb)
	}()

	// Now assert that verification returns an error. We craft the light clients time to be a little ahead of the chain
	// to allow a window for the attack to manifest itself.
	_, err = c.Update(ctx, bTime.Add(time.Duration(forgedHeight)*time.Minute))
	if assert.Error(t, err) {
		assert.Equal(t, light.ErrLightClientAttack, err)
	}

	// Check evidence was sent to the witness against the full node
	evAgainstPrimary := &types.LightClientAttackEvidence{
		ConflictingBlock: &types.LightBlock{
			SignedHeader: primaryHeaders[proofHeight],
			ValidatorSet: primaryValidators[proofHeight],
			VoterSet:     primaryVoters[proofHeight],
		},
		CommonHeight: latestHeight,
	}
	assert.True(t, witness.HasEvidence(evAgainstPrimary))

	// We attempt the same call but now the supporting witness has a block which should
	// immediately conflict in time with the primary
	_, err = c.VerifyLightBlockAtHeight(ctx, forgedHeight, bTime.Add(time.Duration(forgedHeight)*time.Minute))
	if assert.Error(t, err) {
		assert.Equal(t, light.ErrLightClientAttack, err)
	}
	assert.True(t, witness.HasEvidence(evAgainstPrimary))

	// Lastly we test the unfortunate case where the light clients supporting witness doesn't update
	// in enough time
	c, err = light.NewClient(
		ctx,
		chainID,
		light.TrustOptions{
			Period: 4 * time.Hour,
			Height: 1,
			Hash:   primaryHeaders[1].Hash(),
		},
		primary,
		[]provider.Provider{laggingWitness, accomplice},
		dbs.New(dbm.NewMemDB(), chainID),
		types.DefaultVoterParams(),
		light.Logger(log.TestingLogger()),
		light.MaxClockDrift(1*time.Second),
		light.MaxBlockLag(1*time.Second),
	)
	require.NoError(t, err)

	_, err = c.Update(ctx, bTime.Add(time.Duration(forgedHeight)*time.Minute))
	assert.NoError(t, err)

}

// 1. Different nodes therefore a divergent header is produced.
// => light client returns an error upon creation because primary and witness
// have a different view.
func TestClientDivergentTraces1(t *testing.T) {
	primary := mockp.New(genMockNode(chainID, 10, 5, 2, bTime))
	firstBlock, err := primary.LightBlock(ctx, 1)
	require.NoError(t, err)
	witness := mockp.New(genMockNode(chainID, 10, 5, 2, bTime))

	_, err = light.NewClient(
		ctx,
		chainID,
		light.TrustOptions{
			Height: 1,
			Hash:   firstBlock.Hash(),
			Period: 4 * time.Hour,
		},
		primary,
		[]provider.Provider{witness},
		dbs.New(dbm.NewMemDB(), chainID),
		types.DefaultVoterParams(),
		light.Logger(log.TestingLogger()),
		light.MaxRetryAttempts(1),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not match primary")
}

// 2. Two out of three nodes don't respond but the third has a header that matches
// => verification should be successful and all the witnesses should remain
func TestClientDivergentTraces2(t *testing.T) {
	primary := mockp.New(genMockNode(chainID, 10, 5, 2, bTime))
	firstBlock, err := primary.LightBlock(ctx, 1)
	require.NoError(t, err)
	c, err := light.NewClient(
		ctx,
		chainID,
		light.TrustOptions{
			Height: 1,
			Hash:   firstBlock.Hash(),
			Period: 4 * time.Hour,
		},
		primary,
		[]provider.Provider{deadNode, deadNode, primary},
		dbs.New(dbm.NewMemDB(), chainID),
		types.DefaultVoterParams(),
		light.Logger(log.TestingLogger()),
		light.MaxRetryAttempts(1),
	)
	require.NoError(t, err)

	_, err = c.VerifyLightBlockAtHeight(ctx, 10, bTime.Add(1*time.Hour))
	assert.NoError(t, err)
	assert.Equal(t, 3, len(c.Witnesses()))
}

// 3. witness has the same first header, but different second header
// => creation should succeed, but the verification should fail
func TestClientDivergentTraces3(t *testing.T) {
	_, primaryHeaders, primaryVals, primaryVoters := genMockNode(chainID, 10, 5, 2, bTime)
	primary := mockp.New(chainID, primaryHeaders, primaryVals, primaryVoters)

	firstBlock, err := primary.LightBlock(ctx, 1)
	require.NoError(t, err)

	_, mockHeaders, mockVals, mockVoters := genMockNode(chainID, 10, 5, 2, bTime)
	mockHeaders[1] = primaryHeaders[1]
	mockVals[1] = primaryVals[1]
	mockVoters[1] = primaryVoters[1]
	witness := mockp.New(chainID, mockHeaders, mockVals, mockVoters)

	c, err := light.NewClient(
		ctx,
		chainID,
		light.TrustOptions{
			Height: 1,
			Hash:   firstBlock.Hash(),
			Period: 4 * time.Hour,
		},
		primary,
		[]provider.Provider{witness},
		dbs.New(dbm.NewMemDB(), chainID),
		types.DefaultVoterParams(),
		light.Logger(log.TestingLogger()),
		light.MaxRetryAttempts(1),
	)
	require.NoError(t, err)

	_, err = c.VerifyLightBlockAtHeight(ctx, 10, bTime.Add(1*time.Hour))
	assert.Error(t, err)
	assert.Equal(t, 1, len(c.Witnesses()))
}

// 4. Witness has a divergent header but can not produce a valid trace to back it up.
// It should be ignored
func TestClientDivergentTraces4(t *testing.T) {
	_, primaryHeaders, primaryVals, primaryVoters := genMockNode(chainID, 10, 5, 2, bTime)
	primary := mockp.New(chainID, primaryHeaders, primaryVals, primaryVoters)

	firstBlock, err := primary.LightBlock(ctx, 1)
	require.NoError(t, err)

	_, mockHeaders, mockVals, mockVoters := genMockNode(chainID, 10, 5, 2, bTime)
	witness := primary.Copy(chainID)
	witness.AddLightBlock(&types.LightBlock{
		SignedHeader: mockHeaders[10],
		ValidatorSet: mockVals[10],
		VoterSet:     mockVoters[10],
	})

	c, err := light.NewClient(
		ctx,
		chainID,
		light.TrustOptions{
			Height: 1,
			Hash:   firstBlock.Hash(),
			Period: 4 * time.Hour,
		},
		primary,
		[]provider.Provider{witness},
		dbs.New(dbm.NewMemDB(), chainID),
		types.DefaultVoterParams(),
		light.Logger(log.TestingLogger()),
	)
	require.NoError(t, err)

	_, err = c.VerifyLightBlockAtHeight(ctx, 10, bTime.Add(1*time.Hour))
	assert.Error(t, err)
	assert.Equal(t, 1, len(c.Witnesses()))
}
