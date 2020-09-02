package relayer

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"

	"github.com/rs/zerolog/log"

	"github.com/celer-network/go-rollup/utils"

	"github.com/celer-network/rollup-contracts/bindings/go/mainchain"
	"github.com/celer-network/rollup-contracts/bindings/go/sidechain"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Bridge struct {
	mainchainClient         *ethclient.Client
	sidechainClient         *ethclient.Client
	sidechainAuth           *bind.TransactOpts
	sidechainAuthPrivateKey *ecdsa.PrivateKey
	depositWithdrawManager  *mainchain.DepositWithdrawManager
	tokenMapper             *sidechain.TokenMapper
}

//func NewBridge(
//	mainchainClient *ethclient.Client,
//	sidechainClient *ethclient.Client,
//	sidechainAuth *bind.TransactOpts,
//	sidechainAuthPrivateKey *ecdsa.PrivateKey,
//) (*Bridge, error) {
//	depositWithdrawManagerAddress := common.HexToAddress(viper.GetString("depositWithdrawManager"))
//	depositWithdrawManager, err := mainchain.NewDepositWithdrawManager(depositWithdrawManagerAddress, mainchainClient)
//	if err != nil {
//		return nil, err
//	}
//	tokenMapperAddress := common.HexToAddress(viper.GetString("tokenMapper"))
//	tokenMapper, err := sidechain.NewTokenMapper(tokenMapperAddress, sidechainClient)
//	return &Bridge{
//		mainchainClient:         mainchainClient,
//		sidechainClient:         sidechainClient,
//		sidechainAuth:           sidechainAuth,
//		sidechainAuthPrivateKey: sidechainAuthPrivateKey,
//		depositWithdrawManager:  depositWithdrawManager,
//		tokenMapper:             tokenMapper,
//	}, nil
//}

func NewBridge(
	mainchainClient         *ethclient.Client,
	sidechainClient         *ethclient.Client,
	sidechainAuth           *bind.TransactOpts,
	sidechainAuthPrivateKey *ecdsa.PrivateKey,
	depositWithdrawManager  *mainchain.DepositWithdrawManager,
	tokenMapper             *sidechain.TokenMapper,
	) *Bridge {
		return &Bridge{
			mainchainClient:         mainchainClient,
			sidechainClient:         sidechainClient,
			sidechainAuth:           sidechainAuth,
			sidechainAuthPrivateKey: sidechainAuthPrivateKey,
			depositWithdrawManager:  depositWithdrawManager,
			tokenMapper:             tokenMapper,
		}
}

func (b *Bridge) Start() {
	go b.watchMainchainDeposit()
}

func (b *Bridge) relayDeposit(
	mainchainTokenAddress common.Address,
	account common.Address,
	amount *big.Int,
) error {
	sidechainErc20Address, err := b.tokenMapper.MainchainTokenToSidechainToken(&bind.CallOpts{}, mainchainTokenAddress)
	if err != nil {
		return err
	}
	sidechainErc20, err := sidechain.NewSidechainERC20(sidechainErc20Address, b.sidechainClient)
	if err != nil {
		return err
	}
	signature, err := utils.SignPackedData(
		b.sidechainAuthPrivateKey,
		[]string{"address", "uint256"},
		[]interface{}{account, amount},
	)
	if err != nil {
		return err
	}
	log.Print(sidechainErc20Address.Hex(), " ", account.Hex(), " ", amount)
	b.sidechainAuth.GasLimit = 8000000

	nonce, err := b.sidechainClient.PendingNonceAt(context.Background(), b.sidechainAuth.From)
	b.sidechainAuth.Nonce = big.NewInt(int64(nonce))
	fmt.Printf("%s relayDeposit (nonce %v)\n", b.sidechainAuth.From.Hex(), nonce)

	tx, err := sidechainErc20.Deposit(b.sidechainAuth, account, amount, signature)
	if err != nil {
		return err
	}
	receipt, err := utils.WaitMined(context.Background(), b.sidechainClient, tx, 0)
	if err != nil {
		return err
	}
	if receipt.Status != 1 {
		return errors.New("Failed to relay deposit")
	}
	return nil
}

func (b *Bridge) watchMainchainDeposit() {
	log.Print("Watching Mainchain Deposit")
	depositChannel := make(chan *mainchain.DepositWithdrawManagerTokenDeposited)
	b.depositWithdrawManager.WatchTokenDeposited(&bind.WatchOpts{}, depositChannel)
	for {
		select {
		case event := <-depositChannel:
			log.Print("Seeing TokenDeposited event")
			log.Debug().Str("token", event.Token.Hex()).Msg("Bridge caught mainchain deposit")
			err := b.relayDeposit(event.Token, event.Account, event.Amount)
			if err != nil {
				log.Err(err).Send()
			}
		}
	}

}
