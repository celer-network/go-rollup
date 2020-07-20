package demo

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/spf13/cobra"

	"github.com/celer-network/go-rollup/test"
	"github.com/celer-network/go-rollup/utils"
	"github.com/celer-network/rollup-contracts/bindings/go/mainchain"
	"github.com/celer-network/rollup-contracts/bindings/go/sidechain"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func deposit() error {
	ethClientInfo, err := initEthClientInfo()
	if err != nil {
		return err
	}
	mainchainClient := ethClientInfo.mainchainClient
	sidechainClient := ethClientInfo.sidechainClient
	privateKey := ethClientInfo.privateKey
	auth := ethClientInfo.auth
	ctx := context.Background()

	tokenMapperAddress := common.HexToAddress(viper.GetString(configSidechainContractsTokenMapper))
	tokenMapper, err := sidechain.NewTokenMapper(tokenMapperAddress, sidechainClient)
	if err != nil {
		return err
	}
	depositWithdrawManagerAddress :=
		common.HexToAddress(viper.GetString(configMainchainContractsDepositWithdrawManager))
	depositWithdrawManager, err := mainchain.NewDepositWithdrawManager(depositWithdrawManagerAddress, mainchainClient)
	if err != nil {
		return err
	}
	senderAddress := auth.From
	amount := new(big.Int)
	amount.SetString(viper.GetString(flagAmount), 10)

	testTokenAddress := common.HexToAddress(viper.GetString(configMainchainContractsTestToken))
	testToken, err := test.NewERC20(testTokenAddress, mainchainClient)
	if err != nil {
		return err
	}
	allowance, err := testToken.Allowance(&bind.CallOpts{}, senderAddress, depositWithdrawManagerAddress)
	if err != nil {
		return err
	}
	if allowance.Cmp(amount) < 0 {
		tx, approveErr := testToken.Approve(auth, depositWithdrawManagerAddress, amount)
		if approveErr != nil {
			return approveErr
		}
		receipt, waitApproveErr := utils.WaitMined(ctx, mainchainClient, tx, 0)
		if waitApproveErr != nil {
			return waitApproveErr
		}
		if receipt.Status != ethtypes.ReceiptStatusSuccessful {
			return fmt.Errorf("Approve tx %x failed", receipt.TxHash)
		}
	}

	sidechainErc20Address, err := tokenMapper.MainchainTokenToSidechainToken(&bind.CallOpts{}, testTokenAddress)
	if err != nil {
		return err
	}
	sidechainErc20, err := sidechain.NewSidechainERC20(sidechainErc20Address, sidechainClient)
	if err != nil {
		return err
	}
	balanceBefore, err := sidechainErc20.BalanceOf(&bind.CallOpts{}, senderAddress)
	if err != nil {
		return err
	}
	log.Info().Str("balanceBefore", balanceBefore.String()).Send()

	mainchainDepositNonce, err :=
		depositWithdrawManager.DepositNonces(&bind.CallOpts{}, senderAddress, testTokenAddress)
	if err != nil {
		return err
	}

	// Deposit on the mainchain and register account for rollup security. A validator will relay
	// the deposit
	mainchainDepositSig, err := utils.SignPackedData(
		privateKey,
		[]string{"address", "string", "address", "address", "uint256", "uint256"},
		[]interface{}{
			depositWithdrawManagerAddress,
			"deposit",
			senderAddress,
			testTokenAddress,
			amount,
			mainchainDepositNonce,
		},
	)
	log.Info().Msg("Depositing on mainchain")
	auth.GasLimit = 2000000
	tx, err := depositWithdrawManager.Deposit(
		auth,
		senderAddress,
		testTokenAddress,
		amount,
		mainchainDepositSig,
	)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	receipt, err := utils.WaitMined(ctx, mainchainClient, tx, 0)
	if err != nil {
		return err
	}
	if receipt.Status != ethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("Mainchain deposit tx %x failed", receipt.TxHash)
	}
	// TODO: Properly look for the sidechain deposit tx
	relayed := false
	for i := 0; i < 10; i++ {
		time.Sleep(5 * time.Second)
		balanceAfter, err := sidechainErc20.BalanceOf(&bind.CallOpts{}, senderAddress)
		if err != nil {
			return err
		}
		newBalance := new(big.Int)
		if newBalance.Add(balanceBefore, amount).Cmp(balanceAfter) == 0 {
			log.Info().Msg("Deposit relayed on sidechain")
			log.Info().Str("balanceAfter", balanceAfter.String()).Send()
			relayed = true
			break
		}
	}
	if !relayed {
		return errors.New("Sidechain deposit failed")
	}
	return nil
}

func DepositCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deposit",
		Short: "Deposit into the sidechain",
		RunE: func(cmd *cobra.Command, args []string) error {
			return deposit()
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			err := cmd.MarkFlagRequired(flagKeystore)
			if err != nil {
				return err
			}
			err = cmd.MarkFlagRequired(flagAmount)
			if err != nil {
				return err
			}
			err = viper.BindPFlag(flagKeystore, cmd.Flags().Lookup(flagKeystore))
			if err != nil {
				return err
			}
			err = viper.BindPFlag(flagAmount, cmd.Flags().Lookup(flagAmount))
			if err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().String(flagKeystore, "", "Path to keystore file")
	cmd.Flags().String(flagAmount, "", "Deposit amount")
	return cmd
}
