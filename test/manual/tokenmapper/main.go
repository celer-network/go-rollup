package main

import (
	"context"
	"flag"
	"math/big"
	"time"

	"github.com/celer-network/sidechain-contracts/bindings/go/mainchain/rollup"

	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/celer-network/go-rollup/utils"

	"github.com/rs/zerolog/log"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/viper"

	"github.com/celer-network/sidechain-contracts/bindings/go/sidechain"
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
	mapToken(ctx, sidechainConn, sidechainAuth, testTokenAddress, testTokenName, testTokenSymbol, testTokenDecimals)
	registerToken(ctx, mainchainConn, mainchainAuth, testTokenAddress)

	time.Sleep(2)
	account1Auth, err := utils.GetAuthFromKeystore(*account1Keystore, "")
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	account2Auth, err := utils.GetAuthFromKeystore(*account2Keystore, "")
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	depositAndTransfer(ctx, sidechainConn, sidechainAuth, account1Auth, account2Auth, testTokenAddress)
}

func registerToken(ctx context.Context, conn *ethclient.Client, auth *bind.TransactOpts, tokenAddress common.Address) {
	rollupTokenRegistryAddress := common.HexToAddress(viper.GetString("rollupTokenRegistry"))
	rollupTokenRegistry, err := rollup.NewRollupTokenRegistry(rollupTokenRegistryAddress, conn)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	tx, err := rollupTokenRegistry.RegisterToken(auth, tokenAddress)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	waitMined(ctx, conn, tx)
}

func mapToken(
	ctx context.Context,
	conn *ethclient.Client,
	auth *bind.TransactOpts,
	token common.Address,
	name string,
	symbol string,
	decimals uint8,
) {
	tokenMapperAddress := common.HexToAddress(viper.GetString("tokenMapper"))
	log.Printf("Mapping %s to %s", tokenMapperAddress.Hex(), token.Hex())
	tokenMapper, err := sidechain.NewTokenMapper(tokenMapperAddress, conn)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	tx, err := tokenMapper.MapToken(auth, token, name, symbol, decimals)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	waitMined(ctx, conn, tx)
}

func depositAndTransfer(
	ctx context.Context,
	conn *ethclient.Client,
	mapperAuth *bind.TransactOpts,
	auth1 *bind.TransactOpts,
	auth2 *bind.TransactOpts,
	token common.Address) {
	tokenMapperAddress := common.HexToAddress(viper.GetString("tokenMapper"))
	tokenMapper, err := sidechain.NewTokenMapper(tokenMapperAddress, conn)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	sidechainErc20Address, err := tokenMapper.MainchainTokenToSidechainToken(&bind.CallOpts{}, token)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	sidechainErc20, err := sidechain.NewSidechainERC20(sidechainErc20Address, conn)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	tx, err := sidechainErc20.Deposit(mapperAuth, auth1.From, big.NewInt(1), nil)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	waitMined(ctx, conn, tx)
	tx, err = sidechainErc20.Deposit(mapperAuth, auth2.From, big.NewInt(1), nil)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	waitMined(ctx, conn, tx)

	tx, err = sidechainErc20.Transfer0(auth1, auth2.From, big.NewInt(1), nil)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	waitMined(ctx, conn, tx)
}

func waitMined(ctx context.Context, conn *ethclient.Client, tx *ethtypes.Transaction) {
	receipt, err := utils.WaitMined(ctx, conn, tx, 0)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	if receipt.Status != 1 {
		log.Fatal().Err(err).Send()
	}
}
