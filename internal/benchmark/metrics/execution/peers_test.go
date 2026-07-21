package execution

import (
	"context"
	"net/http"
	"net/http/httptest"
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
