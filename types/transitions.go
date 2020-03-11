package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type TransitionType int

const (
	TransitionTypeInitialDeposit TransitionType = iota
	TransitionTypeDeposit
	TransitionTypeWithdraw
	TransitionTypeTransfer
)

type Transition interface {
	GetTransitionType() TransitionType
	Serialize(*Serializer) ([]byte, error)
}

type InitialDepositTransition struct {
	TransitionType   *big.Int
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

func createInitialDepositTransitionArguments(r *typeRegistry) abi.Arguments {
	return abi.Arguments([]abi.Argument{
		{Name: "transitionType", Type: r.uint256Ty, Indexed: false},
		{Name: "stateRoot", Type: r.bytes32Ty, Indexed: false},
		{Name: "accountSlotIndex", Type: r.uint256Ty, Indexed: false},
		{Name: "account", Type: r.addressTy, Indexed: false},
		{Name: "tokenIndex", Type: r.uint256Ty, Indexed: false},
		{Name: "amount", Type: r.uint256Ty, Indexed: false},
		{Name: "signature", Type: r.bytesTy, Indexed: false},
	})
}

func (transition *InitialDepositTransition) Serialize(s *Serializer) ([]byte, error) {
	var stateRoot [32]byte
	copy(stateRoot[:], transition.StateRoot)
	return s.initialDepositTransitionArguments.Pack(
		transition.TransitionType,
		stateRoot,
		transition.AccountSlotIndex,
		transition.Account,
		transition.TokenIndex,
		transition.Amount,
		transition.Signature,
	)
}

type DepositTransition struct {
	TransitionType   *big.Int
	StateRoot        []byte
	AccountSlotIndex *big.Int
	TokenIndex       *big.Int
	Amount           *big.Int
	Signature        []byte
}

func (*DepositTransition) GetTransitionType() TransitionType {
	return TransitionTypeDeposit
}

func createDepositTransitionArguments(r *typeRegistry) abi.Arguments {
	return abi.Arguments([]abi.Argument{
		{Name: "transitionType", Type: r.uint256Ty, Indexed: false},
		{Name: "stateRoot", Type: r.bytes32Ty, Indexed: false},
		{Name: "accountSlotIndex", Type: r.uint256Ty, Indexed: false},
		{Name: "tokenIndex", Type: r.uint256Ty, Indexed: false},
		{Name: "amount", Type: r.uint256Ty, Indexed: false},
		{Name: "signature", Type: r.bytesTy, Indexed: false},
	})
}

func (transition *DepositTransition) Serialize(s *Serializer) ([]byte, error) {
	var stateRoot [32]byte
	copy(stateRoot[:], transition.StateRoot)
	return s.depositTransitionArguments.Pack(
		transition.TransitionType,
		stateRoot,
		transition.AccountSlotIndex,
		transition.TokenIndex,
		transition.Amount,
		transition.Signature,
	)
}

type WithdrawTransition struct {
	TransitionType   *big.Int
	StateRoot        []byte
	AccountSlotIndex *big.Int
	TokenIndex       *big.Int
	Amount           *big.Int
	Signature        []byte
}

func (*WithdrawTransition) GetTransitionType() TransitionType {
	return TransitionTypeWithdraw
}

func createWithdrawTransitionArguments(r *typeRegistry) abi.Arguments {
	return abi.Arguments([]abi.Argument{
		{Name: "transitionType", Type: r.uint256Ty, Indexed: false},
		{Name: "stateRoot", Type: r.bytes32Ty, Indexed: false},
		{Name: "accountSlotIndex", Type: r.uint256Ty, Indexed: false},
		{Name: "tokenIndex", Type: r.uint256Ty, Indexed: false},
		{Name: "amount", Type: r.uint256Ty, Indexed: false},
		{Name: "signature", Type: r.bytesTy, Indexed: false},
	})
}

func (transition *WithdrawTransition) Serialize(s *Serializer) ([]byte, error) {
	var stateRoot [32]byte
	copy(stateRoot[:], transition.StateRoot)
	return s.withdrawTransitionArguments.Pack(
		transition.TransitionType,
		stateRoot,
		transition.AccountSlotIndex,
		transition.TokenIndex,
		transition.Amount,
		transition.Signature,
	)
}

type TransferTransition struct {
	TransitionType     *big.Int
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

func createTransferTransitionArguments(r *typeRegistry) abi.Arguments {
	return abi.Arguments([]abi.Argument{
		{Name: "transitionType", Type: r.uint256Ty, Indexed: false},
		{Name: "stateRoot", Type: r.bytes32Ty, Indexed: false},
		{Name: "senderSlotIndex", Type: r.uint256Ty, Indexed: false},
		{Name: "recipientSlotIndex", Type: r.uint256Ty, Indexed: false},
		{Name: "tokenIndex", Type: r.uint256Ty, Indexed: false},
		{Name: "amount", Type: r.uint256Ty, Indexed: false},
		{Name: "nonce", Type: r.uint256Ty, Indexed: false},
		{Name: "signature", Type: r.bytesTy, Indexed: false},
	})
}

func (transition *TransferTransition) Serialize(s *Serializer) ([]byte, error) {
	var stateRoot [32]byte
	copy(stateRoot[:], transition.StateRoot)
	return s.transferTransitionArguments.Pack(
		transition.TransitionType,
		stateRoot,
		transition.SenderSlotIndex,
		transition.RecipientSlotIndex,
		transition.TokenIndex,
		transition.Amount,
		transition.Nonce,
		transition.Signature,
	)
}
