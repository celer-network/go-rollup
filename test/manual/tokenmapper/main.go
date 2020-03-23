package main

import (
	"flag"

	"github.com/celer-network/go-rollup/test"

	"github.com/rs/zerolog/log"

	"github.com/spf13/viper"
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

	test.RunTokenMapper(*mainchainKeystore, *sidechainKeystore, *account1Keystore, *account2Keystore)
}
