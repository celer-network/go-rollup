package utils

import (
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

func GetAuthFromKeystore(path string, password string) (*bind.TransactOpts, error) {
	ksBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	key, err := keystore.DecryptKey(ksBytes, password)
	if err != nil {
		return nil, err
	}
	return bind.NewKeyedTransactor(key.PrivateKey), nil
}

func generatePrefixedHash(data []byte) []byte {
	return crypto.Keccak256([]byte("\x19Ethereum Signed Message:\n32"), crypto.Keccak256(data))
}
