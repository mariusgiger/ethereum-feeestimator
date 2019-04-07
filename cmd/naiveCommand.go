package cmd

import (
	"github.com/ethereum/go-ethereum/eth/gasprice"
	"github.com/spf13/cobra"
	"github.com/mariusgiger/ethereum-feeestimator/pkg/naive"
)

var naiveCmd = &cobra.Command{
	Use:   "naive",
	Short: "Suggests a naive gas price",
	Long:  `Suggests a naive gas price.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		config := gasprice.Config{
			Blocks:     naiveOptions.numberOfBlocks,
			Percentile: naiveOptions.percentile,
		}
		estimator := naive.NewEstimator(logger, config, rpcClient)
		return estimator.Run()
	},
}

var (
	naiveOptions struct {
		numberOfBlocks int
		percentile     int
	}
)

func init() {
	RootCmd.AddCommand(naiveCmd)

	//TODO find a good value
	naiveCmd.Flags().IntVarP(&naiveOptions.numberOfBlocks, "numberOfBlocks", "n", 20, "number of blocks that are used for the estimate")
	naiveCmd.Flags().IntVarP(&naiveOptions.percentile, "percentile", "p", 60, "percentile of gas prices to be used (value between 1-100, higher is safer)")
}
