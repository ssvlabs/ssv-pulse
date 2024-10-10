package consensus

import (
	"bufio"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser"
)

const (
	partialSignatureLogRecord      = "🧩 reconstructed partial signatures"
	attestationSubmissionLogRecord = "✅ successfully submitted attestation"
)

type (
	logEntry struct {
		Timestamp     parser.MultiFormatTime `json:"T"`
		Message       string                 `json:"M"`
		DutyID        string                 `json:"duty_id"`
		Round         uint8                  `json:"round"`
		Signers       []parser.SignerID      `json:"signers"`
		ConsensusTime string                 `json:"consensus_time"`
	}

	Stats struct {
		OperatorConsensusTimes            map[parser.SignerID][]time.Duration
		OperatorConsensusParticipation    map[parser.SignerID]uint16
		SuccessfullySubmittedAttestations uint16
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
	scanner := bufio.NewScanner(s.logFile)

	var (
		stats Stats = Stats{
			OperatorConsensusTimes:         make(map[uint32][]time.Duration),
			OperatorConsensusParticipation: make(map[uint32]uint16),
		}
		operatorConsensusParticipation = make(map[parser.DutyID]struct {
			Signers   []parser.SignerID
			Timestamp time.Time
		})
		consensusTimes = make(map[parser.DutyID]time.Duration)
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
		if strings.Contains(entry.Message, attestationSubmissionLogRecord) && entry.Round == 1 {
			consensusDuration, err := stringToDuration(entry.ConsensusTime, time.Second)
			if err != nil {
				return stats, err
			}
			consensusTimes[entry.DutyID] = consensusDuration
		}
	}
	if err := scanner.Err(); err != nil {
		slog.With("err", err).Error("error reading log file")
		return stats, err
	}

	for dutyID, signers := range operatorConsensusParticipation {
		signerConsensusTime, exist := consensusTimes[dutyID]
		if exist {
			for _, signerID := range signers.Signers {
				stats.OperatorConsensusTimes[signerID] = append(stats.OperatorConsensusTimes[signerID], signerConsensusTime)
				stats.OperatorConsensusParticipation[signerID]++
			}
		}
	}

	stats.SuccessfullySubmittedAttestations = uint16(len(consensusTimes))
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
