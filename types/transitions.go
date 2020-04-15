package types

import (
	"fmt"
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
	GetSignature() []byte
	GetStateRoot() [32]byte
	Serialize(*Serializer) ([]byte, error)
}

type InitialDepositTransition struct {
	TransitionType   *big.Int
	StateRoot        [32]byte
	AccountSlotIndex *big.Int
	Account          common.Address
	TokenIndex       *big.Int
	Amount           *big.Int
	Signature        []byte
}

func (*InitialDepositTransition) GetTransitionType() TransitionType {
	return TransitionTypeInitialDeposit
}

func (t *InitialDepositTransition) GetSignature() []byte {
	return t.Signature
}

func (t *InitialDepositTransition) GetStateRoot() [32]byte {
	return t.StateRoot
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
	copy(stateRoot[:], transition.StateRoot[:])
	data, err := s.initialDepositTransitionArguments.Pack(
		transition.TransitionType,
		stateRoot,
		transition.AccountSlotIndex,
		transition.Account,
		transition.TokenIndex,
		transition.Amount,
		transition.Signature,
	)
	return data, err
}

func (s *Serializer) DeserializeInitialDepositTransition(data []byte) (*InitialDepositTransition, error) {
	var transition InitialDepositTransition
	err := s.initialDepositTransitionArguments.Unpack(&transition, data)
	if err != nil {
		return nil, fmt.Errorf("Deserialize InitialDepositTransition, data %v: %w", data, err)
	}
	return &transition, nil
}

type DepositTransition struct {
	TransitionType   *big.Int
	StateRoot        [32]byte
	AccountSlotIndex *big.Int
	TokenIndex       *big.Int
	Amount           *big.Int
	Signature        []byte
}

func (*DepositTransition) GetTransitionType() TransitionType {
	return TransitionTypeDeposit
}

func (t *DepositTransition) GetSignature() []byte {
	return t.Signature
}

func (t *DepositTransition) GetStateRoot() [32]byte {
	return t.StateRoot
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
	copy(stateRoot[:], transition.StateRoot[:])
	return s.depositTransitionArguments.Pack(
		transition.TransitionType,
		stateRoot,
		transition.AccountSlotIndex,
		transition.TokenIndex,
		transition.Amount,
		transition.Signature,
	)
}

func (s *Serializer) DeserializeDepositTransition(data []byte) (*DepositTransition, error) {
	var transition DepositTransition
	err := s.depositTransitionArguments.Unpack(&transition, data)
	if err != nil {
		return nil, fmt.Errorf("Deserialize DepositTransition, data %v: %w", data, err)
	}
	return &transition, nil
}

type WithdrawTransition struct {
	TransitionType   *big.Int
	StateRoot        [32]byte
	AccountSlotIndex *big.Int
	TokenIndex       *big.Int
	Amount           *big.Int
	Signature        []byte
}

func (*WithdrawTransition) GetTransitionType() TransitionType {
	return TransitionTypeWithdraw
}

func (t *WithdrawTransition) GetSignature() []byte {
	return t.Signature
}

func (t *WithdrawTransition) GetStateRoot() [32]byte {
	return t.StateRoot
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
	copy(stateRoot[:], transition.StateRoot[:])
	return s.withdrawTransitionArguments.Pack(
		transition.TransitionType,
		stateRoot,
		transition.AccountSlotIndex,
		transition.TokenIndex,
		transition.Amount,
		transition.Signature,
	)
}

func (s *Serializer) DeserializeWithdrawTransition(data []byte) (*WithdrawTransition, error) {
	var transition WithdrawTransition
	err := s.withdrawTransitionArguments.Unpack(&transition, data)
	if err != nil {
		return nil, fmt.Errorf("Deserialize WithdrawTransition, data %v: %w", data, err)
	}
	return &transition, nil
}

type TransferTransition struct {
	TransitionType     *big.Int
	StateRoot          [32]byte
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

func (t *TransferTransition) GetSignature() []byte {
	return t.Signature
}

func (t *TransferTransition) GetStateRoot() [32]byte {
	return t.StateRoot
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
	copy(stateRoot[:], transition.StateRoot[:])
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

func (s *Serializer) DeserializeTransferTransition(data []byte) (*TransferTransition, error) {
	var transition TransferTransition
	err := s.transferTransitionArguments.Unpack(&transition, data)
	if err != nil {
		return nil, fmt.Errorf("Deserialize TransferTransition, data %v: %w", data, err)
	}
	return &transition, nil
}

func (s *Serializer) DeserializeTransition(data []byte) (Transition, error) {
	transitionType := new(big.Int).SetBytes(data[0:32]).Uint64()
	switch TransitionType(transitionType) {
	case TransitionTypeInitialDeposit:
		return s.DeserializeInitialDepositTransition(data)
	case TransitionTypeDeposit:
		return s.DeserializeDepositTransition(data)
	case TransitionTypeWithdraw:
		return s.DeserializeWithdrawTransition(data)
	case TransitionTypeTransfer:
		return s.DeserializeTransferTransition(data)
	}
	return nil, fmt.Errorf("Unknown transition type %d", transitionType)
}
