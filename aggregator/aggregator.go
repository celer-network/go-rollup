package aggregator

import (
	"errors"
	"io/ioutil"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog/log"

	"github.com/celer-network/go-rollup/db"
	"github.com/celer-network/go-rollup/statemachine"
	"github.com/celer-network/go-rollup/types"
	"github.com/celer-network/sidechain-contracts/bindings/go/mainchain/rollup"
	"github.com/spf13/viper"
)

type Aggregator struct {
	db                    *db.DB
	stateMachine          *statemachine.StateMachine
	pendingBlock          *types.RollupBlock
	txGenerator           *TransactionGenerator
	blockSubmitter        *BlockSubmitter
	numTransitionsInBlock int
}

func NewAggregator(dbDir string, mainchainKeystore string) (*Aggregator, error) {
	db, err := db.NewDB(dbDir)
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
	}
	key, err := keystore.DecryptKey(mainchainKeystoreBytes, "")
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	mainchainAuth := bind.NewKeyedTransactor(key.PrivateKey)

	mainchainClient, err := ethclient.Dial(viper.GetString("mainchainEndpoint"))
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	rollupChainAddress := viper.GetString("rollupChain")
	rollupChain, err := rollup.NewRollupChain(common.HexToAddress(rollupChainAddress), mainchainClient)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	return &Aggregator{
		db:             db,
		stateMachine:   statemachine.NewStateMachine(db, serializer),
		txGenerator:    NewTransactionGenerator(db, mainchainClient, rollupChain),
		blockSubmitter: NewBlockSubmitter(mainchainClient, mainchainAuth, serializer, rollupChain),

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
