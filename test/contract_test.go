package test

import (
	"context"
	"math/big"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	"github.com/celer-network/go-rollup/aggregator"
	"github.com/celer-network/go-rollup/utils"
	"github.com/celer-network/rollup-contracts/bindings/go/sidechain"
)

func TestDummyApp(t *testing.T) {
	log.Logger = log.With().Caller().Logger()
	mainchainProc, err := StartMainchain()
	if err != nil {
		t.Fatal(err)
	}
	mainchainPid := mainchainProc.Pid
	defer syscall.Kill(mainchainPid, syscall.SIGTERM)

	sidechainProc, err := StartSidechain()
	if err != nil {
		t.Fatal(err)
	}
	sidechainPid := sidechainProc.Pid
	defer syscall.Kill(sidechainPid, syscall.SIGTERM)

	time.Sleep(5 * time.Second)

	DeployMainchainContracts()
	DeploySidechainContracts()
	SetupConfig()

	mainchainConn, err := ethclient.Dial(viper.GetString("mainchainEndpoint"))
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	mainchainAuth, err := utils.GetAuthFromKeystore(mainchainEtherbaseKeystore, "")
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	sidechainConn, err := ethclient.Dial(viper.GetString("sidechainEndpoint"))
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	sidechainAuth, err := utils.GetAuthFromKeystore(sidechainEtherbaseKeystore, "")
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	testTokenAddressStr := viper.GetString("testTokenAddress")
	testTokenSymbol := viper.GetString("testTokenSymbol")
	testTokenName := viper.GetString("testTokenName")
	testTokenDecimals := uint8(viper.GetUint("testTokenDecimals"))
	testTokenAddress := common.HexToAddress(testTokenAddressStr)

	ctx := context.Background()
	MapToken(ctx, sidechainConn, sidechainAuth, testTokenAddress, testTokenName, testTokenSymbol, testTokenDecimals)
	RegisterToken(ctx, mainchainConn, mainchainAuth, testTokenAddress)

	tokenMapperAddress := common.HexToAddress(viper.GetString("tokenMapper"))
	tokenMapper, err := sidechain.NewTokenMapper(tokenMapperAddress, sidechainConn)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	sidechainErc20Address, err := tokenMapper.MainchainTokenToSidechainToken(&bind.CallOpts{}, testTokenAddress)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	user0Auth, err := utils.GetAuthFromKeystore(user0Keystore, "")
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	user1Auth, err := utils.GetAuthFromKeystore(user1Keysetore, "")
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
	checkTxStatus(receipt.Status, "Deploy DummyApp")
	log.Printf("Deployed DummyApp at 0x%x\n", dummyAppAddress)

	aggregator, err := aggregator.NewAggregator(node0AggregatorDbDir, node0ValidatorDbDir, node1Keystore, node1Keystore, false, false)
	if err != nil {
		t.Fatal(err)
	}
	aggregator.Start()
	time.Sleep(2 * time.Second)

	dummyApp, err := sidechain.NewDummyApp(dummyAppAddress, sidechainConn)
	playerOnePrivateKey, err := utils.GetPrivateKayFromKeystore(user0Keystore, "")
	playerOneSig, err := utils.SignPackedData(
		playerOnePrivateKey,
		[]string{"address", "address", "uint256", "uint256"},
		[]interface{}{
			user0Auth.From,
			dummyAppAddress,
			big.NewInt(1),
			big.NewInt(0),
		},
	)
	dummyApp.PlayerOneDeposit(user0Auth, playerOneSig)
	dummyApp.PlayerTwoWithdraw(user1Auth)

	err = os.RemoveAll(testRootDir)
	if err != nil {
		t.Fatal(err)
	}
}
