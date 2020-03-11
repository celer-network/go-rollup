package test

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/celer-network/sidechain-contracts/bindings/go/sidechain"

	"github.com/celer-network/go-rollup/utils"
	"github.com/celer-network/sidechain-contracts/bindings/go/mainchain/rollup"
	"gopkg.in/yaml.v2"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	ethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/rs/zerolog/log"
)

type MainchainContractAddresses struct {
	RollupChain            common.Address `yaml:"rollupChain"`
	RollupMerkleUtils      common.Address `yaml:"rollupMerkleUtils"`
	RollupTokenRegistry    common.Address `yaml:"rollupTokenRegistry"`
	DepositWithdrawManager common.Address `yaml:"depositWithdrawManager"`
	TransitionEvaluator    common.Address `yaml:"transitionEvaluator"`
	TestToken              common.Address `yaml:"testToken"`
}

type SidechainContractAddresses struct {
	TokenMapper common.Address `yaml:"tokenMapper"`
}

type TestTokenInfo struct {
	TestTokenAddress  common.Address `yaml:"testTokenAddress"`
	TestTokenSymbol   string         `yaml:"testTokenSymbol"`
	TestTokenName     string         `yaml:"testTokenName"`
	TestTokenDecimals uint8          `yaml:"testTokenDecimals"`
}

const (
	testRootDir                  = "/tmp/celer_rollup_test"
	mainchainEthEndpoint         = "ws://127.0.0.1:8546"
	sidechainEthEndpoint         = "ws://127.0.0.1:8548"
	aggregatorAddressStr         = "0x6a6d2a97da1c453a4e099e8054865a0a59728863"
	mainchainEtherbaseKeystore   = "env/keystore/mainchain_etherbase.json"
	sidechainEtherbaseKeystore   = "env/keystore/sidechain_etherbase.json"
	aggregatorKeystore           = "env/aggregator.json"
	mainchainEtherbaseAddressStr = "0xb5bb8b7f6f1883e0c01ffb8697024532e6f3238c"
	sidechainEtherbaseAddressStr = "0xba756d65a1a03f07d205749f35e2406e4a8522ad"
	repo                         = "github.com/celer-network/go-rollup"
)

var (
	mainchainEtherBaseAddress = common.HexToAddress(mainchainEtherbaseAddressStr)
	sidechainEtherBaseAddress = common.HexToAddress(sidechainEtherbaseAddressStr)
	aggregatorAddress         = common.HexToAddress(aggregatorAddressStr)

	configDir = filepath.Join(testRootDir, "config")
)

// toBuild map package subpath to binary file name eg. aggregator/cmd -> aggregator means
// build aggregator/cmd and output aggregator
var toBuild = map[string]string{
	"aggregator/cmd": "aggregator",
}

func getEthClient(endpoint string) (*ethclient.Client, error) {
	ws, err := ethrpc.Dial(endpoint)
	if err != nil {
		return nil, err
	}
	conn := ethclient.NewClient(ws)
	return conn, nil
}

func sleep(second time.Duration) {
	time.Sleep(second * time.Second)
}

func writeBytes(bytes []byte, path string) {
	_, err := os.Stat(configDir)
	if err != nil {
		if os.IsNotExist(err) {
			os.Mkdir(configDir, 0777)
		} else {
			log.Fatal().Err(err).Send()
		}
	}
	err = ioutil.WriteFile(path, bytes, 0644)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
}

func saveMainchainContractAddresses(addresses *MainchainContractAddresses) {
	bytes, err := yaml.Marshal(addresses)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	writeBytes(bytes, filepath.Join(configDir, "mainchain_contract_addresses.yaml"))
}

func saveSidechainContractAddresses(addresses *SidechainContractAddresses) {
	bytes, err := yaml.Marshal(addresses)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	writeBytes(bytes, filepath.Join(configDir, "sidechain_contract_addresses.yaml"))
}

func saveTestTokenInfo(info *TestTokenInfo) {
	bytes, err := yaml.Marshal(info)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	writeBytes(bytes, filepath.Join(configDir, "test_token.yaml"))
}

// func StartChain() (*os.Process, error) {
// 	log.Print("outRootDir", outRootDir, "envDir", envDir)
// 	chainDataDir := outRootDir + "chaindata"
// 	logFname := outRootDir + "chain.log"
// 	if err := os.MkdirAll(chainDataDir, os.ModePerm); err != nil {
// 		return nil, err
// 	}

// 	cmdCopy := exec.Command("cp", "-a", "keystore", chainDataDir)
// 	cmdCopy.Dir, _ = filepath.Abs(envDir)
// 	log.Infoln("copy", filepath.Join(envDir, "keystore"), filepath.Join(chainDataDir, "keystore"))
// 	if err := cmdCopy.Run(); err != nil {
// 		return nil, err
// 	}

// 	// geth init
// 	cmdInit := exec.Command("geth", "--datadir", chainDataDir, "init", envDir+"/celer_genesis.json")
// 	// set cmd.Dir because relative files are under testing/env
// 	cmdInit.Dir, _ = filepath.Abs(envDir)
// 	if err := cmdInit.Run(); err != nil {
// 		return nil, err
// 	}
// 	// actually run geth, blocking. set syncmode full to avoid bloom mem cache by fast sync
// 	cmd := exec.Command("geth", "--networkid", "883", "--cache", "256", "--nousb", "--syncmode", "full", "--nodiscover", "--maxpeers", "0",
// 		"--netrestrict", "127.0.0.1/8", "--datadir", chainDataDir, "--keystore", filepath.Join(chainDataDir, "keystore"), "--targetgaslimit", "8000000",
// 		"--mine", "--allow-insecure-unlock", "--unlock", "0", "--password", "empty_password.txt", "--rpc", "--rpccorsdomain", "*",
// 		"--rpcapi", "admin,debug,eth,miner,net,personal,shh,txpool,web3")
// 	cmd.Dir = cmdInit.Dir

// 	logF, _ := os.Create(logFname)
// 	cmd.Stderr = logF
// 	cmd.Stdout = logF
// 	if err := cmd.Start(); err != nil {
// 		return nil, err
// 	}
// 	fmt.Println("geth pid:", cmd.Process.Pid)
// 	// in case geth exits with non-zero, exit test early
// 	// if geth is killed by ethProc.Signal, it exits w/ 0
// 	go func() {
// 		if err := cmd.Wait(); err != nil {
// 			fmt.Println("geth process failed:", err)
// 			os.Exit(1)
// 		}
// 	}()
// 	return cmd.Process, nil
// }

func DeployMainchainContracts() *MainchainContractAddresses {
	conn, err := ethclient.Dial(filepath.Join(testRootDir, "mainchaindata/geth.ipc"))
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	etherbaseAuth, err := utils.GetAuthFromKeystore(mainchainEtherbaseKeystore, "")
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	ctx := context.Background()

	log.Print("Deploying RollupTokenRegistry...")
	rollupTokenRegistryAddress, tx, _, err := rollup.DeployRollupTokenRegistry(etherbaseAuth, conn)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	receipt, err := utils.WaitMined(ctx, conn, tx, 0)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	checkTxStatus(receipt.Status, "Deploy RollupTokenRegistry")
	log.Printf("Deployed RollupTokenRegistry at 0x%x\n", rollupTokenRegistryAddress)

	log.Print("Deploying RollupMerkleUtils...")
	rollupMerkleUtilsAddress, tx, _, err := rollup.DeployRollupMerkleUtils(etherbaseAuth, conn)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	receipt, err = utils.WaitMined(ctx, conn, tx, 0)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	checkTxStatus(receipt.Status, "Deploy RollupMerkleUtils")
	log.Printf("Deployed RollupMerkleUtils at 0x%x\n", rollupMerkleUtilsAddress)

	log.Print("Deploying TransitionEvaluator...")
	transitionEvaluatorAddress, tx, _, err := rollup.DeployTransitionEvaluator(etherbaseAuth, conn, rollupTokenRegistryAddress)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	receipt, err = utils.WaitMined(ctx, conn, tx, 0)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	checkTxStatus(receipt.Status, "Deploy TransitionEvaluator")
	log.Printf("Deployed TransitionEvaluator at 0x%x\n", transitionEvaluatorAddress)

	log.Print("Deploying RollupChain...")
	rollupChainAddress, tx, _, err :=
		rollup.DeployRollupChain(
			etherbaseAuth,
			conn,
			transitionEvaluatorAddress,
			rollupMerkleUtilsAddress,
			rollupTokenRegistryAddress,
			aggregatorAddress,
		)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	receipt, err = utils.WaitMined(ctx, conn, tx, 0)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	checkTxStatus(receipt.Status, "Deploy RollupChain")
	log.Printf("Deployed RollupChain at 0x%x\n", rollupChainAddress)

	log.Print("Deploying DepositWithdrawManager...")
	depositWithdrawManagerAddress, tx, _, err :=
		rollup.DeployDepositWithdrawManager(
			etherbaseAuth,
			conn,
			rollupChainAddress,
			transitionEvaluatorAddress,
			rollupTokenRegistryAddress,
		)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	receipt, err = utils.WaitMined(ctx, conn, tx, 0)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	checkTxStatus(receipt.Status, "Deploy DepositWithdrawManager")
	log.Printf("Deployed DepositWithdrawManager at 0x%x\n", depositWithdrawManagerAddress)

	initAmt := new(big.Int)
	initAmt.SetString("500000000000000000000000000000000000000000000", 10)
	erc20Address, tx, erc20, err := DeployERC20(etherbaseAuth, conn, initAmt, "Moon", 18, "MOON")
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	receipt, err = utils.WaitMined(ctx, conn, tx, 0)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	checkTxStatus(receipt.Status, "Deploy ERC20")
	log.Printf("Deployed ERC20 at 0x%x\n", erc20Address)

	info := &TestTokenInfo{
		TestTokenAddress:  erc20Address,
		TestTokenName:     "Moon",
		TestTokenSymbol:   "MOON",
		TestTokenDecimals: 18,
	}
	saveTestTokenInfo(info)

	// Transfer ERC20 to aggregator
	moonAmt := new(big.Int)
	moonAmt.SetString("500000000000000000000000000000", 10)
	addrs := []common.Address{aggregatorAddress}
	for _, addr := range addrs {
		tx, err = erc20.Transfer(etherbaseAuth, addr, moonAmt)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		utils.WaitMined(ctx, conn, tx, 0)
	}
	log.Printf("Sent MOON to recipients")

	a := &MainchainContractAddresses{
		RollupChain:            rollupChainAddress,
		RollupMerkleUtils:      rollupMerkleUtilsAddress,
		RollupTokenRegistry:    rollupTokenRegistryAddress,
		TransitionEvaluator:    transitionEvaluatorAddress,
		DepositWithdrawManager: depositWithdrawManagerAddress,
		TestToken:              erc20Address,
	}
	saveMainchainContractAddresses(a)
	return a
}

func DeploySidechainContracts() *SidechainContractAddresses {
	conn, err := ethclient.Dial(filepath.Join(testRootDir, "sidechaindata/geth.ipc"))
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	etherbaseAuth, err := utils.GetAuthFromKeystore(sidechainEtherbaseKeystore, "")
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	ctx := context.Background()

	log.Print("Deploying TokenMapper...")
	tokenMapperAddress, tx, _, err := sidechain.DeployTokenMapper(etherbaseAuth, conn)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	receipt, err := utils.WaitMined(ctx, conn, tx, 0)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	checkTxStatus(receipt.Status, "Deploy TokenMapper")
	log.Printf("Deployed TokenMapper at 0x%x\n", tokenMapperAddress)

	a := &SidechainContractAddresses{
		TokenMapper: tokenMapperAddress,
	}
	saveSidechainContractAddresses(a)
	return a
}

func buildBins(rootDir string) error {
	for pkg, bin := range toBuild {
		fmt.Println("Building", pkg, "->", bin)
		cmd := exec.Command("go", "build", "-o", rootDir+bin, repo+pkg)
		cmd.Stderr, _ = os.OpenFile(rootDir+"build.err", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		err := cmd.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func checkErr(e error, msg string) {
	if e != nil {
		fmt.Println("Err:", msg, e)
		os.Exit(1)
	}
}

// if status isn't 1 (success), log.Fatal
func checkTxStatus(s uint64, txname string) {
	if s != 1 {
		log.Fatal().Msg(txname + " tx failed")
	}
	log.Info().Msg(txname + " tx success")
}
