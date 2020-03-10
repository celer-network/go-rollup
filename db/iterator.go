package db

import (
	"bytes"
	"errors"

	"github.com/dgraph-io/badger/v2"
)

type Iterator struct {
	start   []byte
	end     []byte
	reverse bool
	iter    *badger.Iterator
}

func (db *DB) Iterator(start, end []byte) *Iterator {
	badgerTx := db.db.NewTransaction(true)

	var reverse bool

	// if end is bigger then start, then reverse order
	if bytes.Compare(start, end) == 1 {
		reverse = true
	} else {
		reverse = false
	}

	opt := badger.DefaultIteratorOptions
	opt.PrefetchValues = false
	opt.Reverse = reverse

	badgerIter := badgerTx.NewIterator(opt)

	badgerIter.Seek(start)

	retIter := &Iterator{
		start:   start,
		end:     end,
		reverse: reverse,
		iter:    badgerIter,
	}
	return retIter
}

func (iter *Iterator) Next() error {
	if iter.Valid() {
		iter.iter.Next()
		return nil
	} else {
		return errors.New("Invalid iterator")
	}
}

func (iter *Iterator) Valid() bool {

	if !iter.iter.Valid() {
		return false
	}

	if iter.end != nil {
		if iter.reverse == false {
			if bytes.Compare(iter.end, iter.iter.Item().Key()) <= 0 {
				return false
			}
		} else {
			if bytes.Compare(iter.iter.Item().Key(), iter.end) <= 0 {
				return false
			}
		}
	}

	return true
}

func (iter *Iterator) Key() (key []byte) {
	return iter.iter.Item().Key()
}

func (iter *Iterator) Value() (value []byte, err error) {
	retVal, err := iter.iter.Item().ValueCopy(nil)

	if err != nil {
		//FIXME: test and handle errs
		return nil, err
	}

	return retVal, nil
}
