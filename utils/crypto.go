package crypto

import (
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

func generatePrefixedHash(data []byte) []byte {
	return crypto.Keccak256([]byte("\x19Ethereum Signed Message:\n32"), crypto.Keccak256(data))
}
