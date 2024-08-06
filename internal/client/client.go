package client

import (
	"context"

	eth2client "github.com/attestantio/go-eth2-client"
	"github.com/attestantio/go-eth2-client/auto"
	"github.com/rs/zerolog"
)

func Get(ctx context.Context, beaconNodeAddr string) (eth2client.Service, error) {
	var client eth2client.Service
	client, err := auto.New(
		ctx,
		auto.WithLogLevel(zerolog.DebugLevel),
		auto.WithAddress(beaconNodeAddr),
	)
	if err != nil {
		return client, err
	}
	return client, nil
}
