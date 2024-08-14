package consensus

import (
	"net/http"
	"time"

	"github.com/ssvlabs/ssv-benchmark/internal/platform/metric"
)

type LatencyMetric struct {
	url       string
	latencies []time.Duration
}

func NewLatencyMetric(url string) *LatencyMetric {
	return &LatencyMetric{
		url: url,
	}
}

func (l *LatencyMetric) Get() (time.Duration, error) {
	var latency time.Duration
	start := time.Now()
	res, err := http.Get(l.url)
	if err != nil {
		return latency, err
	}
	defer res.Body.Close()

	end := time.Now()

	latency = end.Sub(start)

	l.latencies = append(l.latencies, latency)

	return latency, nil
}

func (l *LatencyMetric) Aggregate() (min, p10, p50, p90, max time.Duration) {
	min = metric.CalculatePercentile(l.latencies, 0)
	p10 = metric.CalculatePercentile(l.latencies, 10)
	p50 = metric.CalculatePercentile(l.latencies, 50)
	p90 = metric.CalculatePercentile(l.latencies, 90)
	max = metric.CalculatePercentile(l.latencies, 100)

	return
}
