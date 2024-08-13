package benchmark

import (
	"context"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ssvlabsinfra/ssv-benchmark/internal/benchmark/config"
	"github.com/ssvlabsinfra/ssv-benchmark/internal/benchmark/consensus"
	"github.com/ssvlabsinfra/ssv-benchmark/internal/benchmark/execution"
	"github.com/ssvlabsinfra/ssv-benchmark/internal/benchmark/infrastructure"
	"github.com/ssvlabsinfra/ssv-benchmark/internal/benchmark/ssv"
	"github.com/ssvlabsinfra/ssv-benchmark/internal/platform/cmd"
	"github.com/ssvlabsinfra/ssv-benchmark/internal/platform/lifecycle"
	"github.com/ssvlabsinfra/ssv-benchmark/internal/platform/metric"
)

const (
	consensusAddrFlag = "consensusAddr"
	executionAddrFlag = "executionAddr"
	ssvAddrFlag       = "ssvAddr"
	networkFlag       = "network"
)

func init() {
	cmd.AddPersistentStringFlag(CMD, consensusAddrFlag, "", "Consensus client address (beacon node API) with scheme (HTTP/HTTPS) and port, e.g. https://lighthouse:5052", true)
	cmd.AddPersistentStringFlag(CMD, executionAddrFlag, "", "Execution client address with scheme (HTTP/HTTPS) and port, e.g. https://geth:8545", true)
	cmd.AddPersistentStringFlag(CMD, ssvAddrFlag, "", "SSV API address with scheme (HTTP/HTTPS) and port, e.g. http://ssv-node:16000", true)
	cmd.AddPersistentStringFlag(CMD, networkFlag, "", "Ethereum network to use, either 'mainnet' or 'holesky'", true)
}

var CMD = &cobra.Command{
	Use:   "benchmark",
	Short: "Run benchmarks of ssv node",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlag(consensusAddrFlag, cmd.PersistentFlags().Lookup(consensusAddrFlag)); err != nil {
			return err
		}
		if err := viper.BindPFlag(executionAddrFlag, cmd.PersistentFlags().Lookup(executionAddrFlag)); err != nil {
			return err
		}
		if err := viper.BindPFlag(ssvAddrFlag, cmd.PersistentFlags().Lookup(ssvAddrFlag)); err != nil {
			return err
		}
		if err := viper.BindPFlag(networkFlag, cmd.PersistentFlags().Lookup(networkFlag)); err != nil {
			return err
		}
		consensusAddr := viper.GetString(consensusAddrFlag)
		executionAddr := viper.GetString(executionAddrFlag)
		ssvAddr := viper.GetString(ssvAddrFlag)
		networkName := viper.GetString(networkFlag)

		ctx, cancel := context.WithCancel(context.Background())

		isValid, err := config.IsValid(consensusAddr, executionAddr, ssvAddr, networkName)
		if !isValid {
			panic(err.Error())
		}

		metricService := New(map[metric.Group]MetricService{
			metric.ConsensusGroup:      consensus.New(consensusAddr),
			metric.ExecutionGroup:      execution.New(executionAddr),
			metric.SSVGroup:            ssv.New(ssvAddr),
			metric.InfrastructureGroup: infrastructure.New(),
		})

		go metricService.Start(ctx)

		lifecycle.ListenForApplicationShutDown(func() {
			cancel()
			slog.Info("terminating the application")
		}, make(chan os.Signal))
		return nil
	},
}
