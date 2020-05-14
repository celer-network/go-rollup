package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"math/big"
	"os"
	"time"

	"google.golang.org/grpc"

	"github.com/celer-network/go-rollup/relayer"

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

	user0PrivateKey, err := utils.GetPrivateKayFromKeystore(*user0Keystore, "")
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	user0Auth, err := utils.GetAuthFromKeystore(*user0Keystore, "")
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	user1PrivateKey, err := utils.GetPrivateKayFromKeystore(*user1Keystore, "")
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	user1Auth, err := utils.GetAuthFromKeystore(*user1Keystore, "")
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

	depositWithdrawManagerAddress := common.HexToAddress(viper.GetString("depositWithdrawManager"))
	depositWithdrawManager, err := mainchain.NewDepositWithdrawManager(depositWithdrawManagerAddress, mainchainConn)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	user0Address := user0Auth.From
	user1Address := user1Auth.From
	amount := big.NewInt(1)
	testToken, err := test.NewERC20(testTokenAddress, mainchainConn)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	tx, err := testToken.Approve(user0Auth, depositWithdrawManagerAddress, amount)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	receipt, err := utils.WaitMined(ctx, mainchainConn, tx, 0)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	tx, err = testToken.Approve(user1Auth, depositWithdrawManagerAddress, amount)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	receipt, err = utils.WaitMined(ctx, mainchainConn, tx, 0)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	user0MainchainDepositNonce, err := depositWithdrawManager.DepositNonces(&bind.CallOpts{}, user0Address, testTokenAddress)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	user0MainchainDepositSig, err := utils.SignPackedData(
		user0PrivateKey,
		[]string{"address", "string", "address", "address", "uint256", "uint256"},
		[]interface{}{
			depositWithdrawManagerAddress,
			"deposit",
			user0Address,
			testTokenAddress,
			amount,
			user0MainchainDepositNonce,
		},
	)
	user1MainchainDepositNonce, err := depositWithdrawManager.DepositNonces(&bind.CallOpts{}, user1Address, testTokenAddress)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	user1MainchainDepositSig, err := utils.SignPackedData(
		user1PrivateKey,
		[]string{"address", "string", "address", "address", "uint256", "uint256"},
		[]interface{}{
			depositWithdrawManagerAddress,
			"deposit",
			user1Address,
			testTokenAddress,
			amount,
			user1MainchainDepositNonce,
		},
	)
	mainchainAuth.GasLimit = 8000000
	tx, err = depositWithdrawManager.Deposit(
		mainchainAuth,
		user0Address,
		testTokenAddress,
		amount,
		user0MainchainDepositSig,
	)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	receipt, err = utils.WaitMined(ctx, mainchainConn, tx, 0)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	if receipt.Status != 1 {
		log.Fatal().Str("tx", tx.Hash().Hex()).Err(errors.New("Mainchain deposit for user 0 failed")).Send()
	}
	tx, err = depositWithdrawManager.Deposit(mainchainAuth, user1Address, testTokenAddress, amount, user1MainchainDepositSig)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	receipt, err = utils.WaitMined(ctx, mainchainConn, tx, 0)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	if receipt.Status != 1 {
		log.Fatal().Err(errors.New("Mainchain deposit for user 1 failed")).Send()
	}

	sidechainErc20Address, err := tokenMapper.MainchainTokenToSidechainToken(&bind.CallOpts{}, testTokenAddress)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	sidechainErc20, err := sidechain.NewSidechainERC20(sidechainErc20Address, sidechainConn)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	for i := 0; i < 5; i++ {
		time.Sleep(time.Second)
		user0Balance, err := sidechainErc20.BalanceOf(&bind.CallOpts{}, user0Address)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		if user0Balance.Cmp(big.NewInt(1)) == 0 {
			break
		}
		if i == 4 {
			log.Fatal().Err(errors.New("Sidechain deposit failed")).Send()
		}
	}

	user0SidechainWithdrawSig, err := utils.SignPackedData(
		user0PrivateKey,
		[]string{"address", "address", "uint256", "uint256"},
		[]interface{}{
			user0Auth.From,
			testTokenAddress,
			amount,        // amount
			big.NewInt(0), // withdrawNonce
		},
	)
	tx, err = sidechainErc20.Withdraw(user0Auth, user0Auth.From, amount, user0SidechainWithdrawSig)
	receipt, err = utils.WaitMined(ctx, sidechainConn, tx, 0)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	if receipt.Status != 1 {
		log.Fatal().Err(errors.New("Sidechain withdraw failed")).Send()
	}

	log.Info().Msg("Press Enter to withdraw on mainchain")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
	// time.Sleep(5 * time.Second)

	user0MainchainWithdrawNonce, err :=
		depositWithdrawManager.WithdrawNonces(&bind.CallOpts{}, user0Address, testTokenAddress)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	user0MainchainWithdrawSig, err := utils.SignPackedData(
		user0PrivateKey,
		[]string{"address", "string", "address", "address", "uint256", "uint256"},
		[]interface{}{
			depositWithdrawManagerAddress,
			"withdraw",
			user0Address,
			testTokenAddress,
			amount,
			user0MainchainWithdrawNonce,
		},
	)
	grpcConn, err := grpc.Dial("127.0.0.1:6666", grpc.WithInsecure())
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	relayerClient := relayer.NewRelayerRpcClient(grpcConn)
	resp, err := relayerClient.Withdraw(
		context.Background(),
		&relayer.WithdrawRequest{
			Account:           user0Address.Hex(),
			RollupBlockNumber: 0,
			TransitionIndex:   2,
			Signature:         user0MainchainWithdrawSig,
		},
	)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	txHash := resp.TransactionHash
	receipt, err = utils.WaitMinedWithTxHash(ctx, mainchainConn, txHash, 0)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	if receipt.Status != 1 {
		log.Fatal().Err(errors.New("Mainchain withdraw failed")).Send()
	}

}
