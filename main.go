package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	eth2client "github.com/attestantio/go-eth2-client"
	"github.com/attestantio/go-eth2-client/api"
	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/auto"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/carlmjohnson/requests"
	"github.com/rs/zerolog"
	"github.com/sourcegraph/conc/pool"
	"golang.org/x/exp/maps"

	"github.com/aquasecurity/table"
)

type AddressInfo struct {
	URL  string
	Name string
}

func main() {
	ctx := context.Background()

	// Define and parse command-line flags
	addressesFlag := flag.String("addresses", "", "Comma-separated list of name=url pairs, e.g. 'Beacon1=http://eth2-lh-mainnet-5052.bloxinfra.com,Beacon 2=http://mainnet-standalone-v3.bloxinfra.com:5052'")
	flag.Parse()

	// Parse addresses
	if *addressesFlag == "" {
		log.Fatal("addresses flag is required")
	}
	addresses := parseAddresses(*addressesFlag)

	receivals := map[phase0.Slot]map[string]time.Time{}
	knownSlotRoots := map[phase0.Slot]phase0.Root{}
	receivedBlockRoots := map[phase0.Slot]map[string]phase0.Root{}
	peers := map[string]int{}
	mu := sync.Mutex{}

	for _, addressInfo := range addresses {
		go func(addressInfo AddressInfo) {
			client, err := auto.New(
				ctx,
				auto.WithLogLevel(zerolog.DebugLevel),
				auto.WithAddress(addressInfo.URL),
			)
			if err != nil {
				panic(err)
			}
			err = client.(eth2client.EventsProvider).Events(
				ctx,
				[]string{"head"},
				func(event *v1.Event) {
					mu.Lock()
					defer mu.Unlock()

					data := event.Data.(*v1.HeadEvent)
					if receivals[data.Slot] == nil {
						receivals[data.Slot] = map[string]time.Time{}
					}
					receivals[data.Slot][addressInfo.URL] = time.Now()
					knownSlotRoots[data.Slot] = data.Block
				},
			)
			if err != nil {
				panic(err)
			}

			// Request attestation data and fill in block roots at the 4th second of every slot.
			for {
				slot := currentSlot() + 1
				time.Sleep(time.Until(slotTime(slot).Add(4 * time.Second)))

				ctx, _ := context.WithTimeout(ctx, 6*time.Second)
				p := pool.New().WithContext(ctx)
				p.Go(func(ctx context.Context) error {
					attestationData, err := client.(eth2client.AttestationDataProvider).AttestationData(
						ctx,
						&api.AttestationDataOpts{
							Slot:           slot,
							CommitteeIndex: 0,
							Common:         api.CommonOpts{Timeout: 6 * time.Second},
						},
					)
					if err != nil {
						return err
					}
					mu.Lock()
					if receivedBlockRoots[slot] == nil {
						receivedBlockRoots[slot] = map[string]phase0.Root{}
					}
					receivedBlockRoots[slot][addressInfo.Name] = attestationData.Data.BeaconBlockRoot
					mu.Unlock()
					return nil
				})
				p.Go(func(ctx context.Context) error {
					var resp struct {
						Data struct {
							Connected string `json:"connected"`
						}
					}
					err := requests.URL(fmt.Sprintf("%s/eth/v1/node/peer_count", addressInfo.URL)).
						ToJSON(&resp).
						Fetch(ctx)
					if err != nil {
						return err
					}
					mu.Lock()
					n, err := strconv.Atoi(resp.Data.Connected)
					if err != nil {
						return err
					}
					peers[addressInfo.Name] = n
					mu.Unlock()
					return nil
				})
				if err := p.Wait(); err != nil {
					log.Printf("error: %v", err)
				}
			}
		}(addressInfo)
	}

	// Sleep until next slot, and then print the performance
	startSlot := currentSlot() + 1
	slot := startSlot
	for {
		time.Sleep(time.Until(slotTime(slot)))

		mu.Lock()

		// Compute performances.
		type performance struct {
			name                string
			missing             int
			received            int
			peers               int
			delay               time.Duration
			freshAttestations   int
			missingAttestations int
			correctness         float64
		}
		performances := map[string]*performance{}
		for _, addressInfo := range addresses {
			p := &performance{name: addressInfo.Name, peers: peers[addressInfo.Name]}
			performances[addressInfo.Name] = p
			for s := startSlot; s < slot; s++ {
				receivals := receivals[s]
				if receivals == nil {
					p.missing++
					continue
				}
				receival, ok := receivals[addressInfo.URL]
				if !ok {
					p.missing++
					continue
				}
				p.received++
				p.delay += receival.Sub(slotTime(s))

				slotRoot, ok := knownSlotRoots[s]
				if !ok {
					p.missingAttestations++
					continue
				}
				blockRoots, ok := receivedBlockRoots[s]
				if !ok {
					p.missingAttestations++
					continue
				}
				if blockRoots[addressInfo.Name] == slotRoot {
					p.freshAttestations++
				}
			}
		}

		// Sort by correctness.
		performanceList := maps.Values(performances)
		for _, p := range performanceList {
			p.correctness = float64(p.freshAttestations) / float64(p.received)
		}
		sort.Slice(performanceList, func(i, j int) bool {
			return performanceList[i].correctness > performanceList[j].correctness
		})

		// Print.
		tbl := table.New(os.Stdout)
		tbl.SetHeaders("Address", "Peers", "Blocks (Missing)", "Delay", "Correctness (Missing)")
		for _, performance := range performanceList {
			delay := time.Duration(0)
			if performance.received > 0 {
				delay = performance.delay / time.Duration(performance.received)
			}
			tbl.AddRow(
				performance.name,
				fmt.Sprintf("%d", performance.peers),
				fmt.Sprintf("%d (%d)", performance.received, performance.missing),
				delay.String(),
				fmt.Sprintf("%.2f%% (%d)", performance.correctness*100, performance.missingAttestations),
			)
		}
		tbl.Render()

		mu.Unlock()

		slot++
	}
}

func parseAddresses(addressesFlag string) []AddressInfo {
	var addresses []AddressInfo
	pairs := strings.Split(addressesFlag, ",")
	for _, pair := range pairs {
		parts := strings.Split(pair, "=")
		if len(parts) != 2 {
			log.Fatalf("Invalid address format: %s", pair)
		}
		addresses = append(addresses, AddressInfo{
			Name: parts[0],
			URL:  parts[1],
		})
	}
	return addresses
}

var genesisTime = time.Unix(1606824023, 0)

func slotTime(slot phase0.Slot) time.Time {
	return genesisTime.Add(time.Duration(slot) * 12 * time.Second)
}

func currentSlot() phase0.Slot {
	return phase0.Slot(time.Since(genesisTime) / (12 * time.Second))
}
