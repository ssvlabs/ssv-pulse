package monitor

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/ssvlabsinfra/ssv-benchmark/internal/benchmark/client"
)

type (
	PeersMonitor struct {
		consensusClientAddr, executionClientAddr, ssvClientAddr string
	}
)

func NewPeers(consensusClientAddr, executionClientAddr, ssvClientAddr string) *PeersMonitor {
	return &PeersMonitor{
		consensusClientAddr: consensusClientAddr,
		executionClientAddr: executionClientAddr,
		ssvClientAddr:       ssvClientAddr,
	}
}

func (p *PeersMonitor) Measure() (map[client.Type]uint32, error) {
	peers := make(map[client.Type]uint32)
	consensusPeers, err := p.fetchConsensusPeers()
	if err != nil {
		return peers, errors.Join(err, errors.New("failed to fetch consensus client peers"))
	}
	peers[client.Consensus] = consensusPeers

	return peers, nil
}

func (p *PeersMonitor) fetchConsensusPeers() (uint32, error) {
	var (
		resp struct {
			Data struct {
				Connected string `json:"connected"`
			} `json:"data"`
		}
		peers uint32
	)
	res, err := http.Get(fmt.Sprintf("%s/eth/v1/node/peer_count", p.consensusClientAddr))
	if err != nil {
		return peers, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return peers, err
	}
	if err = json.Unmarshal(body, &resp); err != nil {
		return peers, err
	}

	convertedPeers, err := strconv.Atoi(resp.Data.Connected)
	if err != nil {
		return uint32(convertedPeers), err
	}
	return uint32(convertedPeers), nil
}
