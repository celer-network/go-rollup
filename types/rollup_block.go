package types

import "encoding/json"

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

func (block *RollupBlock) SerializeTransactions(s *Serializer) ([][]byte, error) {
	transitions := block.Transitions
	serializedTransitions := make([][]byte, len(transitions))
	for i, transition := range block.Transitions {
		serializedTransition, err := transition.Serialize(s)
		if err != nil {
			return nil, err
		}
		serializedTransitions[i] = serializedTransition
	}
	return serializedTransitions, nil
}

func (s *Serializer) DeserializeRollupBlock(block [][]byte, blockNumber uint64) (*RollupBlock, error) {
	transitions := make([]Transition, len(block))
	for i, transitionData := range block {
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
