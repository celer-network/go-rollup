package statemachine

import (
	"errors"
	"math/big"

	"github.com/rs/zerolog/log"

	"github.com/celer-network/sidechain-rollup-aggregator/serialization"
	"github.com/ethereum/go-ethereum/common"

	smt "github.com/aergoio/SMT"
	"github.com/aergoio/aergo-lib/db"
	"github.com/celer-network/sidechain-rollup-aggregator/types"
)

var errAccountNotFound = errors.New("Account not found")

type StateMachine struct {
	db                       db.DB
	tree                     *smt.SMT
	addressToKey             map[string][]byte
	lastKey                  *big.Int
	tokenAddressToTokenIndex map[string]*big.Int
}

func NewStateMachine(db db.DB) *StateMachine {
	// TODO: restore from db

	return &StateMachine{
		db:      db,
		tree:    smt.NewSMT(nil, smt.Hasher, db),
		lastKey: big.NewInt(-1),
	}
}

func (sm *StateMachine) ApplyTransaction(signedTx *types.SignedTransaction) (*types.StateUpdate, error) {
	log.Print("Validating address")
	tx := signedTx.Transaction
	var accountInfoUpdates []*types.AccountInfoUpdate
	var err error
	switch tx.GetTransactionType() {
	case types.TransactionTypeDeposit:
		accountInfoUpdates, err = sm.applyDeposit(tx.(*types.DepositTransaction))
		if err != nil {
			return nil, err
		}
	case types.TransactionTypeWithdraw:
		accountInfoUpdates, err = sm.applyWithdraw(tx.(*types.WithdrawTransaction))
		if err != nil {
			return nil, err
		}
	case types.TransactionTypeTransfer:
		accountInfoUpdates, err = sm.applyTransfer(tx.(*types.TransferTransaction))
		if err != nil {
			return nil, err
		}
	}
	var entries []*types.StateUpdateEntry
	for _, update := range accountInfoUpdates {
		info := update.Info
		newAccount := update.NewAccount
		key := sm.addressToKey[info.Account.Hex()]
		proof, proofErr := sm.tree.MerkleProof(key)
		if proofErr != nil {
			return nil, err
		}
		entries = append(entries, &types.StateUpdateEntry{
			SlotIndex:      new(big.Int).SetBytes(sm.addressToKey[info.Account.Hex()]),
			InclusionProof: proof,
			AccountInfo:    info,
			NewAccount:     newAccount,
		})
	}
	return &types.StateUpdate{
		Transaction: signedTx,
		StateRoot:   sm.tree.Root,
		Entries:     entries,
	}, nil

}

func (sm *StateMachine) applyDeposit(tx *types.DepositTransaction) ([]*types.AccountInfoUpdate, error) {
	tokenIndexBigInt, exists := sm.tokenAddressToTokenIndex[tx.Token.Hex()]
	if !exists {
		return nil, errors.New("Token not mapped")
	}
	tokenIndex := tokenIndexBigInt.Uint64()

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
			accountInfo = &types.AccountInfo{
				Account:  account,
				Balances: make([]*big.Int, tokenIndex+1),
				Nonces:   make([]*big.Int, tokenIndex+1),
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
	return []*types.AccountInfoUpdate{
		&types.AccountInfoUpdate{
			Info:       updatedAccount,
			NewAccount: newAccount,
		}}, nil
}

func (sm *StateMachine) applyWithdraw(tx *types.WithdrawTransaction) ([]*types.AccountInfoUpdate, error) {
	tokenIndexBigInt, exists := sm.tokenAddressToTokenIndex[tx.Token.Hex()]
	if !exists {
		return nil, errors.New("Token not mapped")
	}
	tokenIndex := tokenIndexBigInt.Uint64()

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
	return []*types.AccountInfoUpdate{
		&types.AccountInfoUpdate{
			Info:       updatedAccount,
			NewAccount: false,
		}}, nil
}

func (sm *StateMachine) applyTransfer(tx *types.TransferTransaction) ([]*types.AccountInfoUpdate, error) {
	tokenIndexBigInt, exists := sm.tokenAddressToTokenIndex[tx.Token.Hex()]
	if !exists {
		return nil, errors.New("Token not mapped")
	}
	tokenIndex := tokenIndexBigInt.Uint64()

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
	if tokenIndexInt > len(senderBalances) {
		return nil, errors.New("Insufficient balance")
	}

	nonce := senderNonces[tokenIndex]
	if nonce.Add(nonce, big.NewInt(1)).Cmp(tx.Nonce) != 0 {
		return nil, errors.New("Invalid nonce")
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
	err = sm.setAccountInfo(sender, updatedSender)
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

func (sm *StateMachine) createAccountKey(address common.Address) []byte {
	newKey := sm.lastKey.Add(sm.lastKey, big.NewInt(1))
	newKeyBytes := newKey.Bytes()
	sm.addressToKey[address.Hex()] = newKeyBytes
	return newKeyBytes
}

func (sm *StateMachine) getAccountInfo(address common.Address) (*types.AccountInfo, error) {
	key, exists := sm.addressToKey[address.Hex()]
	if !exists {
		return nil, errors.New("Account not found")
	}
	data, err := sm.tree.Get(key)
	if err != nil {
		return nil, err
	}
	info, err := serialization.DeserializeAccountInfo(data)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (sm *StateMachine) setAccountInfo(address common.Address, info *types.AccountInfo) error {
	data, err := serialization.SerializeAccountInfo(info)
	if err != nil {
		return err
	}
	key := sm.addressToKey[address.Hex()]
	_, err = sm.tree.Update([][]byte{key}, [][]byte{data})
	if err != nil {
		return err
	}
	return sm.tree.Commit()
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
