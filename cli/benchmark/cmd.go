package benchmark_cli

import (
	"context"
	"sync"

	eth2client "github.com/attestantio/go-eth2-client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ssvlabsinfra/ssv-benchmark/cli/flags"
	"github.com/ssvlabsinfra/ssv-benchmark/src/benchmark"
	"github.com/ssvlabsinfra/ssv-benchmark/src/benchmark/client"
	"github.com/ssvlabsinfra/ssv-benchmark/src/benchmark/configs"
	"github.com/ssvlabsinfra/ssv-benchmark/src/benchmark/monitor"
	"github.com/ssvlabsinfra/ssv-benchmark/src/benchmark/monitor/listener"
)

func init() {
	flags.AddPersistentStringSliceFlag(Run, "address", []string{}, "Comma-separated list of urls, e.g. 'http://eth2-lh-mainnet-5052.bloxinfra.com,http://mainnet-standalone-v3.bloxinfra.com:5052'", true)
	flags.AddPersistentStringFlag(Run, "network", "", "Network to use, either 'mainnet' or 'holesky'", true)
}

var Run = &cobra.Command{
	Use:   "benchmark",
	Short: "Run benchmarks of ssv node",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlag("address", cmd.PersistentFlags().Lookup("address")); err != nil {
			return err
		}
		if err := viper.BindPFlag("network", cmd.PersistentFlags().Lookup("network")); err != nil {
			return err
		}
		addresses := viper.GetStringSlice("address")
		network := viper.GetString("network")
		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		cfg, err := configs.Init(addresses, network)
		if err != nil {
			panic(err.Error())
		}

		var wg sync.WaitGroup
		for _, addr := range cfg.BeaconNodeAddrs {
			wg.Add(2)
			client, err := client.Get(ctx, string(addr))
			if err != nil {
				panic(err.Error())
			}

			listenerSvc := listener.New(client.(eth2client.EventsProvider))
			go func() {
				if err := listenerSvc.Start(ctx); err != nil {
					panic(err.Error())
				}
			}()

			metricSvc := benchmark.NewService(
				string(addr),
				cfg.Network,
				monitor.NewPeers(string(addr)),
				monitor.NewLatency(listenerSvc, cfg.Network),
				monitor.NewBlocks(listenerSvc))
			go metricSvc.Start(ctx)
		}

		wg.Wait()
		return nil
	},
}
