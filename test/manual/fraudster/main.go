package main

import (
	"bufio"
	"flag"
	"io/ioutil"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/spf13/viper"

	"github.com/celer-network/go-rollup/types"
	"github.com/celer-network/rollup-contracts/bindings/go/mainchain/rollup"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog/log"
)

var (
	config            = flag.String("config", "/tmp/celer_rollup_test/config", "Config directory")
	fraudsterDb       = flag.String("config", "/tmp/celer_rollup_test/fraudsterDb", "DB directory")
	mainchainKeystore = flag.String("mainchainkeystore", "env/keystore/aggregator1.json", "Path to mainchain keystore")
)

type Fraudster struct {
	mainchainClient *ethclient.Client
	mainchainAuth   *bind.TransactOpts
	serializer      *types.Serializer
	rollupChain     *rollup.RollupChain
}

func NewFraudster(
	mainchainClient *ethclient.Client,
	mainchainAuth *bind.TransactOpts,
	serializer *types.Serializer,
	rollupChain *rollup.RollupChain,
) *Fraudster {
	return &Fraudster{
		mainchainClient: mainchainClient,
		mainchainAuth:   mainchainAuth,
		serializer:      serializer,
		rollupChain:     rollupChain,
	}
}

func (f *Fraudster) submitFraudBlock(pendingBlock *types.RollupBlock) error {
	// serializedBlock, err := pendingBlock.SerializeTransactions(f.serializer)
	// if err != nil {
	// 	return err
	// }
	// log.Print("Submitting fraud block ", pendingBlock.BlockNumber)
	// tx, err := f.rollupChain.CommitBlock(f.mainchainAuth, serializedBlock)
	// if err != nil {
	// 	return err
	// }
	// receipt, err := utils.WaitMined(context.Background(), f.mainchainClient, tx, 0)
	// if err != nil {
	// 	return err
	// }
	// if receipt.Status != 1 {
	// 	return errors.New("Failed to submit block")
	// }
	return nil
}

func main() {
	flag.Parse()

	serializer, err := types.NewSerializer()
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	mainchainKeystoreBytes, err := ioutil.ReadFile(*mainchainKeystore)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	mainchainKey, err := keystore.DecryptKey(mainchainKeystoreBytes, "")
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	mainchainAuth := bind.NewKeyedTransactor(mainchainKey.PrivateKey)

	mainchainClient, err := ethclient.Dial(viper.GetString("mainchainEndpoint"))
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	rollupChainAddress := viper.GetString("rollupChain")
	rollupChain, err := rollup.NewRollupChain(common.HexToAddress(rollupChainAddress), mainchainClient)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	fraudster := NewFraudster(mainchainClient, mainchainAuth, serializer, rollupChain)
	goodTransition := &types.DepositTransition{
		TransitionType:   big.NewInt(int64(types.TransitionTypeDeposit)),
		StateRoot:        [32]byte{},
		AccountSlotIndex: big.NewInt(0),
		TokenIndex:       big.NewInt(0),
		Amount:           big.NewInt(1),
		Signature:        []byte{},
	}
	fraudTransition := &types.DepositTransition{
		TransitionType:   big.NewInt(int64(types.TransitionTypeDeposit)),
		StateRoot:        [32]byte{},
		AccountSlotIndex: big.NewInt(0),
		TokenIndex:       big.NewInt(0),
		Amount:           big.NewInt(0),
		Signature:        []byte{},
	}
	block := &types.RollupBlock{
		BlockNumber: 0,
		Transitions: []types.Transition{goodTransition, fraudTransition},
	}

	// Submit fraud block manually
	bufio.NewReader(os.Stdin).ReadBytes('\n')

	err = fraudster.submitFraudBlock(block)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
}
