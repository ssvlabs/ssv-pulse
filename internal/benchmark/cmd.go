package benchmark

import (
	"context"
	"sync"

	eth2client "github.com/attestantio/go-eth2-client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ssvlabsinfra/ssv-benchmark/internal/benchmark/client"
	"github.com/ssvlabsinfra/ssv-benchmark/internal/benchmark/config"
	"github.com/ssvlabsinfra/ssv-benchmark/internal/benchmark/monitor"
	"github.com/ssvlabsinfra/ssv-benchmark/internal/benchmark/monitor/listener"
	"github.com/ssvlabsinfra/ssv-benchmark/internal/platform/cmd"
	"github.com/ssvlabsinfra/ssv-benchmark/internal/platform/network"
	"github.com/ssvlabsinfra/ssv-benchmark/internal/platform/output"
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

		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		isValid, err := config.IsValid(consensusAddr, executionAddr, ssvAddr, networkName)
		if !isValid {
			panic(err.Error())
		}

		var wg sync.WaitGroup
		wg.Add(2)

		client, err := client.GetConsensus(ctx, string(consensusAddr))
		if err != nil {
			panic(err.Error())
		}

		listenerSvc := listener.New(client.(eth2client.EventsProvider))
		go func() {
			if err := listenerSvc.Start(ctx); err != nil {
				panic(err.Error())
			}
		}()

		metricSvc := New(
			network.Name(networkName),
			monitor.NewPeers(consensusAddr, executionAddr, ssvAddr),
			monitor.NewLatency(listenerSvc, network.Name(networkName)),
			monitor.NewBlocks(listenerSvc),
			monitor.NewMemory(),
			monitor.NewCPU(),
			output.NewConsole([]string{
				"Slot",
				"Latency (Min | p10 | p50 | p90 | Max)",
				"Peers (Consensus | Execution | SSV)",
				"Blocks (Received | Missed)",
				"Memory (Total | Used | Cached | Free) MB",
				"CPU (System | User)",
			}))

		go metricSvc.Start(ctx)

		wg.Wait()
		return nil
	},
}
