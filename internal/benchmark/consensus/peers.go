package consensus

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

func getPeers(url string) (uint16, error) {
	var (
		resp struct {
			Data struct {
				Connected string `json:"connected"`
			} `json:"data"`
		}
		peers uint16
	)
	res, err := http.Get(fmt.Sprintf("%s/eth/v1/node/peer_count", url))
	if err != nil {
		return peers, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return peers, fmt.Errorf("received unsuccessful status code when fetching Consensus Client Peer count. Code: '%d'", res.StatusCode)
	}

	if err = json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return peers, err
	}

	convertedPeers, err := strconv.Atoi(resp.Data.Connected)
	if err != nil {
		return uint16(convertedPeers), err
	}
	return uint16(convertedPeers), nil
}
