package db

import (
	"time"

	"github.com/celer-network/go-rollup/log"
	"github.com/dgraph-io/badger/v2"
)

type Transaction struct {
	db        *DB
	tx        *badger.Txn
	createT   time.Time
	setCount  uint
	delCount  uint
	keySize   uint64
	valueSize uint64
}

func (transaction *Transaction) Set(namespace []byte, key []byte, value []byte) error {
	// TODO Updating trie nodes may require many updates but ErrTxnTooBig is not handled
	key = prependNamespace(namespace, key)
	key = convNilToBytes(key)
	value = convNilToBytes(value)

	err := transaction.tx.Set(key, value)
	if err != nil {
		return err
	}

	transaction.setCount++
	transaction.keySize += uint64(len(key))
	transaction.valueSize += uint64(len(value))
	return nil
}

func (transaction *Transaction) Delete(namespace []byte, key []byte) error {
	// TODO Reverting trie may require many updates but ErrTxnTooBig is not handled
	key = prependNamespace(namespace, key)
	key = convNilToBytes(key)

	err := transaction.tx.Delete(key)
	if err != nil {
		return err
	}

	transaction.delCount++
	return nil
}

func (transaction *Transaction) Commit() error {
	writeStartT := time.Now()
	err := transaction.tx.Commit()
	writeEndT := time.Now()

	if writeEndT.Sub(writeStartT) > time.Millisecond*100 {
		// write warn log when write tx take too long time (100ms)
		logger.Warn().Str("name", transaction.db.name).Str("callstack1", log.SkipCaller(2)).Str("callstack2", log.SkipCaller(3)).
			Dur("prepareTime", writeStartT.Sub(transaction.createT)).
			Dur("takenTime", writeEndT.Sub(writeStartT)).
			Uint("delCount", transaction.delCount).Uint("setCount", transaction.setCount).
			Uint64("setKeySize", transaction.keySize).Uint64("setValueSize", transaction.valueSize).
			Msg("commit takes long time")
	}

	return err
}

func (transaction *Transaction) Discard() {
	transaction.tx.Discard()
}
