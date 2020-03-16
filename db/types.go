package db

// DB is an general interface to access at storage data
type DB interface {
	Type() string
	Set(namespace []byte, key []byte, value []byte) error
	Delete(namespace []byte, key []byte) error
	Get(namespace []byte, key []byte) ([]byte, bool, error)
	Exist(namespace []byte, key []byte) (bool, error)
	Iterator(start []byte, end []byte) Iterator
	NewTx() Transaction
	NewBulk() Bulk
	Close() error
}

// Transaction is used to batch multiple operations
type Transaction interface {
	Set(namespace []byte, key []byte, value []byte) error
	Delete(namespace []byte, key []byte) error
	Commit() error
	Discard()
}

// Bulk is used to batch multiple transactions
// This will internally commit transactions when reach maximum tx size
type Bulk interface {
	Set(namespace []byte, key []byte, value []byte) error
	Delete(namespace []byte, key []byte) error
	Flush() error
	DiscardLast()
}

// Iterator is used to navigate specific key ranges
type Iterator interface {
	Next() error
	Valid() bool
	Key() ([]byte, error)
	Value() ([]byte, error)
}
