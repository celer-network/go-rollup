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

type InclusionProof [][]byte

type StateUpdateEntry struct {
	SlotIndex      *big.Int
	InclusionProof InclusionProof
	AccountInfo    *AccountInfo
	NewAccount     bool
}

type StateUpdate struct {
	Transaction *SignedTransaction
	StateRoot   []byte
	Entries     []*StateUpdateEntry
}
