package test

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/spf13/viper"

	"github.com/celer-network/rollup-contracts/bindings/go/sidechain"

	"github.com/celer-network/go-rollup/utils"
	"github.com/celer-network/rollup-contracts/bindings/go/mainchain/rollup"
	"gopkg.in/yaml.v2"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	ethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/rs/zerolog/log"
)

type MainchainContractAddresses struct {
	AccountRegistry        common.Address `yaml:"accountRegistry"`
	TokenRegistry          common.Address `yaml:"tokenRegistry"`
	ValidatorRegistry      common.Address `yaml:"validatorRegistry"`
	RollupChain            common.Address `yaml:"rollupChain"`
	MerkleUtils            common.Address `yaml:"merkleUtils"`
	DepositWithdrawManager common.Address `yaml:"depositWithdrawManager"`
	TransitionEvaluator    common.Address `yaml:"transitionEvaluator"`
	TestToken              common.Address `yaml:"testToken"`
}

type SidechainContractAddresses struct {
	TokenMapper    common.Address `yaml:"tokenMapper"`
	BlockCommittee common.Address `yaml:"blockCommittee"`
}

type TestTokenInfo struct {
	TestTokenAddress  common.Address `yaml:"testTokenAddress"`
	TestTokenSymbol   string         `yaml:"testTokenSymbol"`
	TestTokenName     string         `yaml:"testTokenName"`
	TestTokenDecimals uint8          `yaml:"testTokenDecimals"`
}

const (
	testRootDir                  = "/tmp/celer_rollup_test"
	envDir                       = "env"
	mainchainEthEndpoint         = "ws://127.0.0.1:8546"
	sidechainEthEndpoint         = "ws://127.0.0.1:8548"
	aggregatorAddressStr         = "0x6a6d2a97da1c453a4e099e8054865a0a59728863"
	mainchainEtherbaseAddressStr = "0xb5bb8b7f6f1883e0c01ffb8697024532e6f3238c"
	sidechainEtherbaseAddressStr = "0xba756d65a1a03f07d205749f35e2406e4a8522ad"
	repo                         = "github.com/celer-network/go-rollup"
)

var (
	mainchainEtherBaseAddress = common.HexToAddress(mainchainEtherbaseAddressStr)
	sidechainEtherBaseAddress = common.HexToAddress(sidechainEtherbaseAddressStr)
	aggregatorAddress         = common.HexToAddress(aggregatorAddressStr)

	testConfigDir              = filepath.Join(testRootDir, "config")
	mainchainDataDir           = filepath.Join(testRootDir, "mainchaindata")
	sidechainDataDir           = filepath.Join(testRootDir, "sidechaindata")
	aggregator1DbDir           = filepath.Join(testRootDir, "aggregator1Db")
	validator1DbDir            = filepath.Join(testRootDir, "validator1Db")
	aggregator2DbDir           = filepath.Join(testRootDir, "aggregator2Db")
	validator2DbDir            = filepath.Join(testRootDir, "validator2Db")
	emptyPasswordFile          = filepath.Join(envDir, "empty_password.txt")
	mainchainGenesis           = filepath.Join(envDir, "mainchain_genesis.json")
	sidechainGenesis           = filepath.Join(envDir, "sidechain_genesis.json")
	keystoreDir                = filepath.Join(envDir, "keystore")
	envConfigDir               = filepath.Join(envDir, "config")
	mainchainEtherbaseKeystore = filepath.Join(keystoreDir, "mainchain_etherbase.json")
	sidechainEtherbaseKeystore = filepath.Join(keystoreDir, "sidechain_etherbase.json")
	node1Keystore              = filepath.Join(keystoreDir, "node1.json")
	account1Keystore           = filepath.Join(keystoreDir, "account1.json")
	account2Keystore           = filepath.Join(keystoreDir, "account2.json")
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
	_, err := os.Stat(testConfigDir)
	if err != nil {
		if os.IsNotExist(err) {
			os.Mkdir(testConfigDir, 0777)
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
	writeBytes(bytes, filepath.Join(testConfigDir, "mainchain_contract_addresses.yaml"))
}

func saveSidechainContractAddresses(addresses *SidechainContractAddresses) {
	bytes, err := yaml.Marshal(addresses)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	writeBytes(bytes, filepath.Join(testConfigDir, "sidechain_contract_addresses.yaml"))
}

func saveTestTokenInfo(info *TestTokenInfo) {
	bytes, err := yaml.Marshal(info)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	writeBytes(bytes, filepath.Join(testConfigDir, "test_token.yaml"))
}

func SetupConfig() error {
	cmdCopy := exec.Command("cp", "-a", envConfigDir, testRootDir)
	err := cmdCopy.Run()
	if err != nil {
		return err
	}
	viper.AddConfigPath(testConfigDir)
	viper.SetConfigName("parameters")
	err = viper.MergeInConfig()
	if err != nil {
		return err
	}
	viper.SetConfigName("ethereum_networks")
	err = viper.MergeInConfig()
	if err != nil {
		return err
	}
	viper.SetConfigName("mainchain_contract_addresses")
	err = viper.MergeInConfig()
	if err != nil {
		return err
	}
	viper.SetConfigName("sidechain_contract_addresses")
	err = viper.MergeInConfig()
	if err != nil {
		return err
	}
	viper.SetConfigName("test_token")
	return viper.MergeInConfig()
}

func StartMainchain() (*os.Process, error) {
	return StartChain(
		883,
		mainchainDataDir,
		"mainchain.log",
		mainchainGenesis,
		mainchainEtherbaseAddressStr,
		30303,
		8545,
		8546,
	)
}

func StartSidechain() (*os.Process, error) {
	return StartChain(
		884,
		sidechainDataDir,
		"sidechain.log",
		sidechainGenesis,
		sidechainEtherbaseAddressStr,
		30304,
		8547,
		8548,
	)
}

func StartChain(
	networkId int,
	chainDataDir string,
	logFilename string,
	genesis string,
	etherbase string,
	port int,
	rpcPort int,
	wsPort int,
) (*os.Process, error) {
	if err := os.MkdirAll(chainDataDir, os.ModePerm); err != nil {
		log.Print(err)
		return nil, err
	}

	cmdCopy := exec.Command("cp", "-a", keystoreDir, chainDataDir)
	if err := cmdCopy.Run(); err != nil {
		log.Print(cmdCopy, err)
		return nil, err
	}

	// geth init
	cmdInit := exec.Command(
		"geth", "--datadir", chainDataDir, "init", genesis)
	if err := cmdInit.Run(); err != nil {
		log.Print(cmdInit, err)
		return nil, err
	}
	// actually run geth, blocking. set syncmode full to avoid bloom mem cache by fast sync
	cmd := exec.Command(
		"geth", "--networkid", strconv.Itoa(networkId), "--cache", "256", "--nousb", "--syncmode", "full", "--nodiscover", "--maxpeers", "0",
		"--netrestrict", "127.0.0.1/8", "--datadir", chainDataDir, "--keystore", filepath.Join(chainDataDir, "keystore"), "--targetgaslimit", "10000000",
		"--mine", "--allow-insecure-unlock", "--unlock", etherbase, "--etherbase", etherbase, "--password", emptyPasswordFile,
		"--port", strconv.Itoa(port),
		"--rpc", "--rpcport", strconv.Itoa(rpcPort), "--rpccorsdomain", "*",
		"--rpcapi", "admin,debug,eth,miner,net,personal,txpool,web3",
		"--ws", "--wsport", strconv.Itoa(wsPort), "--wsorigins", "*",
		"--wsapi", "admin,debug,eth,miner,net,personal,txpool,web3",
	)
	cmd.Dir = cmdInit.Dir

	logF, err := os.Create(filepath.Join(testRootDir, logFilename))
	if err != nil {
		return nil, err
	}
	cmd.Stderr = logF
	cmd.Stdout = logF
	if err := cmd.Start(); err != nil {
		log.Print(cmd, err)
		return nil, err
	}
	log.Printf("geth pid: %d\n", cmd.Process.Pid)
	// in case geth exits with non-zero, exit test early
	// if geth is killed by ethProc.Signal, it exits w/ 0
	go func() {
		if err := cmd.Wait(); err != nil {
			log.Print("geth process failed: ", err)
			os.Exit(1)
		}
	}()
	return cmd.Process, nil
}

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

	validatorRegistryAddress, tx, validatorRegistry, err := rollup.DeployValidatorRegistry(
		etherbaseAuth,
		conn,
		[]common.Address{aggregatorAddress},
	)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	receipt, err := utils.WaitMined(ctx, conn, tx, 0)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	checkTxStatus(receipt.Status, "Deploy ValidatorRegistry")
	log.Printf("Deployed ValidatorRegistry at 0x%x\n", validatorRegistryAddress)

	log.Print("Deploying AccountRegistry...")
	accountRegistryAddress, tx, _, err := rollup.DeployAccountRegistry(etherbaseAuth, conn)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	receipt, err = utils.WaitMined(ctx, conn, tx, 0)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	checkTxStatus(receipt.Status, "Deploy AccountRegistry")
	log.Printf("Deployed AccountRegistry at 0x%x\n", accountRegistryAddress)

	log.Print("Deploying TokenRegistry...")
	tokenRegistryAddress, tx, _, err := rollup.DeployTokenRegistry(etherbaseAuth, conn)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	receipt, err = utils.WaitMined(ctx, conn, tx, 0)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	checkTxStatus(receipt.Status, "Deploy TokenRegistry")
	log.Printf("Deployed TokenRegistry at 0x%x\n", tokenRegistryAddress)

	log.Print("Deploying MerkleUtils...")
	merkleUtilsAddress, tx, _, err := rollup.DeployMerkleUtils(etherbaseAuth, conn)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	receipt, err = utils.WaitMined(ctx, conn, tx, 0)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	checkTxStatus(receipt.Status, "Deploy MerkleUtils")
	log.Printf("Deployed MerkleUtils at 0x%x\n", merkleUtilsAddress)

	log.Print("Deploying TransitionEvaluator...")
	transitionEvaluatorAddress, tx, _, err :=
		rollup.DeployTransitionEvaluator(
			etherbaseAuth,
			conn,
			accountRegistryAddress,
			tokenRegistryAddress)
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
			merkleUtilsAddress,
			tokenRegistryAddress,
			validatorRegistryAddress,
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
			accountRegistryAddress,
			tokenRegistryAddress,
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

	log.Print("Setting RollupChain for ValidatorRegistry...")
	tx, err = validatorRegistry.SetRollupChainAddress(etherbaseAuth, rollupChainAddress)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	receipt, err = utils.WaitMined(ctx, conn, tx, 0)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	checkTxStatus(receipt.Status, "Set RollupChain for ValidatorRegistry")

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
		MerkleUtils:            merkleUtilsAddress,
		AccountRegistry:        accountRegistryAddress,
		TokenRegistry:          tokenRegistryAddress,
		ValidatorRegistry:      validatorRegistryAddress,
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

	blockCommitteeAddress, tx, _, err := sidechain.DeployBlockCommittee(
		etherbaseAuth,
		conn,
		[]common.Address{aggregatorAddress},
	)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	receipt, err = utils.WaitMined(ctx, conn, tx, 0)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	checkTxStatus(receipt.Status, "Deploy BlockCommittee")
	log.Printf("Deployed BlockCommittee at 0x%x\n", blockCommitteeAddress)

	a := &SidechainContractAddresses{
		TokenMapper:    tokenMapperAddress,
		BlockCommittee: blockCommitteeAddress,
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
