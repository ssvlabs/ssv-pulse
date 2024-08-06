package internal

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/aquasecurity/table"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ssvlabsinfra/ssv-benchmark/configs"
)

type (
	peersMonitor interface {
		Measure() (uint32, error)
	}
	latencyMonior interface {
		Measure(slot phase0.Slot) (min, max, avg time.Duration)
	}
	blocksMonitor interface {
		Measure(slot phase0.Slot) (received, missed uint32)
	}

	Service struct {
		addr          string
		network       configs.NetworkName
		peersMonitor  peersMonitor
		latencyMonior latencyMonior
		blocksMonior  blocksMonitor
	}
)

func NewService(
	addr string,
	network configs.NetworkName,
	peersMonitor peersMonitor,
	latencyMonitor latencyMonior,
	blocksMonitor blocksMonitor,
) *Service {
	return &Service{
		addr:          addr,
		network:       network,
		peersMonitor:  peersMonitor,
		latencyMonior: latencyMonitor,
		blocksMonior:  blocksMonitor,
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
				min, max, avg := s.latencyMonior.Measure(slot)

				peers, err := s.peersMonitor.Measure()
				if err != nil {
					slog.Error("error fetching peer count")
				}
				received, missed := s.blocksMonior.Measure(slot)

				tbl := table.New(os.Stdout)
				tbl.SetHeaders("Address", "Slot", "Latency (Min/Avg/Max)", "Peers", "Blocks (Received/Missed)")
				tbl.AddRow(
					s.addr,
					fmt.Sprintf("%d", slot),
					fmt.Sprintf("%s/%s/%s", min, avg, max),
					fmt.Sprintf("%d", peers),
					fmt.Sprintf("%d/%d", received, missed),
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
