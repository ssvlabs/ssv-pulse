package consensus

import (
	"net/http"
	"time"

	"github.com/ssvlabs/ssv-benchmark/internal/platform/metric"
)

var (
	latencies []time.Duration
)

func getLatency(address string) (time.Duration, error) {
	var latency time.Duration
	start := time.Now()
	res, err := http.Get(address)
	if err != nil {
		return latency, err
	}
	defer res.Body.Close()

	end := time.Now()

	latency = end.Sub(start)

	latencies = append(latencies, latency)

	return latency, nil
}

func getAggregatedLatencyValues() (min, p10, p50, p90, max time.Duration) {
	min = metric.CalculatePercentile(latencies, 0)
	p10 = metric.CalculatePercentile(latencies, 10)
	p50 = metric.CalculatePercentile(latencies, 50)
	p90 = metric.CalculatePercentile(latencies, 90)
	max = metric.CalculatePercentile(latencies, 100)

	return
}
