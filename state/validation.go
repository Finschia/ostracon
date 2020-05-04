package state

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/crypto/vrf"
	"github.com/tendermint/tendermint/types"
	dbm "github.com/tendermint/tm-db"
)

//-----------------------------------------------------
// Validate block

func validateBlock(evidencePool EvidencePool, stateDB dbm.DB, state State, round int, block *types.Block) error {
	// Validate internal consistency.
	if err := block.ValidateBasic(); err != nil {
		return err
	}

	// Validate basic info.
	if block.Version != state.Version.Consensus {
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
	if block.Height != state.LastBlockHeight+1 {
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
	if !bytes.Equal(block.ConsensusHash, state.ConsensusParams.Hash()) {
		return fmt.Errorf("wrong Block.Header.ConsensusHash.  Expected %X, got %v",
			state.ConsensusParams.Hash(),
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
	if !bytes.Equal(block.NextVotersHash, state.NextVoters.Hash()) {
		return fmt.Errorf("wrong Block.Header.NextVotersHash.  Expected %X, got %v",
			state.NextVoters.Hash(),
			block.NextVotersHash,
		)
	}

	// Validate block LastCommit.
	if block.Height == 1 {
		if len(block.LastCommit.Signatures) != 0 {
			return errors.New("block at height 1 can't have LastCommit signatures")
		}
	} else {
		if len(block.LastCommit.Signatures) != state.LastVoters.Size() {
			return types.NewErrInvalidCommitSignatures(state.LastVoters.Size(), len(block.LastCommit.Signatures))
		}
		err := state.LastVoters.VerifyCommit(
			state.ChainID, state.LastBlockID, block.Height-1, block.LastCommit)
		if err != nil {
			return err
		}
	}

	// Validate block Time
	if block.Height > 1 {
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
	} else if block.Height == 1 {
		genesisTime := state.LastBlockTime
		if !block.Time.Equal(genesisTime) {
			return fmt.Errorf("block time %v is not equal to genesis time %v",
				block.Time,
				genesisTime,
			)
		}
	}

	// Limit the amount of evidence
	maxNumEvidence, _ := types.MaxEvidencePerBlock(state.ConsensusParams.Block.MaxBytes)
	numEvidence := int64(len(block.Evidence.Evidence))
	if numEvidence > maxNumEvidence {
		return types.NewErrEvidenceOverflow(maxNumEvidence, numEvidence)

	}

	// Validate all evidence.
	for _, ev := range block.Evidence.Evidence {
		if err := VerifyEvidence(stateDB, state, ev); err != nil {
			return types.NewErrEvidenceInvalid(ev, err)
		}
		if evidencePool != nil && evidencePool.IsCommitted(ev) {
			return types.NewErrEvidenceInvalid(ev, errors.New("evidence was already committed"))
		}
	}

	// NOTE: We can't actually verify it's the right proposer because we dont
	// know what round the block was first proposed. So just check that it's
	// a legit address and a known validator.
	if len(block.ProposerAddress) != crypto.AddressSize ||
		!state.Voters.HasAddress(block.ProposerAddress) {
		return fmt.Errorf("block.Header.ProposerAddress, %X, is not a validator",
			block.ProposerAddress,
		)
	}

	// validate proposer
	if !bytes.Equal(block.ProposerAddress.Bytes(),
		state.Voters.SelectProposer(state.LastProofHash, block.Height, block.Round).Address.Bytes()) {
		return fmt.Errorf("block.ProposerAddress, %X, is not the proposer %X",
			block.ProposerAddress,
			state.Voters.SelectProposer(state.LastProofHash, block.Height, block.Round).Address,
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
	_, val := state.Voters.GetByAddress(block.ProposerAddress)
	verified, err := vrf.Verify(val.PubKey.(ed25519.PubKeyEd25519), block.Proof.Bytes(), message)
	if err != nil {
		return types.NewErrInvalidProof(fmt.Sprintf(
			"verification failed: %s; proof: %v, prevProofHash: %v, height=%d, round=%d, addr: %v",
			err.Error(), block.Proof, state.LastProofHash, state.LastBlockHeight, block.Round, block.ProposerAddress))
	} else if !verified {
		return types.NewErrInvalidProof(fmt.Sprintf("proof: %v, prevProofHash: %v, height=%d, round=%d, addr: %v",
			block.Proof, state.LastProofHash, state.LastBlockHeight, block.Round, block.ProposerAddress))
	}

	return nil
}

// VerifyEvidence verifies the evidence fully by checking:
// - it is sufficiently recent (MaxAge)
// - it is from a key who was a validator at the given height
// - it is internally consistent
// - it was properly signed by the alleged equivocator
func VerifyEvidence(stateDB dbm.DB, state State, evidence types.Evidence) error {
	var (
		height         = state.LastBlockHeight
		evidenceParams = state.ConsensusParams.Evidence
	)

	ageNumBlocks := height - evidence.Height()
	if ageNumBlocks > evidenceParams.MaxAgeNumBlocks {
		return fmt.Errorf("evidence from height %d is too old. Min height is %d",
			evidence.Height(), height-evidenceParams.MaxAgeNumBlocks)
	}

	ageDuration := state.LastBlockTime.Sub(evidence.Time())
	if ageDuration > evidenceParams.MaxAgeDuration {
		return fmt.Errorf("evidence created at %v has expired. Evidence can not be older than: %v",
			evidence.Time(), state.LastBlockTime.Add(evidenceParams.MaxAgeDuration))
	}

	_, voterSet, err := LoadValidators(stateDB, evidence.Height())
	if err != nil {
		// TODO: if err is just that we cant find it cuz we pruned, ignore.
		// TODO: if its actually bad evidence, punish peer
		return err
	}

	// The address must have been an active validator at the height.
	// NOTE: we will ignore evidence from H if the key was not a validator
	// at H, even if it is a validator at some nearby H'
	// XXX: this makes lite-client bisection as is unsafe
	// See https://github.com/tendermint/tendermint/issues/3244
	ev := evidence
	height, addr := ev.Height(), ev.Address()
	_, val := voterSet.GetByAddress(addr)
	if val == nil {
		return fmt.Errorf("address %X was not a validator at height %d", addr, height)
	}

	if err := evidence.Verify(state.ChainID, val.PubKey); err != nil {
		return err
	}

	return nil
}
