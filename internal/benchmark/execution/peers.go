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

var peers []uint32

func getPeers(url string) (uint32, error) {
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

	res, err := http.Post(url, "application/json", bytes.NewBuffer(requestBytes))
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

	peers = append(peers, uint32(peerNumber))

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
