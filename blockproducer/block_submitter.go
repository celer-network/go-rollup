package blockproducer

import (
	"github.com/celer-network/sidechain-contracts/bindings/go/mainchain/rollup"
	"github.com/celer-network/sidechain-rollup-aggregator/types"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

type BlockSubmitter struct {
	serializer  *types.Serializer
	rollupChain *rollup.RollupChain
}

func NewBlockSubmitter(serializer *types.Serializer) *BlockSubmitter {
	return &BlockSubmitter{
		serializer: serializer,
	}
}

func (bs *BlockSubmitter) submitBlock(pendingBlock *types.RollupBlock) error {
	serializedBlock, err := pendingBlock.SerializeTransactions(bs.serializer)
	if err != nil {
		return err
	}
	_, err = bs.rollupChain.SubmitBlock(&bind.TransactOpts{}, serializedBlock)
	if err != nil {
		return err
	}

	return nil
}
