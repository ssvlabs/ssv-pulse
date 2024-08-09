package benchmark

import (
	"context"
	"sync"

	eth2client "github.com/attestantio/go-eth2-client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ssvlabsinfra/ssv-benchmark/configs"
	"github.com/ssvlabsinfra/ssv-benchmark/internal/benchmark/client"
	"github.com/ssvlabsinfra/ssv-benchmark/internal/benchmark/monitor"
	"github.com/ssvlabsinfra/ssv-benchmark/internal/benchmark/monitor/listener"
	"github.com/ssvlabsinfra/ssv-benchmark/internal/platform/cmd"
)

const (
	consensusAddrFlag = "consensusAddr"
	executionAddrFlag = "executionAddr"
	ssvAddrFlag       = "ssvAddr"
	networkFlag       = "network"
)

func init() {
	cmd.AddPersistentStringFlag(CMD, consensusAddrFlag, "", "Consensus client address with scheme(http/https)", true)
	cmd.AddPersistentStringFlag(CMD, executionAddrFlag, "", "Execution client address scheme(http/https)", true)
	cmd.AddPersistentStringFlag(CMD, ssvAddrFlag, "", "SSV client address scheme(http/https)", true)
	cmd.AddPersistentStringFlag(CMD, networkFlag, "", "Network to use, either 'mainnet' or 'holesky'", true)
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
		network := viper.GetString(networkFlag)

		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		cfg, err := configs.Init(configs.Address(consensusAddr), configs.Address(executionAddr), configs.Address(ssvAddr), network)
		if err != nil {
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
			cfg.Network,
			monitor.NewPeers(consensusAddr, executionAddr, ssvAddr),
			monitor.NewLatency(listenerSvc, cfg.Network),
			monitor.NewBlocks(listenerSvc))

		go metricSvc.Start(ctx)

		wg.Wait()
		return nil
	},
}
