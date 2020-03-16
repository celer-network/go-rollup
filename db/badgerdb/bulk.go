package badgerdb

import (
	"time"

	"github.com/celer-network/go-rollup/db"

	"github.com/celer-network/go-rollup/log"
	"github.com/dgraph-io/badger/v2"
)

type Bulk struct {
	db        *DB
	bulk      *badger.WriteBatch
	createT   time.Time
	setCount  uint
	delCount  uint
	keySize   uint64
	valueSize uint64
}

func (bulk *Bulk) Set(namespace []byte, key []byte, value []byte) error {
	// TODO Updating trie nodes may require many updates but ErrTxnTooBig is not handled
	key = db.PrependNamespace(namespace, key)
	key = db.ConvNilToBytes(key)
	value = db.ConvNilToBytes(value)

	err := bulk.bulk.Set(key, value)
	if err != nil {
		return err
	}

	bulk.setCount++
	bulk.keySize += uint64(len(key))
	bulk.valueSize += uint64(len(value))
	return err
}

func (bulk *Bulk) Delete(namespace []byte, key []byte) error {
	// TODO Reverting trie may require many updates but ErrTxnTooBig is not handled
	key = db.PrependNamespace(namespace, key)
	key = db.ConvNilToBytes(key)

	err := bulk.bulk.Delete(key)
	if err != nil {
		return err
	}

	bulk.delCount++
	return nil
}

func (bulk *Bulk) Flush() error {
	writeStartT := time.Now()
	err := bulk.bulk.Flush()
	writeEndT := time.Now()

	if writeEndT.Sub(writeStartT) > time.Millisecond*100 || writeEndT.Sub(bulk.createT) > time.Millisecond*500 {
		// write warn log when write bulk tx take too long time (100ms or 500ms total)
		logger.Warn().Str("name", bulk.db.name).Str("callstack1", log.SkipCaller(2)).Str("callstack2", log.SkipCaller(3)).
			Dur("prepareAndCommitTime", writeStartT.Sub(bulk.createT)).
			Uint("delCount", bulk.delCount).Uint("setCount", bulk.setCount).
			Uint64("setKeySize", bulk.keySize).Uint64("setValueSize", bulk.valueSize).
			Dur("flushTime", writeEndT.Sub(writeStartT)).Msg("flush takes long time")
	}

	return err
}

func (bulk *Bulk) DiscardLast() {
	bulk.bulk.Cancel()
}
