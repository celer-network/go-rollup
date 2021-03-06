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
	"github.com/celer-network/go-rollup/utils"
	"github.com/celer-network/rollup-contracts/bindings/go/mainchain"
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
	rollupChain     *mainchain.RollupChain
}

func NewValidator(
	db db.DB,
	serializer *types.Serializer,
	stateMachine *statemachine.StateMachine,
	mainchainClient *ethclient.Client,
	mainchainAuth *bind.TransactOpts,
	rollupChain *mainchain.RollupChain,
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
	go v.watchRollupBlockCommitted()
}

func (v *Validator) watchRollupBlockCommitted() error {
	channel := make(chan *mainchain.RollupChainRollupBlockCommitted)
	sub, err := v.rollupChain.WatchRollupBlockCommitted(&bind.WatchOpts{}, channel)
	if err != nil {
		return err
	}
	for {
		select {
		case event := <-channel:
			log.Debug().Msg("Caught RollupBlock")
			rollupBlock, err := v.serializer.DeserializeRollupBlockFromFields(event.BlockNumber.Uint64(), event.Transitions)
			if err != nil {
				log.Err(err).Msg("Failed to deserialize block")
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
	_, serializedBlock, err := block.Serialize(v.serializer)
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
			log.Err(err).Msg("Failed to validate transaction")
		}
		log.Debug().Msg("Validated transaction")
		if fraudProof != nil {
			log.Debug().Msg("Generating and submitting fraud proof")
			contractFraudProof, err := v.generateContractFraudProof(block, fraudProof)
			if err != nil {
				log.Err(err).Send()
				return
			}
			err = v.submitContractFraudProof(contractFraudProof)
			if err != nil {
				log.Err(err).Send()
			}
		}
	}
}

func (v *Validator) validateTransition(
	transitionPosition *types.TransitionPosition, transition types.Transition) (*types.LocalFraudProof, error) {
	snapshots, err := v.getInputStateSnapshots(transition)
	if err != nil {
		return nil, err
	}
	tx, err := v.getTransactionFromTransitionAndSnapshots(transition, snapshots)
	if err != nil {
		return nil, err
	}
	_, err = v.stateMachine.ApplyTransaction(tx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to apply transaction")
		return &types.LocalFraudProof{
			Position:   transitionPosition,
			Inputs:     snapshots,
			Transition: transition,
		}, nil
		// TODO: Differentiate between state transition error and other errors
	}
	localPostRoot := v.stateMachine.GetStateRoot()
	transitionPostRoot := transition.GetStateRoot()
	if !bytes.Equal(localPostRoot, transitionPostRoot[:]) {
		log.Error().
			Str("localPostRoot", common.Bytes2Hex(localPostRoot)).
			Str("transitionPostRoot", common.Bytes2Hex(transitionPostRoot[:])).
			Msg("State root mismatch")
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
	case types.TransitionTypeCreateAndDeposit:
		// No StateSnapshot for newly created account
		return nil, nil
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
	case types.TransitionTypeCreateAndTransfer:
		createAndTransferTransition := transition.(*types.CreateAndTransferTransition)
		log.Debug().Uint64("nonce", createAndTransferTransition.Nonce.Uint64()).Msg("getInputStateSnapshots createAndTransfer")
		senderSnapshot, err := v.stateMachine.GetStateSnapshot(createAndTransferTransition.SenderSlotIndex.Bytes())
		if err != nil {
			return nil, err
		}
		return []*types.StateSnapshot{
			senderSnapshot,
			nil, // No StateSnapshot for newly created account
		}, err
	case types.TransitionTypeTransfer:
		transferTransition := transition.(*types.TransferTransition)
		log.Debug().Uint64("nonce", transferTransition.Nonce.Uint64()).Msg("getInputStateSnapshots transfer")
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
) (types.Transaction, error) {
	var tx types.Transaction
	switch transition.GetTransitionType() {
	case types.TransitionTypeCreateAndDeposit:
		createAndDepositTransition := transition.(*types.CreateAndDepositTransition)
		account := createAndDepositTransition.Account
		tokenBytes, exists, err :=
			v.db.Get(
				db.NamespaceTokenIndexToTokenAddress,
				createAndDepositTransition.TokenIndex.Bytes())
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, errors.New("Invalid token")
		}
		tx = &types.DepositTransaction{
			Account:   account,
			Token:     common.BytesToAddress(tokenBytes),
			Amount:    createAndDepositTransition.Amount,
			Signature: createAndDepositTransition.Signature,
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
			Account:   account,
			Token:     common.BytesToAddress(tokenBytes),
			Amount:    depositTransition.Amount,
			Signature: depositTransition.Signature,
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
			Account:   account,
			Token:     common.BytesToAddress(tokenBytes),
			Amount:    withdrawTransition.Amount,
			Nonce:     withdrawTransition.Nonce,
			Signature: withdrawTransition.Signature,
		}
	case types.TransitionTypeCreateAndTransfer:
		createAndTransferTransition := transition.(*types.CreateAndTransferTransition)
		senderInfo := snapshots[0].AccountInfo
		sender := senderInfo.Account
		recipient := createAndTransferTransition.Recipient
		tokenIndex := createAndTransferTransition.TokenIndex
		nonce := createAndTransferTransition.Nonce
		tokenBytes, exists, err := v.db.Get(db.NamespaceTokenIndexToTokenAddress, tokenIndex.Bytes())
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, errors.New("Invalid token")
		}
		tx = &types.TransferTransaction{
			Sender:    sender,
			Recipient: recipient,
			Token:     common.BytesToAddress(tokenBytes),
			Amount:    createAndTransferTransition.Amount,
			Nonce:     nonce,
			Signature: transition.GetSignature(),
		}
	case types.TransitionTypeTransfer:
		transferTransition := transition.(*types.TransferTransition)
		senderInfo := snapshots[0].AccountInfo
		sender := senderInfo.Account
		recipient := snapshots[1].AccountInfo.Account
		tokenIndex := transferTransition.TokenIndex
		nonce := transferTransition.Nonce
		tokenBytes, exists, err := v.db.Get(db.NamespaceTokenIndexToTokenAddress, tokenIndex.Bytes())
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, errors.New("Invalid token")
		}
		tx = &types.TransferTransaction{
			Sender:    sender,
			Recipient: recipient,
			Token:     common.BytesToAddress(tokenBytes),
			Amount:    transferTransition.Amount,
			Nonce:     nonce,
			Signature: transition.GetSignature(),
		}
	}

	return tx, nil
}

func (v *Validator) generateContractFraudProof(block *types.RollupBlock, localFraudProof *types.LocalFraudProof) (*types.ContractFraudProof, error) {
	fraudInputs := localFraudProof.Inputs
	transitionStorageSlots := make([]mainchain.DataTypesIncludedStorageSlot, len(fraudInputs))
	for i, input := range fraudInputs {
		inputAccountInfo := input.AccountInfo
		storageSlot := mainchain.DataTypesStorageSlot{
			SlotIndex: input.SlotIndex,
			Value: mainchain.DataTypesAccountInfo{
				Account:        inputAccountInfo.Account,
				Balances:       inputAccountInfo.Balances,
				TransferNonces: inputAccountInfo.TransferNonces,
				WithdrawNonces: inputAccountInfo.WithdrawNonces,
			},
		}
		transitionStorageSlots[i] = mainchain.DataTypesIncludedStorageSlot{
			StorageSlot: storageSlot,
			Siblings:    input.InclusionProof,
		}
	}

	blockInfo, err := types.NewRollupBlockInfo(v.serializer, block)
	if err != nil {
		return nil, err
	}
	position := localFraudProof.Position
	blockNumber := position.BlockNumber
	transitionIndex := position.TransitionIndex
	log.Debug().Uint64("blockNumber", position.BlockNumber).Uint64("transitionIndex", position.TransitionIndex).Msg("Fraud position")
	invalidIncludedTransition, err := blockInfo.GetIncludedTransition(int(transitionIndex))
	if err != nil {
		return nil, err
	}
	var preStateIncludedTransition *mainchain.DataTypesIncludedTransition
	if localFraudProof.Position.TransitionIndex == 0 {
		// Fetch previous block
		previousBlockNumber := big.NewInt((int64(blockNumber - 1)))
		previousBlockData, exists, err := v.db.Get(db.NamespaceRollupBlockNumber, previousBlockNumber.Bytes())
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, errors.New("Non-existent block")
		}
		previousBlock, err := v.serializer.DeserializeRollupBlockFromData(previousBlockData)
		if err != nil {
			return nil, err
		}
		previousBlockInfo, err := types.NewRollupBlockInfo(v.serializer, previousBlock)
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
	v.mainchainAuth.GasLimit = 8000000
	v.mainchainAuth.GasPrice = big.NewInt(10e9) // 10 Gwei
	committerAddress, err := v.rollupChain.CommitterAddress(&bind.CallOpts{})
	if err != nil {
		return err
	}
	// Hack for now
	if bytes.Equal(v.mainchainAuth.From.Bytes(), committerAddress.Bytes()) {
		return nil
	}
	// transitionEvaluatorAddress := common.HexToAddress(viper.GetString("transitionEvaluator"))
	// transitionEvaluator, err := mainchain.NewTransitionEvaluator(transitionEvaluatorAddress, v.mainchainClient)
	// log.Debug().Str("invalidIncludedTransition", common.Bytes2Hex(proof.InvalidIncludedTransition.Transition)).Send()
	// stateRoot, slots, err := transitionEvaluator.GetTransitionStateRootAndAccessList(&bind.CallOpts{}, proof.InvalidIncludedTransition.Transition)
	// if err != nil {
	// 	log.Error().Err(err).Msg("Failed to GetTransitionStateRootAndAccessList")
	// }
	// log.Debug().Str("stateRoot", common.Bytes2Hex(stateRoot[:])).Send()
	// log.Debug().Interface("slots", slots).Uint64("storageSlot0", slots[0].Uint64()).Send()
	// tx, err := v.rollupChain.GetStateRootsAndStorageSlots(
	// 	v.mainchainAuth,
	// 	proof.PreStateIncludedTransition.Transition,
	// 	proof.InvalidIncludedTransition.Transition,
	// )
	// storageSlots := make([]mainchain.DataTypesStorageSlot, len(proof.TransitionStorageSlots))
	// for i, includedSlot := range proof.TransitionStorageSlots {
	// 	storageSlots[i] = includedSlot.StorageSlot
	// 	log.Debug().Int("index", i).Uint64("balance0", storageSlots[i].Value.Balances[0].Uint64()).Send()
	// }
	// tx, err := transitionEvaluator.EvaluateTransition(
	// 	v.mainchainAuth,
	// 	proof.InvalidIncludedTransition.Transition,
	// 	storageSlots,
	// )
	// if err != nil {
	// 	log.Error().Err(err).Msg("Failed to evaluate transition")
	// 	return err
	// }

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
		log.Error().Str("tx", tx.Hash().Hex()).Msg("Failed to submit fraud proof")
		return errors.New("Failed to submit fraud proof")
	}
	log.Debug().Str("tx", tx.Hash().Hex()).Msg("Successfully submitted fraud proof")
	log.Debug().Uint64("gasUsed", receipt.GasUsed).Send()
	return nil
}
