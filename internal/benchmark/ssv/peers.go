package ssv

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ssvlabs/ssv-benchmark/internal/platform/metric"
)

var peers []uint32

func getPeers(url string) (uint32, error) {
	var (
		resp struct {
			Advanced struct {
				Peers uint32 `json:"peers"`
			} `json:"advanced"`
		}
		peerNumber uint32
	)
	res, err := http.Get(fmt.Sprintf("%s/v1/node/health", url))
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
	peers = append(peers, peerNumber)

	return peerNumber, nil
}

func getAggregatedPeersValues() (min, p10, p50, p90, max uint32) {
	min = metric.CalculatePercentile(peers, 0)
	p10 = metric.CalculatePercentile(peers, 10)
	p50 = metric.CalculatePercentile(peers, 50)
	p90 = metric.CalculatePercentile(peers, 90)
	max = metric.CalculatePercentile(peers, 100)

	return
}
