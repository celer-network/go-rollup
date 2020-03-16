// Package smt implements a Sparse Merkle tree.
package smt

import (
	"errors"
	"hash"

	rollupdb "github.com/celer-network/go-rollup/db"
)

const left = 0
const right = 1

var (
	defaultValue = make([]byte, 32)
	initMarker   = []byte("init")
)

// SparseMerkleTree is a Sparse Merkle tree.
type SparseMerkleTree struct {
	hasher  hash.Hash
	db      rollupdb.DB
	root    []byte
	height  int
	hashKey bool
}

// NewSparseMerkleTree creates or restores a Sparse Merkle tree with a DB.
func NewSparseMerkleTree(db rollupdb.DB, hasher hash.Hash, root []byte, height int, hashKey bool) (*SparseMerkleTree, error) {
	smt := SparseMerkleTree{
		hasher:  hasher,
		db:      db,
		height:  height,
		hashKey: hashKey,
	}

	hasherSizeBits := hasher.Size() * 8
	_, exists, err := db.Get(rollupdb.NamespaceSMT, initMarker)
	if err != nil {
		return nil, err
	}
	if !exists {
		bulk := db.NewBulk()
		for i := hasherSizeBits - height; i < hasherSizeBits-1; i++ {
			err := bulk.Set(rollupdb.NamespaceSMT, smt.defaultNode(i), append(smt.defaultNode(i+1), smt.defaultNode(i+1)...))
			if err != nil {
				return nil, err
			}
		}
		err := bulk.Set(rollupdb.NamespaceSMT, smt.defaultNode(hasherSizeBits-1), defaultValue)
		if err != nil {
			return nil, err
		}
		err = bulk.Set(rollupdb.NamespaceSMT, initMarker, []byte{})
		err = bulk.Flush()
		if err != nil {
			return nil, err
		}
	}

	if root != nil {
		smt.SetRoot(root)
	} else {
		rootHash := smt.defaultNode(hasherSizeBits - height)
		smt.SetRoot(rootHash)
	}

	return &smt, nil
}

// Root gets the root of the tree.
func (smt *SparseMerkleTree) Root() []byte {
	return smt.root
}

// SetRoot sets the root of the tree.
func (smt *SparseMerkleTree) SetRoot(root []byte) {
	smt.root = root
}

func (smt *SparseMerkleTree) Height() int {
	return smt.height
}

func (smt *SparseMerkleTree) IsHashKey() bool {
	return smt.hashKey
}

func (smt *SparseMerkleTree) keySize() int {
	return smt.hasher.Size()
}

func (smt *SparseMerkleTree) defaultNode(height int) []byte {
	return defaultNodes(smt.hasher)[height]
}

func (smt *SparseMerkleTree) digest(data []byte) []byte {
	smt.hasher.Write(data)
	sum := smt.hasher.Sum(nil)
	smt.hasher.Reset()
	return sum
}

// Get gets a key from the tree.
func (smt *SparseMerkleTree) Get(key []byte) ([]byte, error) {
	value, err := smt.GetForRoot(key, smt.Root())
	return value, err
}

// GetForRoot gets a key from the tree at a specific root.
func (smt *SparseMerkleTree) GetForRoot(key []byte, root []byte) ([]byte, error) {
	path, err := smt.getPath(key)
	if err != nil {
		return nil, err
	}
	currentHash := root
	for i := 0; i < smt.height-1; i++ {
		currentValue, exists, err := smt.db.Get(rollupdb.NamespaceSMT, currentHash)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, errors.New("Corrupt db")
		}
		if !isLeft(path, i, smt.height) {
			currentHash = currentValue[smt.keySize():]
		} else {
			currentHash = currentValue[:smt.keySize()]
		}
	}

	value, exists, err := smt.db.Get(rollupdb.NamespaceSMT, currentHash)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("Corrupt db")
	}

	return value, nil
}

// Update sets a new value for a key in the tree, returns the new root, and sets the new current root of the tree.
func (smt *SparseMerkleTree) Update(key []byte, value []byte) ([]byte, error) {
	newRoot, err := smt.UpdateForRoot(key, value, smt.Root())
	if err == nil {
		smt.SetRoot(newRoot)
	}
	return newRoot, err
}

// UpdateForRoot sets a new value for a key in the tree at a specific root, and returns the new root.
func (smt *SparseMerkleTree) UpdateForRoot(key []byte, value []byte, root []byte) ([]byte, error) {
	path, err := smt.getPath(key)
	if err != nil {
		return nil, err
	}
	sideNodes, err := smt.sideNodesForRoot(path, root)
	if err != nil {
		return nil, err
	}

	newRoot, err := smt.updateWithSideNodes(path, value, sideNodes)
	return newRoot, err
}

func (smt *SparseMerkleTree) updateWithSideNodes(path []byte, value []byte, sideNodes [][]byte) ([]byte, error) {
	bulk := smt.db.NewBulk()
	currentHash := smt.digest(value)
	err := bulk.Set(rollupdb.NamespaceSMT, currentHash, value)
	if err != nil {
		return nil, err
	}
	currentValue := currentHash

	for i := smt.height - 2; i >= 0; i-- {
		sideNode := make([]byte, smt.keySize())
		copy(sideNode, sideNodes[i])
		if !isLeft(path, i, smt.height) {
			currentValue = append(sideNode, currentValue...)
		} else {
			currentValue = append(currentValue, sideNode...)
		}
		currentHash = smt.digest(currentValue)
		err := bulk.Set(rollupdb.NamespaceSMT, currentHash, currentValue)
		if err != nil {
			return nil, err
		}
		currentValue = currentHash
	}
	err = bulk.Flush()
	if err != nil {
		return nil, err
	}

	return currentHash, nil
}

func (smt *SparseMerkleTree) sideNodesForRoot(path []byte, root []byte) ([][]byte, error) {
	currentValue, exists, err := smt.db.Get(rollupdb.NamespaceSMT, root)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("Corrupt db")
	}

	sideNodes := make([][]byte, smt.height-1)
	for i := 0; i < smt.height-1; i++ {
		if !isLeft(path, i, smt.height) {
			sideNodes[i] = currentValue[:smt.keySize()]
			currentValue, exists, err = smt.db.Get(rollupdb.NamespaceSMT, currentValue[smt.keySize():])
			if err != nil {
				return nil, err
			}
			if !exists {
				return nil, errors.New("Corrupt db")
			}
		} else {
			sideNodes[i] = currentValue[smt.keySize():]
			currentValue, exists, err = smt.db.Get(rollupdb.NamespaceSMT, currentValue[:smt.keySize()])
			if err != nil {
				return nil, err
			}
			if !exists {
				return nil, errors.New("Corrupt db")
			}
		}
	}

	return sideNodes, err
}

// Prove generates a Merkle proof for a key.
func (smt *SparseMerkleTree) Prove(key []byte) ([][]byte, error) {
	proof, err := smt.ProveForRoot(key, smt.Root())
	return proof, err
}

// ProveForRoot generates a Merkle proof for a key, at a specific root.
func (smt *SparseMerkleTree) ProveForRoot(key []byte, root []byte) ([][]byte, error) {
	path, err := smt.getPath(key)
	if err != nil {
		return nil, err
	}
	sideNodes, err := smt.sideNodesForRoot(path, root)
	if err != nil {
		return nil, err
	}

	// Reverse to match contract order
	return reverseProof(sideNodes), nil
}

// ProveCompact generates a compacted Merkle proof for a key.
func (smt *SparseMerkleTree) ProveCompact(key []byte) ([][]byte, error) {
	proof, err := smt.Prove(key)
	if err != nil {
		return nil, err
	}
	compactedProof, err := smt.CompactProof(proof)
	return compactedProof, err
}

// ProveCompactForRoot generates a compacted Merkle proof for a key, at a specific root.
func (smt *SparseMerkleTree) ProveCompactForRoot(key []byte, root []byte) ([][]byte, error) {
	proof, err := smt.ProveForRoot(key, root)
	if err != nil {
		return nil, err
	}
	compactedProof, err := smt.CompactProof(proof)
	return compactedProof, err
}

func (smt *SparseMerkleTree) padKey(key []byte) ([]byte, error) {
	keyLength := len(key)
	requiredKeyLength := smt.hasher.Size()
	if keyLength > requiredKeyLength {
		return nil, errors.New("Key too long")
	}
	padded := make([]byte, requiredKeyLength)
	copy(padded[requiredKeyLength-keyLength:], key)
	return padded, nil
}

func (smt *SparseMerkleTree) getPath(key []byte) ([]byte, error) {
	var path []byte
	var err error
	if smt.hashKey {
		path = smt.digest(key)
	} else {
		path, err = smt.padKey(key)
		if err != nil {
			return nil, err
		}
	}
	return path, nil
}
