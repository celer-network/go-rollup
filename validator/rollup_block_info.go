package validator

import (
	"math/big"

	"github.com/celer-network/go-rollup/db/memorydb"
	"github.com/celer-network/go-rollup/smt"
	"github.com/celer-network/go-rollup/types"
	"github.com/celer-network/sidechain-contracts/bindings/go/mainchain/rollup"
	"golang.org/x/crypto/sha3"
)

type RollupBlockInfo struct {
	blockNumber        *big.Int
	encodedTransitions [][]byte
	smt                *smt.SparseMerkleTree
}

func NewRollupBlockInfo(serializer *types.Serializer, rollupBlock *types.RollupBlock) (*RollupBlockInfo, error) {
	transitions := rollupBlock.Transitions
	numTransitions := len(transitions)
	smt, err := smt.NewSparseMerkleTree(memorydb.NewDB(), sha3.NewLegacyKeccak256(), nil, numTransitions, false)
	if err != nil {
		return nil, err
	}
	encodedTransitions := make([][]byte, numTransitions)
	for i, transition := range transitions {
		encodedTransition, err := transition.Serialize(serializer)
		if err != nil {
			return nil, err
		}
		encodedTransitions[i] = encodedTransition
		_, err = smt.Update(big.NewInt(int64(i)).Bytes(), encodedTransition)
		if err != nil {
			return nil, err
		}
	}
	return &RollupBlockInfo{
		blockNumber:        big.NewInt(int64(rollupBlock.BlockNumber)),
		encodedTransitions: encodedTransitions,
		smt:                smt,
	}, nil
}

func (info *RollupBlockInfo) GetNumTransitions() int {
	return len(info.encodedTransitions)
}

func (info *RollupBlockInfo) GetIncludedTransition(transitionIndex int) (*rollup.DataTypesIncludedTransition, error) {
	inclusionProof, err := info.GetTransitionInclusionProof(transitionIndex)
	if err != nil {
		return nil, err
	}
	return &rollup.DataTypesIncludedTransition{
		Transition:     info.encodedTransitions[transitionIndex],
		InclusionProof: *inclusionProof,
	}, nil
}

func (info *RollupBlockInfo) GetTransitionInclusionProof(transitionIndex int) (*rollup.DataTypesTransitionInclusionProof, error) {
	transitionIndexInt := big.NewInt(int64(transitionIndex))
	proofData, err := info.smt.Prove(transitionIndexInt.Bytes())
	if err != nil {
		return nil, err
	}
	return &rollup.DataTypesTransitionInclusionProof{
		BlockNumber:     info.blockNumber,
		TransitionIndex: transitionIndexInt,
		Siblings:        types.ConvertToInclusionProof(proofData),
	}, nil
}