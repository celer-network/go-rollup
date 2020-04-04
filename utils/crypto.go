package utils

import (
	"crypto/ecdsa"
	"fmt"
	"io/ioutil"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/rs/zerolog/log"
)

func SigIsValid(signer common.Address, data []byte, sig []byte) bool {
	recoveredAddr := RecoverSigner(data, sig)
	return recoveredAddr == signer
}

func RecoverSigner(data []byte, sig []byte) common.Address {
	pubKey, err := crypto.SigToPub(generatePrefixedHash(data), sig)
	if err != nil {
		log.Error().Msg(err.Error())
		return common.Address{}
	}
	recoveredAddr := crypto.PubkeyToAddress(*pubKey)
	return recoveredAddr
}

func GetPrivateKayFromKeystore(path string, password string) (*ecdsa.PrivateKey, error) {
	ksBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	key, err := keystore.DecryptKey(ksBytes, password)
	if err != nil {
		return nil, err
	}
	return key.PrivateKey, nil
}

func GetAuthFromKeystore(path string, password string) (*bind.TransactOpts, error) {
	privateKey, err := GetPrivateKayFromKeystore(path, password)
	if err != nil {
		return nil, err
	}
	return bind.NewKeyedTransactor(privateKey), nil
}

func SignData(privateKey *ecdsa.PrivateKey, data ...[]byte) ([]byte, error) {
	hash := crypto.Keccak256Hash(data...)
	prefixedHash := crypto.Keccak256Hash(
		[]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%v", len(hash))),
		hash.Bytes(),
	)
	return crypto.Sign(prefixedHash.Bytes(), privateKey)
}

func generatePrefixedHash(data []byte) []byte {
	return crypto.Keccak256([]byte("\x19Ethereum Signed Message:\n32"), crypto.Keccak256(data))
}
