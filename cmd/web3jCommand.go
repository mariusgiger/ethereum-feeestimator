package cmd

import (
	"github.com/spf13/cobra"
	"github.com/mariusgiger/ethereum-feeestimator/pkg/web3j"
)

var web3jCommand = &cobra.Command{
	Use:   "web3j",
	Short: "Suggests a gas price using the time based web3j algorithm",
	Long:  `Suggests a gas price using the time based web3j algorithm.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		estimator := web3j.NewEstimator(logger, rpcClient)
		return estimator.Run()
	},
}

func init() {
	RootCmd.AddCommand(web3jCommand)
}
