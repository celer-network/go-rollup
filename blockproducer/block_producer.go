package blockproducer

import (
	"math/big"

	"github.com/aergoio/aergo-lib/db"
	"github.com/celer-network/sidechain-rollup-aggregator/statemachine"
	"github.com/celer-network/sidechain-rollup-aggregator/storage"
	"github.com/celer-network/sidechain-rollup-aggregator/types"
	"github.com/spf13/viper"
)

type BlockProducer struct {
	stateMachine             *statemachine.StateMachine
	pendingBlock             *types.RollupBlock
	txGenerator              *TransactionGenerator
	blockSubmitter           *BlockSubmitter
	tokenAddressToTokenIndex map[string]*big.Int

	numTransitionsInBlock int
}

func NewBlockProducer(mainDbDir string, treeDbDir string) (*BlockProducer, error) {
	mainDb := db.NewDB(db.BadgerImpl, mainDbDir)
	treeDb := db.NewDB(db.BadgerImpl, treeDbDir)
	storage := storage.NewStorage(mainDb)

	// TODO: Sync

	serializer, err := types.NewSerializer()
	if err != nil {
		return nil, err
	}
	numTransitionsInBlock := viper.GetInt("numTransitionsInBlock")

	return &BlockProducer{
		stateMachine:          statemachine.NewStateMachine(storage, treeDb, serializer),
		txGenerator:           NewTransactionGenerator(storage),
		blockSubmitter:        NewBlockSubmitter(serializer),
		numTransitionsInBlock: numTransitionsInBlock,
	}, nil
}

func (bp *BlockProducer) Start() {
	go bp.processTransactions()
	bp.txGenerator.Start()
}

func (bp *BlockProducer) processTransactions() {
	for {
		tx := <-bp.txGenerator.txQueue
		bp.applyTransaction(tx)
	}
}

func (bp *BlockProducer) applyTransaction(signedTransaction *types.SignedTransaction) (*types.SignedStateReceipt, error) {
	stateUpdate, err := bp.stateMachine.ApplyTransaction(signedTransaction)
	if err != nil {
		return nil, err
	}
	err = bp.addToPendingBlock(stateUpdate, signedTransaction)
	if err != nil {
		return nil, err
	}

	if len(bp.pendingBlock.Transitions) > bp.numTransitionsInBlock {
		bp.blockSubmitter.submitBlock(bp.pendingBlock)
	}

	// TODO: Generate receipt and submit block
	return nil, nil
}

func (bp *BlockProducer) addToPendingBlock(stateUpdate *types.StateUpdate, signedTx *types.SignedTransaction) error {
	tx := signedTx.Transaction
	switch tx.GetTransactionType() {
	case types.TransactionTypeDeposit:
		depositTx := tx.(*types.DepositTransaction)
		entry := stateUpdate.Entries[0]
		info := entry.AccountInfo
		if entry.NewAccount {
			bp.pendingBlock.Transitions = append(
				bp.pendingBlock.Transitions,
				&types.InitialDepositTransition{
					StateRoot:        stateUpdate.StateRoot,
					AccountSlotIndex: entry.SlotIndex,
					Account:          info.Account,
					TokenIndex:       bp.tokenAddressToTokenIndex[depositTx.Token.Hex()],
					Amount:           depositTx.Amount,
					Signature:        signedTx.Signature,
				})
		} else {
			bp.pendingBlock.Transitions = append(
				bp.pendingBlock.Transitions,
				&types.DepositTransition{
					StateRoot:        stateUpdate.StateRoot,
					AccountSlotIndex: entry.SlotIndex,
					TokenIndex:       bp.tokenAddressToTokenIndex[depositTx.Token.Hex()],
					Amount:           depositTx.Amount,
					Signature:        signedTx.Signature,
				})

		}

	case types.TransactionTypeWithdraw:
		entry := stateUpdate.Entries[0]
		withdrawTx := tx.(*types.WithdrawTransaction)
		bp.pendingBlock.Transitions = append(
			bp.pendingBlock.Transitions,
			&types.WithdrawTransition{
				StateRoot:        stateUpdate.StateRoot,
				AccountSlotIndex: entry.SlotIndex,
				TokenIndex:       bp.tokenAddressToTokenIndex[withdrawTx.Token.Hex()],
				Amount:           withdrawTx.Amount,
				Signature:        signedTx.Signature,
			})
	case types.TransactionTypeTransfer:
		entries := stateUpdate.Entries
		transferTx := tx.(*types.TransferTransaction)
		bp.pendingBlock.Transitions = append(
			bp.pendingBlock.Transitions,
			&types.TransferTransition{
				StateRoot:          stateUpdate.StateRoot,
				SenderSlotIndex:    entries[0].SlotIndex,
				RecipientSlotIndex: entries[1].SlotIndex,
				TokenIndex:         bp.tokenAddressToTokenIndex[transferTx.Token.Hex()],
				Amount:             transferTx.Amount,
				Nonce:              transferTx.Nonce,
				Signature:          signedTx.Signature,
			})
	}
	return nil
}
