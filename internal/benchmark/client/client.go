package client

import (
	"context"

	eth2client "github.com/attestantio/go-eth2-client"
	"github.com/attestantio/go-eth2-client/auto"
	"github.com/rs/zerolog"
)

type Type string

const (
	Consensus Type = "Consensus"
	Execution Type = "Execution"
	SSV       Type = "SSV"
)

func GetConsensus(ctx context.Context, addr string) (eth2client.Service, error) {
	var client eth2client.Service
	client, err := auto.New(
		ctx,
		auto.WithLogLevel(zerolog.DebugLevel),
		auto.WithAddress(addr),
	)
	if err != nil {
		return client, err
	}
	return client, nil
}
