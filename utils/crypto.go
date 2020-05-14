package utils

import (
	"crypto/ecdsa"
	"io/ioutil"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	solsha3 "github.com/miguelmota/go-solidity-sha3"
	"github.com/rs/zerolog/log"
)

func IsSignatureValid(signer common.Address, data []byte, sig []byte) bool {
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

func SignHash(privateKey *ecdsa.PrivateKey, hash []byte) ([]byte, error) {
	prefixedHash := solsha3.SoliditySHA3WithPrefix(hash)
	log.Debug().Str("prefixedHash", common.Bytes2Hex(prefixedHash)).Send()
	sig, err := crypto.Sign(prefixedHash, privateKey)
	if err != nil {
		return nil, err
	}
	// Use 27/28 for v
	sig[64] = sig[64] + 27
	return sig, nil
}

func SignData(privateKey *ecdsa.PrivateKey, data []byte) ([]byte, error) {
	return SignHash(privateKey, crypto.Keccak256(data))
}

func SignPackedData(privateKey *ecdsa.PrivateKey, types []string, data []interface{}) ([]byte, error) {
	// SoliditySHA3 is equivalent to abi.encodePacked
	// TODO: Maybe always use abi.encode in the contracts
	return SignHash(privateKey, solsha3.SoliditySHA3(types, data))
}

func generatePrefixedHash(data []byte) []byte {
	return crypto.Keccak256([]byte("\x19Ethereum Signed Message:\n32"), crypto.Keccak256(data))
}
