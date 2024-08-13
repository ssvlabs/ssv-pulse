package ssv

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func getPeers(url string) (uint32, error) {
	var (
		resp struct {
			Advanced struct {
				Peers uint32 `json:"peers"`
			} `json:"advanced"`
		}
		peers uint32
	)
	res, err := http.Get(fmt.Sprintf("%s/v1/node/health", url))
	if err != nil {
		return peers, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return peers, fmt.Errorf("received unsuccessful status code when fetching SSV Client Peer count. Code: '%d'", res.StatusCode)
	}

	if err = json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return peers, err
	}

	return resp.Advanced.Peers, nil
}
