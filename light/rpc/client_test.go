package rpc

import (
	"context"
	"encoding/hex"
	"fmt"
	"testing"

	ics23 "github.com/confio/ics23/go"

	abci "github.com/line/ostracon/abci/types"
	"github.com/line/ostracon/crypto/merkle"
	"github.com/line/ostracon/libs/bytes"
	lcmock "github.com/line/ostracon/light/rpc/mocks"
	tmcrypto "github.com/line/ostracon/proto/ostracon/crypto"
	rpcmock "github.com/line/ostracon/rpc/client/mocks"
	ctypes "github.com/line/ostracon/rpc/core/types"
	"github.com/line/ostracon/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestABCIQuery tests ABCIQuery requests and verifies proofs. HAPPY PATH 😀
func TestABCIQuery(t *testing.T) {
	var (
		key   = []byte("foo")
		value = []byte("bar")
	)

	// You can get this proof binary with following code.
	proof := []byte{10, 23, 10, 3, 102, 111, 111, 18, 3, 98, 97, 114, 26, 11, 8, 1, 24, 1, 32, 1, 42, 3, 0, 2, 2}
	var commitmentProof ics23.CommitmentProof
	err := commitmentProof.Unmarshal(proof)
	require.NoError(t, err)

	// We comment out this code to remove the dependency of iavl
	/*
		tree, err := iavl.NewMutableTree(memdb.NewDB(), 100)
		require.NoError(t, err)

		tree.Set(key, value)

		commitmentProof, err := tree.GetMembershipProof(key)
		require.NoError(t, err)
		data, _ := commitmentProof.Marshal()
		fmt.Printf("%v\n", data)
	*/

	op := &testOp{
		Spec:  ics23.IavlSpec,
		Key:   key,
		Proof: &commitmentProof,
	}

	next := &rpcmock.Client{}
	next.On(
		"ABCIQueryWithOptions",
		context.Background(),
		mock.AnythingOfType("string"),
		bytes.HexBytes(key),
		mock.AnythingOfType("client.ABCIQueryOptions"),
	).Return(&ctypes.ResultABCIQuery{
		Response: abci.ResponseQuery{
			Code:   0,
			Key:    key,
			Value:  value,
			Height: 1,
			ProofOps: &tmcrypto.ProofOps{
				Ops: []tmcrypto.ProofOp{op.ProofOp()},
			},
		},
	}, nil)

	lc := &lcmock.LightClient{}
	appHash, _ := hex.DecodeString("5EFD44055350B5CC34DBD26085347A9DBBE44EA192B9286A9FC107F40EA1FAC5")
	lc.On("VerifyLightBlockAtHeight", context.Background(), int64(2), mock.AnythingOfType("time.Time")).Return(
		&types.LightBlock{
			SignedHeader: &types.SignedHeader{
				Header: &types.Header{AppHash: appHash},
			},
		},
		nil,
	)

	c := NewClient(next, lc,
		KeyPathFn(func(_ string, key []byte) (merkle.KeyPath, error) {
			kp := merkle.KeyPath{}
			kp = kp.AppendKey(key, merkle.KeyEncodingURL)
			return kp, nil
		}))
	c.RegisterOpDecoder("ics23:iavl", testOpDecoder)
	res, err := c.ABCIQuery(context.Background(), "/store/accounts/key", key)
	require.NoError(t, err)

	assert.NotNil(t, res)
}

func TestTxSearch(t *testing.T) {

	query := "query/test"
	prove := false
	page := 0
	perPage := 1
	orderBy := ""

	next := &rpcmock.Client{}
	next.On(
		"TxSearch",
		context.Background(),
		query,
		prove,
		&page,
		&perPage,
		orderBy,
	).Return(&ctypes.ResultTxSearch{
		Txs:        nil,
		TotalCount: 0,
	}, nil)

	lc := &lcmock.LightClient{}

	c := NewClient(next, lc)
	res, err := c.TxSearch(context.Background(), query, prove, &page, &perPage, orderBy)
	require.NoError(t, err)
	assert.NotNil(t, res)
}

func TestBlockSearch(t *testing.T) {

	query := "query/test"
	page := 0
	perPage := 1
	orderBy := ""

	next := &rpcmock.Client{}
	next.On(
		"BlockSearch",
		context.Background(),
		query,
		&page,
		&perPage,
		orderBy,
	).Return(&ctypes.ResultBlockSearch{
		Blocks:     nil,
		TotalCount: 0,
	}, nil)

	lc := &lcmock.LightClient{}

	c := NewClient(next, lc)
	res, err := c.BlockSearch(context.Background(), query, &page, &perPage, orderBy)
	require.NoError(t, err)
	assert.NotNil(t, res)
}

type testOp struct {
	Spec  *ics23.ProofSpec
	Key   []byte
	Proof *ics23.CommitmentProof
}

var _ merkle.ProofOperator = testOp{}

func (op testOp) GetKey() []byte {
	return op.Key
}

func (op testOp) ProofOp() tmcrypto.ProofOp {
	bz, err := op.Proof.Marshal()
	if err != nil {
		panic(err.Error())
	}
	return tmcrypto.ProofOp{
		Type: "ics23:iavl",
		Key:  op.Key,
		Data: bz,
	}
}

func (op testOp) Run(args [][]byte) ([][]byte, error) {
	// calculate root from proof
	root, err := op.Proof.Calculate()
	if err != nil {
		return nil, fmt.Errorf("could not calculate root for proof: %v", err)
	}
	// Only support an existence proof or nonexistence proof (batch proofs currently unsupported)
	switch len(args) {
	case 0:
		// Args are nil, so we verify the absence of the key.
		absent := ics23.VerifyNonMembership(op.Spec, root, op.Proof, op.Key)
		if !absent {
			return nil, fmt.Errorf("proof did not verify absence of key: %s", string(op.Key))
		}
	case 1:
		// Args is length 1, verify existence of key with value args[0]
		if !ics23.VerifyMembership(op.Spec, root, op.Proof, op.Key, args[0]) {
			return nil, fmt.Errorf("proof did not verify existence of key %s with given value %x", op.Key, args[0])
		}
	default:
		return nil, fmt.Errorf("args must be length 0 or 1, got: %d", len(args))
	}

	return [][]byte{root}, nil
}

func testOpDecoder(pop tmcrypto.ProofOp) (merkle.ProofOperator, error) {
	proof := &ics23.CommitmentProof{}
	err := proof.Unmarshal(pop.Data)
	if err != nil {
		return nil, err
	}

	op := testOp{
		Key:   pop.Key,
		Spec:  ics23.IavlSpec,
		Proof: proof,
	}
	return op, nil
}

func TestDefaultMerkleKeyPathFn(t *testing.T) {
	f := DefaultMerkleKeyPathFn()
	require.NotNil(t, f)
	{
		path, err := f("", nil)
		require.Error(t, err)
		require.Nil(t, path)
	}
	{
		path, err := f("/store/test-merkle-path/key", []byte("test"))
		require.NoError(t, err)
		require.NotNil(t, path)
		require.Equal(t, "/test-merkle-path/test", path.String())
	}
}
