package cmd

import (
	"github.com/spf13/cobra"
	"github.com/mariusgiger/ethereum-feeestimator/pkg/gasstation/express"
)

var gasExpressCmd = &cobra.Command{
	Use:   "express",
	Short: "Suggests a gas price using the gas station express algorithm",
	Long:  `Suggests a gas price using the gas station express algorithm.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		estimator := express.NewEstimator(logger, rpcClient)
		return estimator.Run()
	},
}

func init() {
	RootCmd.AddCommand(gasExpressCmd)
}
