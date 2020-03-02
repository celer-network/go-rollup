package types

import (
	"math/big"

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

func (info *AccountInfo) Serialize(s *Serializer) ([]byte, error) {
	return s.accountInfoArguments.Pack(
		info.Account,
		info.Balances,
		info.Nonces,
	)
}

func (s *Serializer) DeserializeAccountInfo(bytes []byte) (*AccountInfo, error) {
	var infoMap map[string]interface{}
	err := s.accountInfoArguments.UnpackIntoMap(infoMap, bytes)
	if err != nil {
		return nil, err
	}
	return &AccountInfo{
		Account:  infoMap["account"].(common.Address),
		Balances: infoMap["balances"].([]*big.Int),
		Nonces:   infoMap["nonces"].([]*big.Int),
	}, nil
}
