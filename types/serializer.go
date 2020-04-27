package types

import "github.com/ethereum/go-ethereum/accounts/abi"

type Serializer struct {
	typeRegistry                         *typeRegistry
	accountInfoArguments                 abi.Arguments
	storageSlotArguments                 abi.Arguments
	createAndDepositTransitionArguments  abi.Arguments
	depositTransitionArguments           abi.Arguments
	withdrawTransitionArguments          abi.Arguments
	createAndTransferTransitionArguments abi.Arguments
	transferTransitionArguments          abi.Arguments
}

func NewSerializer() (*Serializer, error) {
	typeRegistry, err := newTypeRegistry()
	if err != nil {
		return nil, err
	}
	storageSlotArguments, err := createStorageSlotArguments(typeRegistry)
	if err != nil {
		return nil, err
	}
	return &Serializer{
		typeRegistry:                         typeRegistry,
		accountInfoArguments:                 createAccountInfoArguments(typeRegistry),
		storageSlotArguments:                 storageSlotArguments,
		createAndDepositTransitionArguments:  createCreateAndDepositTransitionArguments(typeRegistry),
		depositTransitionArguments:           createDepositTransitionArguments(typeRegistry),
		withdrawTransitionArguments:          createWithdrawTransitionArguments(typeRegistry),
		createAndTransferTransitionArguments: createCreateAndTransferTransitionArguments(typeRegistry),
		transferTransitionArguments:          createTransferTransitionArguments(typeRegistry),
	}, nil
}
