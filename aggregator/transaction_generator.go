package aggregator

import (
	"github.com/celer-network/go-rollup/db"
	"github.com/celer-network/go-rollup/types"
	"github.com/celer-network/rollup-contracts/bindings/go/mainchain"
	"github.com/celer-network/rollup-contracts/bindings/go/sidechain"
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
	aggregatorDb    db.DB
	validatorDb     db.DB
	sidechainClient *ethclient.Client
	rollupChain     *mainchain.RollupChain
	tokenRegistry   *mainchain.TokenRegistry
	tokenMapper     *sidechain.TokenMapper
	txQueue         chan types.Transaction
}

func NewTransactionGenerator(
	aggregatorDb db.DB,
	validatorDb db.DB,
	mainchainClient *ethclient.Client,
	rollupChain *mainchain.RollupChain,
) *TransactionGenerator {
	sidechainClient, err := ethclient.Dial(viper.GetString("sideChainEndpoint"))
	if err != nil {
		log.Error().Err(err).Send()
	}

	tokenRegistryAddress := viper.GetString("tokenRegistry")
	log.Printf("tokenRegistryAddress %s", tokenRegistryAddress)
	tokenRegistry, err := mainchain.NewTokenRegistry(common.HexToAddress(tokenRegistryAddress), mainchainClient)
	if err != nil {
		log.Error().Err(err).Send()
	}

	tokenMapperAddress := viper.GetString("tokenMapper")
	tokenMapper, err := sidechain.NewTokenMapper(common.HexToAddress(tokenMapperAddress), sidechainClient)
	if err != nil {
		log.Error().Err(err).Send()
	}

	return &TransactionGenerator{
		aggregatorDb:    aggregatorDb,
		validatorDb:     validatorDb,
		sidechainClient: sidechainClient,
		rollupChain:     rollupChain,
		tokenRegistry:   tokenRegistry,
		tokenMapper:     tokenMapper,
		txQueue:         make(chan types.Transaction, txQueueSize),
	}
}

func (tg *TransactionGenerator) Start() {
	go tg.watchTokenRegistry()
	go tg.watchTokenMapper()
	go tg.watchTransition()
}

func (tg *TransactionGenerator) watchTransition() {
	log.Print("Watch Transition")
	channel := make(chan *mainchain.RollupChainTransition)
	tg.rollupChain.WatchTransition(&bind.WatchOpts{}, channel)
	for {
		select {
		case _ = <-channel:
			//log.Debug().Int("data length", len(event.Data)).Str("data", common.Bytes2Hex(event.Data)).Msg("Caught Transition")
		}
	}
}

func (tg *TransactionGenerator) watchTokenRegistry() error {
	log.Print("Watching TokenRegistry")
	channel := make(chan *mainchain.TokenRegistryTokenRegistered)
	sub, err := tg.tokenRegistry.WatchTokenRegistered(&bind.WatchOpts{}, channel, nil, nil)
	if err != nil {
		log.Err(err).Send()
		return err
	}
	for {
		select {
		case event := <-channel:
			log.Printf("Registered token %s as %s", event.TokenAddress.Hex(), event.TokenIndex.String())
			err := tg.aggregatorDb.Set(
				db.NamespaceTokenAddressToTokenIndex,
				event.TokenAddress.Bytes(),
				event.TokenIndex.Bytes(),
			)
			if err != nil {
				log.Err(err).Send()
			}
			err = tg.validatorDb.Set(
				db.NamespaceTokenAddressToTokenIndex,
				event.TokenAddress.Bytes(),
				event.TokenIndex.Bytes(),
			)
			if err != nil {
				log.Err(err).Send()
			}
			err = tg.aggregatorDb.Set(
				db.NamespaceTokenIndexToTokenAddress,
				event.TokenIndex.Bytes(),
				event.TokenAddress.Bytes(),
			)
			if err != nil {
				log.Err(err).Send()
			}
			err = tg.validatorDb.Set(
				db.NamespaceTokenIndexToTokenAddress,
				event.TokenIndex.Bytes(),
				event.TokenAddress.Bytes(),
			)
			if err != nil {
				log.Err(err).Send()
			}
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
			err := tg.aggregatorDb.Set(
				db.NamespaceMainchainTokenAddressToSidechainTokenAddress,
				event.MainchainToken.Bytes(),
				sidechainErc20Address.Bytes())
			if err != nil {
				log.Err(err).Send()
			}
			err = tg.validatorDb.Set(
				db.NamespaceMainchainTokenAddressToSidechainTokenAddress,
				event.MainchainToken.Bytes(),
				sidechainErc20Address.Bytes())
			if err != nil {
				log.Err(err).Send()
			}
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
			tg.txQueue <- &types.TransferTransaction{
				Sender:    event.Sender,
				Recipient: event.Recipient,
				Token:     event.MainchainToken,
				Amount:    event.Amount,
				Nonce:     event.Nonce,
				Signature: event.Signature,
			}
		case _ = <-transferSub.Err():
		case event := <-depositChannel:
			log.Print("Caught deposit")
			tg.txQueue <- &types.DepositTransaction{
				Account:   event.Account,
				Token:     event.MainchainToken,
				Amount:    event.Amount,
				Signature: event.Signature,
			}
		case _ = <-depositSub.Err():
		case event := <-withdrawChannel:
			log.Print("Caught withdraw")
			tg.txQueue <- &types.WithdrawTransaction{
				Account:   event.Account,
				Token:     event.MainchainToken,
				Amount:    event.Amount,
				Nonce:     event.Nonce,
				Signature: event.Signature,
			}
		case _ = <-withdrawSub.Err():
		}
	}
}
