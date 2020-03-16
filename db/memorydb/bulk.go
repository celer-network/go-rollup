package memorydb

import (
	"container/list"
	"errors"
	"sync"

	"github.com/celer-network/go-rollup/db"
)

type Bulk struct {
	txLock    sync.Mutex
	db        *DB
	opList    *list.List
	isDiscard bool
	isCommit  bool
}

func (bulk *Bulk) Set(namespace []byte, key []byte, value []byte) error {
	bulk.txLock.Lock()
	defer bulk.txLock.Unlock()

	key = db.PrependNamespace(namespace, key)
	key = db.ConvNilToBytes(key)
	value = db.ConvNilToBytes(value)

	bulk.opList.PushBack(&txOp{true, key, value})
	return nil
}

func (bulk *Bulk) Delete(namespace []byte, key []byte) error {
	bulk.txLock.Lock()
	defer bulk.txLock.Unlock()

	key = db.PrependNamespace(namespace, key)
	key = db.ConvNilToBytes(key)

	bulk.opList.PushBack(&txOp{false, key, nil})
	return nil
}

func (bulk *Bulk) Flush() error {
	bulk.txLock.Lock()
	defer bulk.txLock.Unlock()

	if bulk.isDiscard {
		return errors.New("Commit after dicard tx is not allowed")
	} else if bulk.isCommit {
		return errors.New("Commit occures two times")
	}

	db := bulk.db

	db.lock.Lock()
	defer db.lock.Unlock()

	for e := bulk.opList.Front(); e != nil; e = e.Next() {
		op := e.Value.(*txOp)
		if op.isSet {
			db.db[string(op.key)] = op.value
		} else {
			delete(db.db, string(op.key))
		}
	}

	bulk.isCommit = true
	return nil
}

func (bulk *Bulk) DiscardLast() {
	bulk.txLock.Lock()
	defer bulk.txLock.Unlock()

	bulk.isDiscard = true
}
