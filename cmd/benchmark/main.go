package main

import (
	"context"
	"sync"

	eth2client "github.com/attestantio/go-eth2-client"
	"github.com/ssvlabsinfra/ssv-benchmark/configs"
	metric "github.com/ssvlabsinfra/ssv-benchmark/internal"
	"github.com/ssvlabsinfra/ssv-benchmark/internal/client"
	"github.com/ssvlabsinfra/ssv-benchmark/internal/monitor"
	"github.com/ssvlabsinfra/ssv-benchmark/internal/monitor/listener"
)

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cfg, err := configs.Init()
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

		metricSvc := metric.NewService(
			string(addr),
			cfg.Network,
			monitor.NewPeers(string(addr)),
			monitor.NewLatency(listenerSvc, cfg.Network),
			monitor.NewBlocks(listenerSvc))
		go metricSvc.Start(ctx)
	}

	wg.Wait()
}
