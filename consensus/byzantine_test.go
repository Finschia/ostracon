package consensus

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	config2 "github.com/tendermint/tendermint/config"

	"github.com/tendermint/tendermint/libs/service"
	"github.com/tendermint/tendermint/p2p"
	sm "github.com/tendermint/tendermint/state"
	"github.com/tendermint/tendermint/types"
)

//----------------------------------------------
// byzantine failures

// 4 validators. 1 is byzantine. The other three are partitioned into A (1 val) and B (2 vals).
// byzantine validator sends conflicting proposals into A and B,
// and prevotes/precommits on both of them.
// B sees a commit, A doesn't.
// Byzantine validator refuses to prevote.
// Heal partition and ensure A sees the commit
func TestByzantine(t *testing.T) {
	N := 4
	logger := consensusLogger().With("test", "byzantine")
	css, cleanup := randConsensusNet(N, "consensus_byzantine_test", newMockTickerFunc(false), newCounter)
	defer cleanup()

	// get proposer of first block
	proposerIdx, _ := findProposer(css[0])

	// give the byzantine validator a normal ticker
	ticker := NewTimeoutTicker()
	ticker.SetLogger(css[0].Logger)
	css[0].SetTimeoutTicker(ticker)

	switches := make([]*p2p.Switch, N)
	p2pLogger := logger.With("module", "p2p")
	for i := 0; i < N; i++ {
		switches[i] = p2p.MakeSwitch(
			config.P2P,
			i,
			"foo", "1.0.0",
			func(i int, sw *p2p.Switch, config *config2.P2PConfig) *p2p.Switch {
				return sw
			})
		switches[i].SetLogger(p2pLogger.With("validator", i))
	}

	blocksSubs := make([]types.Subscription, N)
	reactors := make([]p2p.Reactor, N)
	for i := 0; i < N; i++ {
		// make first val byzantine
		if i == proposerIdx {
			// NOTE: Now, test validators are MockPV, which by default doesn't
			// do any safety checks.
			css[i].privValidator.(types.MockPV).DisableChecks()
			css[i].decideProposal = func(j int) func(int64, int) {
				return func(height int64, round int) {
					byzantineDecideProposalFunc(t, height, round, css[j], switches[j])
				}
			}(i)
			css[i].doPrevote = func(height int64, round int) {}
		}

		eventBus := css[i].eventBus
		eventBus.SetLogger(logger.With("module", "events", "validator", i))

		var err error
		blocksSubs[i], err = eventBus.Subscribe(context.Background(), testSubscriber, types.EventQueryNewBlock)
		require.NoError(t, err)

		conR := NewReactor(css[i], true, true, 1000) // so we don't start the consensus states
		conR.SetLogger(logger.With("validator", i))
		conR.SetEventBus(eventBus)

		var conRI p2p.Reactor = conR

		// make first val byzantine
		if i == proposerIdx {
			conRI = NewByzantineReactor(conR)
		}

		reactors[i] = conRI
		sm.SaveState(css[i].blockExec.DB(), css[i].state) //for save height 1's validators info
	}

	defer func() {
		for _, r := range reactors {
			if rr, ok := r.(*ByzantineReactor); ok {
				rr.reactor.Switch.Stop()
			} else {
				r.(*Reactor).Switch.Stop()
			}
		}
	}()

	p2p.MakeConnectedSwitches(config.P2P, N, func(i int, s *p2p.Switch, config *config2.P2PConfig) *p2p.Switch {
		// ignore new switch s, we already made ours
		switches[i].AddReactor("CONSENSUS", reactors[i])
		return switches[i]
	}, func(sws []*p2p.Switch, i, j int) {
		// the network starts partitioned with globally active adversary
		if i != proposerIdx && j != proposerIdx {
			return
		}
		p2p.Connect2Switches(sws, i, j)
	})

	// start the non-byz state machines.
	// note these must be started before the byz
	for i := 0; i < N; i++ {
		if i != proposerIdx {
			cr := reactors[i].(*Reactor)
			cr.SwitchToConsensus(cr.conS.GetState(), 0)
		}
	}

	// start the byzantine state machine
	byzR := reactors[proposerIdx].(*ByzantineReactor)
	s := byzR.reactor.conS.GetState()
	byzR.reactor.SwitchToConsensus(s, 0)

	// byz proposer sends one block to peers[0]
	// and the other block to peers[1] and peers[2].
	// note peers and switches order don't match.
	peers := switches[proposerIdx].Peers().List()

	// partition A
	ind0 := getSwitchIndex(switches, peers[0])

	// partition B
	ind1 := getSwitchIndex(switches, peers[1])
	ind2 := getSwitchIndex(switches, peers[2])
	p2p.Connect2Switches(switches, ind1, ind2)

	// FIXME: test stops at the following step after the introduction of VRF elections
	// wait for someone in the big partition (B) to make a block
	<-blocksSubs[ind2].Out()

	t.Log("A block has been committed. Healing partition")
	p2p.Connect2Switches(switches, ind0, ind1)
	p2p.Connect2Switches(switches, ind0, ind2)

	// wait till everyone makes the first new block
	// (one of them already has)
	wg := new(sync.WaitGroup)
	wg.Add(2)
	for i := 0; i < N-1; i++ {
		if i != proposerIdx {
			go func(j int) {
				<-blocksSubs[j].Out()
				wg.Done()
			}(i)
		}
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	tick := time.NewTicker(time.Second * 10)
	select {
	case <-done:
	case <-tick.C:
		for i, reactor := range reactors {
			t.Log(fmt.Sprintf("Consensus Reactor %v", i))
			t.Log(fmt.Sprintf("%v", reactor))
		}
		t.Fatalf("Timed out waiting for all validators to commit first block")
	}
}

// find proposer of current height and round from State
func findProposer(state *State) (int, *types.Validator) {
	proposer := state.Validators.SelectProposer(state.state.LastProofHash, state.Height, state.Round)
	return state.Voters.GetByAddress(proposer.PubKey.Address())
}

//-------------------------------
// byzantine consensus functions

func byzantineDecideProposalFunc(t *testing.T, height int64, round int, cs *State, sw *p2p.Switch) {
	// byzantine user should create two proposals and try to split the vote.
	// Avoid sending on internalMsgQueue and running consensus state.

	// Create a new proposal block from state/txs from the mempool.
	block1, blockParts1 := cs.createProposalBlock(round)
	polRound, propBlockID := cs.ValidRound, types.BlockID{Hash: block1.Hash(), PartsHeader: blockParts1.Header()}
	proposal1 := types.NewProposal(height, round, polRound, propBlockID)
	if err := cs.privValidator.SignProposal(cs.state.ChainID, proposal1); err != nil {
		t.Error(err)
	}

	// Create a new proposal block from state/txs from the mempool.
	block2, blockParts2 := cs.createProposalBlock(round)
	polRound, propBlockID = cs.ValidRound, types.BlockID{Hash: block2.Hash(), PartsHeader: blockParts2.Header()}
	proposal2 := types.NewProposal(height, round, polRound, propBlockID)
	if err := cs.privValidator.SignProposal(cs.state.ChainID, proposal2); err != nil {
		t.Error(err)
	}

	block1Hash := block1.Hash()
	block2Hash := block2.Hash()

	// broadcast conflicting proposals/block parts to peers
	peers := sw.Peers().List()
	t.Logf("Byzantine: broadcasting conflicting proposals to %d peers", len(peers))
	for i, peer := range peers {
		if i < len(peers)/2 {
			go sendProposalAndParts(height, round, cs, peer, proposal1, block1Hash, blockParts1)
		} else {
			go sendProposalAndParts(height, round, cs, peer, proposal2, block2Hash, blockParts2)
		}
	}
}

func sendProposalAndParts(
	height int64,
	round int,
	cs *State,
	peer p2p.Peer,
	proposal *types.Proposal,
	blockHash []byte,
	parts *types.PartSet,
) {
	// proposal
	msg := &ProposalMessage{Proposal: proposal}
	peer.Send(DataChannel, cdc.MustMarshalBinaryBare(msg))

	// parts
	for i := 0; i < parts.Total(); i++ {
		part := parts.GetPart(i)
		msg := &BlockPartMessage{
			Height: height, // This tells peer that this part applies to us.
			Round:  round,  // This tells peer that this part applies to us.
			Part:   part,
		}
		peer.Send(DataChannel, cdc.MustMarshalBinaryBare(msg))
	}

	// votes
	cs.mtx.Lock()
	prevote, _ := cs.signVote(types.PrevoteType, blockHash, parts.Header())
	precommit, _ := cs.signVote(types.PrecommitType, blockHash, parts.Header())
	cs.mtx.Unlock()

	peer.Send(VoteChannel, cdc.MustMarshalBinaryBare(&VoteMessage{prevote}))
	peer.Send(VoteChannel, cdc.MustMarshalBinaryBare(&VoteMessage{precommit}))
}

//----------------------------------------
// byzantine consensus reactor

type ByzantineReactor struct {
	service.Service
	reactor *Reactor
}

func NewByzantineReactor(conR *Reactor) *ByzantineReactor {
	return &ByzantineReactor{
		Service: conR,
		reactor: conR,
	}
}

func (br *ByzantineReactor) SetSwitch(s *p2p.Switch)               { br.reactor.SetSwitch(s) }
func (br *ByzantineReactor) GetChannels() []*p2p.ChannelDescriptor { return br.reactor.GetChannels() }
func (br *ByzantineReactor) AddPeer(peer p2p.Peer) {
	if !br.reactor.IsRunning() {
		return
	}

	// Create peerState for peer
	peerState := NewPeerState(peer).SetLogger(br.reactor.Logger)
	peer.Set(types.PeerStateKey, peerState)

	// Send our state to peer.
	// If we're fast_syncing, broadcast a RoundStepMessage later upon SwitchToConsensus().
	if !br.reactor.fastSync {
		br.reactor.sendNewRoundStepMessage(peer)
	}
}
func (br *ByzantineReactor) RemovePeer(peer p2p.Peer, reason interface{}) {
	br.reactor.RemovePeer(peer, reason)
}
func (br *ByzantineReactor) Receive(chID byte, peer p2p.Peer, msgBytes []byte) {
	br.reactor.Receive(chID, peer, msgBytes)
}
func (br *ByzantineReactor) InitPeer(peer p2p.Peer) p2p.Peer { return peer }
func (br *ByzantineReactor) RecvRoutine() {
	br.reactor.RecvRoutine()
}

func (br *ByzantineReactor) GetRecvChan() chan *p2p.BufferedMsg {
	return br.reactor.GetRecvChan()
}
