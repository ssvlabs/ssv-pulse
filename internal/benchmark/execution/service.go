package execution

import (
	"context"
	"time"

	"github.com/ssvlabs/ssv-benchmark/internal/platform/logger"
	"github.com/ssvlabs/ssv-benchmark/internal/platform/metric"
)

const (
	Peers   metric.Name = "Peers"
	Latency metric.Name = "Latency"
)

type Service struct {
	apiURL   string
	interval time.Duration
}

func New(apiURL string) *Service {
	return &Service{
		apiURL:   apiURL,
		interval: time.Second * 5,
	}
}

func (s *Service) Start(ctx context.Context) (map[metric.Name][]byte, error) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			peers, err := getPeers(s.apiURL)
			if err != nil {
				logger.WriteError(metric.ExecutionGroup, Peers, err)
			} else {
				logger.WriteMetric(metric.ExecutionGroup, Peers, map[string]any{"peers": peers})
			}
		case <-ctx.Done():
			return map[metric.Name][]byte{
				Peers: []byte(metric.FormatPercentiles(getAggregatedPeersValues())),
			}, ctx.Err()
		}
	}
}
