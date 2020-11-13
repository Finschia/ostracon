package types

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/tendermint/tendermint/libs/rand"

	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/bls"
	"github.com/tendermint/tendermint/crypto/composite"
	"github.com/tendermint/tendermint/crypto/ed25519"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
)

// PrivValidator defines the functionality of a local Tendermint validator
// that signs votes and proposals, and never double signs.
type PrivValidator interface {
	GetPubKey() (crypto.PubKey, error)

	SignVote(chainID string, vote *tmproto.Vote) error
	SignProposal(chainID string, proposal *tmproto.Proposal) error

	GenerateVRFProof(message []byte) (crypto.Proof, error)
}

type PrivValidatorsByAddress []PrivValidator

func (pvs PrivValidatorsByAddress) Len() int {
	return len(pvs)
}

func (pvs PrivValidatorsByAddress) Less(i, j int) bool {
	pvi, err := pvs[i].GetPubKey()
	if err != nil {
		panic(err)
	}
	pvj, err := pvs[j].GetPubKey()
	if err != nil {
		panic(err)
	}

	return bytes.Compare(pvi.Address(), pvj.Address()) == -1
}

func (pvs PrivValidatorsByAddress) Swap(i, j int) {
	pvs[i], pvs[j] = pvs[j], pvs[i]
}

//----------------------------------------
// MockPV

// MockPV implements PrivValidator without any safety or persistence.
// Only use it for testing.
type MockPV struct {
	PrivKey              crypto.PrivKey
	breakProposalSigning bool
	breakVoteSigning     bool
}

type PrivKeyType int

const (
	PrivKeyEd25519 PrivKeyType = iota
	PrivKeyComposite
	PrivKeyBLS
)

func NewMockPV(keyType PrivKeyType) MockPV {
	switch keyType {
	case PrivKeyEd25519:
		return MockPV{ed25519.GenPrivKey(), false, false}
	case PrivKeyComposite:
		return MockPV{composite.GenPrivKey(), false, false}
	case PrivKeyBLS:
		return MockPV{bls.GenPrivKey(), false, false}
	default:
		panic(fmt.Sprintf("known pv key type: %d", keyType))
	}
}

func PrivKeyTypeByPubKey(pubKey crypto.PubKey) PrivKeyType {
	switch pubKey.(type) {
	case ed25519.PubKey:
		return PrivKeyEd25519
	case composite.PubKeyComposite:
		return PrivKeyComposite
	case bls.PubKeyBLS12:
		return PrivKeyBLS
	}
	panic(fmt.Sprintf("unknown public key type: %v", pubKey))
}

// NewMockPVWithParams allows one to create a MockPV instance, but with finer
// grained control over the operation of the mock validator. This is useful for
// mocking test failures.
func NewMockPVWithParams(privKey crypto.PrivKey, breakProposalSigning, breakVoteSigning bool) MockPV {
	return MockPV{privKey, breakProposalSigning, breakVoteSigning}
}

// Implements PrivValidator.
func (pv MockPV) GetPubKey() (crypto.PubKey, error) {
	return pv.PrivKey.PubKey(), nil
}

// Implements PrivValidator.
func (pv MockPV) SignVote(chainID string, vote *tmproto.Vote) error {
	useChainID := chainID
	if pv.breakVoteSigning {
		useChainID = "incorrect-chain-id"
	}

	signBytes := VoteSignBytes(useChainID, vote)
	sig, err := pv.PrivKey.Sign(signBytes)
	if err != nil {
		return err
	}
	vote.Signature = sig
	return nil
}

// Implements PrivValidator.
func (pv MockPV) SignProposal(chainID string, proposal *tmproto.Proposal) error {
	useChainID := chainID
	if pv.breakProposalSigning {
		useChainID = "incorrect-chain-id"
	}

	signBytes := ProposalSignBytes(useChainID, proposal)
	sig, err := pv.PrivKey.Sign(signBytes)
	if err != nil {
		return err
	}
	proposal.Signature = sig
	return nil
}

func (pv MockPV) ExtractIntoValidator(stakingPower int64) *Validator {
	pubKey, _ := pv.GetPubKey()
	return &Validator{
		Address:      pubKey.Address(),
		PubKey:       pubKey,
		StakingPower: stakingPower,
	}
}

// GenerateVRFProof implements PrivValidator.
func (pv MockPV) GenerateVRFProof(message []byte) (crypto.Proof, error) {
	return pv.PrivKey.VRFProve(message)
}

// String returns a string representation of the MockPV.
func (pv MockPV) String() string {
	mpv, _ := pv.GetPubKey() // mockPV will never return an error, ignored here
	return fmt.Sprintf("MockPV{%v}", mpv.Address())
}

// XXX: Implement.
func (pv MockPV) DisableChecks() {
	// Currently this does nothing,
	// as MockPV has no safety checks at all.
}

type ErroringMockPV struct {
	MockPV
}

var ErroringMockPVErr = errors.New("erroringMockPV always returns an error")

// Implements PrivValidator.
func (pv *ErroringMockPV) SignVote(chainID string, vote *tmproto.Vote) error {
	return ErroringMockPVErr
}

// Implements PrivValidator.
func (pv *ErroringMockPV) SignProposal(chainID string, proposal *tmproto.Proposal) error {
	return ErroringMockPVErr
}

// NewErroringMockPV returns a MockPV that fails on each signing request. Again, for testing only.

func NewErroringMockPV() *ErroringMockPV {
	return &ErroringMockPV{MockPV{ed25519.GenPrivKey(), false, false}}
}

////////////////////////////////////////
// For testing
func RandomKeyType() PrivKeyType {
	r := rand.Uint32() % 2
	switch r {
	case 0:
		return PrivKeyEd25519
	case 1:
		return PrivKeyComposite
	}
	return PrivKeyEd25519
}
