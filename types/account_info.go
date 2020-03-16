package types

import (
	"fmt"
	"math/big"
	"runtime/debug"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type AccountInfo struct {
	Account  common.Address
	Balances []*big.Int
	Nonces   []*big.Int
}

func createAccountInfoArguments(r *typeRegistry) abi.Arguments {
	return abi.Arguments([]abi.Argument{
		{Name: "account", Type: r.addressTy, Indexed: false},
		{Name: "balances", Type: r.uint256SliceTy, Indexed: false},
		{Name: "nonces", Type: r.uint256SliceTy, Indexed: false},
	})
}

func createAccountInfoArgumentMarshaling() []abi.ArgumentMarshaling {
	return []abi.ArgumentMarshaling{
		{Name: "account", Type: "address"},
		{Name: "balances", Type: "uint256[]"},
		{Name: "nonces", Type: "uint256[]"},
	}
}

func createAccountInfoType(r *typeRegistry) (abi.Type, error) {
	return abi.NewType("tuple", "", createAccountInfoArgumentMarshaling())
}

func (info *AccountInfo) Serialize(s *Serializer) ([]byte, error) {
	data, err := s.accountInfoArguments.Pack(
		info.Account,
		info.Balances,
		info.Nonces,
	)
	if err != nil {
		return nil, fmt.Errorf("Serialize AccountInfo %v: %w", info, err)
	}
	return data, nil
}

func (s *Serializer) DeserializeAccountInfo(data []byte) (*AccountInfo, error) {
	var info AccountInfo
	err := s.accountInfoArguments.Unpack(&info, data)
	if err != nil {
		debug.PrintStack()
		return nil, fmt.Errorf("Deserialize AccountInfo, data %v: %w", data, err)
	}
	return &info, nil
}
