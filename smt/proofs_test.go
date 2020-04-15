package smt

import (
	"crypto/rand"
	"reflect"
	"testing"

	"github.com/celer-network/go-rollup/db/memorydb"
	"github.com/minio/sha256-simd"
)

func TestProofs(t *testing.T) {
	db := memorydb.NewDB()
	smt, err := NewSparseMerkleTree(db, namespaceTestTrie, sha256.New(), nil, 256, true)
	if err != nil {
		t.Error(err)
	}

	badProof := make([][]byte, sha256.New().Size()*8)
	for i := 0; i < len(badProof); i++ {
		badProof[i] = make([]byte, sha256.New().Size())
		rand.Read(badProof[i])
	}

	smt.Update([]byte("testKey"), []byte("testValue"))

	proof, err := smt.Prove([]byte("testKey"))
	if err != nil {
		t.Error("error returned when trying to prove inclusion")
	}
	result := smt.VerifyProof(proof, []byte("testKey"), []byte("testValue"))
	if !result {
		t.Error("valid proof failed to verify")
	}
	result = smt.VerifyProof(proof, []byte("testKey"), []byte("badValue"))
	if result {
		t.Error("invalid proof verification returned true")
	}
	result = smt.VerifyProof(proof, []byte("testKey1"), []byte("testValue"))
	if result {
		t.Error("invalid proof verification returned true")
	}
	result = smt.VerifyProof(badProof, []byte("testKey"), []byte("testValue"))
	if result {
		t.Error("invalid proof verification returned true")
	}

	smt.Update([]byte("testKey2"), []byte("testValue"))

	proof, err = smt.Prove([]byte("testKey"))
	if err != nil {
		t.Error("error returned when trying to prove inclusion")
	}
	result = smt.VerifyProof(proof, []byte("testKey"), []byte("testValue"))
	if !result {
		t.Error("valid proof failed to verify")
	}
	result = smt.VerifyProof(proof, []byte("testKey"), []byte("badValue"))
	if result {
		t.Error("invalid proof verification returned true")
	}
	result = smt.VerifyProof(proof, []byte("testKey2"), []byte("testValue"))
	if result {
		t.Error("invalid proof verification returned true")
	}
	result = smt.VerifyProof(badProof, []byte("testKey"), []byte("testValue"))
	if result {
		t.Error("invalid proof verification returned true")
	}

	proof, err = smt.Prove([]byte("testKey2"))
	if err != nil {
		t.Error("error returned when trying to prove inclusion")
		t.Log(err)
	}
	result = smt.VerifyProof(proof, []byte("testKey2"), []byte("testValue"))
	if !result {
		t.Error("valid proof failed to verify")
	}
	result = smt.VerifyProof(proof, []byte("testKey2"), []byte("badValue"))
	if result {
		t.Error("invalid proof verification returned true")
	}
	result = smt.VerifyProof(proof, []byte("testKey3"), []byte("testValue"))
	if result {
		t.Error("invalid proof verification returned true")
	}
	result = smt.VerifyProof(badProof, []byte("testKey"), []byte("testValue"))
	if result {
		t.Error("invalid proof verification returned true")
	}

	proof, err = smt.Prove([]byte("testKey3"))
	if err != nil {
		t.Error("error returned when trying to prove inclusion on empty key")
		t.Log(err)
	}
	result = smt.VerifyProof(proof, []byte("testKey3"), DefaultValue)
	if !result {
		t.Error("valid proof on empty key failed to verify")
	}
	result = smt.VerifyProof(proof, []byte("testKey3"), []byte("badValue"))
	if result {
		t.Error("invalid proof verification on empty key returned true")
	}
	result = smt.VerifyProof(proof, []byte("testKey2"), DefaultValue)
	if result {
		t.Error("invalid proof verification on empty key returned true")
	}
	result = smt.VerifyProof(badProof, []byte("testKey"), []byte("testValue"))
	if result {
		t.Error("invalid proof verification on empty key returned true")
	}

	compactProof, err := smt.CompactProof(proof)
	decompactedProof, err := smt.DecompactProof(compactProof)
	if !reflect.DeepEqual(proof, decompactedProof) {
		t.Error("compacting and decompacting proof returns a different proof than the original proof")
	}

	badProof2 := make([][]byte, sha256.New().Size()*8+1)
	for i := 0; i < len(badProof); i++ {
		badProof[i] = make([]byte, sha256.New().Size())
		rand.Read(badProof[i])
	}
	badProof3 := make([][]byte, sha256.New().Size()*8-2)
	for i := 0; i < len(badProof); i++ {
		badProof[i] = make([]byte, sha256.New().Size())
		rand.Read(badProof[i])
	}
	badProof4 := make([][]byte, sha256.New().Size()*8)
	for i := 0; i < len(badProof); i++ {
		badProof[i] = make([]byte, sha256.New().Size()-1)
		rand.Read(badProof[i])
	}
	badProof5 := make([][]byte, sha256.New().Size()*8)
	for i := 0; i < len(badProof); i++ {
		badProof[i] = make([]byte, sha256.New().Size()+1)
		rand.Read(badProof[i])
	}
	badProof6 := make([][]byte, sha256.New().Size()*8)
	for i := 0; i < len(badProof); i++ {
		badProof[i] = make([]byte, 1)
		rand.Read(badProof[i])
	}

	result = smt.VerifyProof(badProof2, []byte("testKey3"), DefaultValue)
	if result {
		t.Error("invalid proof verification returned true")
	}
	result = smt.VerifyProof(badProof3, []byte("testKey3"), DefaultValue)
	if result {
		t.Error("invalid proof verification returned true")
	}
	result = smt.VerifyProof(badProof4, []byte("testKey3"), DefaultValue)
	if result {
		t.Error("invalid proof verification returned true")
	}
	result = smt.VerifyProof(badProof5, []byte("testKey3"), DefaultValue)
	if result {
		t.Error("invalid proof verification returned true")
	}
	result = smt.VerifyProof(badProof6, []byte("testKey3"), DefaultValue)
	if result {
		t.Error("invalid proof verification returned true")
	}

	compactProof, err = smt.CompactProof(badProof2)
	if err == nil {
		t.Error("CompactProof did not return error on bad proof size")
	}
	compactProof, err = smt.CompactProof(badProof3)
	if err == nil {
		t.Error("CompactProof did not return error on bad proof size")
	}

	decompactedProof, err = smt.DecompactProof(badProof3)
	if err == nil {
		t.Error("DecompactProof did not return error on bad proof size")
	}
	decompactedProof, err = smt.DecompactProof([][]byte{})
	if err == nil {
		t.Error("DecompactProof did not return error on bad proof size")
	}

	proof, err = smt.ProveCompact([]byte("testKey2"))
	if err != nil {
		t.Error("error returned when trying to prove inclusion")
		t.Log(err)
	}
	result = smt.VerifyCompactProof(proof, []byte("testKey2"), []byte("testValue"))
	if !result {
		t.Error("valid proof failed to verify")
	}
	result = smt.VerifyCompactProof(proof, []byte("testKey2"), []byte("badValue"))
	if result {
		t.Error("invalid proof verification returned true")
	}
	result = smt.VerifyCompactProof(proof, []byte("testKey3"), []byte("testValue"))
	if result {
		t.Error("invalid proof verification returned true")
	}
	result = smt.VerifyCompactProof(badProof, []byte("testKey"), []byte("testValue"))
	if result {
		t.Error("invalid proof verification returned true")
	}

	root := smt.Root()
	smt.Update([]byte("testKey2"), []byte("testValue2"))

	proof, err = smt.ProveCompactForRoot([]byte("testKey2"), root)
	if err != nil {
		t.Error("error returned when trying to prove inclusion")
		t.Log(err)
	}
	result = VerifyCompactProof(proof, root, []byte("testKey2"), []byte("testValue"), smt.hasher, smt.height)
	if !result {
		t.Error("valid proof failed to verify")
	}
	result = VerifyCompactProof(proof, root, []byte("testKey2"), []byte("badValue"), smt.hasher, smt.height)
	if result {
		t.Error("invalid proof verification returned true")
	}
	result = VerifyCompactProof(proof, root, []byte("testKey3"), []byte("testValue"), smt.hasher, smt.height)
	if result {
		t.Error("invalid proof verification returned true")
	}
	result = VerifyCompactProof(badProof, root, []byte("testKey"), []byte("testValue"), smt.hasher, smt.height)
	if result {
		t.Error("invalid proof verification returned true")
	}

	proof, err = smt.ProveForRoot([]byte("testKey2"), root)
	if err != nil {
		t.Error("error returned when trying to prove inclusion")
		t.Log(err)
	}
	result = VerifyProof(proof, root, []byte("testKey2"), []byte("testValue"), smt.hasher, smt.height)
	if !result {
		t.Error("valid proof failed to verify")
	}
	result = VerifyProof(proof, root, []byte("testKey2"), []byte("badValue"), smt.hasher, smt.height)
	if result {
		t.Error("invalid proof verification returned true")
	}
	result = VerifyProof(proof, root, []byte("testKey3"), []byte("testValue"), smt.hasher, smt.height)
	if result {
		t.Error("invalid proof verification returned true")
	}
	result = VerifyProof(badProof, root, []byte("testKey"), []byte("testValue"), smt.hasher, smt.height)
	if result {
		t.Error("invalid proof verification returned true")
	}
}
