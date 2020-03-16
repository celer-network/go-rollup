package memorydb

import (
	"bytes"
	"errors"
	"sort"

	"github.com/celer-network/go-rollup/db"
)

type Iterator struct {
	start     []byte
	end       []byte
	reverse   bool
	keys      []string
	isInvalid bool
	cursor    int
	db        *DB
}

func isKeyInRange(key []byte, start []byte, end []byte, reverse bool) bool {
	if reverse {
		if start != nil && bytes.Compare(start, key) < 0 {
			return false
		}
		if end != nil && bytes.Compare(key, end) <= 0 {
			return false
		}
		return true
	}

	if bytes.Compare(key, start) < 0 {
		return false
	}
	if end != nil && bytes.Compare(end, key) <= 0 {
		return false
	}
	return true

}

func (db *DB) Iterator(start []byte, end []byte) db.Iterator {
	db.lock.Lock()
	defer db.lock.Unlock()

	var reverse bool

	// if end is bigger then start, then reverse order
	if bytes.Compare(start, end) == 1 {
		reverse = true
	} else {
		reverse = false
	}

	var keys sort.StringSlice

	for key := range db.db {
		if isKeyInRange([]byte(key), start, end, reverse) {
			keys = append(keys, key)
		}
	}
	if reverse {
		sort.Sort(sort.Reverse(keys))
	} else {
		sort.Strings(keys)
	}

	return &Iterator{
		start:     start,
		end:       end,
		reverse:   reverse,
		isInvalid: false,
		keys:      keys,
		cursor:    0,
		db:        db,
	}
}

func (iter *Iterator) Next() error {
	if !iter.Valid() {
		return errors.New("Iterator is Invalid")
	}

	iter.cursor++
	return nil
}

func (iter *Iterator) Valid() bool {
	// Once invalid, forever invalid.
	if iter.isInvalid {
		return false
	}

	return 0 <= iter.cursor && iter.cursor < len(iter.keys)
}

func (iter *Iterator) Key() ([]byte, error) {
	if !iter.Valid() {
		return nil, errors.New("Iterator is Invalid")
	}

	return []byte(iter.keys[iter.cursor]), nil
}

func (iter *Iterator) Value() ([]byte, error) {
	if !iter.Valid() {
		return nil, errors.New("Iterator is Invalid")
	}

	key := []byte(iter.keys[iter.cursor])

	value, exists, err := iter.db.Get(nil, key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	return value, nil
}
