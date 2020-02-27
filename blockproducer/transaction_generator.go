package blockproducer

import (
	"github.com/celer-network/sidechain-contracts/bindings/go/mainchain/rollup"
	"github.com/celer-network/sidechain-contracts/bindings/go/sidechain"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type TransactionGenerator struct {
	rollupChain                  *rollup.RollupChain
	sidechainTokenRegistry       *sidechain.SidechainTokenRegistry
	tokenAddressToSidechainErc20 map[string]*sidechain.SidechainERC20
}

func NewTransactionGenerator() *TransactionGenerator {
	mainchainClient, err := ethclient.Dial(viper.GetString("mainchainEndpoint"))
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	sidechainClient, err := ethclient.Dial(viper.GetString("mainChainEndpoint"))
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	rollupChainAddress := viper.GetString("rollupChainAddress")
	rollupChain, err := rollup.NewRollupChain(common.HexToAddress(rollupChainAddress), mainchainClient)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	sidechainTokenRegistryAddress := viper.GetString("sidechainTokenRegistryAddress")
	sidechain.NewSidechainTokenRegistry(common.HexToAddress(sidechainTokenRegistryAddress), sidechainClient)

	return &TransactionGenerator{
		rollupChain:                  rollupChain,
		tokenAddressToSidechainErc20: make(map[string]*sidechain.SidechainERC20),
	}
}

func (tg *TransactionGenerator) Start() {

}
