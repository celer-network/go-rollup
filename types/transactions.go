package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type TransactionType int

const (
	TransactionTypeDeposit TransactionType = iota
	TransactionTypeWithdraw
	TransactionTypeTransfer
)

type Transaction interface {
	GetTransactionType() TransactionType
	GetSignature() []byte
}

type DepositTransaction struct {
	Account   common.Address
	Token     common.Address
	Amount    *big.Int
	Signature []byte
}

func (*DepositTransaction) GetTransactionType() TransactionType {
	return TransactionTypeDeposit
}

func (t *DepositTransaction) GetSignature() []byte {
	return t.Signature
}

type WithdrawTransaction struct {
	Account   common.Address
	Token     common.Address
	Amount    *big.Int
	Nonce     *big.Int
	Signature []byte
}

func (*WithdrawTransaction) GetTransactionType() TransactionType {
	return TransactionTypeWithdraw
}

func (t *WithdrawTransaction) GetSignature() []byte {
	return t.Signature
}

type TransferTransaction struct {
	Sender    common.Address
	Recipient common.Address
	Token     common.Address
	Amount    *big.Int
	Nonce     *big.Int
	Signature []byte
}

func (*TransferTransaction) GetTransactionType() TransactionType {
	return TransactionTypeTransfer
}

func (t *TransferTransaction) GetSignature() []byte {
	return t.Signature
}
