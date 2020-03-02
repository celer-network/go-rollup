package storage

import (
	"github.com/aergoio/aergo-lib/db"
)

var (
	NamespaceTokenAddressToTokenIndex                     = []byte("tokenAddressToTokenIndex")
	NamespaceMainchainTokenAddressToSidechainTokenAddress = []byte("mainchainTokenAddressToSidechainTokenAddress")
)

type Storage struct {
	db db.DB
}

func NewStorage(db db.DB) *Storage {
	return &Storage{
		db: db,
	}
}

func (s *Storage) Exist(namespace []byte, key []byte) bool {
	return s.db.Exist(append(namespace, key...))
}

func (s *Storage) Get(namespace []byte, key []byte) []byte {
	return s.db.Get(append(namespace, key...))
}

func (s *Storage) Set(namespace []byte, key []byte, value []byte) {
	s.db.Set(append(namespace, key...), value)
}
