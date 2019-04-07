package cmd

import (
	"os"

	"github.com/mariusgiger/ethereum-feeestimator/pkg/utils"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger    *zap.Logger
	rpcClient *utils.CachedRPCClient
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "estimator",
	Short: "Ethereum fee estimator",
	Long:  `Ethereum fee estimator.`,
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		logger.Fatal("Something somewhere went terribly wrong", zap.Error(err))
		os.Exit(-1)
	}
}

func init() {
	cfg := zap.NewDevelopmentConfig()
	cfg.OutputPaths = append(cfg.OutputPaths, "./output/estimator.log")

	var err error
	logger, err = cfg.Build(zap.AddStacktrace(zapcore.DPanicLevel))
	if err != nil {
		panic("could not create logger")
	}

	rpcClient = utils.NewCachedRPCClient(logger)
}
