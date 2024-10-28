package analyzer

import (
	"log/slog"
	"maps"
	"math"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ssvlabs/ssv-pulse/configs"
	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser/client"
	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser/commit"
	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser/consensus"
	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser/operator"
	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser/peers"
	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser/prepare"
	"github.com/ssvlabs/ssv-pulse/internal/analyzer/report"
)

const (
	logFilesDirectoryFlag = "log-files-directory"
	operatorsFlag         = "operators"
	clusterFlag           = "cluster"

	command = "analyzer"
)

func init() {
	addFlags(CMD)
	if err := bindFlags(CMD); err != nil {
		panic(err.Error())
	}
}

var CMD = &cobra.Command{
	Use:   command,
	Short: "Read and analyze ssv node logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		isValid, err := configs.Values.Analyzer.Validate()
		if !isValid {
			return err
		}

		fileDirectory := configs.Values.Analyzer.LogFilesDirectory
		files, err := os.ReadDir(fileDirectory)
		if err != nil {
			return err
		}

		var (
			wg                  sync.WaitGroup
			errChan             = make(chan error, len(files))
			peerRecordsChan     = make(chan report.PeerRecord)
			clientRecordsChan   = make(chan report.ClientRecord)
			operatorRecordsChan = make(chan map[uint32]report.OperatorRecord)
			doneChan            = make(chan any)
			clientReport        = report.NewClient()
			peersReport         = report.NewPeers()
			operatorReport      = report.NewOperator()
		)

		var totalFileSizeMB float64
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			stat, err := os.Stat(path.Join(fileDirectory, file.Name()))
			if err != nil {
				slog.With("err", err.Error()).Warn("error fetching the file info, ignoring")
			}
			totalFileSizeMB += float64(stat.Size()) / (1024 * 1024)

			wg.Add(1)
			go func(filePath string) {
				defer wg.Done()
				analyzeFile(filePath, peerRecordsChan, clientRecordsChan, operatorRecordsChan, errChan)
			}(filepath.Join(fileDirectory, file.Name()))
		}

		go func() {
			wg.Wait()
			close(errChan)
			close(peerRecordsChan)
			close(clientRecordsChan)
			close(doneChan)
		}()

		progressTicker := time.NewTicker(3 * time.Second)

		fileOperatorRecords := make(map[uint32][]report.OperatorRecord)

		for {
			select {
			case <-progressTicker.C:
				slog.
					With("count", len(files)).
					With("filesSizeMB", math.Round(totalFileSizeMB)).
					Info("⏳⏳⏳ processing file(s)...")
			case peerRecord, isOpen := <-peerRecordsChan:
				if isOpen {
					peersReport.AddRecord(peerRecord)
				}
			case clientRecord, isOpen := <-clientRecordsChan:
				if isOpen {
					clientReport.AddRecord(clientRecord)
				}
			case operatorRecord, isOpen := <-operatorRecordsChan:
				if isOpen {
					keys := maps.Keys(operatorRecord)
					for key := range keys {
						fileOperatorRecords[key] = append(fileOperatorRecords[key], operatorRecord[key])
					}
				}
			case err := <-errChan:
				if err != nil {
					return err
				}
			case <-doneChan:
				for _, records := range fileOperatorRecords {
					operatorStats := make(map[uint32]report.OperatorRecord)

					commitAvgTotal := make(map[uint32]time.Duration)
					commitAvgRecordCount := make(map[uint32]uint32)
					commitDelayHighest := make(map[uint32]time.Duration)
					commitDelayedPercentTotal := make(map[uint32]map[time.Duration]float32)
					commitDelayedRecordCount := make(map[uint32]map[time.Duration]uint16)

					prepareAvgTotal := make(map[uint32]time.Duration)
					prepareAvgRecordCount := make(map[uint32]uint32)
					prepareDelayHighest := make(map[uint32]time.Duration)
					prepareDelayedPercentTotal := make(map[uint32]map[time.Duration]float32)
					prepareDelayedRecordCount := make(map[uint32]map[time.Duration]uint16)

					consensusAvgTotal := make(map[uint32]time.Duration)
					consensusAvgRecordCount := make(map[uint32]uint32)

					for _, record := range records {
						operatorStats[record.OperatorID] = report.OperatorRecord{
							OperatorID:        record.OperatorID,
							Clusters:          record.Clusters,
							IsLogFileOwner:    record.IsLogFileOwner,
							CommitTotalCount:  operatorStats[record.OperatorID].CommitTotalCount + record.CommitTotalCount,
							PrepareTotalCount: operatorStats[record.OperatorID].PrepareTotalCount + record.PrepareTotalCount,
						}
						commitAvgTotal[record.OperatorID] += record.CommitDelayAvg
						commitAvgRecordCount[record.OperatorID]++

						prepareAvgTotal[record.OperatorID] += record.CommitDelayAvg
						prepareAvgRecordCount[record.OperatorID]++

						if commitDelayHighest[record.OperatorID] < record.CommitDelayHighest {
							commitDelayHighest[record.OperatorID] = record.CommitDelayHighest
						}

						if prepareDelayHighest[record.OperatorID] < record.PrepareDelayHighest {
							prepareDelayHighest[record.OperatorID] = record.PrepareDelayHighest
						}

						consensusAvgTotal[record.OperatorID] += record.ConsensusTimeAvg
						consensusAvgRecordCount[record.OperatorID]++

						for delay, percent := range record.CommitDelayPercent {
							_, ok := commitDelayedPercentTotal[record.OperatorID][delay]
							if !ok {
								commitDelayedPercentTotal[record.OperatorID] = make(map[time.Duration]float32)
							}
							_, ok = commitDelayedRecordCount[record.OperatorID][delay]
							if !ok {
								commitDelayedRecordCount[record.OperatorID] = make(map[time.Duration]uint16)
							}

							commitDelayedPercentTotal[record.OperatorID][delay] += percent
							commitDelayedRecordCount[record.OperatorID][delay]++
						}

						for delay, percent := range record.PrepareDelayPercent {
							_, ok := prepareDelayedPercentTotal[record.OperatorID][delay]
							if !ok {
								prepareDelayedPercentTotal[record.OperatorID] = make(map[time.Duration]float32)
							}
							_, ok = prepareDelayedRecordCount[record.OperatorID][delay]
							if !ok {
								prepareDelayedRecordCount[record.OperatorID] = make(map[time.Duration]uint16)
							}
							prepareDelayedPercentTotal[record.OperatorID][delay] += percent
							prepareDelayedRecordCount[record.OperatorID][delay]++
						}
					}

					for operatorID, record := range operatorStats {
						record.CommitDelayAvg = commitAvgTotal[operatorID] / time.Duration(commitAvgRecordCount[operatorID])
						record.CommitDelayHighest = commitDelayHighest[operatorID]

						for delayDuration, percentSum := range commitDelayedPercentTotal[operatorID] {
							_, ok := record.CommitDelayPercent[delayDuration]
							if !ok {
								record.CommitDelayPercent = make(map[time.Duration]float32)
							}
							record.CommitDelayPercent[delayDuration] = percentSum / float32(commitDelayedRecordCount[operatorID][delayDuration])
						}

						record.PrepareDelayAvg = prepareAvgTotal[operatorID] / time.Duration(prepareAvgRecordCount[operatorID])
						record.PrepareDelayHighest = prepareDelayHighest[operatorID]

						for delayDuration, percentSum := range prepareDelayedPercentTotal[operatorID] {
							_, ok := record.PrepareDelayPercent[delayDuration]
							if !ok {
								record.PrepareDelayPercent = make(map[time.Duration]float32)
							}
							record.PrepareDelayPercent[delayDuration] = percentSum / float32(prepareDelayedRecordCount[operatorID][delayDuration])
						}

						record.ConsensusTimeAvg = consensusAvgTotal[operatorID] / time.Duration(consensusAvgRecordCount[operatorID])

						operatorStats[operatorID] = record
					}

					stats := slices.Collect(maps.Values(operatorStats))

					//move log file owner record to index 0
					sort.Slice(stats, func(i, j int) bool {
						if stats[i].IsLogFileOwner {
							return true
						}
						if stats[j].IsLogFileOwner {
							return false
						}
						return false
					})

					for _, record := range stats {
						operatorReport.AddRecord(record)
					}
				}

				clientReport.Render()
				peersReport.Render()
				operatorReport.Render()
				return nil
			}
		}
	},
}

func addFlags(cobraCMD *cobra.Command) {
	cobraCMD.Flags().String(logFilesDirectoryFlag, "", "Path to the directory containing SSV node log files for analysis, e.g. my-file-dir")
	cobraCMD.Flags().StringSlice(operatorsFlag, []string{}, "Operators to analyze, e.g. 123,321,132,312")
	cobraCMD.Flags().Bool(clusterFlag, false, "Are operators forming the cluster?")
}

func bindFlags(cmd *cobra.Command) error {
	if err := viper.BindPFlag("analyzer.log-files-directory", cmd.Flags().Lookup(logFilesDirectoryFlag)); err != nil {
		return err
	}
	if err := viper.BindPFlag("analyzer.cluster", cmd.Flags().Lookup(clusterFlag)); err != nil {
		return err
	}
	if err := viper.BindPFlag("analyzer.operators", cmd.Flags().Lookup(operatorsFlag)); err != nil {
		return err
	}

	return nil
}

func analyzeFile(
	filePath string,
	peerRecordChan chan<- report.PeerRecord,
	clientRecordChan chan<- report.ClientRecord,
	operatorRecordChan chan<- map[uint32]report.OperatorRecord,
	errorChan chan<- error) {
	clientAnalyzer, err := client.New(filePath, time.Millisecond*800)
	if err != nil {
		errorChan <- err
		return
	}

	commitAnalyzer, err := commit.New(filePath, time.Millisecond*800)
	if err != nil {
		errorChan <- err
		return
	}
	prepareAnalyzer, err := prepare.New(filePath, time.Millisecond*600)
	if err != nil {
		errorChan <- err
		return
	}

	operatorAnalyzer, err := operator.New(filePath)
	if err != nil {
		errorChan <- err
		return
	}

	consensusAnalyzer, err := consensus.New(filePath)
	if err != nil {
		errorChan <- err
		return
	}

	peersAnalyzer, err := peers.New(filePath)
	if err != nil {
		errorChan <- err
		return
	}

	analyzerSvc, err := New(
		peersAnalyzer,
		consensusAnalyzer,
		operatorAnalyzer,
		clientAnalyzer,
		commitAnalyzer,
		prepareAnalyzer,
		configs.Values.Analyzer.Operators,
		configs.Values.Analyzer.Cluster)
	if err != nil {
		errorChan <- err
		return
	}

	result, err := analyzerSvc.Start()
	if err != nil {
		errorChan <- err
		return
	}

	var (
		owner           uint32
		operatorRecords []report.OperatorRecord
	)

	for _, r := range result.OperatorStats {
		if r.IsLogFileOwner {
			owner = r.OperatorID
			peerRecordChan <- report.PeerRecord{
				OperatorID:             r.OperatorID,
				PeerCountAvg:           r.PeersCountAvg,
				PeersSSVClientVersions: r.PeerSSVClientVersions,
				PeerID:                 r.PeerID,
			}
			clientRecordChan <- report.ClientRecord{
				OperatorID:                              r.OperatorID,
				ConsensusClientResponseTimeAvg:          r.ConsensusClientResponseTimeAvg,
				ConsensusClientResponseTimeDelayPercent: r.ConsensusClientResponseTimeDelayPercent,
				ConsensusClientResponseTimeP10:          r.ConsensusClientResponseTimeP10,
				ConsensusClientResponseTimeP50:          r.ConsensusClientResponseTimeP50,
				ConsensusClientResponseTimeP90:          r.ConsensusClientResponseTimeP90,

				SSVClientCrashesTotal: r.SSVClientCrashesTotal,
				SSVClientELUnhealthy:  r.SSVClientELUnhealthy,
				SSVClientCLUnhealthy:  r.SSVClientCLUnhealthy,
			}
		}
		operatorRecords = append(operatorRecords, report.OperatorRecord{
			OperatorID:     r.OperatorID,
			Clusters:       r.Clusters,
			IsLogFileOwner: r.IsLogFileOwner,

			CommitDelayAvg:     r.CommitDelayAvg,
			CommitDelayHighest: r.CommitDelayHighest,
			CommitDelayPercent: r.CommitDelayPercent,
			CommitTotalCount:   r.CommitCount,

			PrepareDelayAvg:     r.PrepareDelayAvg,
			PrepareDelayHighest: r.PrepareDelayHighest,
			PrepareDelayPercent: r.PrepareDelayPercent,
			PrepareTotalCount:   r.PrepareCount,

			ConsensusTimeAvg: r.ConsensusTimeAvg,
		})
	}

	for _, record := range operatorRecords {
		operatorRecordChan <- map[uint32]report.OperatorRecord{
			owner: record,
		}
	}
}
