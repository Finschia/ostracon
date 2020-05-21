package provider

import (
	"github.com/tendermint/tendermint/types"
)

// Provider provides information for the lite client to sync (verification
// happens in the client).
type Provider interface {
	// ChainID returns the blockchain ID.
	ChainID() string

	// SignedHeader returns the SignedHeader that corresponds to the given
	// height.
	//
	// 0 - the latest.
	// height must be >= 0.
	//
	// If the provider fails to fetch the SignedHeader due to the IO or other
	// issues, an error will be returned.
	// If there's no SignedHeader for the given height, ErrSignedHeaderNotFound
	// error is returned.
	SignedHeader(height int64) (*types.SignedHeader, error)

	// VoterSet returns the VoterSet that corresponds to height.
	//
	// 0 - the latest.
	// height must be >= 0.
	//
	// If the provider fails to fetch the VoterSet due to the IO or other
	// issues, an error will be returned.
	// If there's no VoterSet for the given height, ErrValidatorSetNotFound
	// error is returned.
	VoterSet(height int64) (*types.VoterSet, error)
}
