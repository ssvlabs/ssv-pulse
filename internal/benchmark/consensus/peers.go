package consensus

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

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
			Data struct {
				Connected string `json:"connected"`
			} `json:"data"`
		}
		peerNumber uint32
	)
	res, err := http.Get(fmt.Sprintf("%s/eth/v1/node/peer_count", p.url))
	if err != nil {
		return peerNumber, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return peerNumber, fmt.Errorf("received unsuccessful status code when fetching Consensus Client Peer count. Code: '%d'", res.StatusCode)
	}

	if err = json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return peerNumber, err
	}

	peerNr, err := strconv.Atoi(resp.Data.Connected)
	if err != nil {
		return peerNumber, err
	}
	peerNumber = uint32(peerNr)
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
