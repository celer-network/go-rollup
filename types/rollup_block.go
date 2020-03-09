package types

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
