package execution

import (
	"bytes"
	"encoding/json"
	"errors"
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
			Result string `json:"result"`
		}
		peerNumber uint32
	)

	request := struct {
		Jsonrpc string `json:"jsonrpc"`
		Method  string `json:"method"`
	}{
		Jsonrpc: "2.0",
		Method:  "net_peerCount",
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return peerNumber, errors.Join(err, errors.New("failed to marshal RPC request to Execution node during Peers fetching"))
	}

	res, err := http.Post(p.url, "application/json", bytes.NewBuffer(requestBytes))
	if err != nil {
		return peerNumber, errors.Join(err, errors.New("failed sending HTTP request to Execution node during Peers fetching"))
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return peerNumber, fmt.Errorf("received unsuccessful status code when fetching Execution Client Peer count. Code: '%d'", res.StatusCode)
	}

	if err = json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return peerNumber, errors.Join(err, errors.New("failed to decode RPC response from Execution node during Peers fetching"))
	}

	peerCountHex := resp.Result
	peerCount, err := strconv.ParseInt(peerCountHex[2:], 16, 64)
	if err != nil {
		return peerNumber, errors.Join(err, errors.New("failed to convert peer count response from Execution node during Peers fetching"))
	}
	peerNumber = uint32(peerCount)

	p.peers = append(p.peers, uint32(peerNumber))

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
