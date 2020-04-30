package main

import (
	"flag"
	"os"

	"github.com/rs/zerolog/pkgerrors"

	"github.com/celer-network/go-rollup/aggregator"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var (
	config            = flag.String("config", "/tmp/celer_rollup_test/config", "Config directory")
	aggregatorDbDir   = flag.String("aggregatordb", "/tmp/celer_rollup_test/aggregator1Db", "Aggregator DB directory")
	validatorDbDir    = flag.String("validatordb", "/tmp/celer_rollup_test/validator1Db", "Validator DB directory")
	mainchainKeystore = flag.String("mainchainkeystore", "env/keystore/node1.json", "Mainchain keystore file")
	sidechainKeystore = flag.String("sidechainkeystore", "env/keystore/node1.json", "Sidechain keystore file")
	fraudTransfer     = flag.Bool("fraudtransfer", false, "Submit bad state root for transfers")
)

func main() {
	flag.Parse()
	log.Logger = log.With().Caller().Logger()
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	viper.AddConfigPath(*config)
	viper.SetConfigName("parameters")
	viper.MergeInConfig()
	viper.SetConfigName("ethereum_networks")
	viper.MergeInConfig()
	viper.SetConfigName("mainchain_contract_addresses")
	viper.MergeInConfig()
	viper.SetConfigName("sidechain_contract_addresses")
	viper.MergeInConfig()
	aggregator, err :=
		aggregator.NewAggregator(
			*aggregatorDbDir,
			*validatorDbDir,
			*mainchainKeystore,
			*sidechainKeystore,
			*fraudTransfer,
		)
	if err != nil {
		os.Exit(1)
	}
	aggregator.Start()
	<-make(chan interface{})
}
