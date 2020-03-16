package smt

import (
	"bytes"
	"errors"
	"hash"
)

func (smt *SparseMerkleTree) VerifyProof(proof [][]byte, key []byte, value []byte) bool {
	return VerifyProof(proof, smt.root, key, value, smt.hasher, smt.height)
}

func (smt *SparseMerkleTree) VerifyCompactProof(proof [][]byte, key []byte, value []byte) bool {
	return VerifyCompactProof(proof, smt.root, key, value, smt.hasher, smt.height)
}

func (smt *SparseMerkleTree) CompactProof(proof [][]byte) ([][]byte, error) {
	return CompactProof(proof, smt.hasher, smt.height)
}

func (smt *SparseMerkleTree) DecompactProof(proof [][]byte) ([][]byte, error) {
	return DecompactProof(proof, smt.hasher, smt.height)
}

// VerifyProof verifies a Merkle proof.
func VerifyProof(proof [][]byte, root []byte, key []byte, value []byte, hasher hash.Hash, height int) bool {
	hasher.Write(key)
	path := hasher.Sum(nil)
	hasher.Reset()

	hasher.Write(value)
	currentHash := hasher.Sum(nil)
	hasher.Reset()

	if len(proof) != height-1 {
		return false
	}

	for i := height - 2; i >= 0; i-- {
		node := make([]byte, hasher.Size())
		copy(node, proof[height-2-i])
		if len(node) != hasher.Size() {
			return false
		}
		if !isLeft(path, i, height) {
			hasher.Write(append(node, currentHash...))
			currentHash = hasher.Sum(nil)
			hasher.Reset()
		} else {
			hasher.Write(append(currentHash, node...))
			currentHash = hasher.Sum(nil)
			hasher.Reset()
		}
	}

	return bytes.Compare(currentHash, root) == 0
}

// VerifyCompactProof verifies a compacted Merkle proof.
func VerifyCompactProof(proof [][]byte, root []byte, key []byte, value []byte, hasher hash.Hash, height int) bool {
	decompactedProof, err := DecompactProof(proof, hasher, height)
	if err != nil {
		return false
	}
	return VerifyProof(decompactedProof, root, key, value, hasher, height)
}

// CompactProof compacts a proof, to reduce its size.
func CompactProof(proof [][]byte, hasher hash.Hash, height int) ([][]byte, error) {
	if len(proof) != height-1 {
		return nil, errors.New("bad proof size")
	}

	bits := emptyBytes(hasher.Size())
	var compactProof [][]byte
	for i := 0; i < height-1; i++ {
		node := make([]byte, hasher.Size())
		copy(node, proof[i])
		if bytes.Compare(node, defaultNodes(hasher)[i]) == 0 {
			setBit(bits, i)
		} else {
			compactProof = append(compactProof, node)
		}
	}
	return append([][]byte{bits}, compactProof...), nil
}

// DecompactProof decompacts a proof, so that it can be used for VerifyProof.
func DecompactProof(proof [][]byte, hasher hash.Hash, height int) ([][]byte, error) {
	if len(proof) == 0 ||
		len(proof[0]) != hasher.Size() ||
		len(proof) != (height-1-countSetBits(proof[0]))+1 {
		return nil, errors.New("invalid proof size")
	}

	decompactedProof := make([][]byte, height-1)
	bits := proof[0]
	compactProof := proof[1:]
	position := 0
	for i := 0; i < height-1; i++ {
		if !isLeft(bits, i, height) {
			decompactedProof[i] = defaultNodes(hasher)[i]
		} else {
			decompactedProof[i] = compactProof[position]
			position++
		}
	}
	return decompactedProof, nil
}

func reverseProof(proof [][]byte) [][]byte {
	for i := len(proof)/2 - 1; i >= 0; i-- {
		opp := len(proof) - 1 - i
		proof[i], proof[opp] = proof[opp], proof[i]
	}
	return proof
}
