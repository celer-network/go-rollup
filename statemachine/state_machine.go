package statemachine

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/sha3"

	"github.com/ethereum/go-ethereum/common"

	"github.com/celer-network/go-rollup/db"
	"github.com/celer-network/go-rollup/smt"
	"github.com/celer-network/go-rollup/types"
)

const stateTreeHeight = 160

var (
	errAccountNotFound = errors.New("Account not found")
)

type StateMachine struct {
	db         db.DB
	smt        *smt.SparseMerkleTree
	serializer *types.Serializer
}

func NewStateMachine(db db.DB, serializer *types.Serializer) (*StateMachine, error) {
	// TODO: restore from db

	smt, err := smt.NewSparseMerkleTree(db, sha3.NewLegacyKeccak256(), nil, stateTreeHeight, false)
	if err != nil {
		return nil, err
	}
	return &StateMachine{
		db:         db,
		smt:        smt,
		serializer: serializer,
	}, nil
}

func (sm *StateMachine) ApplyTransaction(signedTx *types.SignedTransaction) (*types.StateUpdate, error) {
	log.Print("Apply transaction")
	tx := signedTx.Transaction
	var accountInfoUpdates []*types.AccountInfoUpdate
	var err error
	switch tx.GetTransactionType() {
	case types.TransactionTypeDeposit:
		accountInfoUpdates, err = sm.applyDeposit(tx.(*types.DepositTransaction))
		if err != nil {
			log.Err(err).Stack().Send()
			return nil, err
		}
	case types.TransactionTypeWithdraw:
		accountInfoUpdates, err = sm.applyWithdraw(tx.(*types.WithdrawTransaction))
		if err != nil {
			log.Err(err).Stack().Send()
			return nil, err
		}
	case types.TransactionTypeTransfer:
		accountInfoUpdates, err = sm.applyTransfer(tx.(*types.TransferTransaction))
		if err != nil {
			log.Err(err).Stack().Send()
			return nil, err
		}
	}
	var entries []*types.StateUpdateEntry
	for _, update := range accountInfoUpdates {
		info := update.Info
		account := info.Account
		newAccount := update.NewAccount
		key, exists, err := sm.db.Get(db.NamespaceAccountAddressToKey, account.Bytes())
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, errAccountNotFound
		}
		proof, proofErr := sm.smt.Prove(key)
		if proofErr != nil {
			log.Err(proofErr).Send()
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
	return &types.StateUpdate{
		Transaction: signedTx,
		StateRoot:   sm.smt.Root(),
		Entries:     entries,
	}, nil

}

func (sm *StateMachine) applyDeposit(tx *types.DepositTransaction) ([]*types.AccountInfoUpdate, error) {
	tokenIndex, err := sm.getTokenIndex(tx.Token)
	if err != nil {
		return nil, err
	}

	// Validations
	amount := tx.Amount
	if amount.Cmp(big.NewInt(0)) == -1 {
		return nil, errors.New("Invalid amount")
	}

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
	// TODO: Check nonce

	// Updates
	balances, nonces := maybeExpandBalancesNonces(tokenIndex, accountInfo)
	balance := balances[tokenIndex]
	balance.Add(balance, amount)
	balances[tokenIndex] = balance
	updatedAccount := &types.AccountInfo{
		Account:  account,
		Balances: balances,
		Nonces:   nonces,
	}
	err = sm.setAccountInfo(account, updatedAccount)
	if err != nil {
		return nil, err
	}
	return []*types.AccountInfoUpdate{
		&types.AccountInfoUpdate{
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
	// TODO: Check nonce
	if int(tokenIndex) > len(accountInfo.Balances)-1 {
		return nil, errors.New("Insufficient balance")
	}
	balance := accountInfo.Balances[tokenIndex]
	if balance.Cmp(amount) == -1 {
		return nil, errors.New("Insufficient balance")
	}

	// Updates
	balance.Sub(balance, amount)
	accountInfo.Balances[tokenIndex] = balance
	updatedAccount := &types.AccountInfo{
		Account:  account,
		Balances: accountInfo.Balances,
		Nonces:   accountInfo.Nonces,
	}
	err = sm.setAccountInfo(account, updatedAccount)
	if err != nil {
		return nil, err
	}
	return []*types.AccountInfoUpdate{
		&types.AccountInfoUpdate{
			Info:       updatedAccount,
			NewAccount: false,
		}}, nil
}

func (sm *StateMachine) applyTransfer(tx *types.TransferTransaction) ([]*types.AccountInfoUpdate, error) {
	tokenIndex, err := sm.getTokenIndex(tx.Token)
	if err != nil {
		return nil, err
	}

	// Validations
	sender := tx.Sender
	recipient := tx.Recipient
	senderAccountInfo, err := sm.getAccountInfo(sender)
	if err != nil {
		return nil, err
	}
	recipientAccountInfo, err := sm.getAccountInfo(recipient)
	if err != nil {
		return nil, err
	}
	amount := tx.Amount
	if amount.Cmp(big.NewInt(0)) == -1 {
		return nil, errors.New("Invalid amount")
	}

	senderBalances := senderAccountInfo.Balances
	senderNonces := senderAccountInfo.Nonces
	tokenIndexInt := int(tokenIndex)
	log.Debug().Int("senderBalancesLength", len(senderBalances)).Interface("senderBalances", senderBalances).Send()
	if tokenIndexInt > len(senderBalances) {
		return nil, errors.New("Sender no such token")
	}

	nonce := senderNonces[tokenIndex]
	requiredNonce := nonce.Add(nonce, big.NewInt(1))
	if requiredNonce.Cmp(tx.Nonce) != 0 {
		err := fmt.Errorf("Invalid nonce, required %d got %d", requiredNonce.Uint64(), tx.Nonce.Uint64())
		return nil, err
	}
	senderBalance := senderBalances[tokenIndex]
	if senderBalance.Cmp(amount) == -1 {
		return nil, errors.New("Insufficient balance")
	}

	// Updates
	recipientBalances, recipientNonces := maybeExpandBalancesNonces(tokenIndex, recipientAccountInfo)
	senderBalance.Sub(senderBalance, amount)
	recipientBalance := recipientBalances[tokenIndex]
	recipientBalance.Add(recipientBalance, amount)

	senderBalances[tokenIndex] = senderBalance
	senderNonces[tokenIndex] = nonce
	recipientBalances[tokenIndex] = recipientBalance
	updatedSender := &types.AccountInfo{
		Account:  sender,
		Balances: senderBalances,
		Nonces:   senderNonces,
	}
	updatedRecipient := &types.AccountInfo{
		Account:  recipient,
		Balances: recipientBalances,
		Nonces:   recipientNonces,
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
		&types.AccountInfoUpdate{
			Info:       updatedSender,
			NewAccount: false,
		},
		&types.AccountInfoUpdate{
			Info:       updatedRecipient,
			NewAccount: false,
		},
	}, nil
}

func (sm *StateMachine) createAccount(address common.Address, numTokens uint64) (*types.AccountInfo, error) {
	balances := make([]*big.Int, numTokens)
	for i := 0; i < len(balances); i++ {
		balances[i] = big.NewInt(0)
	}
	nonces := make([]*big.Int, numTokens)
	for i := 0; i < len(nonces); i++ {
		nonces[i] = big.NewInt(0)
	}
	accountInfo := &types.AccountInfo{
		Account:  address,
		Balances: balances,
		Nonces:   nonces,
	}
	lastKeyBytes, exists, err := sm.db.Get(db.NamespaceLastKey, db.EmptyKey)
	if err != nil {
		return nil, err
	}
	var lastKey *big.Int
	if !exists {
		lastKey = big.NewInt(-1)
	} else {
		lastKey = new(big.Int).SetBytes(lastKeyBytes)
	}
	newKey := lastKey.Add(lastKey, big.NewInt(1))
	newKeyBytes := newKey.Bytes()
	data, err := accountInfo.Serialize(sm.serializer)
	// log.Log().Int("data length", len(data)).Send()
	// log.Log().Bytes("data", data).Err(err).Msg("createAccount")
	tx := sm.db.NewTx()
	err = tx.Set(db.NamespaceLastKey, db.EmptyKey, lastKey.Bytes())
	if err != nil {
		return nil, err
	}
	err = tx.Set(db.NamespaceAccountAddressToKey, address.Bytes(), newKeyBytes)
	if err != nil {
		return nil, err
	}
	err = tx.Set(db.NamespaceKeyToAccountInfo, newKeyBytes, data)
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
	key, exists, err := sm.db.Get(db.NamespaceAccountAddressToKey, address.Bytes())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errAccountNotFound
	}
	data, exists, err := sm.db.Get(db.NamespaceKeyToAccountInfo, key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("Corrupt db")
	}
	log.Log().Bytes("data", data).Msg("getAccountInfo")
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
	key, exists, err := sm.db.Get(db.NamespaceAccountAddressToKey, address.Bytes())
	if err != nil {
		return err
	}
	if !exists {
		return errAccountNotFound
	}
	err = sm.db.Set(db.NamespaceKeyToAccountInfo, key, data)
	if err != nil {
		return err
	}
	_, err = sm.smt.Update(key, data)
	return err
}

func (sm *StateMachine) getTokenIndex(tokenAddress common.Address) (uint64, error) {
	log.Printf("getTokenIndex for %s", tokenAddress.Hex())
	tokenIndexBytes, exists, err := sm.db.Get(
		db.NamespaceTokenAddressToTokenIndex,
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

func maybeExpandBalancesNonces(tokenIndex uint64, accountInfo *types.AccountInfo) ([]*big.Int, []*big.Int) {
	balances := accountInfo.Balances
	nonces := accountInfo.Nonces
	tokenIndexInt := int(tokenIndex)
	oldLength := len(accountInfo.Balances)
	if tokenIndexInt > oldLength-1 {
		for i := oldLength - 1; i < tokenIndexInt; i++ {
			balances = append(balances, big.NewInt(0))
			nonces = append(nonces, big.NewInt(0))
		}
	}
	return balances, nonces
}
