package types

import (
	abci "github.com/line/ostracon/abci/types"
	"github.com/line/ostracon/crypto"
	"github.com/line/ostracon/crypto/bls"
	"github.com/line/ostracon/crypto/composite"
	"github.com/line/ostracon/crypto/ed25519"
	cryptoenc "github.com/line/ostracon/crypto/encoding"
	"github.com/line/ostracon/crypto/secp256k1"
	tmproto "github.com/line/ostracon/proto/ostracon/types"
)

//-------------------------------------------------------
// Use strings to distinguish types in ABCI messages

const (
	ABCIPubKeyTypeBls12WithEd25519 = composite.KeyTypeBlsWithEd25519
	ABCIPubKeyTypeEd25519          = ed25519.KeyType
	ABCIPubKeyTypeSecp256k1        = secp256k1.KeyType
	ABCIPubKeyTypeBls12            = bls.KeyType
)

// TODO: Make non-global by allowing for registration of more pubkey types

var ABCIPubKeyTypesToNames = map[string]string{
	ABCIPubKeyTypeBls12WithEd25519: composite.PubKeyName,
	ABCIPubKeyTypeEd25519:          ed25519.PubKeyName,
	ABCIPubKeyTypeSecp256k1:        secp256k1.PubKeyName,
	ABCIPubKeyTypeBls12:            bls.PubKeyName,
}

//-------------------------------------------------------

// OC2PB is used for converting Ostracon ABCI to protobuf ABCI.
// UNSTABLE
var OC2PB = oc2pb{}

type oc2pb struct{}

func (oc2pb) Header(header *Header) tmproto.Header {
	return tmproto.Header{
		Version: header.Version,
		ChainID: header.ChainID,
		Height:  header.Height,
		Time:    header.Time,

		LastBlockId: header.LastBlockID.ToProto(),

		LastCommitHash: header.LastCommitHash,
		DataHash:       header.DataHash,

		VotersHash:         header.VotersHash,
		NextValidatorsHash: header.NextValidatorsHash,
		ConsensusHash:      header.ConsensusHash,
		AppHash:            header.AppHash,
		LastResultsHash:    header.LastResultsHash,

		EvidenceHash:    header.EvidenceHash,
		ProposerAddress: header.ProposerAddress,
	}
}

func (oc2pb) Validator(val *Validator) abci.Validator {
	return abci.Validator{
		Address:     val.PubKey.Address(),
		Power:       val.StakingPower,
		VotingPower: val.VotingPower,
	}
}

func (oc2pb) BlockID(blockID BlockID) tmproto.BlockID {
	return tmproto.BlockID{
		Hash:          blockID.Hash,
		PartSetHeader: OC2PB.PartSetHeader(blockID.PartSetHeader),
	}
}

func (oc2pb) PartSetHeader(header PartSetHeader) tmproto.PartSetHeader {
	return tmproto.PartSetHeader{
		Total: header.Total,
		Hash:  header.Hash,
	}
}

// XXX: panics on unknown pubkey type
func (oc2pb) ValidatorUpdate(val *Validator) abci.ValidatorUpdate {
	pk, err := cryptoenc.PubKeyToProto(val.PubKey)
	if err != nil {
		panic(err)
	}
	return abci.ValidatorUpdate{
		PubKey: pk,
		Power:  val.StakingPower,
	}
}

// XXX: panics on nil or unknown pubkey type
func (oc2pb) ValidatorUpdates(vals *ValidatorSet) []abci.ValidatorUpdate {
	validators := make([]abci.ValidatorUpdate, vals.Size())
	for i, val := range vals.Validators {
		validators[i] = OC2PB.ValidatorUpdate(val)
	}
	return validators
}

func (oc2pb) ConsensusParams(params *tmproto.ConsensusParams) *abci.ConsensusParams {
	return &abci.ConsensusParams{
		Block: &abci.BlockParams{
			MaxBytes: params.Block.MaxBytes,
			MaxGas:   params.Block.MaxGas,
		},
		Evidence:  &params.Evidence,
		Validator: &params.Validator,
	}
}

// XXX: panics on nil or unknown pubkey type
func (oc2pb) NewValidatorUpdate(pubkey crypto.PubKey, power int64) abci.ValidatorUpdate {
	pubkeyABCI, err := cryptoenc.PubKeyToProto(pubkey)
	if err != nil {
		panic(err)
	}
	return abci.ValidatorUpdate{
		PubKey: pubkeyABCI,
		Power:  power,
	}
}

//----------------------------------------------------------------------------

// PB2OC is used for converting protobuf ABCI to Ostracon ABCI.
// UNSTABLE
var PB2OC = pb2tm{}

type pb2tm struct{}

func (pb2tm) ValidatorUpdates(vals []abci.ValidatorUpdate) ([]*Validator, error) {
	tmVals := make([]*Validator, len(vals))
	for i, v := range vals {
		pub, err := cryptoenc.PubKeyFromProto(&v.PubKey)
		if err != nil {
			return nil, err
		}
		tmVals[i] = NewValidator(pub, v.Power)
	}
	return tmVals, nil
}
