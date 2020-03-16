package memorydb

import (
	"container/list"
	"errors"
	"sync"

	"github.com/celer-network/go-rollup/db"
)

type Transaction struct {
	txLock    sync.Mutex
	db        *DB
	opList    *list.List
	isDiscard bool
	isCommit  bool
}

type txOp struct {
	isSet bool
	key   []byte
	value []byte
}

func (transaction *Transaction) Set(namespace []byte, key []byte, value []byte) error {
	transaction.txLock.Lock()
	defer transaction.txLock.Unlock()

	key = db.PrependNamespace(namespace, key)
	key = db.ConvNilToBytes(key)
	value = db.ConvNilToBytes(value)

	transaction.opList.PushBack(&txOp{true, key, value})
	return nil
}

func (transaction *Transaction) Delete(namespace []byte, key []byte) error {
	transaction.txLock.Lock()
	defer transaction.txLock.Unlock()

	key = db.PrependNamespace(namespace, key)
	key = db.ConvNilToBytes(key)

	transaction.opList.PushBack(&txOp{false, key, nil})
	return nil
}

func (transaction *Transaction) Commit() error {
	transaction.txLock.Lock()
	defer transaction.txLock.Unlock()

	if transaction.isDiscard {
		return errors.New("Commit after dicard tx is not allowed")
	} else if transaction.isCommit {
		return errors.New("Commit occures two times")
	}

	db := transaction.db

	db.lock.Lock()
	defer db.lock.Unlock()

	for e := transaction.opList.Front(); e != nil; e = e.Next() {
		op := e.Value.(*txOp)
		if op.isSet {
			db.db[string(op.key)] = op.value
		} else {
			delete(db.db, string(op.key))
		}
	}

	transaction.isCommit = true
	return nil
}

func (transaction *Transaction) Discard() {
	transaction.txLock.Lock()
	defer transaction.txLock.Unlock()

	transaction.isDiscard = true
}
