package test

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/celer-network/rollup-contracts/bindings/go/mainchain"
	"github.com/celer-network/rollup-contracts/bindings/go/sidechain"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/celer-network/go-rollup/utils"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func RunTokenMapper(
	mainchainKeystore string,
	sidechainKeystore string,
	user0Keystore string,
	user1Keystore string,
) {
	mainchainConn, err := ethclient.Dial(viper.GetString("mainchainEndpoint"))
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	mainchainAuth, err := utils.GetAuthFromKeystore(mainchainKeystore, "")
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	sidechainConn, err := ethclient.Dial(viper.GetString("sidechainEndpoint"))
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	sidechainAuth, err := utils.GetAuthFromKeystore(sidechainKeystore, "")
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

	user0Auth, err := utils.GetAuthFromKeystore(user0Keystore, "")
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	user1Auth, err := utils.GetAuthFromKeystore(user1Keystore, "")
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	user0PrivateKey, err := utils.GetPrivateKayFromKeystore(user0Keystore, "")
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	DepositAndTransfer(ctx, sidechainConn, sidechainAuth, user0Auth, user1Auth, testTokenAddress, user0PrivateKey)
}

func RegisterToken(ctx context.Context, conn *ethclient.Client, auth *bind.TransactOpts, tokenAddress common.Address) {
	tokenRegistryAddress := common.HexToAddress(viper.GetString("tokenRegistry"))
	tokenRegistry, err := mainchain.NewTokenRegistry(tokenRegistryAddress, conn)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	tx, err := tokenRegistry.RegisterToken(auth, tokenAddress)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	waitMined(ctx, conn, tx)
}

func MapToken(
	ctx context.Context,
	conn *ethclient.Client,
	auth *bind.TransactOpts,
	token common.Address,
	name string,
	symbol string,
	decimals uint8,
) {
	tokenMapperAddress := common.HexToAddress(viper.GetString("tokenMapper"))
	log.Printf("Mapping %s", token.Hex())
	tokenMapper, err := sidechain.NewTokenMapper(tokenMapperAddress, conn)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	nonce, err := conn.PendingNonceAt(context.Background(), auth.From)
	auth.Nonce = big.NewInt(int64(nonce))
	fmt.Printf("%s MapToken (nonce %v)\n", auth.From.Hex(), nonce)

	tx, err := tokenMapper.MapToken(auth, token, name, symbol, decimals)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	waitMined(ctx, conn, tx)
}

func DepositAndTransfer(
	ctx context.Context,
	conn *ethclient.Client,
	mapperAuth *bind.TransactOpts,
	auth1 *bind.TransactOpts,
	auth2 *bind.TransactOpts,
	token common.Address,
	auth1PrivateKey *ecdsa.PrivateKey,
) {
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
	amount := big.NewInt(1)

	_nonce, err := conn.PendingNonceAt(context.Background(), mapperAuth.From)
	mapperAuth.Nonce = big.NewInt(int64(_nonce))
	fmt.Printf("%s Deposit (nonce %v)\n", mapperAuth.From.Hex(), _nonce)

	tx, err := sidechainErc20.Deposit(mapperAuth, auth1.From, amount, nil)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	waitMined(ctx, conn, tx)
	tx, err = sidechainErc20.Deposit(mapperAuth, auth2.From, amount, nil)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	waitMined(ctx, conn, tx)

	nonce, err := sidechainErc20.TransferNonces(&bind.CallOpts{}, auth1.From)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	signature, err := utils.SignPackedData(
		auth1PrivateKey,
		[]string{"address", "address", "address", "uint256", "uint256"},
		[]interface{}{
			auth1.From,
			auth2.From,
			token,
			amount,
			nonce,
		},
	)
	tx, err = sidechainErc20.Transfer(auth1, auth1.From, auth2.From, amount, signature)
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
