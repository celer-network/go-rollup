package demo

import (
	"crypto/ecdsa"

	"github.com/celer-network/go-rollup/utils"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/viper"
)

const (
	flagKeystore = "keystore"
	flagAmount   = "amount"

	configMainchainEndpoint                        = "mainchain.endpoint"
	configSidechainEndpoint                        = "sidechain.endpoint"
	configMainchainContractsAccountRegistry        = "mainchain.contracts.accountRegistry"
	configMainchainContractsDepositWithdrawManager = "mainchain.contracts.depositWithdrawManager"
	configMainchainContractsTestToken              = "mainchain.contracts.testToken"
	configSidechainContractsTokenMapper            = "sidechain.contracts.tokenMapper"
)

type ethClientInfo struct {
	mainchainClient *ethclient.Client
	sidechainClient *ethclient.Client
	privateKey      *ecdsa.PrivateKey
	auth            *bind.TransactOpts
}

func initEthClientInfo() (*ethClientInfo, error) {
	mainchainClient, err := ethclient.Dial(viper.GetString(configMainchainEndpoint))
	if err != nil {
		return nil, err
	}
	sidechainClient, err := ethclient.Dial(viper.GetString(configSidechainEndpoint))
	if err != nil {
		return nil, err
	}
	keystorePath := viper.GetString(flagKeystore)
	privateKey, err := utils.GetPrivateKayFromKeystore(keystorePath, "")
	if err != nil {
		return nil, err
	}
	auth, err := utils.GetAuthFromKeystore(keystorePath, "")
	if err != nil {
		return nil, err
	}
	return &ethClientInfo{
		mainchainClient: mainchainClient,
		sidechainClient: sidechainClient,
		privateKey:      privateKey,
		auth:            auth,
	}, nil
}
