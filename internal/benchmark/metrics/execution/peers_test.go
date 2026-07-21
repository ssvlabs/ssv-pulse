package execution

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func newTestPeerMetric(url string) *PeerMetric {
	return NewPeerMetric(url, "Peers", time.Second, nil)
}

func TestGivenValidPeerCountResponsesWhenMeasureThenHistogramBacksAggregateResults(t *testing.T) {
	// net_peerCount returns a hex string; 0x10 == 16 peers.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":"0x10"}`))
	}))
	defer server.Close()

	p := newTestPeerMetric(server.URL)
	p.measure(context.Background())
	p.measure(context.Background())

	result := p.AggregateResults()

	// All samples are 16, so every percentile is 16.
	assert.Equal(t, "min=16, p10=16, p50=16, p90=16, max=16", result)
}

func TestGivenEmptyPeerCountResponseWhenMeasureThenAggregateResultsSurfacesUnableToMeasure(t *testing.T) {
	// An empty result string is how nodes that don't support net_peerCount
	// respond; this must surface as UNABLE_TO_MEASURE rather than a bogus 0.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":""}`))
	}))
	defer server.Close()

	p := newTestPeerMetric(server.URL)
	p.measure(context.Background())

	result := p.AggregateResults()

	assert.Contains(t, result, errUnableMeasure.Error())
}

func TestGivenUnreachableHostWhenMeasureThenRecordsZeroWithoutError(t *testing.T) {
	p := newTestPeerMetric("http://127.0.0.1:0") // guaranteed-unusable port
	p.measure(context.Background())

	result := p.AggregateResults()

	// A failed request records a 0 sample rather than an unable-to-measure
	// error, so the histogram (not the error branch) backs the report.
	assert.Equal(t, "min=0, p10=0, p50=0, p90=0, max=0", result)
}

// TestGivenConcurrentMeasureAndAggregateThenNoDataRace reproduces the
// shutdown race: service.go reads AggregateResults while measure goroutines
// may still be running, and the empty-net_peerCount path writes
// measuringErrors. Without the mutex this is a concurrent map write+read
// (fatal error under the race detector). The empty-result server forces the
// write path on every measure.
func TestGivenConcurrentMeasureAndAggregateThenNoDataRace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":""}`))
	}))
	defer server.Close()

	p := newTestPeerMetric(server.URL)

	// The writer is HTTP-bound (slow); the reader must hammer the map for the
	// entire time the writer runs, or their accesses never overlap and the
	// race goes undetected. So the reader loops until the writer signals done
	// rather than running a fixed, quickly-exhausted count.
	var wg sync.WaitGroup
	stop := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(stop)
		for range 300 {
			p.measure(context.Background())
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				_ = p.AggregateResults()
			}
		}
	}()

	wg.Wait()

	assert.Contains(t, p.AggregateResults(), errUnableMeasure.Error())
}
