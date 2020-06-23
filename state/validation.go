package state

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/crypto/vrf"
	"github.com/tendermint/tendermint/types"
)

//-----------------------------------------------------
// Validate block

func validateBlock(state State, round int32, block *types.Block) error {
	// Validate internal consistency.
	if err := block.ValidateBasic(); err != nil {
		return err
	}

	// Validate basic info.
	if block.Version.App != state.Version.Consensus.App ||
		block.Version.Block != state.Version.Consensus.Block {
		return fmt.Errorf("wrong Block.Header.Version. Expected %v, got %v",
			state.Version.Consensus,
			block.Version,
		)
	}
	if block.ChainID != state.ChainID {
		return fmt.Errorf("wrong Block.Header.ChainID. Expected %v, got %v",
			state.ChainID,
			block.ChainID,
		)
	}
	if state.LastBlockHeight == 0 && block.Height != state.InitialHeight {
		return fmt.Errorf("wrong Block.Header.Height. Expected %v for initial block, got %v",
			block.Height, state.InitialHeight)
	}
	if state.LastBlockHeight > 0 && block.Height != state.LastBlockHeight+1 {
		return fmt.Errorf("wrong Block.Header.Height. Expected %v, got %v",
			state.LastBlockHeight+1,
			block.Height,
		)
	}
	// Validate prev block info.
	if !block.LastBlockID.Equals(state.LastBlockID) {
		return fmt.Errorf("wrong Block.Header.LastBlockID.  Expected %v, got %v",
			state.LastBlockID,
			block.LastBlockID,
		)
	}

	// Validate app info
	if !bytes.Equal(block.AppHash, state.AppHash) {
		return fmt.Errorf("wrong Block.Header.AppHash.  Expected %X, got %v",
			state.AppHash,
			block.AppHash,
		)
	}
	hashCP := types.HashConsensusParams(state.ConsensusParams)
	if !bytes.Equal(block.ConsensusHash, hashCP) {
		return fmt.Errorf("wrong Block.Header.ConsensusHash.  Expected %X, got %v",
			hashCP,
			block.ConsensusHash,
		)
	}
	if !bytes.Equal(block.LastResultsHash, state.LastResultsHash) {
		return fmt.Errorf("wrong Block.Header.LastResultsHash.  Expected %X, got %v",
			state.LastResultsHash,
			block.LastResultsHash,
		)
	}
	if !bytes.Equal(block.VotersHash, state.Voters.Hash()) {
		return fmt.Errorf("wrong Block.Header.VotersHash.  Expected %X, got %v",
			state.Voters.Hash(),
			block.VotersHash,
		)
	}
	if !bytes.Equal(block.ValidatorsHash, state.Validators.Hash()) {
		return fmt.Errorf("wrong Block.Header.ValidatorsHash.  Expected %X, got %v",
			state.Validators.Hash(),
			block.ValidatorsHash,
		)
	}
	if !bytes.Equal(block.NextValidatorsHash, state.NextValidators.Hash()) {
		return fmt.Errorf("wrong Block.Header.NextValidatorsHash.  Expected %X, got %v",
			state.NextValidators.Hash(),
			block.NextValidatorsHash,
		)
	}

	// Validate block LastCommit.
	if block.Height == state.InitialHeight {
		if len(block.LastCommit.Signatures) != 0 {
			return errors.New("initial block can't have LastCommit signatures")
		}
	} else {
		// LastCommit.Signatures length is checked in VerifyCommit.
		if err := state.LastVoters.VerifyCommit(
			state.ChainID, state.LastBlockID, block.Height-1, block.LastCommit); err != nil {
			return err
		}
	}

	// NOTE: We can't actually verify it's the right proposer because we don't
	// know what round the block was first proposed. So just check that it's
	// a legit address and a known validator.
	if len(block.ProposerAddress) != crypto.AddressSize {
		return fmt.Errorf("expected ProposerAddress size %d, got %d",
			crypto.AddressSize,
			len(block.ProposerAddress),
		)
	}
	if !state.Validators.HasAddress(block.ProposerAddress) {
		return fmt.Errorf("block.Header.ProposerAddress %X is not a validator",
			block.ProposerAddress,
		)
	}

	// Validate block Time
	switch {
	case block.Height > state.InitialHeight:
		if !block.Time.After(state.LastBlockTime) {
			return fmt.Errorf("block time %v not greater than last block time %v",
				block.Time,
				state.LastBlockTime,
			)
		}
		medianTime := MedianTime(block.LastCommit, state.LastVoters)
		if !block.Time.Equal(medianTime) {
			return fmt.Errorf("invalid block time. Expected %v, got %v",
				medianTime,
				block.Time,
			)
		}

	case block.Height == state.InitialHeight:
		genesisTime := state.LastBlockTime
		if !block.Time.Equal(genesisTime) {
			return fmt.Errorf("block time %v is not equal to genesis time %v",
				block.Time,
				genesisTime,
			)
		}

	default:
		return fmt.Errorf("block height %v lower than initial height %v",
			block.Height, state.InitialHeight)
	}

	// Check evidence doesn't exceed the limit amount of bytes.
	if max, got := state.ConsensusParams.Evidence.MaxBytes, block.Evidence.ByteSize(); got > max {
		return types.NewErrEvidenceOverflow(max, got)
	}

	// validate proposer
	if !bytes.Equal(block.ProposerAddress.Bytes(),
		state.Validators.SelectProposer(state.LastProofHash, block.Height, block.Round).Address.Bytes()) {
		return fmt.Errorf("block.ProposerAddress, %X, is not the proposer %X",
			block.ProposerAddress,
			state.Validators.SelectProposer(state.LastProofHash, block.Height, block.Round).Address,
		)
	}

	// validate round
	// The block round must be less than or equal to the current round
	// If some proposer proposes his ValidBlock as a proposal, then the proposal block round is less than current round
	if block.Round > round {
		return types.NewErrInvalidRound(round, block.Round)
	}

	// validate vrf proof
	message := state.MakeHashMessage(block.Round)
	_, val := state.Validators.GetByAddress(block.ProposerAddress)
	verified, err := vrf.Verify(val.PubKey.(ed25519.PubKey), block.Proof.Bytes(), message)
	if err != nil {
		return types.NewErrInvalidProof(fmt.Sprintf(
			"verification failed: %s; proof: %v, prevProofHash: %v, height=%d, round=%d, addr: %v",
			err.Error(), block.Proof, state.LastProofHash, state.LastBlockHeight, block.Round, block.ProposerAddress))
	} else if !verified {
		return types.NewErrInvalidProof(fmt.Sprintf("proof: %X, lastProofHash: %X, height=%d, round=%d, addr: %v, msg: %X",
			block.Proof, state.LastProofHash, state.LastBlockHeight, block.Round, block.ProposerAddress, message))
	}

	return nil
}
