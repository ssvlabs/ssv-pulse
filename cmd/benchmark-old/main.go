package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
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
	"github.com/ssvlabsinfra/ssv-benchmark/configs"
	"golang.org/x/exp/maps"

	"github.com/aquasecurity/table"
)

func main() {
	ctx := context.Background()

	cfg, err := configs.Init()
	if err != nil {
		panic(err.Error())
	}

	receivals := map[phase0.Slot]map[configs.Address]time.Time{}
	knownSlotRoots := map[phase0.Slot]phase0.Root{}
	receivedBlockRoots := map[phase0.Slot]map[configs.Address]phase0.Root{}
	unreadyBlocks200 := map[configs.Address]int{}
	unreadyBlocks400 := map[configs.Address]int{}
	peers := map[configs.Address]int{}
	var mu sync.Mutex

	for _, address := range cfg.BeaconNodeAddrs {
		go func(address configs.Address) {
			client, err := auto.New(
				ctx,
				auto.WithLogLevel(zerolog.DebugLevel),
				auto.WithAddress(string(address)),
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
						receivals[data.Slot] = map[configs.Address]time.Time{}
					}
					receivals[data.Slot][address] = time.Now()
					knownSlotRoots[data.Slot] = data.Block

					go func() {
						time.Sleep(200 * time.Millisecond)
						resp, err := client.(eth2client.AttestationDataProvider).AttestationData(
							ctx,
							&api.AttestationDataOpts{
								Slot:           data.Slot,
								CommitteeIndex: 0,
							},
						)
						if err != nil {
							log.Printf("failed to fetch attestation data after head event: %v", err)
						} else if resp.Data.BeaconBlockRoot != data.Block {
							mu.Lock()
							unreadyBlocks200[address]++
							mu.Unlock()
							log.Printf("unready block (200ms) at slot %d from %v", data.Slot, address)
						}
					}()

					go func() {
						time.Sleep(400 * time.Millisecond)
						resp, err := client.(eth2client.AttestationDataProvider).AttestationData(
							ctx,
							&api.AttestationDataOpts{
								Slot:           data.Slot,
								CommitteeIndex: 0,
							},
						)
						if err != nil {
							log.Printf("failed to fetch attestation data after head event: %v", err)
						} else if resp.Data.BeaconBlockRoot != data.Block {
							mu.Lock()
							unreadyBlocks400[address]++
							mu.Unlock()
							log.Printf("unready block (400ms) at slot %d from %v", data.Slot, address)
						}
					}()
				},
			)
			if err != nil {
				panic(err)
			}

			// Request attestation data and fill in block roots at the 4th second of every slot.
			for {
				slot := currentSlot(configs.GenesisTime[cfg.Network]) + 1
				time.Sleep(time.Until(slotTime(configs.GenesisTime[cfg.Network], slot).Add(4 * time.Second)))

				ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
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
						receivedBlockRoots[slot] = map[configs.Address]phase0.Root{}
					}
					receivedBlockRoots[slot][address] = attestationData.Data.BeaconBlockRoot
					mu.Unlock()
					return nil
				})
				p.Go(func(ctx context.Context) error {
					var resp struct {
						Data struct {
							Connected string `json:"connected"`
						}
					}
					err := requests.URL(fmt.Sprintf("%s/eth/v1/node/peer_count", address)).
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
					peers[address] = n
					mu.Unlock()
					return nil
				})
				if err := p.Wait(); err != nil {
					log.Printf("error: %v", err)
				}
				cancel()
			}
		}(address)
	}

	// Sleep until next slot, and then print the performance
	startSlot := currentSlot(configs.GenesisTime[cfg.Network]) + 1
	slot := startSlot
	for {
		time.Sleep(time.Until(slotTime(configs.GenesisTime[cfg.Network], slot)))

		mu.Lock()

		// Compute performances.
		type performance struct {
			addr                configs.Address
			missing             int
			received            int
			peers               int
			totalLatency        time.Duration
			minLatency          time.Duration
			maxLatency          time.Duration
			freshAttestations   int
			missingAttestations int
			correctness         float64
		}
		performances := map[configs.Address]*performance{}
		for _, addr := range cfg.BeaconNodeAddrs {
			p := &performance{addr: addr, peers: peers[addr], minLatency: time.Duration(math.MaxInt64)}
			performances[addr] = p
			for s := startSlot; s < slot; s++ {
				receivals := receivals[s]
				if receivals == nil {
					p.missing++
					continue
				}
				receival, ok := receivals[addr]
				if !ok {
					p.missing++
					continue
				}
				p.received++
				latency := receival.Sub(slotTime(configs.GenesisTime[cfg.Network], s))
				p.totalLatency += latency
				if latency < p.minLatency {
					p.minLatency = latency
				}
				if latency > p.maxLatency {
					p.maxLatency = latency
				}

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
				if blockRoots[addr] == slotRoot {
					p.freshAttestations++
				}
			}
			// Handle case where no latency was recorded
			if p.minLatency == time.Duration(math.MaxInt64) {
				p.minLatency = 0
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
		tbl.SetHeaders("Address", "Peers", "Blocks (Missing)", "Latency (Min/Avg/Max)", "Correctness (Missing)", "Unready (200ms/400ms)")
		for _, performance := range performanceList {
			var avgLatency time.Duration
			if performance.received > 0 {
				avgLatency = performance.totalLatency / time.Duration(performance.received)
			}
			tbl.AddRow(
				string(performance.addr),
				fmt.Sprintf("%d", performance.peers),
				fmt.Sprintf("%d (%d)", performance.received, performance.missing),
				fmt.Sprintf("%s/%s/%s", performance.minLatency, avgLatency, performance.maxLatency),
				fmt.Sprintf("%.2f%% (%d)", performance.correctness*100, performance.missingAttestations),
				fmt.Sprintf("%d/%d", unreadyBlocks200[performance.addr], unreadyBlocks400[performance.addr]),
			)
		}
		tbl.Render()

		mu.Unlock()

		slot++
	}
}

func slotTime(genesisTime time.Time, slot phase0.Slot) time.Time {
	return genesisTime.Add(time.Duration(slot) * 12 * time.Second)
}

func currentSlot(genesisTime time.Time) phase0.Slot {
	return phase0.Slot(time.Since(genesisTime) / (12 * time.Second))
}
