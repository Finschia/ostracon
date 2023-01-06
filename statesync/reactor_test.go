package statesync

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	tmabci "github.com/tendermint/tendermint/abci/types"
	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"
	ssproto "github.com/tendermint/tendermint/proto/tendermint/statesync"
	tmversion "github.com/tendermint/tendermint/proto/tendermint/version"

	"github.com/line/ostracon/config"
	"github.com/line/ostracon/libs/log"
	"github.com/line/ostracon/p2p"
	p2pmocks "github.com/line/ostracon/p2p/mocks"
	"github.com/line/ostracon/proxy"
	proxymocks "github.com/line/ostracon/proxy/mocks"
	sm "github.com/line/ostracon/state"
	"github.com/line/ostracon/statesync/mocks"
	"github.com/line/ostracon/types"
	"github.com/line/ostracon/version"
)

func TestReactor_Receive_ChunkRequest(t *testing.T) {
	testcases := map[string]struct {
		request        *ssproto.ChunkRequest
		chunk          []byte
		expectResponse *ssproto.ChunkResponse
	}{
		"chunk is returned": {
			&ssproto.ChunkRequest{Height: 1, Format: 1, Index: 1},
			[]byte{1, 2, 3},
			&ssproto.ChunkResponse{Height: 1, Format: 1, Index: 1, Chunk: []byte{1, 2, 3}}},
		"empty chunk is returned, as nil": {
			&ssproto.ChunkRequest{Height: 1, Format: 1, Index: 1},
			[]byte{},
			&ssproto.ChunkResponse{Height: 1, Format: 1, Index: 1, Chunk: nil}},
		"nil (missing) chunk is returned as missing": {
			&ssproto.ChunkRequest{Height: 1, Format: 1, Index: 1},
			nil,
			&ssproto.ChunkResponse{Height: 1, Format: 1, Index: 1, Missing: true},
		},
	}

	for name, tc := range testcases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			// Mock ABCI connection to return local snapshots
			conn := &proxymocks.AppConnSnapshot{}
			conn.On("LoadSnapshotChunkSync", tmabci.RequestLoadSnapshotChunk{
				Height: tc.request.Height,
				Format: tc.request.Format,
				Chunk:  tc.request.Index,
			}).Return(&tmabci.ResponseLoadSnapshotChunk{Chunk: tc.chunk}, nil)

			// Mock peer to store response, if found
			peer := &p2pmocks.Peer{}
			peer.On("ID").Return(p2p.ID("id"))
			var response *ssproto.ChunkResponse
			if tc.expectResponse != nil {
				peer.On("Send", ChunkChannel, mock.Anything).Run(func(args mock.Arguments) {
					msg, err := decodeMsg(args[1].([]byte))
					require.NoError(t, err)
					response = msg.(*ssproto.ChunkResponse)
				}).Return(true)
			}

			// Start a reactor and send a ssproto.ChunkRequest, then wait for and check response
			cfg := config.DefaultStateSyncConfig()
			r := NewReactor(*cfg, conn, nil, true, 1000)
			err := r.Start()
			require.NoError(t, err)
			t.Cleanup(func() {
				if err := r.Stop(); err != nil {
					t.Error(err)
				}
			})

			r.Receive(ChunkChannel, peer, mustEncodeMsg(tc.request))
			time.Sleep(100 * time.Millisecond)
			assert.Equal(t, tc.expectResponse, response)

			conn.AssertExpectations(t)
			peer.AssertExpectations(t)
		})
	}
}

func TestReactor_Receive_SnapshotsRequest(t *testing.T) {
	testcases := map[string]struct {
		snapshots       []*tmabci.Snapshot
		expectResponses []*ssproto.SnapshotsResponse
	}{
		"no snapshots": {nil, []*ssproto.SnapshotsResponse{}},
		">10 unordered snapshots": {
			[]*tmabci.Snapshot{
				{Height: 1, Format: 2, Chunks: 7, Hash: []byte{1, 2}, Metadata: []byte{1}},
				{Height: 2, Format: 2, Chunks: 7, Hash: []byte{2, 2}, Metadata: []byte{2}},
				{Height: 3, Format: 2, Chunks: 7, Hash: []byte{3, 2}, Metadata: []byte{3}},
				{Height: 1, Format: 1, Chunks: 7, Hash: []byte{1, 1}, Metadata: []byte{4}},
				{Height: 2, Format: 1, Chunks: 7, Hash: []byte{2, 1}, Metadata: []byte{5}},
				{Height: 3, Format: 1, Chunks: 7, Hash: []byte{3, 1}, Metadata: []byte{6}},
				{Height: 1, Format: 4, Chunks: 7, Hash: []byte{1, 4}, Metadata: []byte{7}},
				{Height: 2, Format: 4, Chunks: 7, Hash: []byte{2, 4}, Metadata: []byte{8}},
				{Height: 3, Format: 4, Chunks: 7, Hash: []byte{3, 4}, Metadata: []byte{9}},
				{Height: 1, Format: 3, Chunks: 7, Hash: []byte{1, 3}, Metadata: []byte{10}},
				{Height: 2, Format: 3, Chunks: 7, Hash: []byte{2, 3}, Metadata: []byte{11}},
				{Height: 3, Format: 3, Chunks: 7, Hash: []byte{3, 3}, Metadata: []byte{12}},
			},
			[]*ssproto.SnapshotsResponse{
				{Height: 3, Format: 4, Chunks: 7, Hash: []byte{3, 4}, Metadata: []byte{9}},
				{Height: 3, Format: 3, Chunks: 7, Hash: []byte{3, 3}, Metadata: []byte{12}},
				{Height: 3, Format: 2, Chunks: 7, Hash: []byte{3, 2}, Metadata: []byte{3}},
				{Height: 3, Format: 1, Chunks: 7, Hash: []byte{3, 1}, Metadata: []byte{6}},
				{Height: 2, Format: 4, Chunks: 7, Hash: []byte{2, 4}, Metadata: []byte{8}},
				{Height: 2, Format: 3, Chunks: 7, Hash: []byte{2, 3}, Metadata: []byte{11}},
				{Height: 2, Format: 2, Chunks: 7, Hash: []byte{2, 2}, Metadata: []byte{2}},
				{Height: 2, Format: 1, Chunks: 7, Hash: []byte{2, 1}, Metadata: []byte{5}},
				{Height: 1, Format: 4, Chunks: 7, Hash: []byte{1, 4}, Metadata: []byte{7}},
				{Height: 1, Format: 3, Chunks: 7, Hash: []byte{1, 3}, Metadata: []byte{10}},
			},
		},
	}

	for name, tc := range testcases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			// Mock ABCI connection to return local snapshots
			conn := &proxymocks.AppConnSnapshot{}
			conn.On("ListSnapshotsSync", tmabci.RequestListSnapshots{}).Return(&tmabci.ResponseListSnapshots{
				Snapshots: tc.snapshots,
			}, nil)

			// Mock peer to catch responses and store them in a slice
			responses := []*ssproto.SnapshotsResponse{}
			peer := &p2pmocks.Peer{}
			if len(tc.expectResponses) > 0 {
				peer.On("ID").Return(p2p.ID("id"))
				peer.On("Send", SnapshotChannel, mock.Anything).Run(func(args mock.Arguments) {
					msg, err := decodeMsg(args[1].([]byte))
					require.NoError(t, err)
					responses = append(responses, msg.(*ssproto.SnapshotsResponse))
				}).Return(true)
			}

			// Start a reactor and send a SnapshotsRequestMessage, then wait for and check responses
			cfg := config.DefaultStateSyncConfig()
			r := NewReactor(*cfg, conn, nil, true, 1000)
			err := r.Start()
			require.NoError(t, err)
			t.Cleanup(func() {
				if err := r.Stop(); err != nil {
					t.Error(err)
				}
			})

			r.Receive(SnapshotChannel, peer, mustEncodeMsg(&ssproto.SnapshotsRequest{}))
			time.Sleep(100 * time.Millisecond)
			assert.Equal(t, tc.expectResponses, responses)

			conn.AssertExpectations(t)
			peer.AssertExpectations(t)
		})
	}
}

func makeTestStateSyncReactor(
	t *testing.T, appHash string, height int64, snapshot *snapshot, chunks []*chunk) *Reactor {
	connSnapshot := makeMockAppConnSnapshot(appHash, snapshot, chunks)
	connQuery := makeMockAppConnQuery(appHash, height)

	p2pConfig := config.DefaultP2PConfig()
	p2pConfig.PexReactor = true
	p2pConfig.AllowDuplicateIP = true

	name := "STATE_SYNC_REACTOR_FOR_TEST"
	size := 2
	reactors := make([]*Reactor, size)
	initSwitch := func(i int, s *p2p.Switch, p2pConfig *config.P2PConfig) *p2p.Switch {
		logger := log.TestingLogger()
		cfg := config.DefaultStateSyncConfig()
		reactors[i] = NewReactor(*cfg, connSnapshot, connQuery, true, 1000)
		reactors[i].SetLogger(logger)
		reactors[i].SetSwitch(s)

		s.AddReactor(name, reactors[i])
		s.SetLogger(logger)
		return s
	}
	switches := p2p.MakeConnectedSwitches(p2pConfig, size, initSwitch, p2p.Connect2Switches)

	t.Cleanup(func() {
		for i := 0; i < size; i++ {
			if err := switches[i].Stop(); err != nil {
				t.Error(err)
			}
		}
	})

	return reactors[0]
}

func makeMockAppConnSnapshot(appHash string, snapshot *snapshot, chunks []*chunk) *proxymocks.AppConnSnapshot {
	connSnapshot := &proxymocks.AppConnSnapshot{}

	connSnapshot.On("OfferSnapshotSync", tmabci.RequestOfferSnapshot{
		Snapshot: &tmabci.Snapshot{
			Height: snapshot.Height,
			Format: snapshot.Format,
			Chunks: snapshot.Chunks,
			Hash:   snapshot.Hash,
		},
		AppHash: []byte(appHash),
	}).Return(&tmabci.ResponseOfferSnapshot{Result: tmabci.ResponseOfferSnapshot_ACCEPT}, nil)

	connSnapshot.On("ListSnapshotsSync", tmabci.RequestListSnapshots{}).Return(&tmabci.ResponseListSnapshots{
		Snapshots: []*tmabci.Snapshot{{
			Height:   snapshot.Height,
			Format:   snapshot.Format,
			Chunks:   snapshot.Chunks,
			Hash:     snapshot.Hash,
			Metadata: snapshot.Metadata,
		}},
	}, nil)

	index := len(chunks)
	for i := 0; i < index; i++ {
		connSnapshot.On("ApplySnapshotChunkSync",
			mock.Anything).Return(&tmabci.ResponseApplySnapshotChunk{Result: tmabci.ResponseApplySnapshotChunk_ACCEPT}, nil)
		connSnapshot.On("LoadSnapshotChunkSync", tmabci.RequestLoadSnapshotChunk{
			Height: chunks[i].Height, Format: chunks[i].Format, Chunk: chunks[i].Index,
		}).Return(&tmabci.ResponseLoadSnapshotChunk{Chunk: chunks[i].Chunk}, nil)
	}

	return connSnapshot
}

func makeMockAppConnQuery(appHash string, height int64) *proxymocks.AppConnQuery {
	connQuery := &proxymocks.AppConnQuery{}
	connQuery.On("InfoSync", proxy.RequestInfo).Return(&tmabci.ResponseInfo{
		AppVersion:       testAppVersion,
		LastBlockHeight:  height,
		LastBlockAppHash: []byte(appHash),
	}, nil)
	return connQuery
}

func makeTestStateAndCommit(appHash string, height int64) (sm.State, *types.Commit) {
	blockHash := "block_hash"

	state := sm.State{
		ChainID: "chain",
		Version: tmstate.Version{
			Consensus: tmversion.Consensus{
				Block: version.BlockProtocol,
				App:   testAppVersion,
			},

			Software: version.OCCoreSemVer,
		},

		LastBlockHeight: height,
		LastBlockID:     types.BlockID{Hash: []byte(blockHash)},
		LastBlockTime:   time.Now(),
		LastResultsHash: []byte("last_results_hash"),
		AppHash:         []byte(appHash),

		LastValidators: &types.ValidatorSet{},
		Validators:     &types.ValidatorSet{},
		NextValidators: &types.ValidatorSet{},

		ConsensusParams:                  *types.DefaultConsensusParams(),
		LastHeightConsensusParamsChanged: 1,
	}
	state.ConsensusParams.Version.AppVersion = testAppVersion

	commit := &types.Commit{BlockID: types.BlockID{Hash: []byte(blockHash)}}

	return state, commit
}

func makeMockStateProvider(appHash string, height uint64) *mocks.StateProvider {
	state, commit := makeTestStateAndCommit(appHash, int64(height))

	stateProvider := &mocks.StateProvider{}
	stateProvider.On("AppHash", mock.Anything, mock.Anything).Return([]byte(appHash), nil)
	stateProvider.On("State", mock.Anything, height).Return(state, nil)
	stateProvider.On("Commit", mock.Anything, height).Return(commit, nil)
	return stateProvider
}

func TestSync(t *testing.T) {
	// prepare
	height, format := uint64(1), uint32(1)
	appHash := "app_hash"

	chunks := []*chunk{
		{Height: height, Format: format, Index: 0, Chunk: []byte{1, 1, 0}},
		{Height: height, Format: format, Index: 1, Chunk: []byte{1, 1, 1}},
		{Height: height, Format: format, Index: 2, Chunk: []byte{1, 1, 2}},
	}
	snapshot := &snapshot{Height: height, Format: format, Chunks: uint32(len(chunks)), Hash: []byte{1, 2, 3}}
	reactor := makeTestStateSyncReactor(t, appHash, int64(height), snapshot, chunks)
	stateProvider := makeMockStateProvider(appHash, height)

	// test
	state, previousState, commit, err := reactor.Sync(stateProvider, 5*time.Second)

	// verify
	require.NoError(t, err)
	require.NotNil(t, state)
	require.NotNil(t, previousState)
	require.NotNil(t, commit)
}
