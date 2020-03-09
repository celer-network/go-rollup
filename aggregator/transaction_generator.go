package aggregator

import (
	"github.com/celer-network/go-rollup/storage"
	"github.com/celer-network/go-rollup/types"
	"github.com/celer-network/sidechain-contracts/bindings/go/mainchain/rollup"
	"github.com/celer-network/sidechain-contracts/bindings/go/sidechain"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const (
	txQueueSize = 10
)

type TransactionGenerator struct {
	storage             *storage.Storage
	sidechainClient     *ethclient.Client
	rollupChain         *rollup.RollupChain
	rollupTokenRegistry *rollup.RollupTokenRegistry
	tokenMapper         *sidechain.TokenMapper
	txQueue             chan *types.SignedTransaction
}

func NewTransactionGenerator(
	storage *storage.Storage,
	mainchainClient *ethclient.Client,
	rollupChain *rollup.RollupChain,
) *TransactionGenerator {
	sidechainClient, err := ethclient.Dial(viper.GetString("sideChainEndpoint"))
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	rollupTokenRegistryAddress := viper.GetString("rollupTokenRegistry")
	log.Printf("rollupTokenRegistryAddress %s", rollupTokenRegistryAddress)
	rollupTokenRegistry, err := rollup.NewRollupTokenRegistry(common.HexToAddress(rollupTokenRegistryAddress), mainchainClient)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	tokenMapperAddress := viper.GetString("tokenMapper")
	tokenMapper, err := sidechain.NewTokenMapper(common.HexToAddress(tokenMapperAddress), sidechainClient)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	return &TransactionGenerator{
		storage:             storage,
		sidechainClient:     sidechainClient,
		rollupChain:         rollupChain,
		rollupTokenRegistry: rollupTokenRegistry,
		tokenMapper:         tokenMapper,
		txQueue:             make(chan *types.SignedTransaction, txQueueSize),
	}
}

func (tg *TransactionGenerator) Start() {
	go tg.watchRollupTokenRegistry()
	go tg.watchTokenMapper()
}

func (tg *TransactionGenerator) watchRollupTokenRegistry() error {
	log.Print("Watching RollupTokenRegistry")
	channel := make(chan *rollup.RollupTokenRegistryTokenRegistered)
	sub, err := tg.rollupTokenRegistry.WatchTokenRegistered(&bind.WatchOpts{}, channel, nil, nil)
	if err != nil {
		log.Err(err).Send()
		return err
	}
	for {
		select {
		case event := <-channel:
			log.Printf("Registered token %s as %s", event.TokenAddress.Hex(), event.TokenIndex.String())
			tg.storage.Set(
				storage.NamespaceTokenAddressToTokenIndex,
				event.TokenAddress.Bytes(),
				event.TokenIndex.Bytes())
		case err := <-sub.Err():
			log.Err(err).Send()
			return err
		}
	}
}

func (tg *TransactionGenerator) watchTokenMapper() error {
	channel := make(chan *sidechain.TokenMapperTokenMapped)
	sub, err := tg.tokenMapper.WatchTokenMapped(&bind.WatchOpts{}, channel, nil, nil)
	if err != nil {
		return err
	}
	for {
		select {
		case event := <-channel:
			sidechainErc20Address := event.SidechainToken
			log.Printf("Mapped token %s to %s", event.MainchainToken.Hex(), event.SidechainToken.Hex())
			tg.storage.Set(
				storage.NamespaceMainchainTokenAddressToSidechainTokenAddress,
				event.MainchainToken.Bytes(),
				sidechainErc20Address.Bytes())
			sidechainErc20, err := sidechain.NewSidechainERC20(sidechainErc20Address, tg.sidechainClient)
			if err != nil {
				return err
			}
			log.Printf("Watching %s", sidechainErc20Address.Hex())
			go tg.watchToken(sidechainErc20)
		case err := <-sub.Err():
			return err
		}
	}
}

func (tg *TransactionGenerator) watchToken(contract *sidechain.SidechainERC20) error {
	transferChannel := make(chan *sidechain.SidechainERC20Transfer)
	depositChannel := make(chan *sidechain.SidechainERC20Deposit)
	withdrawChannel := make(chan *sidechain.SidechainERC20Withdraw)
	transferSub, err := contract.WatchTransfer(&bind.WatchOpts{}, transferChannel, nil, nil, nil)
	if err != nil {
		return err
	}
	depositSub, err := contract.WatchDeposit(&bind.WatchOpts{}, depositChannel, nil, nil)
	if err != nil {
		return err
	}
	withdrawSub, err := contract.WatchWithdraw(&bind.WatchOpts{}, withdrawChannel, nil, nil)
	if err != nil {
		return err
	}
	for {
		select {
		case event := <-transferChannel:
			log.Print("Caught transfer")
			tx := &types.TransferTransaction{
				Sender:    event.Sender,
				Recipient: event.Recipient,
				Token:     event.MainchainToken,
				Amount:    event.Amount,
				Nonce:     event.Nonce,
			}
			tg.txQueue <- &types.SignedTransaction{
				Signature:   event.Signature,
				Transaction: tx,
			}
		case event := <-depositChannel:
			log.Print("Caught deposit")
			tx := &types.DepositTransaction{
				Account: event.Account,
				Token:   event.MainchainToken,
				Amount:  event.Amount,
			}
			tg.txQueue <- &types.SignedTransaction{
				Signature:   event.Signature,
				Transaction: tx,
			}
		case event := <-withdrawChannel:
			log.Print("Caught withdraw")
			tx := &types.WithdrawTransaction{
				Account: event.Account,
				Token:   event.MainchainToken,
				Amount:  event.Amount,
			}
			tg.txQueue <- &types.SignedTransaction{
				Signature:   event.Signature,
				Transaction: tx,
			}
		case err := <-transferSub.Err():
			return err
		case err := <-depositSub.Err():
			return err
		case err := <-withdrawSub.Err():
			return err
		}
	}
}
