package types

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
)

type typeRegistry struct {
	addressTy      abi.Type
	bytesTy        abi.Type
	bytes32Ty      abi.Type
	uint256Ty      abi.Type
	uint256SliceTy abi.Type
}

func newTypeRegistry() (*typeRegistry, error) {
	addressTy, err := abi.NewType("address", "", nil)
	if err != nil {
		return nil, err
	}
	bytesTy, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return nil, err
	}
	bytes32Ty, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		return nil, err
	}
	uint256Ty, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}
	uint256SliceTy, err := abi.NewType("uint256[]", "", nil)
	if err != nil {
		return nil, err
	}
	return &typeRegistry{
		addressTy:      addressTy,
		bytesTy:        bytesTy,
		bytes32Ty:      bytes32Ty,
		uint256Ty:      uint256Ty,
		uint256SliceTy: uint256SliceTy,
	}, nil
}
