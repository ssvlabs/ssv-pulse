package benchmark

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"

	"github.com/ssvlabsinfra/ssv-benchmark/internal/benchmark/client"
	"github.com/ssvlabsinfra/ssv-benchmark/internal/benchmark/monitor"
	"github.com/ssvlabsinfra/ssv-benchmark/internal/platform/network"
)

type (
	peersMonitor interface {
		Measure() (map[client.Type]uint32, error)
	}
	latencyMonitor interface {
		Measure(slot phase0.Slot) monitor.Latency
	}
	blocksMonitor interface {
		Measure(slot phase0.Slot) (received, missed uint32)
	}

	memoryMonitor interface {
		Measure() (total, used, cached, free float64, err error)
	}

	cpuMonitor interface {
		Measure() (system, user float64, err error)
	}

	console interface {
		Update(values []string)
	}

	Service struct {
		network        network.Name
		peersMonitor   peersMonitor
		latencyMonitor latencyMonitor
		blocksMonitor  blocksMonitor
		memoryMonitor  memoryMonitor
		cpuMonitor     cpuMonitor
		console        console
	}
)

func New(
	network network.Name,
	peersMonitor peersMonitor,
	latencyMonitor latencyMonitor,
	blocksMonitor blocksMonitor,
	memoryMonitor memoryMonitor,
	cpuMonitor cpuMonitor,
	console console,
) *Service {
	return &Service{
		network:        network,
		peersMonitor:   peersMonitor,
		latencyMonitor: latencyMonitor,
		blocksMonitor:  blocksMonitor,
		memoryMonitor:  memoryMonitor,
		cpuMonitor:     cpuMonitor,
		console:        console,
	}
}

func (s *Service) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			startSlot := currentSlot(network.GenesisTime[s.network]) + 1
			slot := startSlot

			for {
				time.Sleep(time.Until(slotTime(network.GenesisTime[s.network], slot).Add(time.Second * 4)))
				latency := s.latencyMonitor.Measure(slot)

				peers, err := s.peersMonitor.Measure()
				if err != nil {
					slog.With("err", err.Error()).Error("error fetching peer count")
				}
				received, missed := s.blocksMonitor.Measure(slot)

				total, used, cached, free, err := s.memoryMonitor.Measure()
				if err != nil {
					slog.With("err", err.Error()).Error("error fetching memory metric")
				}

				systemCPUPercent, userCPUPercent, err := s.cpuMonitor.Measure()
				if err != nil {
					slog.With("err", err.Error()).Error("error fetching CPU metric")
				}

				s.console.Update([]string{
					fmt.Sprintf("%d", slot),
					fmt.Sprintf("%s | %s | %s | %s | %s", latency.Min, latency.P10, latency.P50, latency.P90, latency.Max),
					fmt.Sprintf("%d | %d | %d", peers[client.Consensus], peers[client.Execution], peers[client.SSV]),
					fmt.Sprintf("%d | %d", received, missed),
					fmt.Sprintf("%.2f | %.2f | %.2f | %.2f", total, used, cached, free),
					fmt.Sprintf("%f %% | %f %%", systemCPUPercent, userCPUPercent),
				})

				slot++
			}

		}
	}
}

func currentSlot(genesisTime time.Time) phase0.Slot {
	return phase0.Slot(time.Since(genesisTime) / (12 * time.Second))
}

func slotTime(genesisTime time.Time, slot phase0.Slot) time.Time {
	return genesisTime.Add(time.Duration(slot) * 12 * time.Second)
}
