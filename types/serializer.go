package types

import "github.com/ethereum/go-ethereum/accounts/abi"

type Serializer struct {
	typeRegistry                      *typeRegistry
	accountInfoArguments              abi.Arguments
	initialDepositTransitionArguments abi.Arguments
	depositTransitionArguments        abi.Arguments
	withdrawTransitionArguments       abi.Arguments
	transferTransitionArguments       abi.Arguments
}

func NewSerializer() (*Serializer, error) {
	typeRegistry, err := newTypeRegistry()
	if err != nil {
		return nil, err
	}
	return &Serializer{
		typeRegistry:                      typeRegistry,
		accountInfoArguments:              createAccountInfoArguments(typeRegistry),
		initialDepositTransitionArguments: createInitialDepositTransitionArguments(typeRegistry),
		depositTransitionArguments:        createDepositTransitionArguments(typeRegistry),
		withdrawTransitionArguments:       createWithdrawTransitionArguments(typeRegistry),
		transferTransitionArguments:       createTransferTransitionArguments(typeRegistry),
	}, nil
}
