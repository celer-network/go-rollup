package blockproducer

import (
	"math/big"

	"github.com/aergoio/aergo-lib/db"
	"github.com/celer-network/sidechain-rollup-aggregator/statemachine"
	"github.com/celer-network/sidechain-rollup-aggregator/types"
)

type BlockProducer struct {
	stateMachine             *statemachine.StateMachine
	pendingBlock             *types.RollupBlock
	txGenerator              *TransactionGenerator
	blockSubmitter           *BlockSubmitter
	tokenAddressToTokenIndex map[string]*big.Int
}

func NewBlockProducer(db db.DB) *BlockProducer {
	// TODO: Sync

	return &BlockProducer{
		stateMachine:   statemachine.NewStateMachine(db),
		txGenerator:    NewTransactionGenerator(),
		blockSubmitter: NewBlockSubmitter(),
	}
}

func (bp *BlockProducer) Start() {
	bp.txGenerator.Start()
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
