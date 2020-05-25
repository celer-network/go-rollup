package main

import (
	"context"
	"errors"
	"flag"
	"math/big"

	"github.com/celer-network/go-rollup/test"
	"github.com/celer-network/go-rollup/utils"
	"github.com/celer-network/rollup-contracts/bindings/go/sidechain"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var (
	config            = flag.String("config", "/tmp/celer-rollup-test/config", "Config directory")
	mainchainKeystore = flag.String("mainchainkeystore", "env/keystore/mainchain_etherbase.json", "Path to mainchain keystore")
	sidechainKeystore = flag.String("sidechainkeystore", "env/keystore/sidechain_etherbase.json", "Path to sidechain keystore")
	user0Keystore     = flag.String("user0keystore", "env/keystore/user0.json", "Path to user 0 keystore")
	user1Keystore     = flag.String("user1keystore", "env/keystore/user1.json", "Path to user 1 keystore")
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

	// test.DeployMainchainContracts()
	// test.DeploySidechainContracts()
	// test.SetupConfig()

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

	user0Auth, err := utils.GetAuthFromKeystore(*user0Keystore, "")
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	user1Auth, err := utils.GetAuthFromKeystore(*user1Keystore, "")
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

	// Start aggregator manually
	// log.Info().Msg("Press Enter")
	// bufio.NewReader(os.Stdin).ReadBytes('\n')

	tx, err = sidechainErc20.Deposit(user0Auth, user0Auth.From, big.NewInt(1), nil)
	utils.WaitMined(ctx, sidechainConn, tx, 0)

	dummyApp, err := sidechain.NewDummyApp(dummyAppAddress, sidechainConn)
	playerOnePrivateKey, err := utils.GetPrivateKayFromKeystore(*user0Keystore, "")
	playerOneSig, err := utils.SignPackedData(
		playerOnePrivateKey,
		[]string{"address", "address", "address", "uint256", "uint256"},
		[]interface{}{
			user0Auth.From,
			dummyAppAddress,
			testTokenAddress,
			big.NewInt(1), // amount
			big.NewInt(0), // nonce
		},
	)
	// log.Debug().Str("playerOneSig", common.Bytes2Hex(playerOneSig)).Send()
	user0Auth.GasLimit = 8000000
	log.Info().Msg("Player 1 deposits into DummyApp and the contract sends the tokens to Player 2")
	tx, err = dummyApp.PlayerOneDeposit(user0Auth, playerOneSig)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	log.Debug().Str("txHash", tx.Hash().Hex()).Send()
	receipt, err = utils.WaitMined(ctx, sidechainConn, tx, 0)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	if receipt.Status != 1 {
		log.Fatal().Str("tx", tx.Hash().Hex()).Err(errors.New("Failed player 1 deposit")).Send()
	}

	user1Auth.GasLimit = 8000000
	log.Info().Msg("Player 2 withdraws the tokens from DummyApp")
	tx, err = dummyApp.PlayerTwoWithdraw(user1Auth)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	receipt, err = utils.WaitMined(ctx, sidechainConn, tx, 0)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	if receipt.Status != 1 {
		log.Fatal().Str("tx", tx.Hash().Hex()).Err(errors.New("Failed player 2 withdraw")).Send()
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
