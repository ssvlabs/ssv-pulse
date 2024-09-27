package commit

import (
	"bufio"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser"
)

const (
	commitMsg = "got commit message"
)

var rankScores = []int{5, 4, 3, 2, 1, 0}

type (
	Stats struct {
		Score int
		Delay time.Duration
	}

	Service struct {
		logFile   *os.File
		operators []uint32
		cluster   bool
	}
)

func New(logFilePath string, operators []uint32, cluster bool) (*Service, error) {
	file, err := os.Open(logFilePath)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to open log file"))
	}
	return &Service{
		logFile:   file,
		operators: operators,
		cluster:   cluster,
	}, nil
}

func (c *Service) Analyze() (map[parser.SignerID]Stats, error) {
	defer c.logFile.Close()
	scanner := bufio.NewScanner(c.logFile)

	commitTimes := make(map[parser.DutyID]map[parser.SignerID]time.Time)

	for scanner.Scan() {
		var entry commitLogEntry
		line := scanner.Text()
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, err
		}

		if strings.Contains(entry.Message, commitMsg) && entry.Round == 1 {
			if _, exists := commitTimes[entry.DutyID]; !exists {
				commitTimes[entry.DutyID] = make(map[parser.SignerID]time.Time)
			}

			// Record the earliest time for each signer
			if existingTime, exists := commitTimes[entry.DutyID][entry.CommitSigners[0]]; !exists || entry.Timestamp.Before(existingTime) {
				commitTimes[entry.DutyID][entry.CommitSigners[0]] = entry.Timestamp
			}
		}
	}
	if err := scanner.Err(); err != nil {
		slog.With("err", err).Error("error reading log file")
		return nil, err
	}

	stats := make(map[parser.SignerID]Stats)

	for _, signers := range commitTimes {
		if c.cluster && len(c.operators) != 0 {
			if !parser.IsCluster(c.operators, signers) {
				continue
			}
		}

		type signerPerformance struct {
			signer   parser.SignerID
			earliest time.Time
		}

		var performances []signerPerformance

		for signer, earliestTime := range signers {
			if len(c.operators) != 0 {
				var ok bool
				for _, operatorID := range c.operators {
					if signer == operatorID {
						ok = true
					}
				}
				if !ok {
					continue
				}
			}
			performances = append(performances, signerPerformance{
				signer:   signer,
				earliest: earliestTime,
			})
		}

		// Sort by earliest time, the earlier the better
		sort.Slice(performances, func(i, j int) bool {
			return performances[i].earliest.Before(performances[j].earliest)
		})

		// Assign scores and calculate delays
		if len(performances) > 0 {
			firstTime := performances[0].earliest

			for rank, performance := range performances {
				if rank < len(rankScores) {
					stats[performance.signer] = Stats{
						Score: stats[performance.signer].Score + rankScores[rank],
						Delay: stats[performance.signer].Delay + performance.earliest.Sub(firstTime),
					}
				}
			}
		}
	}

	return stats, nil
}
