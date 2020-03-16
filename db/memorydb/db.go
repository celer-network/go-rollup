package memorydb

import (
	"container/list"
	"sync"

	rollupdb "github.com/celer-network/go-rollup/db"
)

func NewDB() *DB {
	var db map[string][]byte

	if db == nil {
		db = make(map[string][]byte)
	}

	database := &DB{
		db: db,
	}

	return database
}

// Enforce database and transaction implements interfaces
var _ rollupdb.DB = (*DB)(nil)

type DB struct {
	lock sync.Mutex
	db   map[string][]byte
}

func (db *DB) Type() string {
	return "memorydb"
}

func (db *DB) Set(namespace []byte, key []byte, value []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	key = rollupdb.PrependNamespace(namespace, key)
	key = rollupdb.ConvNilToBytes(key)
	value = rollupdb.ConvNilToBytes(value)

	db.db[string(key)] = value
	return nil
}

func (db *DB) Delete(namespace []byte, key []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	key = rollupdb.PrependNamespace(namespace, key)
	key = rollupdb.ConvNilToBytes(key)

	delete(db.db, string(key))
	return nil
}

func (db *DB) Get(namespace []byte, key []byte) ([]byte, bool, error) {
	db.lock.Lock()
	defer db.lock.Unlock()

	key = rollupdb.PrependNamespace(namespace, key)
	key = rollupdb.ConvNilToBytes(key)

	value, exists := db.db[string(key)]
	return value, exists, nil
}

func (db *DB) Exist(namespace []byte, key []byte) (bool, error) {
	db.lock.Lock()
	defer db.lock.Unlock()

	key = rollupdb.PrependNamespace(namespace, key)
	key = rollupdb.ConvNilToBytes(key)

	_, ok := db.db[string(key)]

	return ok, nil
}

func (db *DB) Close() error {
	return nil
}

func (db *DB) NewTx() rollupdb.Transaction {

	return &Transaction{
		db:        db,
		opList:    list.New(),
		isDiscard: false,
		isCommit:  false,
	}
}

func (db *DB) NewBulk() rollupdb.Bulk {

	return &Bulk{
		db:        db,
		opList:    list.New(),
		isDiscard: false,
		isCommit:  false,
	}
}
