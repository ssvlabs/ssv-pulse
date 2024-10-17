package attestation

import (
	"bufio"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	attestationMsg = "starting QBFT instance"
)

type (
	attestationLogEntry struct {
		Message         string `json:"M"`
		AttestationTime string `json:"attestation_data_time"`
	}

	Stats struct {
		ConsensusClientResponseTimeDelayCount map[time.Duration]uint16
		ConsensusClientResponseTimeTotal      time.Duration
		ConsensusClientResponseDurations      []time.Duration
	}

	Service struct {
		logFile *os.File
		delay   time.Duration
	}
)

func New(logFilePath string, consensusAttestationEndpointDelay time.Duration) (*Service, error) {
	file, err := os.Open(logFilePath)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to open log file"))
	}
	return &Service{
		logFile: file,
		delay:   consensusAttestationEndpointDelay,
	}, nil
}

func (s *Service) Analyze() (Stats, error) {
	defer s.logFile.Close()
	scanner := bufio.NewScanner(s.logFile)

	var stats Stats = Stats{
		ConsensusClientResponseTimeDelayCount: map[time.Duration]uint16{
			s.delay: 0,
		},
	}

	for scanner.Scan() {
		var entry attestationLogEntry
		line := scanner.Text()
		err := json.Unmarshal([]byte(line), &entry)
		if err != nil {
			return stats, err
		}

		if strings.Contains(entry.Message, attestationMsg) {
			isDelayed, attestationTime, err := s.fetchAttestationTime(entry.AttestationTime)
			stats.ConsensusClientResponseDurations = append(stats.ConsensusClientResponseDurations, attestationTime)
			if err != nil {
				return stats, err
			}

			stats.ConsensusClientResponseTimeTotal += attestationTime
			if isDelayed {
				stats.ConsensusClientResponseTimeDelayCount[s.delay]++
			}
		}
	}

	if err := scanner.Err(); err != nil {
		slog.With("err", err).Error("error reading log file")
		return stats, err
	}

	return stats, nil
}

func (s *Service) fetchAttestationTime(attestationTimeLog string) (isDelayed bool, duration time.Duration, err error) {
	var attestationDuration time.Duration

	if strings.Contains(attestationTimeLog, "ms") {
		attestationTimeMS, e := strconv.ParseFloat(strings.Replace(attestationTimeLog, "ms", "", 2), 64)
		if e != nil {
			err = errors.Join(err, errors.New("error fetching attestation time from the log message"))
			return
		}
		attestationDuration = time.Duration(attestationTimeMS * float64(time.Millisecond))
	} else if strings.Contains(attestationTimeLog, "µs") {
		attestationTimeNS, e := strconv.ParseFloat(strings.Replace(attestationTimeLog, "µs", "", 2), 64)
		if e != nil {
			err = errors.Join(err, errors.New("error fetching attestation time from the log message"))
			return
		}
		attestationDuration = time.Duration(attestationTimeNS)
	}

	if attestationDuration != 0 {
		duration = attestationDuration

		if attestationDuration > s.delay {
			isDelayed = true
		}
	}

	return
}
