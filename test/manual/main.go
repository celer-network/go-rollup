package main

import (
	"flag"
	"os/exec"

	"github.com/rs/zerolog/log"

	"github.com/celer-network/go-rollup/test"
)

func main() {
	flag.Parse()
	log.Logger = log.With().Caller().Logger()
	cmdCopy := exec.Command("cp", "-a", "../config", "/tmp/celer_rollup_test")
	err := cmdCopy.Run()
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	test.DeployMainchainContracts()
	test.DeploySidechainContracts()

	<-make(chan bool)
}
