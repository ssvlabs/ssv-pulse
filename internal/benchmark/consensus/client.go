package consensus

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type ClientVersionMetric struct {
	url string
}

func NewClientVersionMetric(url string) *ClientVersionMetric {
	return &ClientVersionMetric{
		url: url,
	}
}

func (c *ClientVersionMetric) Get() (string, error) {
	var (
		resp struct {
			Data struct {
				Version string `json:"version"`
			} `json:"data"`
		}
	)
	res, err := http.Get(fmt.Sprintf("%s/eth/v1/node/version", c.url))
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received unsuccessful status code when fetching Consensus Client Version. Code: '%d'", res.StatusCode)
	}

	if err = json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return "", err
	}

	return resp.Data.Version, nil
}
