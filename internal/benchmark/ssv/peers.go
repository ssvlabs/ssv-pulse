package ssv

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ssvlabs/ssv-benchmark/internal/platform/metric"
)

type PeerMetric struct {
	url   string
	peers []uint32
}

func NewPeerMetric(url string) *PeerMetric {
	return &PeerMetric{
		url: url,
	}
}

func (p *PeerMetric) Get() (uint32, error) {
	var (
		resp struct {
			Advanced struct {
				Peers uint32 `json:"peers"`
			} `json:"advanced"`
		}
		peerNumber uint32
	)
	res, err := http.Get(fmt.Sprintf("%s/v1/node/health", p.url))
	if err != nil {
		return peerNumber, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return peerNumber, fmt.Errorf("received unsuccessful status code when fetching SSV Client Peer count. Code: '%d'", res.StatusCode)
	}

	if err = json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return peerNumber, err
	}
	peerNumber = resp.Advanced.Peers
	p.peers = append(p.peers, peerNumber)

	return peerNumber, nil
}

func (p *PeerMetric) Aggregate() (min, p10, p50, p90, max uint32) {
	min = metric.CalculatePercentile(p.peers, 0)
	p10 = metric.CalculatePercentile(p.peers, 10)
	p50 = metric.CalculatePercentile(p.peers, 50)
	p90 = metric.CalculatePercentile(p.peers, 90)
	max = metric.CalculatePercentile(p.peers, 100)

	return
}
