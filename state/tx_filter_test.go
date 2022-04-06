package state_test

import (
	"os"
	"testing"

	dbm "github.com/line/tm-db/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tmrand "github.com/line/ostracon/libs/rand"
	sm "github.com/line/ostracon/state"
	"github.com/line/ostracon/types"
)

func TestTxFilter(t *testing.T) {
	genDoc := randomGenesisDoc()
	genDoc.ConsensusParams.Block.MaxBytes = 3035
	genDoc.ConsensusParams.Evidence.MaxBytes = 1500

	// Max size of Txs is much smaller than size of block,
	// since we need to account for commits and evidence.
	testCases := []struct {
		tx    types.Tx
		isErr bool
	}{
		{types.Tx(tmrand.Bytes(2122)), false},
		{types.Tx(tmrand.Bytes(2123)), true},
		{types.Tx(tmrand.Bytes(3000)), true},
	}

	for i, tc := range testCases {
		stateDB, err := dbm.NewDB("state", "memdb", os.TempDir())
		require.NoError(t, err)
		stateStore := sm.NewStore(stateDB)
		state, err := stateStore.LoadFromDBOrGenesisDoc(genDoc)
		require.NoError(t, err)

		f := sm.TxPreCheck(state)
		if tc.isErr {
			assert.NotNil(t, f(tc.tx), "#%v", i)
		} else {
			assert.Nil(t, f(tc.tx), "#%v", i)
		}
	}
}
