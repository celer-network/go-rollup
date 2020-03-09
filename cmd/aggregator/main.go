package main

import (
	"flag"
	"os"

	"github.com/celer-network/go-rollup/aggregator"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var (
	config            = flag.String("config", "/tmp/celer_rollup_test/config", "Config directory")
	mainDbDir         = flag.String("maindb", "/tmp/celer_rollup_test/maindb", "Main DB directory")
	treeDbDir         = flag.String("treedb", "/tmp/celer_rollup_test/treedb", "Tree DB directory")
	mainchainKeystore = flag.String("mainchainkeystore", "env/keystore/aggregator.json", "Mainchain keystore file")
)

func main() {
	flag.Parse()
	log.Logger = log.With().Caller().Logger()
	viper.AddConfigPath(*config)
	viper.SetConfigName("parameters")
	viper.MergeInConfig()
	viper.SetConfigName("ethereum_networks")
	viper.MergeInConfig()
	viper.SetConfigName("mainchain_contract_addresses")
	viper.MergeInConfig()
	viper.SetConfigName("sidechain_contract_addresses")
	viper.MergeInConfig()
	aggregator, err := aggregator.NewAggregator(*mainDbDir, *treeDbDir, *mainchainKeystore)
	if err != nil {
		os.Exit(1)
	}
	aggregator.Start()
	<-make(chan interface{})
}
