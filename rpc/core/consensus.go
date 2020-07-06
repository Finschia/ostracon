package core

import (
	cm "github.com/tendermint/tendermint/consensus"
	tmmath "github.com/tendermint/tendermint/libs/math"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	rpctypes "github.com/tendermint/tendermint/rpc/jsonrpc/types"
	sm "github.com/tendermint/tendermint/state"
	"github.com/tendermint/tendermint/types"
	dbm "github.com/tendermint/tm-db"
)

// Validators gets the validator set at the given block height.
//
// If no height is provided, it will fetch the latest validator set. Note the
// voters are sorted by their voting power - this is the canonical order
// for the voters in the set as used in computing their Merkle root.
//
// More: https://docs.tendermint.com/master/rpc/#/Info/validators
func Validators(ctx *rpctypes.Context, heightPtr *int64, page, perPage int) (*ctypes.ResultValidators, error) {
	return validators(ctx, heightPtr, page, perPage, sm.LoadValidators)
}

func validators(ctx *rpctypes.Context, heightPtr *int64, page, perPage int,
	loadFunc func(db dbm.DB, height int64) (*types.ValidatorSet, error)) (
	*ctypes.ResultValidators, error) {
	// The latest validator that we know is the
	// NextValidator of the last block.
	height, err := getHeight(latestUncommittedHeight(), heightPtr)
	if err != nil {
		return nil, err
	}

	vals, err := loadFunc(env.StateDB, height)
	if err != nil {
		return nil, err
	}

	totalCount := len(vals.Validators)
	perPage = validatePerPage(perPage)
	page, err = validatePage(page, perPage, totalCount)
	if err != nil {
		return nil, err
	}

	skipCount := validateSkipCount(page, perPage)

	v := vals.Validators[skipCount : skipCount+tmmath.MinInt(perPage, totalCount-skipCount)]

	return &ctypes.ResultValidators{
		BlockHeight: height,
		Validators:  v}, nil
}

func Voters(ctx *rpctypes.Context, heightPtr *int64, page, perPage int) (*ctypes.ResultVoters, error) {
	return voters(ctx, heightPtr, page, perPage, sm.LoadVoters)
}

func voters(ctx *rpctypes.Context, heightPtr *int64, page, perPage int,
	loadFunc func(db dbm.DB, height int64, voterParam *types.VoterParams) (*types.VoterSet, error)) (
	*ctypes.ResultVoters, error) {
	// The latest validator that we know is the
	// NextValidator of the last block.
	height, err := getHeight(latestUncommittedHeight(), heightPtr)
	if err != nil {
		return nil, err
	}

	voters, err := loadFunc(env.StateDB, height, env.ConsensusState.GetState().VoterParams)
	if err != nil {
		return nil, err
	}

	totalCount := len(voters.Voters)
	perPage = validatePerPage(perPage)
	page, err = validatePage(page, perPage, totalCount)
	if err != nil {
		return nil, err
	}

	skipCount := validateSkipCount(page, perPage)

	v := voters.Voters[skipCount : skipCount+tmmath.MinInt(perPage, totalCount-skipCount)]

	return &ctypes.ResultVoters{
		BlockHeight: height,
		Voters:      v}, nil
}

// DumpConsensusState dumps consensus state.
// UNSTABLE
// More: https://docs.tendermint.com/master/rpc/#/Info/dump_consensus_state
func DumpConsensusState(ctx *rpctypes.Context) (*ctypes.ResultDumpConsensusState, error) {
	// Get Peer consensus states.
	peers := env.P2PPeers.Peers().List()
	peerStates := make([]ctypes.PeerStateInfo, len(peers))
	for i, peer := range peers {
		peerState, ok := peer.Get(types.PeerStateKey).(*cm.PeerState)
		if !ok { // peer does not have a state yet
			continue
		}
		peerStateJSON, err := peerState.ToJSON()
		if err != nil {
			return nil, err
		}
		peerStates[i] = ctypes.PeerStateInfo{
			// Peer basic info.
			NodeAddress: peer.SocketAddr().String(),
			// Peer consensus state.
			PeerState: peerStateJSON,
		}
	}
	// Get self round state.
	roundState, err := env.ConsensusState.GetRoundStateJSON()
	if err != nil {
		return nil, err
	}
	return &ctypes.ResultDumpConsensusState{
		RoundState: roundState,
		Peers:      peerStates}, nil
}

// ConsensusState returns a concise summary of the consensus state.
// UNSTABLE
// More: https://docs.tendermint.com/master/rpc/#/Info/consensus_state
func ConsensusState(ctx *rpctypes.Context) (*ctypes.ResultConsensusState, error) {
	// Get self round state.
	bz, err := env.ConsensusState.GetRoundStateSimpleJSON()
	return &ctypes.ResultConsensusState{RoundState: bz}, err
}

// ConsensusParams gets the consensus parameters at the given block height.
// If no height is provided, it will fetch the latest consensus params.
// More: https://docs.tendermint.com/master/rpc/#/Info/consensus_params
func ConsensusParams(ctx *rpctypes.Context, heightPtr *int64) (*ctypes.ResultConsensusParams, error) {
	// The latest consensus params that we know is the consensus params after the
	// last block.
	height, err := getHeight(latestUncommittedHeight(), heightPtr)
	if err != nil {
		return nil, err
	}

	consensusParams, err := sm.LoadConsensusParams(env.StateDB, height)
	if err != nil {
		return nil, err
	}
	return &ctypes.ResultConsensusParams{
		BlockHeight:     height,
		ConsensusParams: consensusParams}, nil
}
