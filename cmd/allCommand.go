package cmd

import "github.com/spf13/cobra"

// allCmd represents the all command
var allCmd = &cobra.Command{
	Use:   "all",
	Short: "Starts all estimations",
	Long:  `Starts all estimations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger.Info("Starting all services")
		go naiveCmd.RunE(naiveCmd, args)
		go gasExpressCmd.RunE(gasExpressCmd, args)
		return web3jCommand.RunE(web3jCommand, args)
	},
}

func init() {
	RootCmd.AddCommand(allCmd)
}
