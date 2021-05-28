package types

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/tendermint/tendermint/crypto"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"
	tmjson "github.com/tendermint/tendermint/libs/json"
	tmos "github.com/tendermint/tendermint/libs/os"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmtime "github.com/tendermint/tendermint/types/time"
)

const (
	// MaxChainIDLen is a maximum length of the chain ID.
	MaxChainIDLen = 50
)

//------------------------------------------------------------
// core types for a genesis definition
// NOTE: any changes to the genesis definition should
// be reflected in the documentation:
// docs/tendermint-core/using-tendermint.md

// GenesisValidator is an initial validator.
type GenesisValidator struct {
	Address Address       `json:"address"`
	PubKey  crypto.PubKey `json:"pub_key"`
	Power   int64         `json:"power"`
	Name    string        `json:"name"`
}

type VoterParams struct {
	VoterElectionThreshold          int32 `json:"voter_election_threshold"`
	MaxTolerableByzantinePercentage int32 `json:"max_tolerable_byzantine_percentage"`

	// As a unit of precision, if it is 1, it is 0.9, and if it is 2, it is 0.99.
	// The default is 5, with a precision of 0.99999.
	ElectionPrecision int32 `json:"election_precision"`
}

func (vp *VoterParams) DefaultVoterParams() *VoterParams {
	return &VoterParams{
		DefaultVoterElectionThreshold,
		DefaultMaxTolerableByzantinePercentage,
		DefaultElectionPrecision,
	}
}

// GenesisDoc defines the initial conditions for a tendermint blockchain, in particular its validator set.
type GenesisDoc struct {
	GenesisTime     time.Time                `json:"genesis_time"`
	ChainID         string                   `json:"chain_id"`
	InitialHeight   int64                    `json:"initial_height"`
	ConsensusParams *tmproto.ConsensusParams `json:"consensus_params,omitempty"`
	Validators      []GenesisValidator       `json:"validators,omitempty"`
	VoterParams     *VoterParams             `json:"voter_params,omitempty"`
	AppHash         tmbytes.HexBytes         `json:"app_hash"`
	AppState        json.RawMessage          `json:"app_state,omitempty"`
}

// SaveAs is a utility method for saving GenensisDoc as a JSON file.
func (genDoc *GenesisDoc) SaveAs(file string) error {
	genDocBytes, err := tmjson.MarshalIndent(genDoc, "", "  ")
	if err != nil {
		return err
	}
	return tmos.WriteFile(file, genDocBytes, 0644)
}

// ValidatorHash returns the hash of the validator set contained in the GenesisDoc
func (genDoc *GenesisDoc) ValidatorHash() []byte {
	vals := make([]*Validator, len(genDoc.Validators))
	for i, v := range genDoc.Validators {
		vals[i] = NewValidator(v.PubKey, v.Power)
	}
	vset := NewValidatorSet(vals)
	return vset.Hash()
}

// ValidateAndComplete checks that all necessary fields are present
// and fills in defaults for optional fields left empty
func (genDoc *GenesisDoc) ValidateAndComplete() error {
	if genDoc.ChainID == "" {
		return errors.New("genesis doc must include non-empty chain_id")
	}
	if len(genDoc.ChainID) > MaxChainIDLen {
		return fmt.Errorf("chain_id in genesis doc is too long (max: %d)", MaxChainIDLen)
	}
	if genDoc.InitialHeight < 0 {
		return fmt.Errorf("initial_height cannot be negative (got %v)", genDoc.InitialHeight)
	}
	if genDoc.InitialHeight == 0 {
		genDoc.InitialHeight = 1
	}

	if genDoc.ConsensusParams == nil {
		genDoc.ConsensusParams = DefaultConsensusParams()
	} else if err := ValidateConsensusParams(*genDoc.ConsensusParams); err != nil {
		return err
	}

	if genDoc.VoterParams == nil {
		genDoc.VoterParams = DefaultVoterParams()
	} else if err := genDoc.VoterParams.Validate(); err != nil {
		return err
	}

	for i, v := range genDoc.Validators {
		if v.Power == 0 {
			return fmt.Errorf("the genesis file cannot contain validators with no voting power: %v", v)
		}
		if len(v.Address) > 0 && !bytes.Equal(v.PubKey.Address(), v.Address) {
			return fmt.Errorf("incorrect address for validator %v in the genesis file, should be %v", v, v.PubKey.Address())
		}
		if len(v.Address) == 0 {
			genDoc.Validators[i].Address = v.PubKey.Address()
		}
	}

	if genDoc.GenesisTime.IsZero() {
		genDoc.GenesisTime = tmtime.Now()
	}

	return nil
}

// Hash returns the hash of the GenesisDoc
func (genDoc *GenesisDoc) Hash() []byte {
	genDocBytes, err := tmjson.Marshal(genDoc)
	if err != nil {
		panic(err)
	}
	return crypto.Sha256(genDocBytes)
}

//------------------------------------------------------------
// Make genesis state from file

// GenesisDocFromJSON unmarshalls JSON data into a GenesisDoc.
func GenesisDocFromJSON(jsonBlob []byte) (*GenesisDoc, error) {
	genDoc := GenesisDoc{}
	err := tmjson.Unmarshal(jsonBlob, &genDoc)
	if err != nil {
		return nil, err
	}

	if err := genDoc.ValidateAndComplete(); err != nil {
		return nil, err
	}

	return &genDoc, err
}

// GenesisDocFromFile reads JSON data from a file and unmarshalls it into a GenesisDoc.
func GenesisDocFromFile(genDocFile string) (*GenesisDoc, error) {
	jsonBlob, err := ioutil.ReadFile(genDocFile)
	if err != nil {
		return nil, fmt.Errorf("couldn't read GenesisDoc file: %w", err)
	}
	genDoc, err := GenesisDocFromJSON(jsonBlob)
	if err != nil {
		return nil, fmt.Errorf("error reading GenesisDoc at %s: %w", genDocFile, err)
	}
	return genDoc, nil
}

func (vp *VoterParams) Validate() error {
	if vp.VoterElectionThreshold < 0 {
		return fmt.Errorf("VoterElectionThreshold must be greater than or equal to 0. Got %d",
			vp.VoterElectionThreshold)
	}
	if vp.MaxTolerableByzantinePercentage <= 0 || vp.MaxTolerableByzantinePercentage >= 34 {
		return fmt.Errorf("MaxTolerableByzantinePercentage must be in between 1 and 33. Got %d",
			vp.MaxTolerableByzantinePercentage)
	}
	if vp.ElectionPrecision <= 1 || vp.ElectionPrecision > 15 {
		return fmt.Errorf("ElectionPrecision must be in between 2 and 15. Got %d", vp.ElectionPrecision)
	}
	return nil
}

func (vp *VoterParams) ToProto() *tmproto.VoterParams {
	if vp == nil {
		return nil
	}
	return &tmproto.VoterParams{
		VoterElectionThreshold:          vp.VoterElectionThreshold,
		MaxTolerableByzantinePercentage: vp.MaxTolerableByzantinePercentage,
		ElectionPrecision:               vp.ElectionPrecision,
	}
}
func VoterParamsFromProto(pb *tmproto.VoterParams) *VoterParams {
	if pb == nil {
		return nil
	}
	return &VoterParams{
		VoterElectionThreshold:          pb.VoterElectionThreshold,
		MaxTolerableByzantinePercentage: pb.MaxTolerableByzantinePercentage,
		ElectionPrecision:               pb.ElectionPrecision,
	}
}
