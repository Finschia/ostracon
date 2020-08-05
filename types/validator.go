package types

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/tendermint/tendermint/crypto"
	ce "github.com/tendermint/tendermint/crypto/encoding"
	tmrand "github.com/tendermint/tendermint/libs/rand"
	tmproto "github.com/tendermint/tendermint/proto/types"
)

// Volatile state for each Validator
// NOTE: The ProposerPriority, VotingPower is not included in Validator.Hash();
// make sure to update that method if changes are made here
// StakingPower is the potential voting power proportional to the amount of stake,
// and VotingPower is the actual voting power granted by the election process.
// StakingPower is durable and can be changed by staking txs.
// VotingPower is volatile and can be changed at every height.
type Validator struct {
	Address      Address       `json:"address"`
	PubKey       crypto.PubKey `json:"pub_key"`
	StakingPower int64         `json:"staking_power"`

	VotingPower      int64 `json:"voting_power"`
	ProposerPriority int64 `json:"proposer_priority"`
}

func NewValidator(pubKey crypto.PubKey, stakingPower int64) *Validator {
	return &Validator{
		Address:          pubKey.Address(),
		PubKey:           pubKey,
		StakingPower:     stakingPower,
		VotingPower:      0,
		ProposerPriority: 0,
	}
}

// Creates a new copy of the validator so we can mutate ProposerPriority.
// Panics if the validator is nil.
func (v *Validator) Copy() *Validator {
	vCopy := *v
	return &vCopy
}

// Returns the one with higher ProposerPriority.
func (v *Validator) CompareProposerPriority(other *Validator) *Validator {
	if v == nil {
		return other
	}
	switch {
	case v.ProposerPriority > other.ProposerPriority:
		return v
	case v.ProposerPriority < other.ProposerPriority:
		return other
	default:
		result := bytes.Compare(v.Address, other.Address)
		switch {
		case result < 0:
			return v
		case result > 0:
			return other
		default:
			panic("Cannot compare identical validators")
		}
	}
}

func (v *Validator) String() string {
	if v == nil {
		return "nil-Validator"
	}
	return fmt.Sprintf("Validator{%v %v VP:%v A:%v}",
		v.Address,
		v.PubKey,
		v.StakingPower,
		v.ProposerPriority)
}

// ValidatorListString returns a prettified validator list for logging purposes.
func ValidatorListString(vals []*Validator) string {
	chunks := make([]string, len(vals))
	for i, val := range vals {
		chunks[i] = fmt.Sprintf("%s:%d", val.Address, val.StakingPower)
	}

	return strings.Join(chunks, ",")
}

// Bytes computes the unique encoding of a validator with a given voting power.
// These are the bytes that gets hashed in consensus. It excludes address
// as its redundant with the pubkey. This also excludes ProposerPriority
// which changes every round.
func (v *Validator) Bytes() []byte {
	return cdcEncode(struct {
		PubKey       crypto.PubKey
		StakingPower int64
	}{
		v.PubKey,
		v.StakingPower,
	})
}

// ToProto converts Valiator to protobuf
func (v *Validator) ToProto() (*tmproto.Validator, error) {
	if v == nil {
		return nil, errors.New("nil validator")
	}

	pk, err := ce.PubKeyToProto(v.PubKey)
	if err != nil {
		return nil, err
	}

	vp := tmproto.Validator{
		Address:          v.Address,
		PubKey:           pk,
		StakingPower:     v.StakingPower,
		VotingPower:      v.VotingPower,
		ProposerPriority: v.ProposerPriority,
	}

	return &vp, nil
}

// FromProto sets a protobuf Validator to the given pointer.
// It returns an error if the public key is invalid.
func ValidatorFromProto(vp *tmproto.Validator) (*Validator, error) {
	if vp == nil {
		return nil, errors.New("nil validator")
	}

	pk, err := ce.PubKeyFromProto(&vp.PubKey)
	if err != nil {
		return nil, err
	}
	v := new(Validator)
	v.Address = vp.GetAddress()
	v.PubKey = pk
	v.StakingPower = vp.GetStakingPower()
	v.VotingPower = vp.GetVotingPower()
	v.ProposerPriority = vp.GetProposerPriority()

	return v, nil
}

//----------------------------------------
// RandValidator

// RandValidator returns a randomized validator, useful for testing.
// UNSTABLE
func RandValidator(randPower bool, minPower int64) (*Validator, PrivValidator) {
	privVal := NewMockPV()
	stakingPower := minPower
	if randPower {
		stakingPower += int64(tmrand.Uint32())
	}
	pubKey, err := privVal.GetPubKey()
	if err != nil {
		panic(fmt.Errorf("could not retrieve pubkey %w", err))
	}
	val := NewValidator(pubKey, stakingPower)
	return val, privVal
}
