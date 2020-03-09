package aggregator

import (
	"context"
	"errors"

	"github.com/celer-network/go-rollup/types"
	"github.com/celer-network/go-rollup/utils"
	"github.com/celer-network/sidechain-contracts/bindings/go/mainchain/rollup"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog/log"
)

type BlockSubmitter struct {
	mainchainClient *ethclient.Client
	mainchainAuth   *bind.TransactOpts
	serializer      *types.Serializer
	rollupChain     *rollup.RollupChain
}

func NewBlockSubmitter(
	mainchainClient *ethclient.Client,
	mainchainAuth *bind.TransactOpts,
	serializer *types.Serializer,
	rollupChain *rollup.RollupChain,
) *BlockSubmitter {
	return &BlockSubmitter{
		mainchainClient: mainchainClient,
		mainchainAuth:   mainchainAuth,
		serializer:      serializer,
		rollupChain:     rollupChain,
	}
}

func (bs *BlockSubmitter) submitBlock(pendingBlock *types.RollupBlock) error {
	serializedBlock, err := pendingBlock.SerializeTransactions(bs.serializer)
	if err != nil {
		return err
	}
	log.Print("Submitting block ", pendingBlock.BlockNumber)
	tx, err := bs.rollupChain.SubmitBlock(&bind.TransactOpts{}, serializedBlock)
	if err != nil {
		return err
	}
	receipt, err := utils.WaitMined(context.Background(), bs.mainchainClient, tx, 0)
	if err != nil {
		return err
	}
	if receipt.Status != 1 {
		return errors.New("Failed to submit block")
	}
	return nil
}
