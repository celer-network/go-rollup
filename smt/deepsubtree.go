package smt

import (
	"hash"

	rollupdb "github.com/celer-network/go-rollup/db"
)

// DeepSparseMerkleSubTree is a deep Sparse Merkle subtree for working on only a few leafs.
type DeepSparseMerkleSubTree struct {
	*SparseMerkleTree
}

// NewDeepSparseMerkleSubTree creates a new deep Sparse Merkle subtree on an empty DB.
func NewDeepSparseMerkleSubTree(db rollupdb.DB, hasher hash.Hash, height int, hashKey bool) *DeepSparseMerkleSubTree {
	smt := &SparseMerkleTree{
		hasher:  hasher,
		db:      db,
		height:  height,
		hashKey: hashKey,
	}

	return &DeepSparseMerkleSubTree{SparseMerkleTree: smt}
}

// AddBranches adds new branches to the tree.
// These branches are generated by smt.ProveForRoot, and should be verified by VerifyProof first.
// Set updateRoot to true if the current root of the tree should be updated.
func (dsmst *DeepSparseMerkleSubTree) AddBranches(proof [][]byte, key []byte, value []byte, updateRoot bool) ([]byte, error) {
	newRoot, err := dsmst.updateWithSideNodes(dsmst.digest(key), value, reverseProof(proof))
	if err == nil && updateRoot {
		dsmst.SetRoot(newRoot)
	}
	return newRoot, err
}
