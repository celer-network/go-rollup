package main

import (
	"context"
	"errors"
	"flag"
	"math/big"
	"time"

	"github.com/celer-network/go-rollup/test"
	"github.com/celer-network/go-rollup/utils"
	"github.com/celer-network/rollup-contracts/bindings/go/mainchain"
	"github.com/celer-network/rollup-contracts/bindings/go/sidechain"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var (
	config   = flag.String("config", "manual/transfertocontract/config", "Config directory")
	keystore = flag.String("ks", "", "Path to user 0 keystore")
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

	mainchainConn, err := ethclient.Dial(viper.GetString("mainchainEndpoint"))
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	sidechainConn, err := ethclient.Dial(viper.GetString("sidechainEndpoint"))
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	privateKey, err := utils.GetPrivateKayFromKeystore(*keystore, "")
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	auth, err := utils.GetAuthFromKeystore(*keystore, "")
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	testTokenAddressStr := viper.GetString("testTokenAddress")
	testTokenAddress := common.HexToAddress(testTokenAddressStr)

	ctx := context.Background()

	tokenMapperAddress := common.HexToAddress(viper.GetString("tokenMapper"))
	tokenMapper, err := sidechain.NewTokenMapper(tokenMapperAddress, sidechainConn)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	depositWithdrawManagerAddress := common.HexToAddress(viper.GetString("depositWithdrawManager"))
	depositWithdrawManager, err := mainchain.NewDepositWithdrawManager(depositWithdrawManagerAddress, mainchainConn)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	senderAddress := auth.From
	depositAmount := big.NewInt(100)
	testToken, err := test.NewERC20(testTokenAddress, mainchainConn)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	tx, err := testToken.Approve(auth, depositWithdrawManagerAddress, depositAmount)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	receipt, err := utils.WaitMined(ctx, mainchainConn, tx, 0)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	mainchainDepositNonce, err := depositWithdrawManager.DepositNonces(&bind.CallOpts{}, senderAddress, testTokenAddress)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	// Deposit on the mainchain and register account for rollup security. A validator will relay
	// the deposit
	mainchainDepositSig, err := utils.SignPackedData(
		privateKey,
		[]string{"address", "string", "address", "address", "uint256", "uint256"},
		[]interface{}{
			depositWithdrawManagerAddress,
			"deposit",
			senderAddress,
			testTokenAddress,
			depositAmount,
			mainchainDepositNonce,
		},
	)
	log.Info().Msg("Depositing on mainchain")
	auth.GasLimit = 8000000
	tx, err = depositWithdrawManager.Deposit(
		auth,
		senderAddress,
		testTokenAddress,
		depositAmount,
		mainchainDepositSig,
	)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	receipt, err = utils.WaitMined(ctx, mainchainConn, tx, 0)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	if receipt.Status != 1 {
		log.Fatal().Str("tx", tx.Hash().Hex()).Err(errors.New("Mainchain deposit for sender failed")).Send()
	}
	sidechainErc20Address, err := tokenMapper.MainchainTokenToSidechainToken(&bind.CallOpts{}, testTokenAddress)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	sidechainErc20, err := sidechain.NewSidechainERC20(sidechainErc20Address, sidechainConn)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	for i := 0; i < 100; i++ {
		time.Sleep(5 * time.Second)
		user0Balance, err := sidechainErc20.BalanceOf(&bind.CallOpts{}, senderAddress)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		if user0Balance.Cmp(big.NewInt(1)) == 0 {
			log.Info().Msg("Depositing relayed on sidechain")
			break
		}
		if i == 4 {
			log.Fatal().Err(errors.New("Sidechain deposit failed")).Send()
		}
	}

	log.Print("Deploying DummyApp...")
	dummyAppAddress, tx, _, err := sidechain.DeployDummyApp(auth, sidechainConn, sidechainErc20Address)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	receipt, err = utils.WaitMined(ctx, sidechainConn, tx, 0)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	if receipt.Status != 1 {
		log.Fatal().Err(errors.New("Failed deployment")).Send()
	}
	log.Printf("Deployed DummyApp at 0x%x\n", dummyAppAddress)

	sidechainErc20, err = sidechain.NewSidechainERC20(sidechainErc20Address, sidechainConn)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	nonce, err := sidechainErc20.TransferNonces(&bind.CallOpts{}, senderAddress)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	dummyApp, err := sidechain.NewDummyApp(dummyAppAddress, sidechainConn)
	playerOneSig, err := utils.SignPackedData(
		privateKey,
		[]string{"address", "address", "address", "uint256", "uint256"},
		[]interface{}{
			senderAddress,
			dummyAppAddress,
			testTokenAddress,
			big.NewInt(1), // amount
			nonce,
		},
	)
	auth.GasLimit = 8000000
	log.Info().Msg("Player 1 deposits into DummyApp and the contract sends the tokens to Player 2")
	tx, err = dummyApp.PlayerOneDeposit(auth, playerOneSig)
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
}
