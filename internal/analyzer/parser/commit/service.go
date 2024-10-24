package commit

import (
	"bufio"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser"
)

const (
	proposeMsg = "leader broadcasting proposal message"
	commitMsg  = "got commit message"

	parserName = "commit"
)

type (
	Stats struct {
		Count uint16
		DelayAvg,
		DelayHighest time.Duration
		DelayedPercent map[time.Duration]float32
		DelayTotal     time.Duration
	}

	Service struct {
		logFile *os.File
		delay   time.Duration
	}
)

func New(logFilePath string, delay time.Duration) (*Service, error) {
	file, err := os.Open(logFilePath)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to open log file"))
	}
	return &Service{
		logFile: file,
		delay:   delay,
	}, nil
}

func (s *Service) Analyze() (map[parser.SignerID]Stats, error) {
	defer s.logFile.Close()
	scanner := bufio.NewScanner(s.logFile)

	proposeTime := make(map[parser.DutyID]time.Time)
	commitTimes := make(map[parser.DutyID]map[parser.SignerID]time.Time)

	for scanner.Scan() {
		var entry commitLogEntry
		line := scanner.Text()
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, err
		}

		if strings.Contains(entry.Message, proposeMsg) {
			proposeTime[entry.DutyID] = entry.Timestamp.Time
		}

		if strings.Contains(entry.Message, commitMsg) && entry.Round == 1 {
			if _, exists := commitTimes[entry.DutyID]; !exists {
				commitTimes[entry.DutyID] = make(map[parser.SignerID]time.Time)
			}

			// Record the earliest time for each signer
			// Can same signer submit commit message more than once per duty?
			if existingTime, exists := commitTimes[entry.DutyID][entry.CommitSigners[0]]; !exists || entry.Timestamp.Before(existingTime) {
				commitTimes[entry.DutyID][entry.CommitSigners[0]] = entry.Timestamp.Time
			}
		}
	}
	if err := scanner.Err(); err != nil {
		slog.
			With("err", err).
			With("parserName", parserName).
			With("fileName", s.logFile.Name()).
			Error("error reading log file")

		return nil, err
	}

	totalDelay := make(map[parser.SignerID]time.Duration)
	highestDelay := make(map[parser.SignerID]time.Duration)
	delayed := make(map[parser.SignerID]map[time.Duration]uint16)
	count := make(map[parser.SignerID]uint16)

	stats := make(map[parser.SignerID]Stats)

	for dutyID, signers := range commitTimes {
		dutyProposeTime, exist := proposeTime[dutyID]
		if !exist {
			continue
		}

		for signerID, commitTime := range signers {
			stats[signerID] = Stats{}
			totalDelay[signerID] += commitTime.Sub(dutyProposeTime)
			count[signerID]++

			delay := commitTime.Sub(dutyProposeTime)
			if delay > highestDelay[signerID] {
				highestDelay[signerID] = commitTime.Sub(dutyProposeTime)
			}

			if delayed[signerID] == nil {
				delayed[signerID] = map[time.Duration]uint16{s.delay: 0}
			}
			if delay > s.delay {
				delayed[signerID][s.delay]++
			}
		}
	}

	for signerID, signerStats := range stats {
		signerStats.Count = count[signerID]
		_, ok := signerStats.DelayedPercent[s.delay]
		if !ok {
			signerStats.DelayedPercent = make(map[time.Duration]float32)
		}

		signerStats.DelayedPercent[s.delay] = float32(delayed[signerID][s.delay]) / float32(count[signerID]) * 100
		signerStats.DelayHighest = highestDelay[signerID]
		signerStats.DelayTotal = totalDelay[signerID]
		signerStats.DelayAvg = signerStats.DelayTotal / time.Duration(signerStats.Count)

		stats[signerID] = signerStats
	}

	return stats, nil
}
