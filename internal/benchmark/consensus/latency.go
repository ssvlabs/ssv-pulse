package consensus

import (
	"net/http"
	"time"

	"github.com/ssvlabsinfra/ssv-benchmark/internal/platform/metric"
)

var (
	min, max  time.Duration
	latencies []time.Duration
)

func getLatency(address string) (Min, P10, P50, P90, Max time.Duration, err error) {
	start := time.Now()
	res, err := http.Get(address)
	if err != nil {
		return Min, P10, P50, P90, Max, err
	}
	defer res.Body.Close()

	end := time.Now()

	latency := end.Sub(start)

	if len(latencies) == 0 {
		min = latency
	} else if latency < min {
		min = latency
	}
	if latency > max {
		max = latency
	}

	latencies = append(latencies, latency)
	p10 := metric.CalculatePercentile(latencies, 10)
	p50 := metric.CalculatePercentile(latencies, 50)
	p90 := metric.CalculatePercentile(latencies, 90)

	return min, p10, p50, p90, max, err
}
