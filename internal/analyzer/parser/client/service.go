package client

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
	"github.com/ssvlabs/ssv-pulse/internal/platform/metric"
)

const (
	attestationMsg = "starting QBFT instance"

	parserName = "client"
)

type (
	attestationLogEntry struct {
		Message         string `json:"M"`
		AttestationTime string `json:"attestation_data_time"`
	}

	Stats struct {
		ConsensusClientResponseTimeDelayPercent map[time.Duration]float32
		ConsensusClientResponseTimeAvg,
		ConsensusClientResponseTimeP10,
		ConsensusClientResponseTimeP50,
		ConsensusClientResponseTimeP90 time.Duration
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
	scanner := parser.NewScanner(s.logFile)

	var (
		stats Stats = Stats{
			ConsensusClientResponseTimeDelayPercent: map[time.Duration]float32{
				s.delay: 0,
			}}

		attestationEndpointResponseTimes   []time.Duration
		attestationEndpointResponseTimeSum time.Duration
		attestationEndpointTotalDelayed    = make(map[time.Duration]uint32)
	)

	for scanner.Scan() {
		var entry attestationLogEntry
		line := scanner.Text()
		err := json.Unmarshal([]byte(line), &entry)
		if err != nil {
			return stats, err
		}

		if strings.Contains(entry.Message, attestationMsg) {
			isDelayed, responseTime, err := s.fetchAttestationTime(entry.AttestationTime)
			if err != nil {
				return stats, err
			}
			attestationEndpointResponseTimes = append(attestationEndpointResponseTimes, responseTime)
			attestationEndpointResponseTimeSum += responseTime

			if isDelayed {
				stats.ConsensusClientResponseTimeDelayPercent[s.delay]++
				attestationEndpointTotalDelayed[s.delay]++
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

	if len(attestationEndpointResponseTimes) > 0 {
		percentiles := metric.CalculatePercentiles(attestationEndpointResponseTimes, 10, 50, 90)

		stats.ConsensusClientResponseTimeAvg = attestationEndpointResponseTimeSum / time.Duration(len(attestationEndpointResponseTimes))
		stats.ConsensusClientResponseTimeP10 = percentiles[10]
		stats.ConsensusClientResponseTimeP50 = percentiles[50]
		stats.ConsensusClientResponseTimeP90 = percentiles[90]

		stats.ConsensusClientResponseTimeDelayPercent[s.delay] =
			float32(attestationEndpointTotalDelayed[s.delay]) / float32(len(attestationEndpointResponseTimes)) * 100
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
