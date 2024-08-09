package benchmark

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/aquasecurity/table"
	"github.com/attestantio/go-eth2-client/spec/phase0"

	"github.com/ssvlabsinfra/ssv-benchmark/configs"
	"github.com/ssvlabsinfra/ssv-benchmark/internal/benchmark/client"
)

type (
	peersMonitor interface {
		Measure() (map[client.Type]uint32, error)
	}
	latencyMonitor interface {
		Measure(slot phase0.Slot) (min, max, avg time.Duration)
	}
	blocksMonitor interface {
		Measure(slot phase0.Slot) (received, missed uint32)
	}

	memoryMonitor interface {
		Measure() (total, used, cached, free float64, err error)
	}

	Service struct {
		network        configs.NetworkName
		peersMonitor   peersMonitor
		latencyMonitor latencyMonitor
		blocksMonitor  blocksMonitor
		memoryMonitor  memoryMonitor
	}
)

func New(
	network configs.NetworkName,
	peersMonitor peersMonitor,
	latencyMonitor latencyMonitor,
	blocksMonitor blocksMonitor,
	memoryMonitor memoryMonitor,
) *Service {
	return &Service{
		network:        network,
		peersMonitor:   peersMonitor,
		latencyMonitor: latencyMonitor,
		blocksMonitor:  blocksMonitor,
		memoryMonitor:  memoryMonitor,
	}
}

func (s *Service) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			slog.Info("cancellation context received. Terminating service")
		default:
			startSlot := currentSlot(configs.GenesisTime[s.network]) + 1
			slot := startSlot

			for {
				time.Sleep(time.Until(slotTime(configs.GenesisTime[s.network], slot).Add(time.Second * 4)))
				min, max, avg := s.latencyMonitor.Measure(slot)

				peers, err := s.peersMonitor.Measure()
				if err != nil {
					slog.With("err", err.Error()).Error("error fetching peer count")
				}
				received, missed := s.blocksMonitor.Measure(slot)

				total, used, cached, free, err := s.memoryMonitor.Measure()
				if err != nil {
					slog.With("err", err.Error()).Error("error fetching memory metric")
				}

				tbl := table.New(os.Stdout)
				tbl.SetHeaders("Slot", "Latency (Min | Avg | Max)", "Peers (Consensus | Execution | SSV)", "Blocks (Received | Missed)", "Memory (Total | Used | Cached | Free) MB")
				tbl.AddRow(
					fmt.Sprintf("%d", slot),
					fmt.Sprintf("%s | %s | %s", min, avg, max),
					fmt.Sprintf("%d | %d | %d", peers[client.Consensus], peers[client.Execution], peers[client.SSV]),
					fmt.Sprintf("%d | %d", received, missed),
					fmt.Sprintf("%.2f | %.2f | %.2f | %.2f", total, used, cached, free),
				)
				tbl.Render()
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
