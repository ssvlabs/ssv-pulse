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
)

type (
	Stats struct {
		Count           uint16
		AverageDelay    time.Duration
		HighestDelay    time.Duration
		MoreSecondDelay uint16
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

func (p *Service) Analyze() (map[parser.SignerID]Stats, error) {
	defer p.logFile.Close()
	scanner := bufio.NewScanner(p.logFile)

	leaderProposeTime := make(map[parser.DutyID]time.Time)
	prepareSignerTimes := make(map[parser.DutyID]map[parser.SignerID]time.Time)

	for scanner.Scan() {
		var entry prepareLogEntry
		line := scanner.Text()
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, err
		}

		if strings.Contains(entry.Message, leaderProposeMsg) {
			leaderProposeTime[entry.DutyID] = entry.Timestamp
		}

		if strings.Contains(entry.Message, prepareMsg) && entry.Round == 1 {
			if _, exists := prepareSignerTimes[entry.DutyID]; !exists {
				prepareSignerTimes[entry.DutyID] = make(map[parser.SignerID]time.Time)
			}

			// Record the earliest time for each signer
			if existingTime, exists := prepareSignerTimes[entry.DutyID][entry.PrepareSigners[0]]; !exists || entry.Timestamp.Before(existingTime) {
				prepareSignerTimes[entry.DutyID][entry.PrepareSigners[0]] = entry.Timestamp
			}
		}
	}
	if err := scanner.Err(); err != nil {
		slog.With("err", err).Error("error reading log file")
		return nil, err
	}

	stats := p.calcPrepareTimes(leaderProposeTime, prepareSignerTimes)

	return stats, nil
}

func (r *Service) calcPrepareTimes(leaderProposeTime map[parser.DutyID]time.Time, prepareSignerTimes map[parser.DutyID]map[parser.SignerID]time.Time) map[parser.SignerID]Stats {
	proposeStats := make(map[parser.SignerID]Stats)
	prepareMessageCount := make(map[parser.SignerID]uint16)
	prepareMessageCountMoreSecond := make(map[parser.SignerID]uint16)
	averageTimePrepareMessage := make(map[parser.SignerID]time.Duration)
	totalTimePrepareMessage := make(map[parser.SignerID]time.Duration)
	highestTimePrepareMessage := make(map[parser.SignerID]time.Duration)

	for dutyID, signers := range prepareSignerTimes {
		leaderProposeMessageTime, exist := leaderProposeTime[dutyID]
		if !exist {
			continue
		}
		if r.cluster && len(r.operators) != 0 {
			if !parser.IsCluster(r.operators, signers) {
				continue
			}
		}
		for signer, prepareMessageTimeStamp := range signers {
			if len(r.operators) != 0 {
				var ok bool
				for _, ID := range r.operators {
					if signer == ID {
						ok = true
					}
				}
				if !ok {
					continue
				}
			}
			if prepareMessageTimeStamp.Before(leaderProposeMessageTime) {
				slog.Error("error: got prepare message before leader propose message")
				break
			}
			delay := prepareMessageTimeStamp.Sub(leaderProposeMessageTime)
			prepareMessageCount[signer] = prepareMessageCount[signer] + 1
			totalTimePrepareMessage[signer] = totalTimePrepareMessage[signer] + delay
			if highestTimePrepareMessage[signer] < delay {
				highestTimePrepareMessage[signer] = delay
			}
			if delay > time.Second {
				prepareMessageCountMoreSecond[signer] = prepareMessageCountMoreSecond[signer] + 1
			}

			averageTimePrepareMessage[signer] = time.Duration(totalTimePrepareMessage[signer].Nanoseconds() / int64(prepareMessageCount[signer]))

			proposeStats[signer] = Stats{
				Count:           prepareMessageCount[signer],
				AverageDelay:    averageTimePrepareMessage[signer],
				HighestDelay:    highestTimePrepareMessage[signer],
				MoreSecondDelay: prepareMessageCountMoreSecond[signer],
			}
		}
	}

	return proposeStats
}
