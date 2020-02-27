package blockproducer

import "github.com/celer-network/sidechain-contracts/bindings/go/mainchain/rollup"

type BlockSubmitter struct {
	rollupChain *rollup.RollupChain
}

func NewBlockSubmitter() *BlockSubmitter {
	return nil
}
