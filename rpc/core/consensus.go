package core

import (
	cm "github.com/tendermint/tendermint/consensus"
	tmmath "github.com/tendermint/tendermint/libs/math"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	rpctypes "github.com/tendermint/tendermint/rpc/jsonrpc/types"
	"github.com/tendermint/tendermint/types"
)

// Validators gets the validators set at the given block height.
//
// If no height is provided, it will fetch the latest validator set. Note the
// voters are sorted by their voting power - this is the canonical order
// for the voters in the set as used in computing their Merkle root.
//
// More: https://docs.tendermint.com/master/rpc/#/Info/validators
func Validators(ctx *rpctypes.Context, heightPtr *int64, pagePtr, perPagePtr *int) (*ctypes.ResultValidators, error) {

	// The latest validator that we know is the NextValidator of the last block.
	height, err := getHeight(latestUncommittedHeight(), heightPtr)
	if err != nil {
		return nil, err
	}

	vals, err := env.StateStore.LoadValidators(height)
	if err != nil {
		return nil, err
	}

	voters, err := env.StateStore.LoadVoters(height, env.ConsensusState.GetState().VoterParams)
	if err != nil {
		return nil, err
	}

	totalCount := len(vals.Validators)
	perPage := validatePerPage(perPagePtr)
	page, err := validatePage(pagePtr, perPage, totalCount)
	if err != nil {
		return nil, err
	}

	skipCount := validateSkipCount(page, perPage)

	v := vals.Validators[skipCount : skipCount+tmmath.MinInt(perPage, totalCount-skipCount)]

	// Retrieve to the indices where selected as Voters in Validators.
	votersIndices := make([]int32, 0, len(v))
	for i := range v {
		idx, voter := voters.GetByAddress(v[i].Address)
		if idx >= 0 {
			votersIndices = append(votersIndices, int32(i))
			v[i] = voter		// replace to preserve its VotingPower
		}
	}

	return &ctypes.ResultValidators{
		BlockHeight:  height,
		Validators:   v,
		VoterIndices: votersIndices,
		Count:        len(v),
		Total:        totalCount}, nil
}

// Voters gets the voters set at the given block height.
//
// If no height is provided, it will fetch the latest validator set. Note the
// voters are sorted by their voting power - this is the canonical order
// for the voters in the set as used in computing their Merkle root.
//
// More: https://docs.tendermint.com/master/rpc/#/Info/validators
func Voters(ctx *rpctypes.Context, heightPtr *int64, pagePtr, perPagePtr *int) (*ctypes.ResultVoters, error) {
	return voters(ctx, heightPtr, pagePtr, perPagePtr,
		func(height int64, voterParams *types.VoterParams) (*types.VoterSet, error) {
			return env.StateStore.LoadVoters(height, voterParams)
		})
}

func voters(ctx *rpctypes.Context, heightPtr *int64, pagePtr, perPagePtr *int,
	loadFunc func(height int64, voterParams *types.VoterParams) (*types.VoterSet, error)) (*ctypes.ResultVoters, error) {

	// The latest validator that we know is the NextValidator of the last block.
	height, err := getHeight(latestUncommittedHeight(), heightPtr)
	if err != nil {
		return nil, err
	}

	vals, err := loadFunc(height, env.ConsensusState.GetState().VoterParams)
	if err != nil {
		return nil, err
	}

	totalCount := len(vals.Voters)
	perPage := validatePerPage(perPagePtr)
	page, err := validatePage(pagePtr, perPage, totalCount)
	if err != nil {
		return nil, err
	}

	skipCount := validateSkipCount(page, perPage)

	v := vals.Voters[skipCount : skipCount+tmmath.MinInt(perPage, totalCount-skipCount)]

	return &ctypes.ResultVoters{
		BlockHeight: height,
		Voters:      v,
		Count:       len(v),
		Total:       totalCount}, nil
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

	consensusParams, err := env.StateStore.LoadConsensusParams(height)
	if err != nil {
		return nil, err
	}
	return &ctypes.ResultConsensusParams{
		BlockHeight:     height,
		ConsensusParams: consensusParams}, nil
}
