package main

import (
	"flag"
	"os"

	"github.com/celer-network/sidechain-rollup-aggregator/blockproducer"
	"github.com/spf13/viper"
)

var (
	config    = flag.String("config", "", "Config directory")
	mainDbDir = flag.String("maindb", "", "Main DB directory")
	treeDbDir = flag.String("treedb", "", "Tree DB directory")
)

func main() {
	flag.Parse()
	viper.AddConfigPath(*config)
	producer, err := blockproducer.NewBlockProducer(*mainDbDir, *treeDbDir)
	if err != nil {
		os.Exit(1)
	}
	producer.Start()
}
