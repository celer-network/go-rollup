package validator

import (
	"bytes"
	"context"
	"errors"
	"math/big"

	"github.com/rs/zerolog/log"

	"github.com/celer-network/go-rollup/db"
	"github.com/celer-network/go-rollup/statemachine"
	"github.com/celer-network/go-rollup/types"
	"github.com/celer-network/goCeler/utils"
	"github.com/celer-network/sidechain-contracts/bindings/go/mainchain/rollup"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Validator struct {
	db              db.DB
	serializer      *types.Serializer
	stateMachine    *statemachine.StateMachine
	mainchainClient *ethclient.Client
	mainchainAuth   *bind.TransactOpts
	rollupChain     *rollup.RollupChain
}

func NewValidator(
	db db.DB,
	serializer *types.Serializer,
	stateMachine *statemachine.StateMachine,
	mainchainClient *ethclient.Client,
	mainchainAuth *bind.TransactOpts,
	rollupChain *rollup.RollupChain,
) *Validator {
	return &Validator{
		db:              db,
		serializer:      serializer,
		stateMachine:    stateMachine,
		mainchainClient: mainchainClient,
		mainchainAuth:   mainchainAuth,
		rollupChain:     rollupChain,
	}
}

func (v *Validator) Start() {
	go v.watchNewRollupBlock()
}

func (v *Validator) watchNewRollupBlock() error {
	channel := make(chan *rollup.RollupChainNewRollupBlock)
	sub, err := v.rollupChain.WatchNewRollupBlock(&bind.WatchOpts{}, channel)
	if err != nil {
		return err
	}
	for {
		select {
		case event := <-channel:
			rollupBlock, err := v.serializer.DeserializeRollupBlock(event.Block, event.BlockNumber.Uint64())
			if err != nil {
				return err
			}
			v.validateBlock(rollupBlock)
		case err := <-sub.Err():
			return err
		}
	}
}

func (v *Validator) validateBlock(block *types.RollupBlock) {
	// TODO: Validate block number
	serializedBlock, err := block.SerializeForStorage()
	if err != nil {
		log.Err(err).Send()
	}
	err = v.db.Set(db.NamespaceRollupBlockNumber, big.NewInt(int64(block.BlockNumber)).Bytes(), serializedBlock)
	if err != nil {
		log.Err(err).Send()
	}

	for i, transition := range block.Transitions {
		transitionPosition := &types.TransitionPosition{
			BlockNumber:     block.BlockNumber,
			TransitionIndex: uint64(i),
		}
		fraudProof, err := v.validateTransition(transitionPosition, transition)
		if err != nil {
			log.Err(err).Send()
		}
		if fraudProof != nil {
			contractFraudProof, err := v.generateContractFraudProof(block, fraudProof)
			if err != nil {
				log.Err(err).Send()
			}
			err = v.submitContractFraudProof(contractFraudProof)
			if err != nil {
				log.Err(err).Send()
			}
		}
	}

}

func (v *Validator) validateTransition(transitionPosition *types.TransitionPosition, transition types.Transition) (*types.LocalFraudProof, error) {
	snapshots, err := v.getInputStateSnapshots(transition)
	if err != nil {
		return nil, err
	}
	signedTx, err := v.getTransactionFromTransitionAndSnapshots(transition, snapshots)
	if err != nil {
		return nil, err
	}
	_, err = v.stateMachine.ApplyTransaction(signedTx)
	if err != nil {
		return &types.LocalFraudProof{
			Position:   transitionPosition,
			Inputs:     snapshots,
			Transition: transition,
		}, nil
		// TODO: Differentiate between state transition error and other errors
	}
	localPostRoot := v.stateMachine.GetStateRoot()
	transitionPostRoot := transition.GetStateRoot()
	if !bytes.Equal(localPostRoot, transitionPostRoot) {
		return &types.LocalFraudProof{
			Position:   transitionPosition,
			Inputs:     snapshots,
			Transition: transition,
		}, nil

	}
	return nil, nil
}

func (v *Validator) getInputStateSnapshots(transition types.Transition) ([]*types.StateSnapshot, error) {
	switch transition.GetTransitionType() {
	case types.TransitionTypeInitialDeposit:
		initialDepositTransition := transition.(*types.InitialDepositTransition)
		snapshot, err := v.stateMachine.GetStateSnapshot(initialDepositTransition.AccountSlotIndex.Bytes())
		if err != nil {
			return nil, err
		}
		return []*types.StateSnapshot{
			snapshot,
		}, nil
	case types.TransitionTypeDeposit:
		depositTransition := transition.(*types.DepositTransition)
		snapshot, err := v.stateMachine.GetStateSnapshot(depositTransition.AccountSlotIndex.Bytes())
		if err != nil {
			return nil, err
		}
		return []*types.StateSnapshot{
			snapshot,
		}, nil
	case types.TransitionTypeWithdraw:
		withdrawTransition := transition.(*types.WithdrawTransition)
		snapshot, err := v.stateMachine.GetStateSnapshot(withdrawTransition.AccountSlotIndex.Bytes())
		if err != nil {
			return nil, err
		}
		return []*types.StateSnapshot{
			snapshot,
		}, nil
	case types.TransitionTypeTransfer:
		transferTransition := transition.(*types.TransferTransition)
		senderSnapshot, err := v.stateMachine.GetStateSnapshot(transferTransition.SenderSlotIndex.Bytes())
		if err != nil {
			return nil, err
		}
		recipientSnapshot, err := v.stateMachine.GetStateSnapshot(transferTransition.RecipientSlotIndex.Bytes())
		if err != nil {
			return nil, err
		}
		return []*types.StateSnapshot{
			senderSnapshot,
			recipientSnapshot,
		}, nil
	}
	return nil, errors.New("Invalid transition type")
}

func (v *Validator) getTransactionFromTransitionAndSnapshots(
	transition types.Transition,
	snapshots []*types.StateSnapshot,
) (*types.SignedTransaction, error) {
	var tx types.Transaction
	switch transition.GetTransitionType() {
	case types.TransitionTypeInitialDeposit:
		initialDepositTransition := transition.(*types.InitialDepositTransition)
		account := snapshots[0].AccountInfo.Account
		tokenBytes, exists, err := v.db.Get(db.NamespaceTokenIndexToTokenAddress, initialDepositTransition.TokenIndex.Bytes())
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, errors.New("Invalid token")
		}
		tx = &types.DepositTransaction{
			Account: account,
			Token:   common.BytesToAddress(tokenBytes),
			Amount:  initialDepositTransition.Amount,
		}
	case types.TransitionTypeDeposit:
		depositTransition := transition.(*types.DepositTransition)
		account := snapshots[0].AccountInfo.Account
		tokenBytes, exists, err := v.db.Get(db.NamespaceTokenIndexToTokenAddress, depositTransition.TokenIndex.Bytes())
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, errors.New("Invalid token")
		}
		tx = &types.DepositTransaction{
			Account: account,
			Token:   common.BytesToAddress(tokenBytes),
			Amount:  depositTransition.Amount,
		}
	case types.TransitionTypeWithdraw:
		withdrawTransition := transition.(*types.WithdrawTransition)
		account := snapshots[0].AccountInfo.Account
		tokenBytes, exists, err := v.db.Get(db.NamespaceTokenIndexToTokenAddress, withdrawTransition.TokenIndex.Bytes())
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, errors.New("Invalid token")
		}
		tx = &types.WithdrawTransaction{
			Account: account,
			Token:   common.BytesToAddress(tokenBytes),
			Amount:  withdrawTransition.Amount,
		}
	case types.TransitionTypeTransfer:
		withdrawTransition := transition.(*types.WithdrawTransition)
		account := snapshots[0].AccountInfo.Account
		tokenBytes, exists, err := v.db.Get(db.NamespaceTokenIndexToTokenAddress, withdrawTransition.TokenIndex.Bytes())
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, errors.New("Invalid token")
		}
		tx = &types.WithdrawTransaction{
			Account: account,
			Token:   common.BytesToAddress(tokenBytes),
			Amount:  withdrawTransition.Amount,
		}
	}

	return &types.SignedTransaction{
		Signature:   transition.GetSignature(),
		Transaction: tx,
	}, nil
}

func (v *Validator) generateContractFraudProof(block *types.RollupBlock, localFraudProof *types.LocalFraudProof) (*types.ContractFraudProof, error) {
	fraudInputs := localFraudProof.Inputs
	transitionStorageSlots := make([]rollup.DataTypesIncludedStorageSlot, len(fraudInputs))
	for i, input := range fraudInputs {
		inputAccountInfo := input.AccountInfo
		storageSlot := rollup.DataTypesStorageSlot{
			SlotIndex: input.SlotIndex,
			Value: rollup.DataTypesAccountInfo{
				Account:  inputAccountInfo.Account,
				Balances: inputAccountInfo.Balances,
				Nonces:   inputAccountInfo.Nonces,
			},
		}
		transitionStorageSlots[i] = rollup.DataTypesIncludedStorageSlot{
			StorageSlot: storageSlot,
			Siblings:    input.InclusionProof,
		}
	}

	blockInfo, err := NewRollupBlockInfo(v.serializer, block)
	if err != nil {
		return nil, err
	}
	position := localFraudProof.Position
	blockNumber := position.BlockNumber
	transitionIndex := position.TransitionIndex
	invalidIncludedTransition, err := blockInfo.GetIncludedTransition(int(transitionIndex))
	if err != nil {
		return nil, err
	}
	var preStateIncludedTransition *rollup.DataTypesIncludedTransition
	if localFraudProof.Position.TransitionIndex == 0 {
		// Fetch previous block
		previousBlockNumber := big.NewInt((int64(blockNumber - 1)))
		previousBlockData, exists, err := v.db.Get(db.NamespaceRollupBlockNumber, previousBlockNumber.Bytes())
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, err
		}
		previousBlock, err := types.DeserializeRollupBlockFromStorage(previousBlockData)
		if err != nil {
			return nil, err
		}
		previousBlockInfo, err := NewRollupBlockInfo(v.serializer, previousBlock)
		if err != nil {
			return nil, err
		}
		preStateIncludedTransition, err = previousBlockInfo.GetIncludedTransition(previousBlockInfo.GetNumTransitions() - 1)
		if err != nil {
			return nil, err
		}
	} else {
		preStateIncludedTransition, err = blockInfo.GetIncludedTransition(int(transitionIndex - 1))
		if err != nil {
			return nil, err
		}
	}
	return &types.ContractFraudProof{
		InvalidIncludedTransition:  *invalidIncludedTransition,
		PreStateIncludedTransition: *preStateIncludedTransition,
		TransitionStorageSlots:     transitionStorageSlots,
	}, nil
}

func (v *Validator) submitContractFraudProof(proof *types.ContractFraudProof) error {
	tx, err := v.rollupChain.ProveTransitionInvalid(
		v.mainchainAuth,
		proof.PreStateIncludedTransition,
		proof.InvalidIncludedTransition,
		proof.TransitionStorageSlots,
	)
	if err != nil {
		return err
	}
	receipt, err := utils.WaitMined(context.Background(), v.mainchainClient, tx, 0)
	if err != nil {
		return err
	}
	if receipt.Status != 1 {
		return errors.New("Failed to submit fraud proof")
	}
	return nil
}
