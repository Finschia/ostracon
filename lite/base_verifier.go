package lite

import (
	"bytes"
	"github.com/tendermint/tendermint/crypto/vrf"

	"github.com/pkg/errors"

	lerr "github.com/tendermint/tendermint/lite/errors"
	"github.com/tendermint/tendermint/types"
)

var _ Verifier = (*BaseVerifier)(nil)

// BaseVerifier lets us check the validity of SignedHeaders at height or
// later, requiring sufficient votes (> 2/3) from the given voterSet.
// To verify blocks produced by a blockchain with mutable validator sets,
// use the DynamicVerifier.
// TODO: Handle unbonding time.
type BaseVerifier struct {
	chainID     string
	height      int64
	valSet      *types.ValidatorSet
	voterParams *types.VoterParams
}

// NewBaseVerifier returns a new Verifier initialized with a validator set at
// some height.
func NewBaseVerifier(chainID string, height int64, valset *types.ValidatorSet,
	voterParams *types.VoterParams) *BaseVerifier {
	if valset.IsNilOrEmpty() {
		panic("NewBaseVerifier requires a valid voterSet")
	}
	return &BaseVerifier{
		chainID:     chainID,
		height:      height,
		valSet:      valset,
		voterParams: voterParams,
	}
}

// Implements Verifier.
func (bv *BaseVerifier) ChainID() string {
	return bv.chainID
}

// Implements Verifier.
func (bv *BaseVerifier) Verify(signedHeader types.SignedHeader) error {

	// We can't verify commits for a different chain.
	if signedHeader.ChainID != bv.chainID {
		return errors.Errorf("BaseVerifier chainID is %v, cannot verify chainID %v",
			bv.chainID, signedHeader.ChainID)
	}

	// We can't verify commits older than bv.height.
	if signedHeader.Height < bv.height {
		return errors.Errorf("BaseVerifier height is %v, cannot verify height %v",
			bv.height, signedHeader.Height)
	}

	// We can't verify with the wrong validator set.
	if !bytes.Equal(signedHeader.ValidatorsHash,
		bv.valSet.Hash()) {
		return lerr.ErrUnexpectedValidators(signedHeader.ValidatorsHash, bv.valSet.Hash())
	}

	// Do basic sanity checks.
	err := signedHeader.ValidateBasic(bv.chainID)
	if err != nil {
		return errors.Wrap(err, "in verify")
	}

	proofHash, err := vrf.ProofToHash(signedHeader.Proof.Bytes())
	if err != nil {
		return errors.Wrap(err, "in verify")
	}
	voters := types.SelectVoter(bv.valSet, proofHash, bv.voterParams)
	if !bytes.Equal(signedHeader.VotersHash, voters.Hash()) {
		return errors.Errorf("header's voter hash is %X, but voters hash is %X",
			signedHeader.VotersHash, voters.Hash())
	}
	// Check commit signatures.
	err = voters.VerifyCommit(
		bv.chainID, signedHeader.Commit.BlockID,
		signedHeader.Height, signedHeader.Commit)
	if err != nil {
		return errors.Wrap(err, "in verify")
	}

	return nil
}
