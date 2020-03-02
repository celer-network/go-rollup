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
