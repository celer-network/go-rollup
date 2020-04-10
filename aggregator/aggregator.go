package aggregator

import (
	"errors"
	"io/ioutil"
	"math/big"

	"github.com/celer-network/go-rollup/bridge"

	"github.com/celer-network/go-rollup/db"
	"github.com/celer-network/go-rollup/validator"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog/log"

	rollupdb "github.com/celer-network/go-rollup/db"
	"github.com/celer-network/go-rollup/db/badgerdb"
	"github.com/celer-network/go-rollup/statemachine"
	"github.com/celer-network/go-rollup/types"
	"github.com/celer-network/sidechain-contracts/bindings/go/mainchain/rollup"
	"github.com/spf13/viper"
)

type Aggregator struct {
	db                    rollupdb.DB
	stateMachine          *statemachine.StateMachine
	pendingBlock          *types.RollupBlock
	txGenerator           *TransactionGenerator
	blockSubmitter        *BlockSubmitter
	validator             *validator.Validator
	bridge                *bridge.Bridge
	numTransitionsInBlock int
}

func NewAggregator(dbDir string, mainchainKeystore string, sidechainKeystore string) (*Aggregator, error) {
	db, err := badgerdb.NewDB(dbDir)
	if err != nil {
		return nil, err
	}

	// TODO: Sync

	serializer, err := types.NewSerializer()
	if err != nil {
		return nil, err
	}
	numTransitionsInBlock := viper.GetInt("numTransitionsInBlock")

	mainchainKeystoreBytes, err := ioutil.ReadFile(mainchainKeystore)
	if err != nil {
		log.Fatal().Err(err).Send()
		return nil, err
	}
	mainchainKey, err := keystore.DecryptKey(mainchainKeystoreBytes, "")
	if err != nil {
		log.Fatal().Err(err).Send()
		return nil, err
	}
	mainchainAuth := bind.NewKeyedTransactor(mainchainKey.PrivateKey)

	sidechainKeystoreBytes, err := ioutil.ReadFile(sidechainKeystore)
	if err != nil {
		log.Fatal().Err(err).Send()
		return nil, err
	}
	sidechainKey, err := keystore.DecryptKey(sidechainKeystoreBytes, "")
	if err != nil {
		log.Fatal().Err(err).Send()
		return nil, err
	}
	sidechainAuth := bind.NewKeyedTransactor(sidechainKey.PrivateKey)

	mainchainClient, err := ethclient.Dial(viper.GetString("mainchainEndpoint"))
	if err != nil {
		log.Fatal().Err(err).Send()
		return nil, err
	}

	sidechainClient, err := ethclient.Dial(viper.GetString("sidechainEndpoint"))
	if err != nil {
		log.Fatal().Err(err).Send()
		return nil, err
	}

	rollupChainAddress := viper.GetString("rollupChain")
	rollupChain, err := rollup.NewRollupChain(common.HexToAddress(rollupChainAddress), mainchainClient)
	if err != nil {
		log.Fatal().Err(err).Send()
		return nil, err
	}

	aggregatorStateMachine, err := statemachine.NewStateMachine(db, rollupdb.NamespaceAggregatorTrie, serializer)
	if err != nil {
		log.Fatal().Err(err).Send()
		return nil, err
	}

	transactionGenerator := NewTransactionGenerator(db, mainchainClient, rollupChain)
	blockSubmitter := NewBlockSubmitter(mainchainClient, mainchainAuth, serializer, rollupChain)

	validatorStateMachine, err := statemachine.NewStateMachine(db, rollupdb.NamespaceValidatorTrie, serializer)
	validator := validator.NewValidator(
		db,
		serializer,
		validatorStateMachine,
		mainchainClient,
		mainchainAuth,
		rollupChain,
	)

	bridge, err := bridge.NewBridge(
		mainchainClient,
		sidechainClient,
		sidechainAuth,
		sidechainKey.PrivateKey,
	)

	return &Aggregator{
		db:             db,
		stateMachine:   aggregatorStateMachine,
		txGenerator:    transactionGenerator,
		blockSubmitter: blockSubmitter,
		validator:      validator,
		bridge:         bridge,

		pendingBlock:          types.NewRollupBlock(0),
		numTransitionsInBlock: numTransitionsInBlock,
	}, nil
}

func (a *Aggregator) Start() {
	go a.processTransactions()
	a.txGenerator.Start()
}

func (a *Aggregator) processTransactions() {
	for {
		tx := <-a.txGenerator.txQueue
		a.applyTransaction(tx)
	}
}

func (a *Aggregator) applyTransaction(signedTransaction *types.SignedTransaction) (*types.SignedStateReceipt, error) {
	stateUpdate, err := a.stateMachine.ApplyTransaction(signedTransaction)
	if err != nil {
		return nil, err
	}
	log.Printf("Adding to pending block %d", signedTransaction.Transaction.GetTransactionType())
	err = a.addToPendingBlock(stateUpdate, signedTransaction)
	if err != nil {
		return nil, err
	}

	log.Debug().Int("numPendingTxn", len(a.pendingBlock.Transitions)).Int("numTxnInBlock", a.numTransitionsInBlock).Send()
	if len(a.pendingBlock.Transitions) >= a.numTransitionsInBlock {
		submitErr := a.blockSubmitter.submitBlock(a.pendingBlock)
		if submitErr != nil {
			log.Err(submitErr).Msg("Submit error")
			return nil, submitErr
		}
		a.pendingBlock = types.NewRollupBlock(a.pendingBlock.BlockNumber + 1)
	}

	// TODO: Generate receipt?
	return nil, nil
}

func (a *Aggregator) addToPendingBlock(stateUpdate *types.StateUpdate, signedTx *types.SignedTransaction) error {
	tx := signedTx.Transaction
	switch tx.GetTransactionType() {
	case types.TransactionTypeDeposit:
		depositTx := tx.(*types.DepositTransaction)
		entry := stateUpdate.Entries[0]
		info := entry.AccountInfo
		tokenIndex, exists, err := a.db.Get(db.NamespaceTokenAddressToTokenIndex, depositTx.Token.Bytes())
		if err != nil {
			return err
		}
		if !exists {
			return errors.New("Invalid token")
		}
		if entry.NewAccount {
			a.pendingBlock.Transitions = append(
				a.pendingBlock.Transitions,
				&types.InitialDepositTransition{
					TransitionType:   big.NewInt(int64(types.TransitionTypeInitialDeposit)),
					StateRoot:        stateUpdate.StateRoot,
					AccountSlotIndex: entry.SlotIndex,
					Account:          info.Account,
					TokenIndex:       new(big.Int).SetBytes(tokenIndex),
					Amount:           depositTx.Amount,
					Signature:        signedTx.Signature,
				})
		} else {
			a.pendingBlock.Transitions = append(
				a.pendingBlock.Transitions,
				&types.DepositTransition{
					TransitionType:   big.NewInt(int64(types.TransitionTypeDeposit)),
					StateRoot:        stateUpdate.StateRoot,
					AccountSlotIndex: entry.SlotIndex,
					TokenIndex:       new(big.Int).SetBytes(tokenIndex),
					Amount:           depositTx.Amount,
					Signature:        signedTx.Signature,
				})

		}
	case types.TransactionTypeWithdraw:
		entry := stateUpdate.Entries[0]
		withdrawTx := tx.(*types.WithdrawTransaction)
		tokenIndex, exists, err := a.db.Get(db.NamespaceTokenAddressToTokenIndex, withdrawTx.Token.Bytes())
		if err != nil {
			return err
		}
		if !exists {
			return errors.New("Invalid token")
		}
		a.pendingBlock.Transitions = append(
			a.pendingBlock.Transitions,
			&types.WithdrawTransition{
				TransitionType:   big.NewInt(int64(types.TransitionTypeWithdraw)),
				StateRoot:        stateUpdate.StateRoot,
				AccountSlotIndex: entry.SlotIndex,
				TokenIndex:       new(big.Int).SetBytes(tokenIndex),
				Amount:           withdrawTx.Amount,
				Signature:        signedTx.Signature,
			})
	case types.TransactionTypeTransfer:
		entries := stateUpdate.Entries
		transferTx := tx.(*types.TransferTransaction)
		tokenIndex, exists, err := a.db.Get(db.NamespaceTokenAddressToTokenIndex, transferTx.Token.Bytes())
		if err != nil {
			return err
		}
		if !exists {
			return errors.New("Invalid token")
		}
		a.pendingBlock.Transitions = append(
			a.pendingBlock.Transitions,
			&types.TransferTransition{
				TransitionType:     big.NewInt(int64(types.TransitionTypeTransfer)),
				StateRoot:          stateUpdate.StateRoot,
				SenderSlotIndex:    entries[0].SlotIndex,
				RecipientSlotIndex: entries[1].SlotIndex,
				TokenIndex:         new(big.Int).SetBytes(tokenIndex),
				Amount:             transferTx.Amount,
				Nonce:              transferTx.Nonce,
				Signature:          signedTx.Signature,
			})
	}
	return nil
}
