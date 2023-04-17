package mempool

import (
	"testing"

	abci "github.com/Finschia/ostracon/abci/types"
	"github.com/stretchr/testify/require"
)

func TestPostCheckMaxGas(t *testing.T) {
	tests := []struct {
		res       *abci.ResponseCheckTx
		postCheck PostCheckFunc
		ok        bool
	}{
		{&abci.ResponseCheckTx{GasWanted: 10}, PostCheckMaxGas(10), true},
		{&abci.ResponseCheckTx{GasWanted: 10}, PostCheckMaxGas(-1), true},
		{&abci.ResponseCheckTx{GasWanted: -1}, PostCheckMaxGas(10), false},
		{&abci.ResponseCheckTx{GasWanted: 11}, PostCheckMaxGas(10), false},
	}
	for tcIndex, tt := range tests {
		err := tt.postCheck(nil, tt.res)
		if tt.ok {
			require.NoError(t, err, "postCheck should not return error, on test case %d", tcIndex)
		} else {
			require.Error(t, err, "postCheck should return error, on test case %d", tcIndex)
		}
	}
}
