package aggregator

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"sync"

	rollupdb "github.com/celer-network/go-rollup/db"

	"github.com/celer-network/go-rollup/db/memorydb"
	"github.com/celer-network/go-rollup/smt"
	"golang.org/x/crypto/sha3"

	"github.com/celer-network/go-rollup/types"
	"github.com/celer-network/go-rollup/utils"
	"github.com/celer-network/rollup-contracts/bindings/go/mainchain"
	"github.com/celer-network/rollup-contracts/bindings/go/sidechain"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog/log"
)

type BlockSubmitter struct {
	mainchainClient         *ethclient.Client
	mainchainAuth           *bind.TransactOpts
	mainchainAuthPrivateKey *ecdsa.PrivateKey
	sidechainClient         *ethclient.Client
	sidechainAuth           *bind.TransactOpts
	sidechainAuthPrivateKey *ecdsa.PrivateKey
	serializer              *types.Serializer
	rollupChain             *mainchain.RollupChain
	validatorRegistry       *mainchain.ValidatorRegistry
	blockCommittee          *sidechain.BlockCommittee
	currentProposer         common.Address
	currentCommitter        common.Address
	lock                    sync.Mutex
}

func NewBlockSubmitter(
	mainchainClient *ethclient.Client,
	mainchainAuth *bind.TransactOpts,
	mainchainAuthPrivatekey *ecdsa.PrivateKey,
	sidechainClient *ethclient.Client,
	sidechainAuth *bind.TransactOpts,
	sidechainAuthPrivateKey *ecdsa.PrivateKey,
	serializer *types.Serializer,
	rollupChain *mainchain.RollupChain,
	validatorRegistry *mainchain.ValidatorRegistry,
	blockCommittee *sidechain.BlockCommittee,
) *BlockSubmitter {
	return &BlockSubmitter{
		mainchainClient:         mainchainClient,
		mainchainAuth:           mainchainAuth,
		mainchainAuthPrivateKey: mainchainAuthPrivatekey,
		sidechainClient:         sidechainClient,
		sidechainAuth:           sidechainAuth,
		sidechainAuthPrivateKey: sidechainAuthPrivateKey,
		serializer:              serializer,
		rollupChain:             rollupChain,
		validatorRegistry:       validatorRegistry,
		blockCommittee:          blockCommittee,
	}
}

func (bs *BlockSubmitter) Start() {
	go bs.watchBlockCommittee()
}

func (bs *BlockSubmitter) watchBlockCommittee() error {
	log.Print("Watching BlockCommittee")
	blockProposedChannel := make(chan *sidechain.BlockCommitteeBlockProposed)
	blockConsensusReachedChannel := make(chan *sidechain.BlockCommitteeBlockConsensusReached)

	blockProposedSub, err := bs.blockCommittee.WatchBlockProposed(&bind.WatchOpts{}, blockProposedChannel)
	if err != nil {
		return err
	}
	blockConsensusReachedSub, err :=
		bs.blockCommittee.WatchBlockConsensusReached(&bind.WatchOpts{}, blockConsensusReachedChannel)
	if err != nil {
		return err
	}
	for {
		select {
		case event := <-blockProposedChannel:
			bs.lock.Lock()
			log.Debug().Uint64("blockNumber", event.BlockNumber.Uint64()).Msg("Caught BlockProposed")
			bs.submitSignature(event.BlockNumber, event.Transitions)
			bs.lock.Unlock()
		case _ = <-blockProposedSub.Err():
		case event := <-blockConsensusReachedChannel:
			bs.lock.Lock()
			log.Debug().Uint64("blockNumber", event.Proposal.BlockNumber.Uint64()).Msg("Caught BlockConsensusReached")
			bs.commitBlock(&event.Proposal, event.Signatures)
			bs.lock.Unlock()
		case _ = <-blockConsensusReachedSub.Err():
		}
	}
}

func (bs *BlockSubmitter) proposeBlock(pendingBlock *types.RollupBlock) (bool, error) {
	proposerAddress, err := bs.blockCommittee.CurrentProposer(&bind.CallOpts{})
	if err != nil {
		return false, err
	}
	// Hack for now
	if !bytes.Equal(bs.sidechainAuth.From.Bytes(), proposerAddress.Bytes()) {
		return false, nil
	}
	serializedTransitions, encodedBlock, err := pendingBlock.SerializeForSubmission(bs.serializer)
	if err != nil {
		return false, err
	}
	signature, err := utils.SignData(bs.sidechainAuthPrivateKey, encodedBlock)
	log.Debug().Uint64("blockNumber", pendingBlock.BlockNumber).Msg("Proposing block")
	bs.sidechainAuth.GasLimit = 10000000
	tx, err :=
		bs.blockCommittee.ProposeBlock(
			bs.sidechainAuth,
			new(big.Int).SetUint64(pendingBlock.BlockNumber),
			serializedTransitions,
			signature)
	if err != nil {
		log.Error().Err(err).Send()
		return false, err
	}
	receipt, err := utils.WaitMined(context.Background(), bs.sidechainClient, tx, 0)
	if err != nil {
		return false, err
	}
	if receipt.Status != 1 {
		log.Error().Str("tx", tx.Hash().Hex()).Msg("Failed to propose block")
		return false, errors.New("Failed to propose block")
	}
	log.Debug().Str("tx", tx.Hash().Hex()).Msg("Proposed block")
	return true, nil
}

func (bs *BlockSubmitter) commitBlock(
	proposal *sidechain.BlockCommitteeBlockProposal, signatures [][]byte) error {
	committerAddress, err := bs.rollupChain.CommitterAddress(&bind.CallOpts{})
	if err != nil {
		return err
	}
	// Hack for now
	if !bytes.Equal(bs.mainchainAuth.From.Bytes(), committerAddress.Bytes()) {
		return nil
	}
	log.Debug().Uint64("blockNumber", proposal.BlockNumber.Uint64()).Msg("Committing block")
	tx, err :=
		bs.rollupChain.CommitBlock(
			bs.mainchainAuth,
			proposal.BlockNumber,
			proposal.Transitions,
			signatures,
		)
	if err != nil {
		return err
	}
	receipt, err := utils.WaitMined(context.Background(), bs.mainchainClient, tx, 0)
	if err != nil {
		return err
	}
	if receipt.Status != 1 {
		return errors.New("Failed to commit block")
	}
	log.Debug().Str("tx", tx.Hash().Hex()).Msg("Committed block")
	block, _ := bs.rollupChain.Blocks(&bind.CallOpts{}, big.NewInt(0))
	log.Printf("Contract block root hash: %s", common.Bytes2Hex(block.RootHash[:]))
	tree, _ := smt.NewSparseMerkleTree(memorydb.NewDB(), rollupdb.NamespaceRollupBlockTrie, sha3.NewLegacyKeccak256(), nil, int(block.BlockSize.Uint64()), false)
	for i, encodedTransition := range proposal.Transitions {
		//log.Debug().Str("encodedTransition", common.Bytes2Hex(encodedTransition)).Send()
		_, _ = tree.Update(big.NewInt(int64(i)).Bytes(), encodedTransition)
	}
	log.Printf("Local block root hash: %s", common.Bytes2Hex(tree.Root()))

	return nil
}

func (bs *BlockSubmitter) submitSignature(blockNumber *big.Int, transitions [][]byte) error {
	proposerAddress, err := bs.blockCommittee.CurrentProposer(&bind.CallOpts{})
	if err != nil {
		return err
	}
	// Hack for now
	if bytes.Equal(bs.mainchainAuth.From.Bytes(), proposerAddress.Bytes()) {
		return nil
	}
	encodedBlock, err := types.EncodeBlock(bs.serializer, blockNumber, transitions)
	if err != nil {
		return err
	}
	signature, err := utils.SignData(bs.sidechainAuthPrivateKey, encodedBlock)
	if err != nil {
		return err
	}
	log.Debug().Uint64("blockNumber", blockNumber.Uint64()).Msg("Submitting signature for block")
	bs.sidechainAuth.GasLimit = 10000000
	tx, err := bs.blockCommittee.SignBlock(bs.sidechainAuth, bs.sidechainAuth.From, signature)
	if err != nil {
		return err
	}
	receipt, err := utils.WaitMined(context.Background(), bs.sidechainClient, tx, 0)
	if err != nil {
		return err
	}
	if receipt.Status != 1 {
		log.Error().Str("tx", tx.Hash().Hex()).Msg("Failed to submit signature")
		return errors.New("Failed to submit block proposal signature")
	}
	log.Debug().Str("tx", tx.Hash().Hex()).Msg("Submitted signature")
	return nil
}
