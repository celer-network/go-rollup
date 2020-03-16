package badgerdb

import (
	"context"
	"time"

	rollupdb "github.com/celer-network/go-rollup/db"
	"github.com/celer-network/go-rollup/log"
	"github.com/dgraph-io/badger/v2"
	"github.com/dgraph-io/badger/v2/options"
)

const (
	badgerDbDiscardRatio   = 0.5 // run gc when 50% of samples can be collected
	badgerDbGcInterval     = 10 * time.Minute
	badgerDbGcSize         = 1 << 20 // 1 MB
	badgerValueLogFileSize = 1<<26 - 1
)

var logger *extendedLog

// NewDB creates new database or load existing database in the directory
func NewDB(dir string) (*DB, error) {
	logger = &extendedLog{Logger: log.NewLogger("db")}
	db, err := newBadgerDB(dir)

	if err != nil {
		return nil, err
	}

	return db, nil
}

func (db *DB) runBadgerGC() {
	ticker := time.NewTicker(1 * time.Minute)

	lastGcT := time.Now()
	_, lastDbVlogSize := db.db.Size()
	for {
		select {
		case <-ticker.C:
			// check current db size
			currentDblsmSize, currentDbVlogSize := db.db.Size()

			// exceed badgerDbGcInterval time or badgerDbGcSize is increase slowly (it means resource is free)
			if time.Now().Sub(lastGcT) > badgerDbGcInterval || lastDbVlogSize+badgerDbGcSize > currentDbVlogSize {
				startGcT := time.Now()
				logger.Debug().Str("name", db.name).Int64("lsmSize", currentDblsmSize).Int64("vlogSize", currentDbVlogSize).Msg("Start to GC at badger")
				err := db.db.RunValueLogGC(badgerDbDiscardRatio)
				if err != nil {
					if err == badger.ErrNoRewrite {
						logger.Debug().Str("name", db.name).Str("msg", err.Error()).Msg("Nothing to GC at badger")
					} else {
						logger.Error().Str("name", db.name).Err(err).Msg("Fail to GC at badger")
					}
					lastDbVlogSize = currentDbVlogSize
				} else {
					afterGcDblsmSize, afterGcDbVlogSize := db.db.Size()

					logger.Debug().Str("name", db.name).Int64("lsmSize", afterGcDblsmSize).Int64("vlogSize", afterGcDbVlogSize).
						Dur("takenTime", time.Now().Sub(startGcT)).Msg("Finish to GC at badger")
					lastDbVlogSize = afterGcDbVlogSize
				}
				lastGcT = time.Now()
			}

		case <-db.ctx.Done():
			return
		}
	}
}

// newBadgerDB create a DB instance that uses badger db and implements DB interface.
// An input parameter, dir, is a root directory to store db files.
func newBadgerDB(dir string) (*DB, error) {
	// set option file
	opts := badger.DefaultOptions(dir)

	// TODO : options tuning.
	// Quick fix to prevent RAM usage from going to the roof when adding 10Million new keys during tests
	opts.ValueLogLoadingMode = options.FileIO
	opts.TableLoadingMode = options.FileIO
	opts.ValueThreshold = 1024 // store values, whose size is smaller than 1k, to a lsm tree -> to invoke flushing memtable

	// to reduce size of value log file for low throughput of cloud; 1GB -> 64 MB
	// Time to read or write 1GB file in cloud (normal disk, not high provisioned) takes almost 20 seconds for GC
	opts.ValueLogFileSize = badgerValueLogFileSize

	//opts.MaxTableSize = 1 << 20 // 2 ^ 20 = 1048576, max mempool size invokes updating vlog header for gc

	// set aergo-lib logger instead of default badger stderr logger
	opts.Logger = logger

	// open badger db
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	ctx, cancelFunc := context.WithCancel(context.Background())

	database := &DB{
		db:         db,
		ctx:        ctx,
		cancelFunc: cancelFunc,
		name:       dir,
	}

	go database.runBadgerGC()

	return database, nil
}

// Enforce database and transaction implements interfaces
var _ rollupdb.DB = (*DB)(nil)

type DB struct {
	db         *badger.DB
	ctx        context.Context
	cancelFunc context.CancelFunc
	name       string
}

func (db *DB) Type() string {
	return "badgerdb"
}

func (db *DB) Set(namespace []byte, key []byte, value []byte) error {
	key = rollupdb.PrependNamespace(namespace, key)
	key = rollupdb.ConvNilToBytes(key)
	value = rollupdb.ConvNilToBytes(value)

	err := db.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})

	return err
}

func (db *DB) Delete(namespace []byte, key []byte) error {
	key = rollupdb.PrependNamespace(namespace, key)
	key = rollupdb.ConvNilToBytes(key)

	err := db.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})

	return err
}

func (db *DB) Get(namespace []byte, key []byte) ([]byte, bool, error) {
	key = rollupdb.PrependNamespace(namespace, key)
	key = rollupdb.ConvNilToBytes(key)

	var val []byte
	err := db.db.View(func(txn *badger.Txn) error {

		item, err := txn.Get(key)
		if err != nil {
			return err
		}

		getVal, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}

		val = getVal

		return nil
	})

	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}

	return val, true, nil
}

func (db *DB) Exist(namespace []byte, key []byte) (bool, error) {
	key = rollupdb.PrependNamespace(namespace, key)
	key = rollupdb.ConvNilToBytes(key)

	var isExist bool

	err := db.db.View(func(txn *badger.Txn) error {

		_, err := txn.Get(key)
		if err != nil {
			return err
		}

		isExist = true

		return nil
	})

	if err != nil {
		if err == badger.ErrKeyNotFound {
			return false, nil
		}
		return false, err
	}

	return isExist, nil
}

func (db *DB) Close() error {

	db.cancelFunc() // wait until gc goroutine is finished
	return db.db.Close()
}

func (db *DB) NewTx() rollupdb.Transaction {
	badgerTx := db.db.NewTransaction(true)

	retTransaction := &Transaction{
		db:      db,
		tx:      badgerTx,
		createT: time.Now(),
	}

	return retTransaction
}

func (db *DB) NewBulk() rollupdb.Bulk {
	badgerWriteBatch := db.db.NewWriteBatch()

	retBulk := &Bulk{
		db:      db,
		bulk:    badgerWriteBatch,
		createT: time.Now(),
	}

	return retBulk
}
