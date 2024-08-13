package execution

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

func getPeers(url string) (uint32, error) {
	var (
		resp struct {
			Result string `json:"result"`
		}
		peers uint32
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
		return peers, errors.Join(err, errors.New("failed to marshal RPC request to Execution node during Peers fetching"))
	}

	res, err := http.Post(url, "application/json", bytes.NewBuffer(requestBytes))
	if err != nil {
		return peers, errors.Join(err, errors.New("failed sending HTTP request to Execution node during Peers fetching"))
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return peers, fmt.Errorf("received unsuccessful status code when fetching Execution Client Peer count. Code: '%d'", res.StatusCode)
	}

	if err = json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return peers, errors.Join(err, errors.New("failed to decode RPC response from Execution node during Peers fetching"))
	}

	peerCountHex := resp.Result
	peerCount, err := strconv.ParseInt(peerCountHex[2:], 16, 64)
	if err != nil {
		return peers, errors.Join(err, errors.New("failed to convert peer count response from Execution node during Peers fetching"))
	}
	peers = uint32(peerCount)

	return peers, nil
}
