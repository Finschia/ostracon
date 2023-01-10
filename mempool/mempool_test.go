package mempool

import (
	"testing"

	ocabci "github.com/line/ostracon/abci/types"
	"github.com/stretchr/testify/require"
)

func TestPostCheckMaxGas(t *testing.T) {
	tests := []struct {
		res       *ocabci.ResponseCheckTx
		postCheck PostCheckFunc
		ok        bool
	}{
		{&ocabci.ResponseCheckTx{GasWanted: 10}, PostCheckMaxGas(10), true},
		{&ocabci.ResponseCheckTx{GasWanted: 10}, PostCheckMaxGas(-1), true},
		{&ocabci.ResponseCheckTx{GasWanted: -1}, PostCheckMaxGas(10), false},
		{&ocabci.ResponseCheckTx{GasWanted: 11}, PostCheckMaxGas(10), false},
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
