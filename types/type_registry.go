package types

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
)

type typeRegistry struct {
	addressTy      abi.Type
	bytesTy        abi.Type
	bytesSliceTy   abi.Type
	bytes32Ty      abi.Type
	bytes32SliceTy abi.Type
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
	bytesSliceTy, err := abi.NewType("bytes[]", "", nil)
	if err != nil {
		return nil, err
	}
	bytes32Ty, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		return nil, err
	}
	bytes32SliceTy, err := abi.NewType("bytes32[]", "", nil)
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
		bytesSliceTy:   bytesSliceTy,
		bytes32Ty:      bytes32Ty,
		bytes32SliceTy: bytes32SliceTy,
		uint256Ty:      uint256Ty,
		uint256SliceTy: uint256SliceTy,
	}, nil
}
