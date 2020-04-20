package types

import (
	"fmt"
	"math/big"

	"github.com/celer-network/rollup-contracts/bindings/go/mainchain/rollup"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

type LocalFraudProof struct {
	Position   *TransitionPosition
	Inputs     []*StateSnapshot
	Transition Transition
}

type StorageSlot struct {
	SlotIndex   *big.Int
	AccountInfo *AccountInfo
}

func createStorageSlotArguments(r *typeRegistry) (abi.Arguments, error) {
	accountInfoType, err := createAccountInfoType(r)
	if err != nil {
		return nil, err
	}
	return abi.Arguments([]abi.Argument{
		{Name: "slotIndex", Type: r.uint256Ty, Indexed: false},
		{Name: "accountInfo", Type: accountInfoType, Indexed: false},
	}), nil
}

func createStorageSlotType(r *typeRegistry) (abi.Type, error) {
	return abi.NewType("tuple", "", []abi.ArgumentMarshaling{
		{Name: "slotIndex", Type: "uint256"},
		{Name: "accountInfo", Type: "tuple", Components: createAccountInfoArgumentMarshaling()},
	})
}

func (slot *StorageSlot) Serialize(s *Serializer) ([]byte, error) {
	data, err := s.storageSlotArguments.Pack(
		slot.SlotIndex,
		slot.AccountInfo,
	)
	if err != nil {
		return nil, fmt.Errorf("Serialize StorageSlot %v: %w", slot, err)
	}
	return data, nil
}

type IncludedStorageSlot struct {
	StorageSlot *StorageSlot
	Siblings    [][32]byte
}

func createIncludedStorageSlotArguments(r *typeRegistry) (abi.Arguments, error) {
	storageSlotType, err := createStorageSlotType(r)
	if err != nil {
		return nil, err
	}
	return abi.Arguments([]abi.Argument{
		{Name: "storageSlot", Type: storageSlotType, Indexed: false},
		{Name: "siblings", Type: r.bytes32SliceTy, Indexed: false},
	}), nil
}

type TransitionInclusionProof struct {
	BlockNumber     *big.Int
	TransitionIndex *big.Int
	Siblings        [][32]byte
}

func createTransitionInclusionProofArgumentMarshaling() []abi.ArgumentMarshaling {
	return []abi.ArgumentMarshaling{
		{Name: "blockNumber", Type: "uint256"},
		{Name: "transitionIndex", Type: "uint256"},
		{Name: "siblings", Type: "bytes32"},
	}
}

func createTransitionInclusionProofType(r *typeRegistry) (abi.Type, error) {
	return abi.NewType("tuple", "", createTransitionInclusionProofArgumentMarshaling())
}

type IncludedTransition struct {
	Transition     []byte
	InclusionProof *TransitionInclusionProof
}

func createIncludedTransitionType(r *typeRegistry) (abi.Type, error) {
	return abi.NewType("tuple", "", []abi.ArgumentMarshaling{
		{Name: "transition", Type: "bytes"},
		{Name: "inclusionProof", Type: "tuple", Components: createTransitionInclusionProofArgumentMarshaling()},
	})
}

type ContractFraudProof struct {
	PreStateIncludedTransition rollup.DataTypesIncludedTransition
	InvalidIncludedTransition  rollup.DataTypesIncludedTransition
	TransitionStorageSlots     []rollup.DataTypesIncludedStorageSlot
}

func ConvertToInclusionProof(data [][]byte) InclusionProof {
	proof := make([][32]byte, len(data))
	for i, sibling := range data {
		var arr [32]byte
		copy(arr[:], sibling)
		proof[i] = arr
	}
	return proof
}
