package monitor

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

type (
	PeersMonitor struct {
		addr string
	}
	peersRPCResponse struct {
		Data struct {
			Connected string `json:"connected"`
		} `json:"data"`
	}
)

func NewPeers(addr string) *PeersMonitor {
	return &PeersMonitor{
		addr: addr,
	}
}

func (p *PeersMonitor) Measure() (uint32, error) {
	var (
		resp  peersRPCResponse
		peers int
	)
	res, err := http.Get(fmt.Sprintf("%s/eth/v1/node/peer_count", p.addr))
	if err != nil {
		return uint32(peers), err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return uint32(peers), err
	}
	if err = json.Unmarshal(body, &resp); err != nil {
		return uint32(peers), err
	}

	peers, err = strconv.Atoi(resp.Data.Connected)
	if err != nil {
		return uint32(peers), err
	}
	return uint32(peers), nil
}
