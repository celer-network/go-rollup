package types

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

type RollupBlock struct {
	BlockNumber uint64
	Transitions []Transition
}

func NewRollupBlock(blockNumber uint64) *RollupBlock {
	return &RollupBlock{
		BlockNumber: blockNumber,
		Transitions: nil,
	}
}

func (block *RollupBlock) SerializeForStorage() ([]byte, error) {
	// TODO: Check gob?
	return json.Marshal(block)
}

func DeserializeRollupBlockFromStorage(data []byte) (*RollupBlock, error) {
	var block RollupBlock
	err := json.Unmarshal(data, &block)
	if err != nil {
		return nil, err
	}
	return &block, nil
}

func (s *Serializer) DeserializeRollupBlock(
	blockNumber uint64, rawTransitions [][]byte) (*RollupBlock, error) {
	transitions := make([]Transition, len(rawTransitions))
	for i, transitionData := range rawTransitions {
		transition, err := s.DeserializeTransition(transitionData)
		if err != nil {
			return nil, err
		}
		transitions[i] = transition
	}
	return &RollupBlock{
		BlockNumber: blockNumber,
		Transitions: transitions,
	}, nil
}

func (block *RollupBlock) SerializeForSubmission(s *Serializer) ([][]byte, []byte, error) {
	transitions := block.Transitions
	serializedTransitions := make([][]byte, len(transitions))
	for i, transition := range block.Transitions {
		serializedTransition, err := transition.Serialize(s)
		if err != nil {
			return nil, nil, err
		}
		serializedTransitions[i] = serializedTransition
	}
	encodedBlock, err := EncodeBlock(s, new(big.Int).SetUint64(block.BlockNumber), serializedTransitions)
	if err != nil {
		return nil, nil, err
	}
	return serializedTransitions, encodedBlock, nil
}

func EncodeBlock(s *Serializer, blockNumber *big.Int, serializedTransitions [][]byte) ([]byte, error) {
	rollupBlockArguments := abi.Arguments([]abi.Argument{
		{Name: "blockNumber", Type: s.typeRegistry.uint256Ty, Indexed: false},
		{Name: "transitions", Type: s.typeRegistry.bytesSliceTy, Indexed: false},
	})
	encodedBlock, err := rollupBlockArguments.Pack(
		blockNumber,
		serializedTransitions)
	if err != nil {
		return nil, err
	}
	return encodedBlock, nil
}
