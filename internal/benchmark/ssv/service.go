package ssv

import (
	"context"
	"time"

	"github.com/ssvlabs/ssv-benchmark/internal/platform/logger"
	"github.com/ssvlabs/ssv-benchmark/internal/platform/metric"
)

const (
	Peers metric.Name = "Peers"
)

type Service struct {
	apiURL string
}

func New(apiURL string) *Service {
	return &Service{
		apiURL: apiURL,
	}
}

func (s *Service) Start(ctx context.Context) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			peers, err := getPeers(s.apiURL)
			if err != nil {
				logger.WriteError(metric.SSVGroup, Peers, err)
			} else {
				logger.WriteMetric(metric.SSVGroup, Peers, map[string]any{"peers": peers})
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
