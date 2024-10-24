package prepare

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
	prepareMsg       = "got prepare message"
	leaderProposeMsg = "leader broadcasting proposal message"

	parserName               = "prepare"
	scannerBufferMaxCapacity = 1024 * 1024
)

type (
	Stats struct {
		Count          uint16
		DelayAvg       time.Duration
		DelayHighest   time.Duration
		DelayedPercent map[time.Duration]float32
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

func (p *Service) Analyze() (map[parser.SignerID]Stats, error) {
	defer p.logFile.Close()

	scanner := parser.NewScanner(p.logFile)

	leaderProposeTime := make(map[parser.DutyID]time.Time)
	prepareSignerTimes := make(map[parser.DutyID]map[parser.SignerID]time.Time)

	for scanner.Scan() {
		var entry prepareLogEntry
		line := scanner.Text()
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, err
		}

		if strings.Contains(entry.Message, leaderProposeMsg) {
			leaderProposeTime[entry.DutyID] = entry.Timestamp.Time
		}

		if strings.Contains(entry.Message, prepareMsg) && entry.Round == 1 {
			if _, exists := prepareSignerTimes[entry.DutyID]; !exists {
				prepareSignerTimes[entry.DutyID] = make(map[parser.SignerID]time.Time)
			}

			// Record the earliest time for each signer
			if existingTime, exists := prepareSignerTimes[entry.DutyID][entry.PrepareSigners[0]]; !exists || entry.Timestamp.Before(existingTime) {
				prepareSignerTimes[entry.DutyID][entry.PrepareSigners[0]] = entry.Timestamp.Time
			}
		}
	}
	if err := scanner.Err(); err != nil {
		logger := slog.
			With("parserName", parserName).
			With("fileName", p.logFile.Name())
		if err == bufio.ErrTooLong {
			logger.Warn("the log line was too long, continue reading..")
		} else {
			logger.
				With("err", err).
				Error("error reading log file")

			return nil, err
		}
	}

	stats := p.calcPrepareTimes(leaderProposeTime, prepareSignerTimes)

	return stats, nil
}

func (s *Service) calcPrepareTimes(leaderProposeTime map[parser.DutyID]time.Time, prepareSignerTimes map[parser.DutyID]map[parser.SignerID]time.Time) map[parser.SignerID]Stats {
	prepareStats := make(map[parser.SignerID]Stats)
	prepareMessageCount := make(map[parser.SignerID]uint16)
	prepareDelayedMessageCount := make(map[parser.SignerID]uint16)
	averageTimePrepareMessage := make(map[parser.SignerID]time.Duration)
	totalTimePrepareMessage := make(map[parser.SignerID]time.Duration)
	highestTimePrepareMessage := make(map[parser.SignerID]time.Duration)

	for dutyID, signers := range prepareSignerTimes {
		leaderProposeMessageTime, exist := leaderProposeTime[dutyID]
		if !exist {
			continue
		}
		for signer, prepareMessageTimeStamp := range signers {
			if prepareMessageTimeStamp.Before(leaderProposeMessageTime) {
				slog.Error("error: got prepare message before leader propose message")
				break
			}
			delay := prepareMessageTimeStamp.Sub(leaderProposeMessageTime)
			prepareMessageCount[signer]++
			totalTimePrepareMessage[signer] = totalTimePrepareMessage[signer] + delay
			if highestTimePrepareMessage[signer] < delay {
				highestTimePrepareMessage[signer] = delay
			}
			if delay > s.delay {
				prepareDelayedMessageCount[signer]++
			}

			averageTimePrepareMessage[signer] = time.Duration(totalTimePrepareMessage[signer].Nanoseconds() / int64(prepareMessageCount[signer]))

			prepareStats[signer] = Stats{
				Count:          prepareMessageCount[signer],
				DelayAvg:       averageTimePrepareMessage[signer],
				DelayHighest:   highestTimePrepareMessage[signer],
				DelayedPercent: map[time.Duration]float32{s.delay: float32(prepareDelayedMessageCount[signer]) / float32(prepareMessageCount[signer]) * 100},
			}
		}
	}

	return prepareStats
}
