package smt

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/celer-network/go-rollup/db/memorydb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/minio/sha256-simd"
	"golang.org/x/crypto/sha3"
)

var namespaceTestTrie = []byte("tt")

func TestSMTNumericalKey(t *testing.T) {
	db := memorydb.NewDB()
	smt, err := NewSparseMerkleTree(db, namespaceTestTrie, sha3.NewLegacyKeccak256(), nil, 4, false)
	if err != nil {
		t.Error(err)
	}
	var value []byte
	value, err = smt.Get([]byte("testKey"))
	if err != nil {
		t.Error("returned error when getting empty key")
	}
	if bytes.Compare(DefaultValue, value) != 0 {
		t.Error("did not get default value when getting empty key")
	}

	if bytes.Compare(smt.Root(), common.Hex2Bytes("3b8ec09e026fdc305365dfc94e189a81b38c7597b3d941c279f042e8206e0bd8")) != 0 {
		t.Error("Unexpected empty root")
	}
	newRoot, err := smt.Update(big.NewInt(0).Bytes(), []byte("asdf"))
	if err != nil {
		t.Error(err)
	}
	if bytes.Compare(newRoot, common.Hex2Bytes("33fabbdf8a5902941d46caa04e371bfd745c537230941fdd1e380f8194918a93")) != 0 {
		t.Error("Unexpected updated root")
	}
	newRoot, _ = smt.Update(big.NewInt(1).Bytes(), []byte("asdf"))
	t.Log("newRoot ", common.Bytes2Hex(newRoot))
	// smt.Update(big.NewInt(2).Bytes(), []byte("asdf"))
	// smt.Update(big.NewInt(3).Bytes(), []byte("asdf"))
	t.Log("root ", common.Bytes2Hex(smt.Root()))
	proof, _ := smt.Prove(big.NewInt(0).Bytes())
	for _, node := range proof {
		t.Log(common.Bytes2Hex(node))
	}
}

func TestSparseMerkleTree(t *testing.T) {
	db := memorydb.NewDB()
	smt, err := NewSparseMerkleTree(db, namespaceTestTrie, sha256.New(), nil, 256, true)
	if err != nil {
		t.Error(err)
	}
	var value []byte
	value, err = smt.Get([]byte("testKey"))
	if err != nil {
		t.Error("returned error when getting empty key")
	}
	if bytes.Compare(DefaultValue, value) != 0 {
		t.Error("did not get default value when getting empty key")
	}

	t.Log("empty root", common.Bytes2Hex(smt.Root()))
	newRoot, err := smt.Update(big.NewInt(0).Bytes(), []byte("asdf"))
	if err != nil {
		t.Error(err)
	}
	t.Log("height is now", smt.height)
	t.Log("root is now", common.Bytes2Hex(newRoot))

	_, err = smt.Update([]byte("testKey"), []byte("testValue"))
	if err != nil {
		t.Error("returned error when updating empty key")
	}
	value, err = smt.Get([]byte("testKey"))
	if err != nil {
		t.Error("returned error when getting non-empty key")
	}
	if bytes.Compare([]byte("testValue"), value) != 0 {
		t.Error("did not get correct value when getting non-empty key")
	}

	_, err = smt.Update([]byte("testKey"), []byte("testValue2"))
	if err != nil {
		t.Error("returned error when updating non-empty key")
	}
	value, err = smt.Get([]byte("testKey"))
	if err != nil {
		t.Error("returned error when getting non-empty key")
	}
	if bytes.Compare([]byte("testValue2"), value) != 0 {
		t.Error("did not get correct value when getting non-empty key")
	}

	_, err = smt.Update([]byte("testKey2"), []byte("testValue"))
	if err != nil {
		t.Error("returned error when updating empty second key")
	}
	value, err = smt.Get([]byte("testKey2"))
	if err != nil {
		t.Error("returned error when getting non-empty second key")
	}
	if bytes.Compare([]byte("testValue"), value) != 0 {
		t.Error("did not get correct value when getting non-empty second key")
	}

	value, err = smt.Get([]byte("testKey"))
	if err != nil {
		t.Error("returned error when getting non-empty key")
	}
	if bytes.Compare([]byte("testValue2"), value) != 0 {
		t.Error("did not get correct value when getting non-empty key")
	}

	root := smt.Root()
	smt.Update([]byte("testKey"), []byte("testValue3"))

	value, err = smt.GetForRoot([]byte("testKey"), root)
	if err != nil {
		t.Error("returned error when getting non-empty key")
	}
	if bytes.Compare([]byte("testValue2"), value) != 0 {
		t.Error("did not get correct value when getting non-empty key")
	}

	root, err = smt.UpdateForRoot([]byte("testKey3"), []byte("testValue4"), root)

	value, err = smt.GetForRoot([]byte("testKey3"), root)
	if err != nil {
		t.Error("returned error when getting non-empty key")
	}
	if bytes.Compare([]byte("testValue4"), value) != 0 {
		t.Error("did not get correct value when getting non-empty key")
	}

	value, err = smt.GetForRoot([]byte("testKey"), root)
	if err != nil {
		t.Error("returned error when getting non-empty key")
	}
	if bytes.Compare([]byte("testValue2"), value) != 0 {
		t.Error("did not get correct value when getting non-empty key")
	}

	smt2, err := NewSparseMerkleTree(db, namespaceTestTrie, sha256.New(), smt.Root(), smt.Height(), smt.IsHashKey())
	if err != nil {
		t.Error("error importing smt")
	}

	value, err = smt2.Get([]byte("testKey"))
	if err != nil {
		t.Error("returned error when getting non-empty key")
	}
	if bytes.Compare([]byte("testValue3"), value) != 0 {
		t.Error("did not get correct value when getting non-empty key")
	}
}
