package relayer

import (
	"context"
	"errors"
	fmt "fmt"
	"math/big"
	"net"

	"github.com/rs/zerolog/log"

	rollupdb "github.com/celer-network/go-rollup/db"
	"github.com/celer-network/go-rollup/statemachine"
	"github.com/celer-network/go-rollup/types"
	"github.com/celer-network/go-rollup/utils"
	"github.com/celer-network/rollup-contracts/bindings/go/mainchain"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"google.golang.org/grpc"
)

type WithdrawManager struct {
	grpcPort               int
	mainchainClient        *ethclient.Client
	mainchainAuth          *bind.TransactOpts
	depositWithdrawManager *mainchain.DepositWithdrawManager
	serializer             *types.Serializer
	stateMachine           *statemachine.StateMachine
	aggregatorDb           rollupdb.DB
}

func NewWithdrawManager(
	grpcPort int,
	mainchainClient *ethclient.Client,
	mainchainAuth *bind.TransactOpts,
	depositWithdrawManager *mainchain.DepositWithdrawManager,
	serializer *types.Serializer,
	stateMachine *statemachine.StateMachine,
	aggregatorDb rollupdb.DB,
) *WithdrawManager {
	return &WithdrawManager{
		grpcPort:               grpcPort,
		mainchainClient:        mainchainClient,
		mainchainAuth:          mainchainAuth,
		depositWithdrawManager: depositWithdrawManager,
		serializer:             serializer,
		stateMachine:           stateMachine,
		aggregatorDb:           aggregatorDb,
	}
}

func (m *WithdrawManager) Start() {
	grpcServer := grpc.NewServer()
	RegisterRelayerRpcServer(grpcServer, m)
	errChan := make(chan error)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", m.grpcPort))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to listen")
	}
	log.Info().Int("grpcPort", m.grpcPort).Msg("Serving relayer gRPC")
	go func() {
		errChan <- grpcServer.Serve(lis)
	}()
}

func (m *WithdrawManager) Withdraw(
	ctx context.Context, request *WithdrawRequest) (*WithdrawResponse, error) {
	txHash, err := m.withdraw(
		common.HexToAddress(request.Account),
		request.RollupBlockNumber,
		request.TransitionIndex,
		request.Signature,
	)
	if err != nil {
		return nil, err
	}
	return &WithdrawResponse{TransactionHash: txHash}, nil
}

func (m *WithdrawManager) withdraw(
	account common.Address,
	rollupBlockNumber int64,
	transitionIndex int64,
	signature []byte,
) (string, error) {
	// TODO: Check account / signature

	blockData, exists, err :=
		m.aggregatorDb.Get(
			rollupdb.NamespaceRollupBlockNumber,
			big.NewInt(rollupBlockNumber).Bytes())
	if !exists {
		return "", errors.New("Block not found")
	}
	if err != nil {
		log.Err(err).Send()
		return "", err
	}
	log.Debug().Str("blockData", common.Bytes2Hex(blockData)).Msg("Block data")
	block, err := m.serializer.DeserializeRollupBlockFromData(blockData)
	if err != nil {
		log.Err(err).Send()
		return "", err
	}
	blockInfo, err := types.NewRollupBlockInfo(m.serializer, block)
	if err != nil {
		log.Err(err).Send()
		return "", err
	}
	transition := block.Transitions[transitionIndex]
	if transition.GetTransitionType() != types.TransitionTypeWithdraw {
		return "", errors.New("Invalid transition type")
	}
	includedTransition, err := blockInfo.GetIncludedTransition(int(transitionIndex))
	if err != nil {
		log.Err(err).Send()
		return "", err
	}

	m.mainchainAuth.GasLimit = 8000000
	tx, err := m.depositWithdrawManager.Withdraw(
		m.mainchainAuth,
		account,
		*includedTransition,
		signature,
	)
	log.Debug().Str("txHash", tx.Hash().Hex()).Msg("Mainchain withdraw")
	if err != nil {
		log.Err(err).Send()
		return "", err
	}
	receipt, err := utils.WaitMined(context.Background(), m.mainchainClient, tx, 0)
	if err != nil {
		return "", err
	}
	if receipt.Status != 1 {
		return "", errors.New("Failed to submit withdraw transaction")
	}
	return tx.Hash().Hex(), nil
}
