package demo

import (
	"context"
	"fmt"
	"math/big"

	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/spf13/cobra"

	"github.com/celer-network/go-rollup/utils"
	"github.com/celer-network/rollup-contracts/bindings/go/sidechain"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const (
	flagRecipient = "recipient"
)

func transferToAccount() error {
	ethClientInfo, err := initEthClientInfo()
	if err != nil {
		return err
	}
	sidechainClient := ethClientInfo.sidechainClient
	privateKey := ethClientInfo.privateKey
	auth := ethClientInfo.auth
	ctx := context.Background()

	testTokenAddress := common.HexToAddress(viper.GetString(configMainchainContractsTestToken))
	tokenMapperAddress := common.HexToAddress(viper.GetString(configSidechainContractsTokenMapper))
	tokenMapper, err := sidechain.NewTokenMapper(tokenMapperAddress, sidechainClient)
	if err != nil {
		return err
	}

	senderAddress := auth.From
	sidechainErc20Address, err :=
		tokenMapper.MainchainTokenToSidechainToken(&bind.CallOpts{}, testTokenAddress)
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

	// Transfer on the sidechain
	recipientAddress := common.HexToAddress(viper.GetString(flagRecipient))
	nonce, err := sidechainErc20.TransferNonces(&bind.CallOpts{}, senderAddress)
	if err != nil {
		return err
	}
	amount := new(big.Int)
	amount.SetString(viper.GetString(flagAmount), 10)
	signature, err := utils.SignPackedData(
		privateKey,
		[]string{"address", "address", "address", "uint256", "uint256"},
		[]interface{}{
			senderAddress,
			recipientAddress,
			testTokenAddress,
			amount,
			nonce,
		},
	)
	log.Info().Msg("Transferring on sidechain from sender to recipient")
	tx, err := sidechainErc20.Transfer(auth, senderAddress, recipientAddress, amount, signature)
	if err != nil {
		return err
	}
	receipt, err := utils.WaitMined(ctx, sidechainClient, tx, 0)
	if err != nil {
		return err
	}
	if receipt.Status != ethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("Transfer tx %x failed", receipt.TxHash)
	}

	balanceAfter, err := sidechainErc20.BalanceOf(&bind.CallOpts{}, senderAddress)
	if err != nil {
		return err
	}
	log.Info().Str("balanceAfter", balanceAfter.String()).Send()
	return nil
}

func TransferToAccountCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer-to-account",
		Short: "Transfer to an external account on the sidechain",
		RunE: func(cmd *cobra.Command, args []string) error {
			return transferToAccount()
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			err := cmd.MarkFlagRequired(flagKeystore)
			if err != nil {
				return err
			}
			err = cmd.MarkFlagRequired(flagRecipient)
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
			err = viper.BindPFlag(flagRecipient, cmd.Flags().Lookup(flagRecipient))
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
	cmd.Flags().String(flagRecipient, "", "Recipient ETH address")
	cmd.Flags().String(flagAmount, "", "Transfer amount")
	return cmd
}
