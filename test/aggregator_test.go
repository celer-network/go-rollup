package test

import (
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/celer-network/go-rollup/aggregator"
)

func TestSubmitBlock(t *testing.T) {
	log.Logger = log.With().Caller().Logger()
	mainchainProc, err := StartMainchain()
	if err != nil {
		t.Fatal(err)
	}
	mainchainPid := mainchainProc.Pid
	defer syscall.Kill(mainchainPid, syscall.SIGTERM)

	sidechainProc, err := StartSidechain()
	if err != nil {
		t.Fatal(err)
	}
	sidechainPid := sidechainProc.Pid
	defer syscall.Kill(sidechainPid, syscall.SIGTERM)

	time.Sleep(5 * time.Second)

	DeployMainchainContracts()
	DeploySidechainContracts()
	SetupConfig()

	aggregator, err := aggregator.NewAggregator(aggregatorDbDir, aggregatorKeystore, aggregatorKeystore)
	if err != nil {
		t.Fatal(err)
	}
	aggregator.Start()
	time.Sleep(2 * time.Second)

	RunTokenMapper(
		mainchainEtherbaseKeystore,
		sidechainEtherbaseKeystore,
		account1Keystore,
		account2Keystore,
	)

	time.Sleep(2 * time.Second)
	err = os.RemoveAll(testRootDir)
	if err != nil {
		t.Fatal(err)
	}
}
