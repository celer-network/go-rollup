package aggregator

import (
	"bytes"
	"context"
	"errors"
	"math/big"

	rollupdb "github.com/celer-network/go-rollup/db"

	"github.com/celer-network/go-rollup/db/memorydb"
	"github.com/celer-network/go-rollup/smt"
	"golang.org/x/crypto/sha3"

	"github.com/celer-network/go-rollup/types"
	"github.com/celer-network/go-rollup/utils"
	"github.com/celer-network/sidechain-contracts/bindings/go/mainchain/rollup"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
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
	aggregatorAddress, err := bs.rollupChain.AggregatorAddress(&bind.CallOpts{})
	if err != nil {
		return err
	}
	// Hack for now
	if !bytes.Equal(bs.mainchainAuth.From.Bytes(), aggregatorAddress.Bytes()) {
		return nil
	}
	serializedBlock, err := pendingBlock.SerializeTransactions(bs.serializer)
	if err != nil {
		return err
	}
	log.Print("Submitting block ", pendingBlock.BlockNumber)
	tx, err := bs.rollupChain.SubmitBlock(bs.mainchainAuth, serializedBlock)
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
	block, _ := bs.rollupChain.Blocks(&bind.CallOpts{}, big.NewInt(0))
	log.Printf("Contract block root hash: %s", common.Bytes2Hex(block.RootHash[:]))
	tree, _ := smt.NewSparseMerkleTree(memorydb.NewDB(), rollupdb.NamespaceRollupBlockTrie, sha3.NewLegacyKeccak256(), nil, int(block.BlockSize.Uint64()), false)
	transitions := pendingBlock.Transitions
	encodedTransitions := make([][]byte, len(transitions))
	for i, transition := range transitions {
		encodedTransition, _ := transition.Serialize(bs.serializer)
		log.Debug().Str("encodedTransition", common.Bytes2Hex(encodedTransition)).Send()
		//log.Printf("Local encodedTransition hash %s", sha3.NewLegacyKeccak256().Sum(encodedTransition))
		encodedTransitions[i] = encodedTransition
		_, _ = tree.Update(big.NewInt(int64(i)).Bytes(), encodedTransition)
	}
	log.Printf("Local block root hash: %s", common.Bytes2Hex(tree.Root()))

	return nil
}
