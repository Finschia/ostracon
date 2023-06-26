package state

import (
	"errors"
	"fmt"
	"time"

	abci "github.com/tendermint/tendermint/abci/types"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	ocabci "github.com/Finschia/ostracon/abci/types"
	"github.com/Finschia/ostracon/crypto"
	cryptoenc "github.com/Finschia/ostracon/crypto/encoding"
	"github.com/Finschia/ostracon/libs/fail"
	"github.com/Finschia/ostracon/libs/log"
	mempl "github.com/Finschia/ostracon/mempool"
	tmstate "github.com/Finschia/ostracon/proto/ostracon/state"
	"github.com/Finschia/ostracon/proxy"
	"github.com/Finschia/ostracon/types"
	canonictime "github.com/Finschia/ostracon/types/time"
	vrf "github.com/oasisprotocol/curve25519-voi/primitives/ed25519/extra/ecvrf"
)

//-----------------------------------------------------------------------------
// BlockExecutor handles block execution and state updates.
// It exposes ApplyBlock(), which validates & executes the block, updates state w/ ABCI responses,
// then commits and updates the mempool atomically, then saves state.

// BlockExecutor provides the context and accessories for properly executing a block.
type BlockExecutor struct {
	// save state, validators, consensus params, abci responses here
	store Store

	// execute the app against this
	proxyApp proxy.AppConnConsensus

	// events
	eventBus types.BlockEventPublisher

	// manage the mempool lock during commit
	// and update both with block results after commit.
	mempool mempl.Mempool
	evpool  EvidencePool

	logger log.Logger

	metrics *Metrics
}

type CommitStepTimes struct {
	CommitExecuting  types.StepDuration
	CommitCommitting types.StepDuration
	CommitRechecking types.StepDuration
	Current          *types.StepDuration
}

func (st *CommitStepTimes) ToNextStep(from, next *types.StepDuration) time.Time {
	now := canonictime.Now()
	if st.Current == from {
		from.End, next.Start = now, now
		st.Current = next
	}
	return now
}

func (st *CommitStepTimes) ToCommitCommitting() time.Time {
	return st.ToNextStep(&st.CommitExecuting, &st.CommitCommitting)
}

func (st *CommitStepTimes) ToCommitRechecking() time.Time {
	return st.ToNextStep(&st.CommitCommitting, &st.CommitRechecking)
}

type BlockExecutorOption func(executor *BlockExecutor)

func BlockExecutorWithMetrics(metrics *Metrics) BlockExecutorOption {
	return func(blockExec *BlockExecutor) {
		blockExec.metrics = metrics
	}
}

// NewBlockExecutor returns a new BlockExecutor with a NopEventBus.
// Call SetEventBus to provide one.
func NewBlockExecutor(
	stateStore Store,
	logger log.Logger,
	proxyApp proxy.AppConnConsensus,
	mempool mempl.Mempool,
	evpool EvidencePool,
	options ...BlockExecutorOption,
) *BlockExecutor {
	res := &BlockExecutor{
		store:    stateStore,
		proxyApp: proxyApp,
		eventBus: types.NopEventBus{},
		mempool:  mempool,
		evpool:   evpool,
		logger:   logger,
		metrics:  NopMetrics(),
	}

	for _, option := range options {
		option(res)
	}

	return res
}

func (blockExec *BlockExecutor) Store() Store {
	return blockExec.store
}

// SetEventBus - sets the event bus for publishing block related events.
// If not called, it defaults to types.NopEventBus.
func (blockExec *BlockExecutor) SetEventBus(eventBus types.BlockEventPublisher) {
	blockExec.eventBus = eventBus
}

// CreateProposalBlock calls state.MakeBlock with evidence from the evpool
// and txs from the mempool. The max bytes must be big enough to fit the commit.
// Up to 1/10th of the block space is allcoated for maximum sized evidence.
// The rest is given to txs, up to the max gas.
func (blockExec *BlockExecutor) CreateProposalBlock(
	height int64,
	state State, commit *types.Commit,
	proposerAddr []byte,
	round int32,
	proof crypto.Proof,
	maxTxs int64,
) (*types.Block, *types.PartSet) {

	maxBytes := state.ConsensusParams.Block.MaxBytes
	maxGas := state.ConsensusParams.Block.MaxGas

	evidence, evSize := blockExec.evpool.PendingEvidence(state.ConsensusParams.Evidence.MaxBytes)

	// Fetch a limited amount of valid txs
	maxDataBytes := types.MaxDataBytes(maxBytes, evSize, state.Validators.Size())

	txs := blockExec.mempool.ReapMaxBytesMaxGasMaxTxs(maxDataBytes, maxGas, maxTxs)

	return state.MakeBlock(height, txs, commit, evidence, proposerAddr, round, proof)
}

// ValidateBlock validates the given block against the given state.
// If the block is invalid, it returns an error.
// Validation does not mutate state, but does require historical information from the stateDB,
// ie. to verify evidence from a validator at an old height.
func (blockExec *BlockExecutor) ValidateBlock(state State, round int32, block *types.Block) error {
	err := validateBlock(state, round, block)
	if err != nil {
		return err
	}
	return blockExec.evpool.CheckEvidence(block.Evidence.Evidence)
}

// ApplyBlock validates the block against the state, executes it against the app,
// fires the relevant events, commits the app, and saves the new state and responses.
// It returns the new state and the block height to retain (pruning older blocks).
// It's the only function that needs to be called
// from outside this package to process and commit an entire block.
// It takes a blockID to avoid recomputing the parts hash.
func (blockExec *BlockExecutor) ApplyBlock(
	state State, blockID types.BlockID, block *types.Block, stepTimes *CommitStepTimes,
) (State, int64, error) {

	// When doing ApplyBlock, we don't need to check whether the block.Round is same to current round,
	// so we just put block.Round for the current round parameter
	if err := blockExec.ValidateBlock(state, block.Round, block); err != nil {
		return state, 0, ErrInvalidBlock(err)
	}

	execStartTime := time.Now().UnixNano()
	abciResponses, err := execBlockOnProxyApp(
		blockExec.logger, blockExec.proxyApp, block, blockExec.store, state.InitialHeight,
	)
	execEndTime := time.Now().UnixNano()

	execTimeMs := float64(execEndTime-execStartTime) / 1000000
	blockExec.metrics.BlockProcessingTime.Observe(execTimeMs)
	blockExec.metrics.BlockExecutionTime.Set(execTimeMs)

	if err != nil {
		return state, 0, ErrProxyAppConn(err)
	}

	fail.Fail() // XXX

	// Save the results before we commit.
	if err := blockExec.store.SaveABCIResponses(block.Height, abciResponses); err != nil {
		return state, 0, err
	}

	fail.Fail() // XXX

	// validate the validator updates and convert to ostracon types
	abciValUpdates := abciResponses.EndBlock.ValidatorUpdates
	err = validateValidatorUpdates(abciValUpdates, state.ConsensusParams.Validator)
	if err != nil {
		return state, 0, fmt.Errorf("error in validator updates: %v", err)
	}

	validatorUpdates, err := types.PB2OC.ValidatorUpdates(abciValUpdates)
	if err != nil {
		return state, 0, err
	}
	if len(validatorUpdates) > 0 {
		blockExec.logger.Debug("updates to validators", "updates", types.ValidatorListString(validatorUpdates))
	}

	// Update the state with the block and responses.
	state, err = updateState(state, blockID, &block.Header, &block.Entropy, abciResponses, validatorUpdates)
	if err != nil {
		return state, 0, fmt.Errorf("commit failed for application: %v", err)
	}

	if stepTimes != nil {
		stepTimes.ToCommitCommitting()
	}

	// Lock mempool, commit app state, update mempoool.
	commitStartTime := time.Now().UnixNano()
	appHash, retainHeight, err := blockExec.Commit(state, block, abciResponses.DeliverTxs, stepTimes)
	commitEndTime := time.Now().UnixNano()

	commitTimeMs := float64(commitEndTime-commitStartTime) / 1000000
	blockExec.metrics.BlockCommitTime.Set(commitTimeMs)

	if err != nil {
		return state, 0, fmt.Errorf("commit failed for application: %v", err)
	}

	// Update evpool with the latest state.
	blockExec.evpool.Update(state, block.Evidence.Evidence)

	fail.Fail() // XXX

	// Update the app hash and save the state.
	state.AppHash = appHash
	if err := blockExec.store.Save(state); err != nil {
		return state, 0, err
	}

	fail.Fail() // XXX

	// Can't use stepTimes at this point as it gets wrapped up by the caller of this function
	blockGenerationTimeMs := float64((time.Now().UnixNano() - execStartTime) / 1000000.0)
	numTxs := len(block.Txs)
	tps := 0
	if blockGenerationTimeMs > 0 {
		tps = int(float64(numTxs) / blockGenerationTimeMs * 1000.0)
	}
	blockExec.logger.Info(
		"block generated",
		"height", block.Height,
		"num_txs", numTxs,
		"generation_time", blockGenerationTimeMs/1000.0,
		"tps", tps,
	)

	// Events are fired after everything else.
	// NOTE: if we crash between Commit and Save, events wont be fired during replay
	fireEvents(blockExec.logger, blockExec.eventBus, block, abciResponses, validatorUpdates)

	return state, retainHeight, nil
}

// Commit locks the mempool, runs the ABCI Commit message, and updates the
// mempool.
// It returns the result of calling abci.Commit (the AppHash) and the height to retain (if any).
// The Mempool must be locked during commit and update because state is
// typically reset on Commit and old txs must be replayed against committed
// state before new txs are run in the mempool, lest they be invalid.
func (blockExec *BlockExecutor) Commit(
	state State,
	block *types.Block,
	deliverTxResponses []*abci.ResponseDeliverTx,
	stepTimes *CommitStepTimes,
) ([]byte, int64, error) {
	blockExec.mempool.Lock()
	defer blockExec.mempool.Unlock()

	// while mempool is Locked, flush to ensure all async requests have completed
	// in the ABCI app before Commit.
	err := blockExec.mempool.FlushAppConn()
	if err != nil {
		blockExec.logger.Error("client error during mempool.FlushAppConn", "err", err)
		return nil, 0, err
	}

	// Commit block, get hash back
	appCommitStartTime := time.Now().UnixNano()
	res, err := blockExec.proxyApp.CommitSync()
	appCommitEndTime := time.Now().UnixNano()

	appCommitTimeMs := float64(appCommitEndTime-appCommitStartTime) / 1000000
	blockExec.metrics.BlockAppCommitTime.Set(appCommitTimeMs)

	if err != nil {
		blockExec.logger.Error("client error during proxyAppConn.CommitSync", "err", err)
		return nil, 0, err
	}

	// ResponseCommit has no error code - just data

	if stepTimes != nil {
		stepTimes.ToCommitRechecking()
	}

	// Update mempool.
	updateMempoolStartTime := time.Now().UnixNano()
	err = blockExec.mempool.Update(
		block,
		deliverTxResponses,
		TxPreCheck(state),
		TxPostCheck(state),
	)
	updateMempoolEndTime := time.Now().UnixNano()

	updateMempoolTimeMs := float64(updateMempoolEndTime-updateMempoolStartTime) / 1000000
	blockExec.metrics.BlockUpdateMempoolTime.Set(updateMempoolTimeMs)

	blockExec.logger.Info(
		"committed state",
		"height", block.Height,
		"num_txs", len(block.Txs),
		"app_hash", fmt.Sprintf("%X", res.Data),
		"commit_time", float64(int(appCommitTimeMs))/1000.0,
		"update_mempool_time", float64(int(updateMempoolTimeMs))/1000.0,
	)

	return res.Data, res.RetainHeight, err
}

//---------------------------------------------------------
// Helper functions for executing blocks and updating state

// Executes block's transactions on proxyAppConn.
// Returns a list of transaction results and updates to the validator set
func execBlockOnProxyApp(
	logger log.Logger,
	proxyAppConn proxy.AppConnConsensus,
	block *types.Block,
	store Store,
	initialHeight int64,
) (*tmstate.ABCIResponses, error) {
	var validTxs, invalidTxs = 0, 0

	txIndex := 0
	abciResponses := new(tmstate.ABCIResponses)
	dtxs := make([]*abci.ResponseDeliverTx, len(block.Txs))
	abciResponses.DeliverTxs = dtxs

	// Execute transactions and get hash.
	proxyCb := func(req *ocabci.Request, res *ocabci.Response) {
		if r, ok := res.Value.(*ocabci.Response_DeliverTx); ok {
			// TODO: make use of res.Log
			// TODO: make use of this info
			// Blocks may include invalid txs.
			txRes := r.DeliverTx
			if txRes.Code == ocabci.CodeTypeOK {
				validTxs++
			} else {
				logger.Debug("invalid tx", "code", txRes.Code, "log", txRes.Log)
				invalidTxs++
			}

			abciResponses.DeliverTxs[txIndex] = txRes
			txIndex++
		}
	}
	proxyAppConn.SetGlobalCallback(proxyCb)

	commitInfo := getBeginBlockValidatorInfo(block, store, initialHeight)

	byzVals := make([]abci.Evidence, 0)
	for _, evidence := range block.Evidence.Evidence {
		byzVals = append(byzVals, evidence.ABCI()...)
	}

	// Begin block
	var err error
	pbh := block.Header.ToProto()
	if pbh == nil {
		return nil, errors.New("nil header")
	}

	pbe := block.Entropy.ToProto()
	if pbe == nil {
		return nil, errors.New("nil entropy")
	}

	abciResponses.BeginBlock, err = proxyAppConn.BeginBlockSync(ocabci.RequestBeginBlock{
		Hash:                block.Hash(),
		Header:              *pbh,
		LastCommitInfo:      commitInfo,
		ByzantineValidators: byzVals,
		Entropy:             *pbe,
	})
	if err != nil {
		logger.Error("error in proxyAppConn.BeginBlock", "err", err)
		return nil, err
	}

	startTime := time.Now()
	// run txs of block
	for _, tx := range block.Txs {
		proxyAppConn.DeliverTxAsync(abci.RequestDeliverTx{Tx: tx}, nil)
		if err := proxyAppConn.Error(); err != nil {
			return nil, err
		}
	}
	endTime := time.Now()
	execTime := endTime.Sub(startTime)

	// End block.
	abciResponses.EndBlock, err = proxyAppConn.EndBlockSync(abci.RequestEndBlock{Height: block.Height})
	if err != nil {
		logger.Error("error in proxyAppConn.EndBlock", "err", err)
		return nil, err
	}

	tps := 0
	if execTime.Milliseconds() > 0 {
		tps = int(float64(validTxs+invalidTxs) / float64(execTime.Milliseconds()) * 1000.0)
	}
	logger.Info("executed block", "height", block.Height, "num_valid_txs", validTxs,
		"num_invalid_txs", invalidTxs, "exec_time", float64(execTime.Milliseconds())/1000.0, "tps", tps)
	return abciResponses, nil
}

func getBeginBlockValidatorInfo(block *types.Block, store Store,
	initialHeight int64) abci.LastCommitInfo {
	voteInfos := make([]abci.VoteInfo, block.LastCommit.Size())
	// Initial block -> LastCommitInfo.Votes are empty.
	// Remember that the first LastCommit is intentionally empty, so it makes
	// sense for LastCommitInfo.Votes to also be empty.
	if block.Height > initialHeight {
		lastValSet, err := store.LoadValidators(block.Height - 1)
		if err != nil {
			panic(err)
		}

		// Sanity check that commit size matches validator set size - only applies
		// after first block.
		var (
			commitSize = block.LastCommit.Size()
			valSetLen  = len(lastValSet.Validators)
		)
		if commitSize != valSetLen {
			panic(fmt.Sprintf(
				"commit size (%d) doesn't match valset length (%d) at height %d\n\n%v\n\n%v",
				commitSize, valSetLen, block.Height, block.LastCommit.Signatures, lastValSet.Validators,
			))
		}

		for i, val := range lastValSet.Validators {
			commitSig := block.LastCommit.Signatures[i]
			voteInfos[i] = abci.VoteInfo{
				Validator:       types.OC2PB.Validator(val),
				SignedLastBlock: !commitSig.Absent(),
			}
		}
	}

	return abci.LastCommitInfo{
		Round: block.LastCommit.Round,
		Votes: voteInfos,
	}
}

func validateValidatorUpdates(abciUpdates []abci.ValidatorUpdate,
	params tmproto.ValidatorParams) error {
	for _, valUpdate := range abciUpdates {
		if valUpdate.GetPower() < 0 {
			return fmt.Errorf("voting power can't be negative %v", valUpdate)
		} else if valUpdate.GetPower() == 0 {
			// continue, since this is deleting the validator, and thus there is no
			// pubkey to check
			continue
		}

		// Check if validator's pubkey matches an ABCI type in the consensus params
		pk, err := cryptoenc.PubKeyFromProto(&valUpdate.PubKey)
		if err != nil {
			return err
		}

		if !types.IsValidPubkeyType(params, pk.Type()) {
			return fmt.Errorf("validator %v is using pubkey %s, which is unsupported for consensus",
				valUpdate, pk.Type())
		}
	}
	return nil
}

// updateState returns a new State updated according to the header and responses.
func updateState(
	state State,
	blockID types.BlockID,
	header *types.Header,
	entropy *types.Entropy,
	abciResponses *tmstate.ABCIResponses,
	validatorUpdates []*types.Validator,
) (State, error) {

	// Copy the valset so we can apply changes from EndBlock
	// and update s.LastValidators and s.Validators.
	nValSet := state.NextValidators.Copy()

	// Update the validator set with the latest abciResponses.
	lastHeightValsChanged := state.LastHeightValidatorsChanged
	if len(validatorUpdates) > 0 {
		err := nValSet.UpdateWithChangeSet(validatorUpdates)
		if err != nil {
			return state, fmt.Errorf("error changing validator set: %v", err)
		}
		// Change results from this height but only applies to the next next height.
		lastHeightValsChanged = header.Height + 1 + 1
	}

	// Update validator proposer priority and set state variables.
	nValSet.IncrementProposerPriority(1)

	// Update the params with the latest abciResponses.
	nextParams := state.ConsensusParams
	lastHeightParamsChanged := state.LastHeightConsensusParamsChanged
	if abciResponses.EndBlock.ConsensusParamUpdates != nil {
		// NOTE: must not mutate s.ConsensusParams
		nextParams = types.UpdateConsensusParams(state.ConsensusParams, abciResponses.EndBlock.ConsensusParamUpdates)
		err := types.ValidateConsensusParams(nextParams)
		if err != nil {
			return state, fmt.Errorf("error updating consensus params: %v", err)
		}

		state.Version.Consensus.App = nextParams.Version.AppVersion

		// Change results from this height but only applies to the next height.
		lastHeightParamsChanged = header.Height + 1
	}

	nextVersion := state.Version

	// get proof hash from vrf proof
	proofHash, err := vrf.ProofToHash(entropy.Proof.Bytes())
	if err != nil {
		return state, fmt.Errorf("error get proof of hash: %v", err)
	}

	// NOTE: the AppHash has not been populated.
	// It will be filled on state.Save.
	return State{
		Version:                          nextVersion,
		ChainID:                          state.ChainID,
		InitialHeight:                    state.InitialHeight,
		LastBlockHeight:                  header.Height,
		LastBlockID:                      blockID,
		LastBlockTime:                    header.Time,
		LastProofHash:                    proofHash,
		NextValidators:                   nValSet,
		Validators:                       state.NextValidators.Copy(),
		LastValidators:                   state.Validators.Copy(),
		LastHeightValidatorsChanged:      lastHeightValsChanged,
		ConsensusParams:                  nextParams,
		LastHeightConsensusParamsChanged: lastHeightParamsChanged,
		LastResultsHash:                  ABCIResponsesResultsHash(abciResponses),
		AppHash:                          nil,
	}, nil
}

// Fire NewBlock, NewBlockHeader.
// Fire TxEvent for every tx.
// NOTE: if Ostracon crashes before commit, some or all of these events may be published again.
func fireEvents(
	logger log.Logger,
	eventBus types.BlockEventPublisher,
	block *types.Block,
	abciResponses *tmstate.ABCIResponses,
	validatorUpdates []*types.Validator,
) {
	if err := eventBus.PublishEventNewBlock(types.EventDataNewBlock{
		Block:            block,
		ResultBeginBlock: *abciResponses.BeginBlock,
		ResultEndBlock:   *abciResponses.EndBlock,
	}); err != nil {
		logger.Error("failed publishing new block", "err", err)
	}

	if err := eventBus.PublishEventNewBlockHeader(types.EventDataNewBlockHeader{
		Header:           block.Header,
		NumTxs:           int64(len(block.Txs)),
		ResultBeginBlock: *abciResponses.BeginBlock,
		ResultEndBlock:   *abciResponses.EndBlock,
	}); err != nil {
		logger.Error("failed publishing new block header", "err", err)
	}

	if len(block.Evidence.Evidence) != 0 {
		for _, ev := range block.Evidence.Evidence {
			if err := eventBus.PublishEventNewEvidence(types.EventDataNewEvidence{
				Evidence: ev,
				Height:   block.Height,
			}); err != nil {
				logger.Error("failed publishing new evidence", "err", err)
			}
		}
	}

	for i, tx := range block.Data.Txs {
		if err := eventBus.PublishEventTx(types.EventDataTx{TxResult: abci.TxResult{
			Height: block.Height,
			Index:  uint32(i),
			Tx:     tx,
			Result: *(abciResponses.DeliverTxs[i]),
		}}); err != nil {
			logger.Error("failed publishing event TX", "err", err)
		}
	}

	if len(validatorUpdates) > 0 {
		if err := eventBus.PublishEventValidatorSetUpdates(
			types.EventDataValidatorSetUpdates{ValidatorUpdates: validatorUpdates}); err != nil {
			logger.Error("failed publishing event", "err", err)
		}
	}
}

//----------------------------------------------------------------------------------------------------
// Execute block without state. TODO: eliminate

// ExecCommitBlock executes and commits a block on the proxyApp without validating or mutating the state.
// It returns the application root hash (result of abci.Commit).
func ExecCommitBlock(
	appConnConsensus proxy.AppConnConsensus,
	block *types.Block,
	logger log.Logger,
	store Store,
	initialHeight int64,
) ([]byte, error) {
	_, err := execBlockOnProxyApp(logger, appConnConsensus, block, store, initialHeight)
	if err != nil {
		logger.Error("failed executing block on proxy app", "height", block.Height, "err", err)
		return nil, err
	}

	// Commit block, get hash back
	res, err := appConnConsensus.CommitSync()
	if err != nil {
		logger.Error("client error during proxyAppConn.CommitSync", "err", res)
		return nil, err
	}

	// ResponseCommit has no error or log, just data
	return res.Data, nil
}
