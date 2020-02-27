package serialization

import (
	"math/big"

	"github.com/celer-network/sidechain-rollup-aggregator/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

func createAccountInfoArguments() (abi.Arguments, error) {
	address, err := abi.NewType("address", "", nil)
	if err != nil {
		return nil, err
	}
	uint256Slice, err := abi.NewType("uint256[]", "", nil)
	if err != nil {
		return nil, err
	}
	return abi.Arguments([]abi.Argument{
		{Name: "account", Type: address, Indexed: false},
		{Name: "balances", Type: uint256Slice, Indexed: false},
		{Name: "nonces", Type: uint256Slice, Indexed: false},
	}), nil
}

func DeserializeAccountInfo(bytes []byte) (*types.AccountInfo, error) {
	arguments, err := createAccountInfoArguments()
	if err != nil {
		return nil, err
	}
	var infoMap map[string]interface{}
	err = arguments.UnpackIntoMap(infoMap, bytes)
	if err != nil {
		return nil, err
	}
	return &types.AccountInfo{
		Account:  infoMap["account"].(common.Address),
		Balances: infoMap["balances"].([]*big.Int),
		Nonces:   infoMap["nonces"].([]*big.Int),
	}, nil
}

func SerializeAccountInfo(info *types.AccountInfo) ([]byte, error) {
	arguments, err := createAccountInfoArguments()
	if err != nil {
		return nil, err
	}
	return arguments.Pack(info.Account, info.Balances, info.Nonces)
}
