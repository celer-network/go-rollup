package main

import (
	"github.com/rs/zerolog/log"

	"github.com/celer-network/go-rollup/test/demo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	flagConfig = "config"
)

func main() {
	cobra.EnableCommandSorting = false
	log.Logger = log.With().Caller().Logger()

	rootCmd := &cobra.Command{
		Use:   "rollupdemo",
		Short: "rollup demo program",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			err := viper.BindPFlags(cmd.Flags())
			if err != nil {
				return err
			}
			viper.SetConfigFile(viper.GetString(flagConfig))
			return viper.ReadInConfig()
		},
	}

	// Construct Root Command
	rootCmd.AddCommand(
		demo.DepositCommand(),
		demo.TransferToAccountCommand(),
		demo.TransferToContractCommand(),
	)

	rootCmd.PersistentFlags().String(flagConfig, "./config/ropsten/config.yaml", "config path")
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal().Err(err).Send()
	}
}
