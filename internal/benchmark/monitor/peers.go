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

	ssvPeers, err := p.fetchSSVPeers()
	if err != nil {
		return peers, errors.Join(err, errors.New("failed to fetch SSV client peers"))
	}

	peers[client.Consensus] = consensusPeers
	peers[client.SSV] = ssvPeers

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

	if res.StatusCode != http.StatusOK {
		return peers, fmt.Errorf("received unsuccessful status code when fetching Consensus Client Peer count. Code: '%d'", res.StatusCode)
	}

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

func (p *PeersMonitor) fetchSSVPeers() (uint32, error) {
	var (
		resp struct {
			Advanced struct {
				Peers uint32 `json:"peers"`
			} `json:"advanced"`
		}
		peers uint32
	)
	res, err := http.Get(fmt.Sprintf("%s/v1/node/health", p.ssvClientAddr))
	if err != nil {
		return peers, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return peers, fmt.Errorf("received unsuccessful status code when fetching SSV Client Peer count. Code: '%d'", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return peers, err
	}
	if err = json.Unmarshal(body, &resp); err != nil {
		return peers, err
	}

	return resp.Advanced.Peers, nil
}
