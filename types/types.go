package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type RollupBlock struct {
	BlockNumber uint64
	Transitions []Transition
}

type TransitionType int

const (
	TransitionTypeInitialDeposit TransitionType = iota
	TransitionTypeDeposit
	TransitionTypeWithdraw
	TransitionTypeTransfer
)

type Transition interface {
	GetTransitionType() TransitionType
}

type InitialDepositTransition struct {
	StateRoot        []byte
	AccountSlotIndex *big.Int
	Account          common.Address
	TokenIndex       *big.Int
	Amount           *big.Int
	Signature        []byte
}

func (*InitialDepositTransition) GetTransitionType() TransitionType {
	return TransitionTypeInitialDeposit
}

type DepositTransition struct {
	StateRoot        []byte
	AccountSlotIndex *big.Int
	TokenIndex       *big.Int
	Amount           *big.Int
	Signature        []byte
}

func (*DepositTransition) GetTransitionType() TransitionType {
	return TransitionTypeDeposit
}

type WithdrawTransition struct {
	StateRoot        []byte
	AccountSlotIndex *big.Int
	TokenIndex       *big.Int
	Amount           *big.Int
	Signature        []byte
}

func (*WithdrawTransition) GetTransitionType() TransitionType {
	return TransitionTypeWithdraw
}

type TransferTransition struct {
	StateRoot          []byte
	SenderSlotIndex    *big.Int
	RecipientSlotIndex *big.Int
	TokenIndex         *big.Int
	Amount             *big.Int
	Nonce              *big.Int
	Signature          []byte
}

func (*TransferTransition) GetTransitionType() TransitionType {
	return TransitionTypeTransfer
}

type TransactionType int

const (
	TransactionTypeDeposit TransactionType = iota
	TransactionTypeWithdraw
	TransactionTypeTransfer
)

type Transaction interface {
	GetTransactionType() TransactionType
}

type DepositTransaction struct {
	Account common.Address
	Token   common.Address
	Amount  *big.Int
}

func (*DepositTransaction) GetTransactionType() TransactionType {
	return TransactionTypeDeposit
}

type WithdrawTransaction struct {
	Account common.Address
	Token   common.Address
	Amount  *big.Int
}

func (*WithdrawTransaction) GetTransactionType() TransactionType {
	return TransactionTypeWithdraw
}

type TransferTransaction struct {
	Sender    common.Address
	Recipient common.Address
	Token     common.Address
	Amount    *big.Int
	Nonce     *big.Int
}

func (*TransferTransaction) GetTransactionType() TransactionType {
	return TransactionTypeTransfer
}

type SignedTransaction struct {
	Signature   []byte
	Transaction Transaction
}

type SignedStateReceipt struct {
}

type AccountInfo struct {
	Account  common.Address
	Balances []*big.Int
	Nonces   []*big.Int
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
