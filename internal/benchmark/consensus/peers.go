package consensus

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ssvlabs/ssv-benchmark/internal/platform/metric"
)

var peers []uint32

func getPeers(url string) (uint32, error) {
	var (
		resp struct {
			Data struct {
				Connected string `json:"connected"`
			} `json:"data"`
		}
		peerNumber uint32
	)
	res, err := http.Get(fmt.Sprintf("%s/eth/v1/node/peer_count", url))
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
