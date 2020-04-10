package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"math/big"
	"os"

	"github.com/celer-network/go-rollup/test"
	"github.com/celer-network/go-rollup/utils"
	"github.com/celer-network/sidechain-contracts/bindings/go/sidechain"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var (
	config            = flag.String("config", "/tmp/celer_rollup_test/config", "Config directory")
	mainchainKeystore = flag.String("mainchainkeystore", "env/keystore/mainchain_etherbase.json", "Path to mainchain keystore")
	sidechainKeystore = flag.String("sidechainkeystore", "env/keystore/sidechain_etherbase.json", "Path to sidechain keystore")
	account1Keystore  = flag.String("account1keystore", "env/keystore/account1.json", "Path to account 1 keystore")
	account2Keystore  = flag.String("account2keystore", "env/keystore/account2.json", "Path to account 2 keystore")
)

func main() {
	flag.Parse()

	log.Logger = log.With().Caller().Logger()
	viper.AddConfigPath(*config)
	viper.SetConfigName("ethereum_networks")
	viper.MergeInConfig()
	viper.SetConfigName("mainchain_contract_addresses")
	viper.MergeInConfig()
	viper.SetConfigName("sidechain_contract_addresses")
	viper.MergeInConfig()
	viper.SetConfigName("test_token")
	viper.MergeInConfig()

	test.DeployMainchainContracts()
	test.DeploySidechainContracts()
	test.SetupConfig()

	mainchainConn, err := ethclient.Dial(viper.GetString("mainchainEndpoint"))
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	mainchainAuth, err := utils.GetAuthFromKeystore(*mainchainKeystore, "")
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	sidechainConn, err := ethclient.Dial(viper.GetString("sidechainEndpoint"))
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	sidechainAuth, err := utils.GetAuthFromKeystore(*sidechainKeystore, "")
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	testTokenAddressStr := viper.GetString("testTokenAddress")
	testTokenSymbol := viper.GetString("testTokenSymbol")
	testTokenName := viper.GetString("testTokenName")
	testTokenDecimals := uint8(viper.GetUint("testTokenDecimals"))
	testTokenAddress := common.HexToAddress(testTokenAddressStr)

	ctx := context.Background()
	test.MapToken(ctx, sidechainConn, sidechainAuth, testTokenAddress, testTokenName, testTokenSymbol, testTokenDecimals)
	test.RegisterToken(ctx, mainchainConn, mainchainAuth, testTokenAddress)

	tokenMapperAddress := common.HexToAddress(viper.GetString("tokenMapper"))
	tokenMapper, err := sidechain.NewTokenMapper(tokenMapperAddress, sidechainConn)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	sidechainErc20Address, err := tokenMapper.MainchainTokenToSidechainToken(&bind.CallOpts{}, testTokenAddress)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	go watchTransfer(sidechainErc20Address, sidechainConn)

	account1Auth, err := utils.GetAuthFromKeystore(*account1Keystore, "")
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	account2Auth, err := utils.GetAuthFromKeystore(*account2Keystore, "")
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	log.Print("Deploying DummyApp...")
	dummyAppAddress, tx, _, err := sidechain.DeployDummyApp(sidechainAuth, sidechainConn, sidechainErc20Address)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	receipt, err := utils.WaitMined(ctx, sidechainConn, tx, 0)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	if receipt.Status != 1 {
		log.Fatal().Err(errors.New("Failed deployment")).Send()
	}
	log.Printf("Deployed DummyApp at 0x%x\n", dummyAppAddress)

	sidechainErc20, err := sidechain.NewSidechainERC20(sidechainErc20Address, sidechainConn)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	tx, err = sidechainErc20.Deposit(account1Auth, account1Auth.From, big.NewInt(1), nil)
	utils.WaitMined(ctx, sidechainConn, tx, 0)

	// Start aggregator manually
	bufio.NewReader(os.Stdin).ReadBytes('\n')

	dummyApp, err := sidechain.NewDummyApp(dummyAppAddress, sidechainConn)
	playerOnePrivateKey, err := utils.GetPrivateKayFromKeystore(*account1Keystore, "")
	playerOneSig, err := utils.SignData(
		playerOnePrivateKey,
		[]string{"address", "address", "uint256", "uint256"},
		[]interface{}{
			account1Auth.From,
			dummyAppAddress,
			big.NewInt(1),
			big.NewInt(1),
		},
	)
	log.Debug().Str("playerOneSig", common.Bytes2Hex(playerOneSig)).Send()
	account1Auth.GasLimit = 10000000
	tx, err = dummyApp.PlayerOneDeposit(account1Auth, playerOneSig)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	log.Debug().Str("txHash", tx.Hash().Hex()).Send()
	receipt, err = utils.WaitMined(ctx, sidechainConn, tx, 0)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	if receipt.Status != 1 {
		log.Fatal().Err(errors.New("Failed player 1 deposit")).Send()
	}

	tx, err = dummyApp.PlayerTwoWithdraw(account2Auth)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	receipt, err = utils.WaitMined(ctx, sidechainConn, tx, 0)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	if receipt.Status != 1 {
		log.Fatal().Err(errors.New("Failed player 2 withdraw")).Send()
	}
}

func watchTransfer(sidechainErc20Address common.Address, conn *ethclient.Client) {
	sidechainErc20, err := sidechain.NewSidechainERC20(sidechainErc20Address, conn)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	transferChannel := make(chan *sidechain.SidechainERC20Transfer)
	sidechainErc20.WatchTransfer(&bind.WatchOpts{}, transferChannel, nil, nil, nil)
	for {
		select {
		case event := <-transferChannel:
			log.Debug().
				Str("event", "Caught transfer").
				Str("sender", event.Sender.Hex()).
				Str("recipient", event.Recipient.Hex()).
				Str("token", event.MainchainToken.Hex()).
				Str("amount", event.Amount.String()).
				Str("nonce", event.Nonce.String()).
				Str("signature", common.Bytes2Hex(event.Signature)).
				Send()
		}
	}
}
