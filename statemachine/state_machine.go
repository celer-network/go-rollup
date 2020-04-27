package statemachine

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/sha3"

	"github.com/ethereum/go-ethereum/common"

	rollupdb "github.com/celer-network/go-rollup/db"
	"github.com/celer-network/go-rollup/smt"
	"github.com/celer-network/go-rollup/types"
)

const stateTreeHeight = 160

var (
	errAccountNotFound = errors.New("Account not found")
)

type StateMachine struct {
	db         rollupdb.DB
	smt        *smt.SparseMerkleTree
	serializer *types.Serializer
}

func NewStateMachine(db rollupdb.DB, serializer *types.Serializer) (*StateMachine, error) {
	// TODO: restore from db

	smt, err := smt.NewSparseMerkleTree(db, rollupdb.NamespaceStateTrie, sha3.NewLegacyKeccak256(), nil, stateTreeHeight, false)
	if err != nil {
		return nil, err
	}
	return &StateMachine{
		db:         db,
		smt:        smt,
		serializer: serializer,
	}, nil
}

func (sm *StateMachine) ApplyTransaction(tx types.Transaction) (*types.StateUpdate, error) {
	log.Debug().Msg("Apply transaction")
	var accountInfoUpdates []*types.AccountInfoUpdate
	var err error
	switch tx.GetTransactionType() {
	case types.TransactionTypeDeposit:
		accountInfoUpdates, err = sm.applyDeposit(tx.(*types.DepositTransaction))
		if err != nil {
			log.Error().Err(err).Stack().Send()
			return nil, err
		}
	case types.TransactionTypeWithdraw:
		accountInfoUpdates, err = sm.applyWithdraw(tx.(*types.WithdrawTransaction))
		if err != nil {
			log.Error().Err(err).Stack().Send()
			return nil, err
		}
	case types.TransactionTypeTransfer:
		accountInfoUpdates, err = sm.applyTransfer(tx.(*types.TransferTransaction))
		if err != nil {
			log.Error().Err(err).Stack().Send()
			return nil, err
		}
	}
	var entries []*types.StateUpdateEntry
	for _, update := range accountInfoUpdates {
		info := update.Info
		account := info.Account
		newAccount := update.NewAccount
		key, exists, err := sm.db.Get(rollupdb.NamespaceAccountAddressToKey, account.Bytes())
		if err != nil {
			log.Error().Err(err).Send()
			return nil, err
		}
		if !exists {
			log.Error().Err(errAccountNotFound).Send()
			return nil, errAccountNotFound
		}
		proof, proofErr := sm.smt.Prove(key)
		if proofErr != nil {
			log.Error().Err(proofErr).Send()
			return nil, err
		}
		inclusionProof := types.ConvertToInclusionProof(proof)
		entries = append(entries, &types.StateUpdateEntry{
			SlotIndex:      new(big.Int).SetBytes(key),
			InclusionProof: inclusionProof,
			AccountInfo:    info,
			NewAccount:     newAccount,
		})
	}
	var stateRoot [32]byte
	copy(stateRoot[:], sm.smt.Root())
	log.Debug().
		Int("transactionType", int(tx.GetTransactionType())).
		Str("stateRoot", common.Bytes2Hex(stateRoot[:])).Msg("stateroots")
	return &types.StateUpdate{
		Transaction: tx,
		StateRoot:   stateRoot,
		Entries:     entries,
	}, nil
}

func (sm *StateMachine) applyDeposit(tx *types.DepositTransaction) ([]*types.AccountInfoUpdate, error) {
	tokenIndex, err := sm.getTokenIndex(tx.Token)
	if err != nil {
		return nil, err
	}

	// Create account if not existent
	account := tx.Account
	accountInfo, err := sm.getAccountInfo(account)
	newAccount := false
	if err != nil {
		if errors.Is(err, errAccountNotFound) {
			var createErr error
			accountInfo, createErr = sm.createAccount(account, tokenIndex+1)
			if createErr != nil {
				return nil, err
			}
			newAccount = true
		} else {
			return nil, err
		}
	}

	// Validations
	amount := tx.Amount
	if amount.Cmp(big.NewInt(0)) == -1 {
		return nil, errors.New("Invalid amount")
	}

	// TODO: Check validator signature
	// TODO: nonce?

	// Updates
	balances, transferNonces, withdrawNonces := maybeExpandAccountInfo(tokenIndex, accountInfo)
	oldBalance := balances[tokenIndex]
	newBalance := new(big.Int).Add(oldBalance, amount)
	balances[tokenIndex] = newBalance
	updatedAccount := &types.AccountInfo{
		Account:        account,
		Balances:       balances,
		TransferNonces: transferNonces,
		WithdrawNonces: withdrawNonces,
	}
	err = sm.setAccountInfo(account, updatedAccount)
	if err != nil {
		return nil, err
	}
	return []*types.AccountInfoUpdate{
		{
			Info:       updatedAccount,
			NewAccount: newAccount,
		}}, nil
}

func (sm *StateMachine) applyWithdraw(tx *types.WithdrawTransaction) ([]*types.AccountInfoUpdate, error) {
	tokenIndex, err := sm.getTokenIndex(tx.Token)
	if err != nil {
		return nil, err
	}

	// Validations
	account := tx.Account
	accountInfo, err := sm.getAccountInfo(account)
	if err != nil {
		return nil, err
	}
	amount := tx.Amount
	if amount.Cmp(big.NewInt(0)) == -1 {
		return nil, errors.New("Invalid amount")
	}

	if int(tokenIndex) > len(accountInfo.Balances)-1 {
		return nil, errors.New("Insufficient balance")
	}
	oldBalance := accountInfo.Balances[tokenIndex]
	if oldBalance.Cmp(amount) == -1 {
		return nil, errors.New("Insufficient balance")
	}

	withdrawNonces := accountInfo.WithdrawNonces
	oldWithdrawNonce := withdrawNonces[tokenIndex]
	if oldWithdrawNonce.Cmp(tx.Nonce) != 0 {
		err := fmt.Errorf("Invalid nonce, required %d got %d", oldWithdrawNonce.Uint64(), tx.Nonce.Uint64())
		return nil, err
	}
	newWithdrawNonce := new(big.Int).Add(oldWithdrawNonce, big.NewInt(1))
	accountInfo.WithdrawNonces[tokenIndex] = newWithdrawNonce

	// TODO: Check signature

	// Updates
	newBalance := new(big.Int).Sub(oldBalance, amount)
	accountInfo.Balances[tokenIndex] = newBalance
	updatedAccount := &types.AccountInfo{
		Account:        account,
		Balances:       accountInfo.Balances,
		TransferNonces: accountInfo.TransferNonces,
		WithdrawNonces: accountInfo.WithdrawNonces,
	}
	err = sm.setAccountInfo(account, updatedAccount)
	if err != nil {
		return nil, err
	}
	return []*types.AccountInfoUpdate{
		{
			Info:       updatedAccount,
			NewAccount: false,
		}}, nil
}

func (sm *StateMachine) applyTransfer(tx *types.TransferTransaction) ([]*types.AccountInfoUpdate, error) {
	tokenIndex, err := sm.getTokenIndex(tx.Token)
	if err != nil {
		return nil, err
	}

	// Create account for recipient if not existent
	recipient := tx.Recipient
	recipientAccountInfo, err := sm.getAccountInfo(recipient)
	newRecipient := false
	if err != nil {
		if errors.Is(err, errAccountNotFound) {
			var createErr error
			recipientAccountInfo, createErr = sm.createAccount(recipient, tokenIndex+1)
			if createErr != nil {
				return nil, err
			}
			newRecipient = true
		} else {
			return nil, err
		}
	}

	// Validations
	sender := tx.Sender
	senderAccountInfo, err := sm.getAccountInfo(sender)
	if err != nil {
		return nil, err
	}
	amount := tx.Amount
	if amount.Cmp(big.NewInt(0)) == -1 {
		return nil, errors.New("Invalid amount")
	}

	senderBalances := senderAccountInfo.Balances
	senderTransferNonces := senderAccountInfo.TransferNonces
	tokenIndexInt := int(tokenIndex)
	log.Debug().Int("senderBalancesLength", len(senderBalances)).Interface("senderBalances", senderBalances).Send()
	if tokenIndexInt > len(senderBalances) {
		return nil, errors.New("Sender no such token")
	}

	oldTransferNonce := senderTransferNonces[tokenIndex]
	if oldTransferNonce.Cmp(tx.Nonce) != 0 {
		err := fmt.Errorf("Invalid nonce, required %d got %d", oldTransferNonce.Uint64(), tx.Nonce.Uint64())
		return nil, err
	}
	newTransferNonce := new(big.Int).Add(oldTransferNonce, big.NewInt(1))
	oldSenderBalance := senderBalances[tokenIndex]
	if oldSenderBalance.Cmp(amount) == -1 {
		return nil, errors.New("Insufficient balance")
	}

	// TODO: Check signature

	// Updates
	recipientBalances, recipientTransferNonces, recipientWithdrawNonces :=
		maybeExpandAccountInfo(tokenIndex, recipientAccountInfo)
	newSenderBalance := new(big.Int).Sub(oldSenderBalance, amount)
	oldRecipientBalance := recipientBalances[tokenIndex]
	newRecipientBalance := new(big.Int).Add(oldRecipientBalance, amount)

	senderBalances[tokenIndex] = newSenderBalance
	senderTransferNonces[tokenIndex] = newTransferNonce
	recipientBalances[tokenIndex] = newRecipientBalance
	updatedSender := &types.AccountInfo{
		Account:        sender,
		Balances:       senderBalances,
		TransferNonces: senderTransferNonces,
		WithdrawNonces: senderAccountInfo.WithdrawNonces,
	}
	updatedRecipient := &types.AccountInfo{
		Account:        recipient,
		Balances:       recipientBalances,
		TransferNonces: recipientTransferNonces,
		WithdrawNonces: recipientWithdrawNonces,
	}
	err = sm.setAccountInfo(sender, updatedSender)
	if err != nil {
		return nil, err
	}
	err = sm.setAccountInfo(recipient, updatedRecipient)
	if err != nil {
		return nil, err
	}

	return []*types.AccountInfoUpdate{
		{
			Info:       updatedSender,
			NewAccount: false,
		},
		{
			Info:       updatedRecipient,
			NewAccount: newRecipient,
		},
	}, nil
}

func (sm *StateMachine) createAccount(address common.Address, numTokens uint64) (*types.AccountInfo, error) {
	balances := make([]*big.Int, numTokens)
	for i := 0; i < len(balances); i++ {
		balances[i] = big.NewInt(0)
	}
	transferNonces := make([]*big.Int, numTokens)
	for i := 0; i < len(transferNonces); i++ {
		transferNonces[i] = big.NewInt(0)
	}
	withdrawNonces := make([]*big.Int, numTokens)
	for i := 0; i < len(withdrawNonces); i++ {
		withdrawNonces[i] = big.NewInt(0)
	}
	accountInfo := &types.AccountInfo{
		Account:        address,
		Balances:       balances,
		TransferNonces: transferNonces,
		WithdrawNonces: withdrawNonces,
	}
	lastKeyBytes, exists, err := sm.db.Get(rollupdb.NamespaceLastKey, rollupdb.EmptyKey)
	if err != nil {
		return nil, err
	}
	var lastKey *big.Int
	if !exists {
		lastKey = big.NewInt(-1)
	} else {
		lastKey = new(big.Int).SetBytes(lastKeyBytes)
	}
	newKey := new(big.Int).Add(lastKey, big.NewInt(1))
	newKeyBytes := newKey.Bytes()
	data, err := accountInfo.Serialize(sm.serializer)
	// log.Log().Int("data length", len(data)).Send()
	// log.Log().Bytes("data", data).Err(err).Msg("createAccount")
	tx := sm.db.NewTx()
	err = tx.Set(rollupdb.NamespaceLastKey, rollupdb.EmptyKey, newKey.Bytes())
	if err != nil {
		return nil, err
	}
	err = tx.Set(rollupdb.NamespaceAccountAddressToKey, address.Bytes(), newKeyBytes)
	if err != nil {
		return nil, err
	}
	err = tx.Set(rollupdb.NamespaceKeyToAccountInfo, newKeyBytes, data)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	_, err = sm.smt.Update(newKeyBytes, data)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	return accountInfo, nil
}

func (sm *StateMachine) getAccountInfo(address common.Address) (*types.AccountInfo, error) {
	key, exists, err := sm.db.Get(rollupdb.NamespaceAccountAddressToKey, address.Bytes())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errAccountNotFound
	}
	data, exists, err := sm.db.Get(rollupdb.NamespaceKeyToAccountInfo, key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("Corrupt db")
	}
	//log.Log().Str("data", common.Bytes2Hex(data)).Msg("getAccountInfo")
	info, err := sm.serializer.DeserializeAccountInfo(data)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (sm *StateMachine) setAccountInfo(address common.Address, info *types.AccountInfo) error {
	data, err := info.Serialize(sm.serializer)
	if err != nil {
		return err
	}
	key, exists, err := sm.db.Get(rollupdb.NamespaceAccountAddressToKey, address.Bytes())
	if err != nil {
		return err
	}
	if !exists {
		return errAccountNotFound
	}
	err = sm.db.Set(rollupdb.NamespaceKeyToAccountInfo, key, data)
	if err != nil {
		return err
	}
	_, err = sm.smt.Update(key, data)
	return err
}

func (sm *StateMachine) getTokenIndex(tokenAddress common.Address) (uint64, error) {
	log.Printf("getTokenIndex for %s", tokenAddress.Hex())
	tokenIndexBytes, exists, err := sm.db.Get(
		rollupdb.NamespaceTokenAddressToTokenIndex,
		tokenAddress.Bytes(),
	)
	if err != nil {
		return 0, err
	}
	if !exists {
		return 0, errors.New("Token not mapped")
	}

	return new(big.Int).SetBytes(tokenIndexBytes).Uint64(), nil
}

func (sm *StateMachine) GetStateSnapshot(key []byte) (*types.StateSnapshot, error) {
	infoData, err := sm.smt.Get(key)
	if err != nil {
		return nil, err
	}
	proof, err := sm.smt.Prove(key)
	if err != nil {
		return nil, err
	}
	info, err := sm.serializer.DeserializeAccountInfo(infoData)
	if err != nil {
		return nil, err
	}
	inclusionProof := types.ConvertToInclusionProof(proof)
	return &types.StateSnapshot{
		AccountInfo:    info,
		SlotIndex:      new(big.Int).SetBytes(key),
		StateRoot:      sm.smt.Root(),
		InclusionProof: inclusionProof,
	}, nil
}

func (sm *StateMachine) GetStateRoot() []byte {
	return sm.smt.Root()
}

func maybeExpandAccountInfo(
	tokenIndex uint64,
	accountInfo *types.AccountInfo) ([]*big.Int, []*big.Int, []*big.Int) {
	balances := accountInfo.Balances
	transferNonces := accountInfo.TransferNonces
	withdrawNonces := accountInfo.WithdrawNonces
	tokenIndexInt := int(tokenIndex)
	oldLength := len(accountInfo.Balances)
	if tokenIndexInt > oldLength-1 {
		for i := oldLength - 1; i < tokenIndexInt; i++ {
			balances = append(balances, big.NewInt(0))
			transferNonces = append(transferNonces, big.NewInt(0))
			withdrawNonces = append(withdrawNonces, big.NewInt(0))
		}
	}
	return balances, transferNonces, withdrawNonces
}
