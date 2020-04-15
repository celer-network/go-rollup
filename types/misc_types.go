package types

import (
	"math/big"
)

type SignedStateReceipt struct {
}

type AccountInfoUpdate struct {
	Info       *AccountInfo
	NewAccount bool
}

type InclusionProof [][32]byte

type StateUpdateEntry struct {
	SlotIndex      *big.Int
	InclusionProof InclusionProof
	AccountInfo    *AccountInfo
	NewAccount     bool
}

type StateUpdate struct {
	Transaction *SignedTransaction
	StateRoot   [32]byte
	Entries     []*StateUpdateEntry
}

type StateSnapshot struct {
	SlotIndex      *big.Int
	AccountInfo    *AccountInfo
	StateRoot      []byte
	InclusionProof InclusionProof
}

type TransitionPosition struct {
	BlockNumber     uint64
	TransitionIndex uint64
}
