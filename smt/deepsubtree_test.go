package smt

import (
	"bytes"
	"testing"

	"github.com/celer-network/go-rollup/db/memorydb"
	"github.com/minio/sha256-simd"
)

func TestDeepSparseMerkleSubTree(t *testing.T) {
	dsmstDb := memorydb.NewDB()
	dsmst := NewDeepSparseMerkleSubTree(dsmstDb, sha256.New(), 256, true)
	smtDb := memorydb.NewDB()
	smt, err := NewSparseMerkleTree(smtDb, namespaceTestTrie, sha256.New(), nil, 256, true)
	if err != nil {
		t.Error(err)
	}

	smt.Update([]byte("testKey1"), []byte("testValue1"))
	smt.Update([]byte("testKey2"), []byte("testValue2"))
	smt.Update([]byte("testKey3"), []byte("testValue3"))
	smt.Update([]byte("testKey4"), []byte("testValue4"))

	proof1, _ := smt.Prove([]byte("testKey1"))
	proof2, _ := smt.Prove([]byte("testKey2"))
	dsmst.AddBranches(proof1, []byte("testKey1"), []byte("testValue1"), true)
	dsmst.AddBranches(proof2, []byte("testKey2"), []byte("testValue2"), true)

	value, err := dsmst.Get([]byte("testKey1"))
	if err != nil {
		t.Error("returned error when getting value in deep subtree")
	}
	if bytes.Compare(value, []byte("testValue1")) != 0 {
		t.Error("did not get correct value in deep subtree")
	}
	value, err = dsmst.Get([]byte("testKey2"))
	if err != nil {
		t.Error("returned error when getting value in deep subtree")
	}
	if bytes.Compare(value, []byte("testValue2")) != 0 {
		t.Error("did not get correct value in deep subtree")
	}

	dsmst.Update([]byte("testKey1"), []byte("testValue3"))
	dsmst.Update([]byte("testKey2"), []byte("testValue4"))

	value, err = dsmst.Get([]byte("testKey1"))
	if err != nil {
		t.Error("returned error when getting value in deep subtree")
	}
	if bytes.Compare(value, []byte("testValue3")) != 0 {
		t.Error("did not get correct value in deep subtree")
	}
	value, err = dsmst.Get([]byte("testKey2"))
	if err != nil {
		t.Error("returned error when getting value in deep subtree")
	}
	if bytes.Compare(value, []byte("testValue4")) != 0 {
		t.Error("did not get correct value in deep subtree")
	}

	smt.Update([]byte("testKey1"), []byte("testValue3"))
	smt.Update([]byte("testKey2"), []byte("testValue4"))

	if bytes.Compare(smt.Root(), dsmst.Root()) != 0 {
		t.Error("roots of identical standard tree and subtree do not match")
	}
}
