package consensus

import (
	"bufio"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser"
)

const (
	partialSignatureLogRecord      = "🧩 reconstructed partial signatures"
	attestationSubmissionLogRecord = "✅ successfully submitted attestation"

	parserName = "consensus"
)

type (
	logEntry struct {
		Timestamp     parser.MultiFormatTime `json:"T"`
		Message       string                 `json:"M"`
		DutyID        string                 `json:"duty_id"`
		Slot          uint64                 `json:"slot"`
		Round         uint8                  `json:"round"`
		Signers       []parser.SignerID      `json:"signers"`
		ConsensusTime string                 `json:"consensus_time"`
		BlockRoot     string                 `json:"block_root,omitempty"`
	}

	OperatorStats struct {
		ConsensusTimeAvg time.Duration
	}

	Stats struct {
		OperatorStats                        map[parser.SignerID]OperatorStats
		DuplicateBlockRootSubmissions        uint32
		DuplicateBlockRootSubmissionsPercent float32
	}

	Service struct {
		logFile *os.File
	}
)

func New(logFilePath string) (*Service, error) {
	file, err := os.Open(logFilePath)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to open log file"))
	}
	return &Service{
		logFile: file,
	}, nil
}

func (s *Service) Analyze() (Stats, error) {
	defer s.logFile.Close()
	scanner := parser.NewScanner(s.logFile)

	var (
		stats = Stats{
			OperatorStats: make(map[uint32]OperatorStats),
		}
		operatorConsensusParticipation = make(map[parser.DutyID]struct {
			Signers   []parser.SignerID
			Timestamp time.Time
		})
		consensusTimes = make(map[parser.DutyID]time.Duration)
		blockRootSlots = make(map[parser.BlockRoot][]parser.Slot)
	)

	for scanner.Scan() {
		var entry logEntry
		line := scanner.Text()
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return stats, err
		}

		if strings.Contains(entry.Message, partialSignatureLogRecord) {
			/*
				Since there is no way to verify the round (as the given log record does not contain the roundID field),
				we need to ensure that we save signers from the earliest round, meaning round 1.
			*/
			if consensusRecord, hasRecordForDuty := operatorConsensusParticipation[entry.DutyID]; hasRecordForDuty {
				if entry.Timestamp.Before(consensusRecord.Timestamp) {
					operatorConsensusParticipation[entry.DutyID] = struct {
						Signers   []uint32
						Timestamp time.Time
					}{
						Signers:   entry.Signers,
						Timestamp: entry.Timestamp.Time,
					}
				}
			} else {
				operatorConsensusParticipation[entry.DutyID] = struct {
					Signers   []uint32
					Timestamp time.Time
				}{
					Signers:   entry.Signers,
					Timestamp: entry.Timestamp.Time,
				}
			}
		}

		//only consensus times with round 1 are not diluted
		if strings.Contains(entry.Message, attestationSubmissionLogRecord) {
			if entry.Round == 1 {
				consensusDuration, err := stringToDuration(entry.ConsensusTime, time.Second)
				if err != nil {
					return stats, err
				}
				consensusTimes[entry.DutyID] = consensusDuration
			}

			if entry.BlockRoot != "" {
				duties, ok := blockRootSlots[entry.BlockRoot]
				if !ok {
					blockRootSlots[entry.BlockRoot] = append(blockRootSlots[entry.BlockRoot], entry.Slot)
				} else {
					if !slices.Contains(duties, entry.Slot) {
						blockRootSlots[entry.BlockRoot] = append(blockRootSlots[entry.BlockRoot], entry.Slot)
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		logger := slog.
			With("parserName", parserName).
			With("fileName", s.logFile.Name())
		if err == bufio.ErrTooLong {
			logger.Warn("the log line was too long, continue reading..")
		} else {
			logger.
				With("err", err).
				Error("error reading log file")

			return stats, err
		}
	}

	signerConsensusTimes := make(map[parser.SignerID][]time.Duration)
	for dutyID, signers := range operatorConsensusParticipation {
		signerConsensusTime, exist := consensusTimes[dutyID]
		if exist {
			for _, signerID := range signers.Signers {
				signerConsensusTimes[signerID] = append(signerConsensusTimes[signerID], signerConsensusTime)
			}
		}
	}
	for signerID, consensusTimes := range signerConsensusTimes {
		var (
			consensusDurationsTotal time.Duration
			consensusDurationLen    = len(consensusTimes)
		)
		for _, duration := range consensusTimes {
			consensusDurationsTotal += duration
		}

		if consensusDurationLen > 0 {
			stats.OperatorStats[signerID] = OperatorStats{
				ConsensusTimeAvg: consensusDurationsTotal / time.Duration(consensusDurationLen),
			}
		}
	}

	for _, slots := range blockRootSlots {
		if len(slots) > 1 {
			stats.DuplicateBlockRootSubmissions++
		}
	}

	stats.DuplicateBlockRootSubmissionsPercent = float32(stats.DuplicateBlockRootSubmissions) / float32(len(blockRootSlots)) * 100

	return stats, nil
}

func stringToDuration(s string, unit time.Duration) (time.Duration, error) {
	var duration time.Duration
	seconds, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return duration, err
	}

	duration = time.Duration(seconds * float64(unit))

	return duration, nil
}
