package trie

import (
	"bytes"

	"github.com/minio/sha256-simd"
)

var (
	// Trie default value : [byte(0)]
	DefaultLeaf = []byte{0}
)

const (
	HashLength   = 32
	maxPastTries = 300
)

type Hash [HashLength]byte

// Hasher exports default hash function for trie
var Hasher = func(data ...[]byte) []byte {
	hasher := sha256.New()
	for i := 0; i < len(data); i++ {
		hasher.Write(data[i])
	}
	return hasher.Sum(nil)
}

func bitIsSet(bits []byte, i int) bool {
	return bits[i/8]&(1<<uint(7-i%8)) != 0
}
func bitSet(bits []byte, i int) {
	bits[i/8] |= 1 << uint(7-i%8)
}

// for sorting test data
type DataArray [][]byte

func (d DataArray) Len() int {
	return len(d)
}
func (d DataArray) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}
func (d DataArray) Less(i, j int) bool {
	return bytes.Compare(d[i], d[j]) == -1
}
