package monitor

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
		return peers, errors.Join(err, errors.New("failed to fetch Consensus client peers"))
	}
	peers[client.Consensus] = consensusPeers

	ssvPeers, err := p.fetchSSVPeers()
	if err != nil {
		return peers, errors.Join(err, errors.New("failed to fetch SSV client peers"))
	}
	peers[client.SSV] = ssvPeers

	executionPeers, err := p.fetchExecutionPeers()
	if err != nil {
		return peers, errors.Join(err, errors.New("failed to fetch Execution client peers"))
	}
	peers[client.Execution] = executionPeers

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

	if err = json.NewDecoder(res.Body).Decode(&resp); err != nil {
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

	if err = json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return peers, err
	}

	return resp.Advanced.Peers, nil
}

func (p *PeersMonitor) fetchExecutionPeers() (uint32, error) {
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

	res, err := http.Post(p.executionClientAddr, "application/json", bytes.NewBuffer(requestBytes))
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
