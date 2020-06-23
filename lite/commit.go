package lite

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/tendermint/tendermint/types"
)

// FullCommit contains a SignedHeader (the block header and a commit that signs it),
// the validator set which signed the commit, and the next validator set. The
// next validator set (which is proven from the block header) allows us to
// revert to block-by-block updating of lite Verifier's latest validator set,
// even in the face of arbitrarily large power changes.
type FullCommit struct {
	SignedHeader types.SignedHeader `json:"signed_header"`
	Voters       *types.VoterSet    `json:"voter_set"`
	NextVoters   *types.VoterSet    `json:"next_validator_set"`
}

// NewFullCommit returns a new FullCommit.
func NewFullCommit(signedHeader types.SignedHeader, voterSet, nextVoterSet *types.VoterSet) FullCommit {
	return FullCommit{
		SignedHeader: signedHeader,
		Voters:       voterSet,
		NextVoters:   nextVoterSet,
	}
}

// Validate the components and check for consistency.
// This also checks to make sure that Voters actually
// signed the SignedHeader.Commit.
// If > 2/3 did not sign the Commit from fc.Voters, it
// is not a valid commit!
func (fc FullCommit) ValidateFull(chainID string) error {
	// Ensure that Validators exists and matches the header.
	if fc.Voters.Size() == 0 {
		return errors.New("need FullCommit.Voters")
	}
	if !bytes.Equal(
		fc.SignedHeader.VotersHash,
		fc.Voters.Hash()) {
		return fmt.Errorf("header has vhash %X but voterSet hash is %X",
			fc.SignedHeader.VotersHash,
			fc.Voters.Hash(),
		)
	}
	// Ensure that NextValidators exists and matches the header.
	if fc.NextVoters.Size() == 0 {
		return errors.New("need FullCommit.NextValidators")
	}
	if !bytes.Equal(
		fc.SignedHeader.NextVotersHash,
		fc.NextVoters.Hash()) {
		return fmt.Errorf("header has next vhash %X but next voterSet hash is %X",
			fc.SignedHeader.NextVotersHash,
			fc.NextVoters.Hash(),
		)
	}
	// Validate the header.
	err := fc.SignedHeader.ValidateBasic(chainID)
	if err != nil {
		return err
	}
	// Validate the signatures on the commit.
	hdr, cmt := fc.SignedHeader.Header, fc.SignedHeader.Commit
	return fc.Voters.VerifyCommit(
		hdr.ChainID, cmt.BlockID,
		hdr.Height, cmt)
}

// Height returns the height of the header.
func (fc FullCommit) Height() int64 {
	if fc.SignedHeader.Header == nil {
		panic("should not happen")
	}
	return fc.SignedHeader.Height
}

// ChainID returns the chainID of the header.
func (fc FullCommit) ChainID() string {
	if fc.SignedHeader.Header == nil {
		panic("should not happen")
	}
	return fc.SignedHeader.ChainID
}
