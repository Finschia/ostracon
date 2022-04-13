// This program tests that SignerClient can connect to KMS and make API calls.
// To test, address the KMS connection to port 45666 on the machine running this program and run the following:
//
// $ cd test/kms
// $ go run -tags libsodium .
//
package main

import (
	"fmt"
	"github.com/line/ostracon/crypto/ed25519"
	"github.com/line/ostracon/libs/log"
	tmnet "github.com/line/ostracon/libs/net"
	"github.com/line/ostracon/privval"
	types2 "github.com/line/ostracon/proto/ostracon/types"
	"github.com/line/ostracon/types"
	"net"
	"os"
	"time"
)

const VrfProofSize = 80
const VrfOutputSize = 64

func main() {
	logger := log.NewOCLogger(log.NewSyncWriter(os.Stdout))

	chainId := "test-chain"
	protocol, address := tmnet.ProtocolAndAddress("tcp://0.0.0.0:45666")
	ln, err := net.Listen(protocol, address)
	NoError(err)
	listener := privval.NewTCPListener(ln, ed25519.GenPrivKey())
	endpoint := privval.NewSignerListenerEndpoint(logger, listener)
	client, err := privval.NewSignerClient(endpoint, chainId)
	NoError(err)

	// Ping
	err = client.Ping()
	NoError(err)
	logger.Info("✅ Ping: call")

	// PubKey
	pubKey, err := client.GetPubKey()
	NoError(err)
	logger.Info("✅ PubKey: call")
	if len(pubKey.Bytes()) != ed25519.PubKeySize {
		logger.Error(fmt.Sprintf("❌ PubKey: public key size = %d != %d", len(pubKey.Bytes()), ed25519.PubKeySize))
	} else {
		logger.Info(fmt.Sprintf("✅ PubKey: public key size = %d", len(pubKey.Bytes())))
	}

	// SignVote
	blockId := types.BlockID{
		Hash: make([]byte, 32),
		PartSetHeader: types.PartSetHeader{
			Total: 10,
			Hash:  make([]byte, 32),
		},
	}
	vote := types.Vote{
		Type:             types2.PrevoteType,
		Height:           1,
		Round:            0,
		BlockID:          blockId,
		Timestamp:        time.Now(),
		ValidatorAddress: pubKey.Address(),
		ValidatorIndex:   0,
		Signature:        nil,
	}
	pv := vote.ToProto()
	err = client.SignVote(chainId, pv)
	NoError(err)
	logger.Info("✅ SignVote: call")
	if len(pv.Signature) != ed25519.SignatureSize {
		logger.Error(fmt.Sprintf("❌ SignVote: signature size = %d != %d", len(pv.Signature), ed25519.SignatureSize))
	} else {
		logger.Info(fmt.Sprintf("✅ SignVote: signature size = %d", len(pv.Signature)))
	}
	bytes := types.VoteSignBytes(chainId, pv)
	if !pubKey.VerifySignature(bytes, pv.Signature) {
		logger.Error(fmt.Sprintf("❌ SignVote: signature verification"))
	} else {
		logger.Info("✅ SignVote: signature verification")
	}

	// SignProposal
	proposal := types.Proposal{
		Type:      types2.ProposalType,
		Height:    1,
		Round:     0,
		POLRound:  -1,
		BlockID:   blockId,
		Timestamp: time.Now(),
		Signature: nil,
	}
	pp := proposal.ToProto()
	err = client.SignProposal(chainId, pp)
	NoError(err)
	logger.Info("✅ SignProposal: call")
	if len(pp.Signature) != ed25519.SignatureSize {
		logger.Error(fmt.Sprintf("❌ SignProposal: signature size = %d != %d", len(pp.Signature), ed25519.SignatureSize))
	} else {
		logger.Info(fmt.Sprintf("✅ SignProposal: signature size = %d", len(pp.Signature)))
	}
	bytes = types.ProposalSignBytes(chainId, pp)
	if !pubKey.VerifySignature(bytes, pp.Signature) {
		logger.Error(fmt.Sprintf("❌ SignProposal: signature verification"))
	} else {
		logger.Info("✅ SignProposal: signature verification")
	}

	// VRFProof
	message := []byte("hello, world")
	proof, err := client.GenerateVRFProof(message)
	NoError(err)
	logger.Info("✅ VRFProof: call")
	if len(proof) != VrfProofSize {
		logger.Error(fmt.Sprintf("❌ VRFProof: proof size = %d != %d", len(proof), VrfProofSize))
	} else {
		logger.Info(fmt.Sprintf("✅ VRFProof: proof size = %d", len(proof)))
	}
	output, err := pubKey.VRFVerify(proof, message)
	NoError(err)
	logger.Info("✅ VRFProof: proof verification")
	if len(output) != VrfOutputSize {
		logger.Error(fmt.Sprintf("❌ VRFProof: output size = %d != %d", len(output), VrfOutputSize))
	} else {
		logger.Info(fmt.Sprintf("✅ VRFProof: output size = %d", len(output)))
	}

	logger.Info("All tests finished")
}

func NoError(err error) {
	if err != nil {
		panic(fmt.Sprintf("❌ %v", err))
	}
}
