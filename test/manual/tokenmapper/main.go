package main

import (
	"flag"

	"github.com/celer-network/go-rollup/test"

	"github.com/rs/zerolog/log"

	"github.com/spf13/viper"
)

var (
	config            = flag.String("config", "/tmp/celer-rollup-test/config", "Config directory")
	mainchainKeystore = flag.String("mainchainkeystore", "env/keystore/mainchain_etherbase.json", "Path to mainchain keystore")
	sidechainKeystore = flag.String("sidechainkeystore", "env/keystore/sidechain_etherbase.json", "Path to sidechain keystore")
	user0Keystore     = flag.String("user0keystore", "env/keystore/user0.json", "Path to user 0 keystore")
	user1Keystore     = flag.String("user1keystore", "env/keystore/user1.json", "Path to user 1 keystore")
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

	test.RunTokenMapper(*mainchainKeystore, *sidechainKeystore, *user0Keystore, *user1Keystore)
}
