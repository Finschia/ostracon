package types

const (
	PubKeyEd25519 = "ed25519"
)

func NewValidatorUpdate(keyType string, pubkey []byte, power int64) ValidatorUpdate {
	return ValidatorUpdate{
		// Address:
		PubKey: PubKey{
			Type: keyType,
			Data: pubkey,
		},
		Power: power,
	}
}

func Ed25519ValidatorUpdate(pubkey []byte, power int64) ValidatorUpdate {
	return NewValidatorUpdate(PubKeyEd25519, pubkey, power)
}
