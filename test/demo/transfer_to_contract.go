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

func transferToContract() error {
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
	sidechainErc20Address, err := tokenMapper.MainchainTokenToSidechainToken(&bind.CallOpts{}, testTokenAddress)
	if err != nil {
		return err
	}
	sidechainErc20, err := sidechain.NewSidechainERC20(sidechainErc20Address, sidechainClient)
	if err != nil {
		return err
	}

	log.Print("Deploying DummyApp")
	dummyAppAddress, tx, _, err := sidechain.DeployDummyApp(auth, sidechainClient, sidechainErc20Address)
	if err != nil {
		return err
	}
	receipt, err := utils.WaitMined(ctx, sidechainClient, tx, 0)
	if err != nil {
		return err
	}
	if receipt.Status != ethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("Deployment tx %x failed", receipt.TxHash)
	}
	log.Printf("Deployed DummyApp at %s\n", dummyAppAddress.Hex())

	sidechainErc20, err = sidechain.NewSidechainERC20(sidechainErc20Address, sidechainClient)
	if err != nil {
		return err
	}
	nonce, err := sidechainErc20.TransferNonces(&bind.CallOpts{}, senderAddress)
	if err != nil {
		return err
	}
	dummyApp, err := sidechain.NewDummyApp(dummyAppAddress, sidechainClient)
	amount := new(big.Int)
	amount.SetString(viper.GetString(flagAmount), 10)
	playerOneSig, err := utils.SignPackedData(
		privateKey,
		[]string{"address", "address", "address", "uint256", "uint256"},
		[]interface{}{
			senderAddress,
			dummyAppAddress,
			testTokenAddress,
			amount,
			nonce,
		},
	)
	auth.GasLimit = 2000000
	log.Info().Msg("Transferring tokens to the deployed contract")
	tx, err = dummyApp.PlayerOneDeposit(auth, playerOneSig)
	if err != nil {
		return err
	}
	receipt, err = utils.WaitMined(ctx, sidechainClient, tx, 0)
	if err != nil {
		return err
	}
	if receipt.Status != ethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("Transfer to contract tx %x failed", receipt.TxHash)
	}
	return nil
}

func TransferToContractCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer-to-contract",
		Short: "Deploy a dummy contract on the sidechain and transfer token to it",
		RunE: func(cmd *cobra.Command, args []string) error {
			return transferToContract()
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
	cmd.Flags().String(flagAmount, "", "Transfer amount")
	return cmd
}
