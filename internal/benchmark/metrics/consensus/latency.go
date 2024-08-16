package consensus

import (
	"net/http"
	"time"

	"github.com/ssvlabs/ssv-benchmark/internal/platform/logger"
	"github.com/ssvlabs/ssv-benchmark/internal/platform/metric"
)

const (
	Duration = "Duration"
)

type LatencyMetric struct {
	metric.Base[time.Duration]
	url string
}

func NewLatencyMetric(url, name string, healthCondition []metric.HealthCondition[time.Duration]) *LatencyMetric {
	return &LatencyMetric{
		url: url,
		Base: metric.Base[time.Duration]{
			HealthConditions: healthCondition,
			Name:             name,
		},
	}
}

func (l *LatencyMetric) Measure() {
	var latency time.Duration
	start := time.Now()
	res, err := http.Get(l.url)
	if err != nil {
		l.AddDataPoint(map[string]time.Duration{
			Duration: 0,
		})
		logger.WriteError(metric.ConsensusGroup, l.Name, err)
		return
	}
	defer res.Body.Close()

	end := time.Now()

	latency = end.Sub(start)

	l.AddDataPoint(map[string]time.Duration{
		Duration: latency,
	})

	logger.WriteMetric(metric.ConsensusGroup, l.Name, map[string]any{"duration": latency})
}

func (p *LatencyMetric) AggregateResults() string {
	var values []time.Duration
	for _, point := range p.DataPoints {
		values = append(values, point.Values[Duration])
	}
	return metric.FormatPercentiles(
		metric.CalculatePercentile(values, 0),
		metric.CalculatePercentile(values, 10),
		metric.CalculatePercentile(values, 50),
		metric.CalculatePercentile(values, 90),
		metric.CalculatePercentile(values, 100))
}
