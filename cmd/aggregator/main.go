package main

import (
	"flag"

	"github.com/rs/zerolog/pkgerrors"

	"github.com/celer-network/go-rollup/aggregator"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var (
	config            = flag.String("config", "/tmp/celer-rollup-test/config", "Config directory")
	aggregatorDbDir   = flag.String("aggregatordb", "/tmp/celer-rollup-test/node0_aggregator_db", "Aggregator DB directory")
	validatorDbDir    = flag.String("validatordb", "/tmp/celer-rollup-test/node0_validator_db", "Validator DB directory")
	mainchainKeystore = flag.String("mainchainkeystore", "env/keystore/node0.json", "Mainchain keystore file")
	sidechainKeystore = flag.String("sidechainkeystore", "env/keystore/node0.json", "Sidechain keystore file")
	relayerGrpcPort   = flag.Int("relayergrpcport", 6666, "Relayer gRPC port")
	fraudTransfer     = flag.Bool("fraudtransfer", false, "Submit bad state root for transfers")
	validatorMode     = flag.Bool("validatormode", false, "Run in validator mode")
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
			*relayerGrpcPort,
			*fraudTransfer,
			*validatorMode,
		)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	aggregator.Start()
	<-make(chan interface{})
}
