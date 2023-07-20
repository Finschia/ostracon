package node

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dbm "github.com/tendermint/tm-db"

	"github.com/Finschia/ostracon/abci/example/kvstore"
	cfg "github.com/Finschia/ostracon/config"
	"github.com/Finschia/ostracon/crypto/ed25519"
	"github.com/Finschia/ostracon/evidence"
	"github.com/Finschia/ostracon/libs/log"
	tmrand "github.com/Finschia/ostracon/libs/rand"
	mempl "github.com/Finschia/ostracon/mempool"
	mempoolv0 "github.com/Finschia/ostracon/mempool/v0"

	//mempoolv1 "github.com/Finschia/ostracon/mempool/v1"
	"github.com/Finschia/ostracon/p2p"
	"github.com/Finschia/ostracon/p2p/conn"
	p2pmock "github.com/Finschia/ostracon/p2p/mock"
	p2pmocks "github.com/Finschia/ostracon/p2p/mocks"
	"github.com/Finschia/ostracon/privval"
	"github.com/Finschia/ostracon/proxy"
	sm "github.com/Finschia/ostracon/state"
	"github.com/Finschia/ostracon/store"
	"github.com/Finschia/ostracon/types"
	tmtime "github.com/Finschia/ostracon/types/time"
)

func TestNewOstraconNode(t *testing.T) {
	config := cfg.ResetTestRootWithChainID("TestNewOstraconNode", "new_ostracon_node")
	defer os.RemoveAll(config.RootDir)
	require.Equal(t, "", config.PrivValidatorListenAddr)
	node, err := NewOstraconNode(config, log.TestingLogger())
	require.NoError(t, err)
	pubKey, err := node.PrivValidator().GetPubKey()
	require.NoError(t, err)
	require.NotNil(t, pubKey)
}

func TestNewOstraconNode_WithoutNodeKey(t *testing.T) {
	config := cfg.ResetTestRootWithChainID("TestNewOstraconNode", "new_ostracon_node_wo_node_key")
	defer os.RemoveAll(config.RootDir)
	_ = os.Remove(config.NodeKeyFile())
	_, err := NewOstraconNode(config, log.TestingLogger())
	require.Error(t, err)
}

func TestNodeStartStop(t *testing.T) {
	config := cfg.ResetTestRoot("node_node_test")
	defer os.RemoveAll(config.RootDir)

	// create & start node
	n, err := DefaultNewNode(config, log.TestingLogger())
	require.NoError(t, err)
	err = n.Start()
	require.NoError(t, err)

	t.Logf("Started node %v", n.sw.NodeInfo())

	// wait for the node to produce a block
	blocksSub, err := n.EventBus().Subscribe(context.Background(), "node_test", types.EventQueryNewBlock)
	require.NoError(t, err)
	select {
	case <-blocksSub.Out():
	case <-blocksSub.Cancelled():
		t.Fatal("blocksSub was cancelled")
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for the node to produce a block")
	}

	// stop the node
	go func() {
		err = n.Stop()
		require.NoError(t, err)
	}()

	select {
	case <-n.Quit():
	case <-time.After(5 * time.Second):
		pid := os.Getpid()
		p, err := os.FindProcess(pid)
		if err != nil {
			panic(err)
		}
		err = p.Signal(syscall.SIGABRT)
		fmt.Println(err)
		t.Fatal("timed out waiting for shutdown")
	}
}

func TestSplitAndTrimEmpty(t *testing.T) {
	testCases := []struct {
		s        string
		sep      string
		cutset   string
		expected []string
	}{
		{"a,b,c", ",", " ", []string{"a", "b", "c"}},
		{" a , b , c ", ",", " ", []string{"a", "b", "c"}},
		{" a, b, c ", ",", " ", []string{"a", "b", "c"}},
		{" a, ", ",", " ", []string{"a"}},
		{"   ", ",", " ", []string{}},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.expected, splitAndTrimEmpty(tc.s, tc.sep, tc.cutset), "%s", tc.s)
	}
}

func TestNodeDelayedStart(t *testing.T) {
	config := cfg.ResetTestRoot("node_delayed_start_test")
	defer os.RemoveAll(config.RootDir)
	now := tmtime.Now()

	// create & start node
	n, err := DefaultNewNode(config, log.TestingLogger())
	n.GenesisDoc().GenesisTime = now.Add(2 * time.Second)
	require.NoError(t, err)

	err = n.Start()
	require.NoError(t, err)
	defer n.Stop() //nolint:errcheck // ignore for tests

	startTime := tmtime.Now()
	assert.Equal(t, true, startTime.After(n.GenesisDoc().GenesisTime))
}

func TestNodeSetAppVersion(t *testing.T) {
	config := cfg.ResetTestRoot("node_app_version_test")
	defer os.RemoveAll(config.RootDir)

	// create & start node
	n, err := DefaultNewNode(config, log.TestingLogger())
	require.NoError(t, err)

	// default config uses the kvstore app
	var appVersion = kvstore.ProtocolVersion

	// check version is set in state
	state, err := n.stateStore.Load()
	require.NoError(t, err)
	assert.Equal(t, state.Version.Consensus.App, appVersion)

	// check version is set in node info
	assert.Equal(t, n.nodeInfo.(p2p.DefaultNodeInfo).ProtocolVersion.App, appVersion)
}

func TestNodeSetPrivValTCP(t *testing.T) {
	addr := "tcp://" + testFreeAddr(t)

	config := cfg.ResetTestRoot("node_priv_val_tcp_test")
	defer os.RemoveAll(config.RootDir)
	config.BaseConfig.PrivValidatorListenAddr = addr

	dialer := privval.DialTCPFn(addr, 100*time.Millisecond, ed25519.GenPrivKey())
	dialerEndpoint := privval.NewSignerDialerEndpoint(
		log.TestingLogger(),
		dialer,
	)
	privval.SignerDialerEndpointTimeoutReadWrite(100 * time.Millisecond)(dialerEndpoint)

	signerServer := privval.NewSignerServer(
		dialerEndpoint,
		config.ChainID(),
		types.NewMockPV(),
	)

	go func() {
		err := signerServer.Start()
		if err != nil {
			panic(err)
		}
	}()
	defer signerServer.Stop() //nolint:errcheck // ignore for tests

	n, err := DefaultNewNode(config, log.TestingLogger())
	require.NoError(t, err)
	assert.IsType(t, &privval.RetrySignerClient{}, n.PrivValidator())
}

// address without a protocol must result in error
func TestPrivValidatorListenAddrNoProtocol(t *testing.T) {
	addrNoPrefix := testFreeAddr(t)

	config := cfg.ResetTestRoot("node_priv_val_tcp_test")
	defer os.RemoveAll(config.RootDir)
	config.BaseConfig.PrivValidatorListenAddr = addrNoPrefix

	_, err := DefaultNewNode(config, log.TestingLogger())
	assert.Error(t, err)
}

func TestNodeSetPrivValIPC(t *testing.T) {
	tmpfile := "/tmp/kms." + tmrand.Str(6) + ".sock"
	defer os.Remove(tmpfile) // clean up

	config := cfg.ResetTestRoot("node_priv_val_tcp_test")
	defer os.RemoveAll(config.RootDir)
	config.BaseConfig.PrivValidatorListenAddr = "unix://" + tmpfile

	dialer := privval.DialUnixFn(tmpfile)
	dialerEndpoint := privval.NewSignerDialerEndpoint(
		log.TestingLogger(),
		dialer,
	)
	privval.SignerDialerEndpointTimeoutReadWrite(100 * time.Millisecond)(dialerEndpoint)

	pvsc := privval.NewSignerServer(
		dialerEndpoint,
		config.ChainID(),
		types.NewMockPV(),
	)

	go func() {
		err := pvsc.Start()
		require.NoError(t, err)
	}()
	defer pvsc.Stop() //nolint:errcheck // ignore for tests

	n, err := DefaultNewNode(config, log.TestingLogger())
	require.NoError(t, err)
	assert.IsType(t, &privval.RetrySignerClient{}, n.PrivValidator())
}

// testFreeAddr claims a free port so we don't block on listener being ready.
func testFreeAddr(t *testing.T) string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	return fmt.Sprintf("127.0.0.1:%d", ln.Addr().(*net.TCPAddr).Port)
}

// create a proposal block using real and full
// mempool and evidence pool and validate it.
func TestCreateProposalBlock(t *testing.T) {
	config := cfg.ResetTestRoot("node_create_proposal")
	defer os.RemoveAll(config.RootDir)
	cc := proxy.NewLocalClientCreator(kvstore.NewApplication())
	proxyApp := proxy.NewAppConns(cc)
	err := proxyApp.Start()
	require.Nil(t, err)
	defer proxyApp.Stop() //nolint:errcheck // ignore for tests

	logger := log.TestingLogger()

	var height int64 = 1
	state, stateDB, privVals := state(1, height)
	stateStore := sm.NewStore(stateDB, sm.StoreOptions{
		DiscardABCIResponses: false,
	})
	maxBytes := 16384
	var partSize uint32 = 256
	maxEvidenceBytes := int64(maxBytes / 2)
	state.ConsensusParams.Block.MaxBytes = int64(maxBytes)
	state.ConsensusParams.Evidence.MaxBytes = maxEvidenceBytes
	proposerAddr, _ := state.Validators.GetByIndex(0)

	// Make Mempool
	memplMetrics := mempl.NopMetrics()
	var mempool mempl.Mempool

	switch config.Mempool.Version {
	case cfg.MempoolV0:
		mempool = mempoolv0.NewCListMempool(config.Mempool,
			proxyApp.Mempool(),
			state.LastBlockHeight,
			mempoolv0.WithMetrics(memplMetrics),
			mempoolv0.WithPreCheck(sm.TxPreCheck(state)),
			mempoolv0.WithPostCheck(sm.TxPostCheck(state)))
	case cfg.MempoolV1: // XXX Deprecated MempoolV1
		panic("Deprecated MempoolV1")
		/*
			mempool = mempoolv1.NewTxMempool(logger,
				config.Mempool,
				proxyApp.Mempool(),
				state.LastBlockHeight,
				mempoolv1.WithMetrics(memplMetrics),
				mempoolv1.WithPreCheck(sm.TxPreCheck(state)),
				mempoolv1.WithPostCheck(sm.TxPostCheck(state)),
			)
		*/
	}

	// Make EvidencePool
	evidenceDB := dbm.NewMemDB()
	blockStore := store.NewBlockStore(dbm.NewMemDB())
	evidencePool, err := evidence.NewPool(evidenceDB, stateStore, blockStore)
	require.NoError(t, err)
	evidencePool.SetLogger(logger)

	// fill the evidence pool with more evidence
	// than can fit in a block
	var currentBytes int64
	for currentBytes <= maxEvidenceBytes {
		ev := types.NewMockDuplicateVoteEvidenceWithValidator(height, time.Now(), privVals[0], "test-chain")
		currentBytes += int64(len(ev.Bytes()))
		evidencePool.ReportConflictingVotes(ev.VoteA, ev.VoteB)
	}

	evList, size := evidencePool.PendingEvidence(state.ConsensusParams.Evidence.MaxBytes)
	require.Less(t, size, state.ConsensusParams.Evidence.MaxBytes+1)
	evData := &types.EvidenceData{Evidence: evList}
	require.EqualValues(t, size, evData.ByteSize())

	// fill the mempool with more txs
	// than can fit in a block
	txLength := 100
	for i := 0; i <= maxBytes/txLength; i++ {
		tx := tmrand.Bytes(txLength)
		err := mempool.CheckTxSync(tx, nil, mempl.TxInfo{})
		assert.NoError(t, err)
	}

	blockExec := sm.NewBlockExecutor(
		stateStore,
		logger,
		proxyApp.Consensus(),
		mempool,
		evidencePool,
	)

	commit := types.NewCommit(height-1, 0, types.BlockID{}, nil)
	message := state.MakeHashMessage(0)
	proof, _ := privVals[0].GenerateVRFProof(message)
	block, _ := blockExec.CreateProposalBlock(
		height,
		state, commit,
		proposerAddr,
		0,
		proof,
		0,
	)

	// check that the part set does not exceed the maximum block size
	partSet := block.MakePartSet(partSize)
	assert.Less(t, partSet.ByteSize(), int64(maxBytes))

	partSetFromHeader := types.NewPartSetFromHeader(partSet.Header())
	for partSetFromHeader.Count() < partSetFromHeader.Total() {
		added, err := partSetFromHeader.AddPart(partSet.GetPart(int(partSetFromHeader.Count())))
		require.NoError(t, err)
		require.True(t, added)
	}
	assert.EqualValues(t, partSetFromHeader.ByteSize(), partSet.ByteSize())

	err = blockExec.ValidateBlock(state, 0, block)
	assert.NoError(t, err)
}

func TestMaxProposalBlockSize(t *testing.T) {
	config := cfg.ResetTestRoot("node_create_proposal")
	defer os.RemoveAll(config.RootDir)
	cc := proxy.NewLocalClientCreator(kvstore.NewApplication())
	proxyApp := proxy.NewAppConns(cc)
	err := proxyApp.Start()
	require.Nil(t, err)
	defer proxyApp.Stop() //nolint:errcheck // ignore for tests

	logger := log.TestingLogger()

	var height int64 = 1
	state, stateDB, privVals := state(1, height)
	stateStore := sm.NewStore(stateDB, sm.StoreOptions{
		DiscardABCIResponses: false,
	})
	var maxBytes int64 = 16384
	var partSize uint32 = 256
	state.ConsensusParams.Block.MaxBytes = maxBytes
	proposerAddr, _ := state.Validators.GetByIndex(0)

	// Make Mempool
	memplMetrics := mempl.NopMetrics()
	var mempool mempl.Mempool
	switch config.Mempool.Version {
	case cfg.MempoolV0:
		mempool = mempoolv0.NewCListMempool(config.Mempool,
			proxyApp.Mempool(),
			state.LastBlockHeight,
			mempoolv0.WithMetrics(memplMetrics),
			mempoolv0.WithPreCheck(sm.TxPreCheck(state)),
			mempoolv0.WithPostCheck(sm.TxPostCheck(state)))
	case cfg.MempoolV1: // XXX Deprecated
		/*
			mempool = mempoolv1.NewTxMempool(logger,
				config.Mempool,
				proxyApp.Mempool(),
				state.LastBlockHeight,
				mempoolv1.WithMetrics(memplMetrics),
				mempoolv1.WithPreCheck(sm.TxPreCheck(state)),
				mempoolv1.WithPostCheck(sm.TxPostCheck(state)),
			)
		*/
	}

	// fill the mempool with one txs just below the maximum size
	txLength := int(types.MaxDataBytesNoEvidence(maxBytes, 1))
	tx := tmrand.Bytes(txLength - 4) // to account for the varint
	err = mempool.CheckTxSync(tx, nil, mempl.TxInfo{})
	assert.NoError(t, err)

	blockExec := sm.NewBlockExecutor(
		stateStore,
		logger,
		proxyApp.Consensus(),
		mempool,
		sm.EmptyEvidencePool{},
	)

	commit := types.NewCommit(height-1, 0, types.BlockID{}, nil)
	message := state.MakeHashMessage(0)
	proof, _ := privVals[0].GenerateVRFProof(message)
	block, _ := blockExec.CreateProposalBlock(
		height,
		state, commit,
		proposerAddr,
		0,
		proof,
		0,
	)

	pb, err := block.ToProto()
	require.NoError(t, err)
	assert.Less(t, int64(pb.Size()), maxBytes)

	// check that the part set does not exceed the maximum block size
	partSet := block.MakePartSet(partSize)
	assert.EqualValues(t, partSet.ByteSize(), int64(pb.Size()))
}

func TestNodeNewNodeCustomReactors(t *testing.T) {
	config := cfg.ResetTestRoot("node_new_node_custom_reactors_test")
	defer os.RemoveAll(config.RootDir)

	cr := p2pmock.NewReactor()
	cr.Channels = []*conn.ChannelDescriptor{
		{
			ID:                  byte(0x31),
			Priority:            5,
			SendQueueCapacity:   100,
			RecvMessageCapacity: 100,
		},
	}
	customBlockchainReactor := p2pmock.NewReactor()

	nodeKey, err := p2p.LoadOrGenNodeKey(config.NodeKeyFile())
	require.NoError(t, err)

	n, err := NewNode(config,
		privval.LoadOrGenFilePV(config.PrivValidatorKeyFile(), config.PrivValidatorStateFile()),
		nodeKey,
		proxy.DefaultClientCreator(config.ProxyApp, config.ABCI, config.DBDir()),
		DefaultGenesisDocProviderFunc(config),
		DefaultDBProvider,
		DefaultMetricsProvider(config.Instrumentation),
		log.TestingLogger(),
		CustomReactors(map[string]p2p.Reactor{"FOO": cr, "BLOCKCHAIN": customBlockchainReactor}),
	)
	require.NoError(t, err)

	err = n.Start()
	require.NoError(t, err)
	defer n.Stop() //nolint:errcheck // ignore for tests

	assert.True(t, cr.IsRunning())
	assert.Equal(t, cr, n.Switch().Reactor("FOO"))

	assert.True(t, customBlockchainReactor.IsRunning())
	assert.Equal(t, customBlockchainReactor, n.Switch().Reactor("BLOCKCHAIN"))

	channels := n.NodeInfo().(p2p.DefaultNodeInfo).Channels
	assert.Contains(t, channels, mempl.MempoolChannel)
	assert.Contains(t, channels, cr.Channels[0].ID)
}

func TestNodeNewNodeTxIndexIndexer(t *testing.T) {
	config := cfg.ResetTestRoot("node_new_node_tx_index_indexer_test")
	defer os.RemoveAll(config.RootDir)

	doTest := func(doProvider func(ctx *DBContext) (dbm.DB, error)) (*Node, error) {
		nodeKey, err := p2p.LoadOrGenNodeKey(config.NodeKeyFile())
		require.NoError(t, err)

		return NewNode(config,
			privval.LoadOrGenFilePV(config.PrivValidatorKeyFile(), config.PrivValidatorStateFile()),
			nodeKey,
			proxy.DefaultClientCreator(config.ProxyApp, config.ABCI, config.DBDir()),
			DefaultGenesisDocProviderFunc(config),
			doProvider,
			DefaultMetricsProvider(config.Instrumentation),
			log.TestingLogger(),
		)
	}

	{
		// Change to panic-provider for test
		n, err := doTest(func(ctx *DBContext) (dbm.DB, error) { return nil, fmt.Errorf("test error") })
		require.Error(t, err)
		require.Nil(t, n)
	}
	{
		// Change to non-default-value for test
		config.TxIndex.Indexer = ""
		n, err := doTest(DefaultDBProvider)
		require.NoError(t, err)
		require.NotNil(t, n)
	}
	{
		// Change to psql for test
		config.TxIndex.Indexer = "psql"
		n, err := doTest(DefaultDBProvider)
		require.Error(t, err)
		require.Equal(t, "no psql-conn is set for the \"psql\" indexer", err.Error())
		require.Nil(t, n)

		// config.TxIndex.PsqlConn = "cannot test with no-import postgres driver"
		// n, err = doTest(DefaultDBProvider)
		// require.Error(t, err)
		// require.Equal(t, "creating psql indexer: sql: unknown driver \"postgres\" (forgotten import?)", err.Error())
		// require.Nil(t, n)

		config.TxIndex.PsqlConn = makeTestPsqlConn(t)
		n, err = doTest(DefaultDBProvider)
		require.NoError(t, err)
		require.NotNil(t, n)
	}
}

func makeTestPsqlConn(t *testing.T) string {
	user := "postgres"
	password := "secret"
	port := "5432"
	dsn := "postgres://%s:%s@localhost:%s/%s?sslmode=disable"
	dbName := "postgres"

	pool, err := dockertest.NewPool(os.Getenv("DOCKER_URL"))
	if err != nil {
		require.NoError(t, err)
	}
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "13",
		Env: []string{
			"POSTGRES_USER=" + user,
			"POSTGRES_PASSWORD=" + password,
			"POSTGRES_DB=" + dbName,
			"listen_addresses = '*'",
		},
		ExposedPorts: []string{port},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	})
	if err != nil {
		require.NoError(t, err)
	}
	return fmt.Sprintf(dsn, user, password, resource.GetPort(port+"/tcp"), dbName)
}

func state(nVals int, height int64) (sm.State, dbm.DB, []types.PrivValidator) {
	privVals := make([]types.PrivValidator, nVals)
	vals := make([]types.GenesisValidator, nVals)
	for i := 0; i < nVals; i++ {
		secret := []byte(fmt.Sprintf("test%d", i))
		pk := ed25519.GenPrivKeyFromSecret(secret)
		privVal := types.NewMockPVWithParams(pk, false, false)
		privVals[i] = privVal
		vals[i] = types.GenesisValidator{
			Address: privVal.PrivKey.PubKey().Address(),
			PubKey:  privVal.PrivKey.PubKey(),
			Power:   1000,
			Name:    fmt.Sprintf("test%d", i),
		}
	}
	s, _ := sm.MakeGenesisState(&types.GenesisDoc{
		ChainID:    "test-chain",
		Validators: vals,
		AppHash:    nil,
	})

	// save validators to db for 2 heights
	stateDB := dbm.NewMemDB()
	stateStore := sm.NewStore(stateDB, sm.StoreOptions{
		DiscardABCIResponses: false,
	})
	if err := stateStore.Save(s); err != nil {
		panic(err)
	}

	for i := 1; i < int(height); i++ {
		s.LastBlockHeight++
		s.LastValidators = s.Validators.Copy()
		if err := stateStore.Save(s); err != nil {
			panic(err)
		}
	}
	return s, stateDB, privVals
}

func TestNodeInvalidNodeInfoCustomReactors(t *testing.T) {
	config := cfg.ResetTestRoot("node_new_node_custom_reactors_test")
	defer os.RemoveAll(config.RootDir)

	cr := p2pmock.NewReactor()
	cr.Channels = []*conn.ChannelDescriptor{
		{
			ID:                  byte(0x31),
			Priority:            5,
			SendQueueCapacity:   100,
			RecvMessageCapacity: 100,
		},
	}
	customBlockchainReactor := p2pmock.NewReactor()

	nodeKey, err := p2p.LoadOrGenNodeKey(config.NodeKeyFile())
	require.NoError(t, err)

	_, err = NewInvalidNode(config,
		privval.LoadOrGenFilePV(config.PrivValidatorKeyFile(), config.PrivValidatorStateFile()),
		nodeKey,
		proxy.DefaultClientCreator(config.ProxyApp, config.ABCI, config.DBDir()),
		DefaultGenesisDocProviderFunc(config),
		DefaultDBProvider,
		DefaultMetricsProvider(config.Instrumentation),
		log.TestingLogger(),
		CustomReactors(map[string]p2p.Reactor{"FOO": cr, "BLOCKCHAIN": customBlockchainReactor}),
	)
	require.NoError(t, err)
}

func TestSaveAndLoadBigGensisFile(t *testing.T) {
	stateDB, err := dbm.NewGoLevelDB("state", os.TempDir())
	require.NoError(t, err)
	config := cfg.ResetTestRoot("node_big_genesis_test")
	defer os.RemoveAll(config.RootDir)
	n, err := DefaultNewNode(config, log.TestingLogger())
	require.NoError(t, err)
	newChainID := strings.Repeat("a", 200000000) // about 200MB
	n.genesisDoc.ChainID = newChainID
	err = saveGenesisDoc(stateDB, n.genesisDoc)
	require.NoError(t, err)
	g, err := loadGenesisDoc(stateDB)
	require.NoError(t, err)
	require.Equal(t, newChainID, g.ChainID)
	stateDB.Close()
}

func NewInvalidNode(config *cfg.Config,
	privValidator types.PrivValidator,
	nodeKey *p2p.NodeKey,
	clientCreator proxy.ClientCreator,
	genesisDocProvider GenesisDocProvider,
	dbProvider DBProvider,
	metricsProvider MetricsProvider,
	logger log.Logger,
	options ...Option) (*Node, error) {
	n, err := NewNode(config,
		privValidator,
		nodeKey,
		clientCreator,
		genesisDocProvider,
		dbProvider,
		metricsProvider,
		logger,
	)
	if err != nil {
		return nil, err
	}

	transport, _ := createTransport(config, &p2pmocks.NodeInfo{}, nodeKey, n.proxyApp)
	n.transport = transport

	for _, option := range options {
		option(n)
	}

	return n, nil
}
