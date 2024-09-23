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
	attestationDelay = time.Second
	attestationMsg   = "starting QBFT instance"
)

type (
	attestationLogEntry struct {
		Message         string `json:"M"`
		AttestationTime string `json:"attestation_data_time"`
	}

	Stats struct {
		AttestationDelayCount uint16
		AttestationTimeTotal  time.Duration
		AttestationDurations  []time.Duration
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

func (r *Service) Analyze() (Stats, error) {
	defer r.logFile.Close()
	scanner := bufio.NewScanner(r.logFile)

	stats := Stats{}

	for scanner.Scan() {
		var entry attestationLogEntry
		line := scanner.Text()
		err := json.Unmarshal([]byte(line), &entry)
		if err != nil {
			return stats, err
		}

		if strings.Contains(entry.Message, attestationMsg) {
			isDelayed, attestationTime, err := fetchAttestationTime(entry.AttestationTime)
			stats.AttestationDurations = append(stats.AttestationDurations, attestationTime)
			if err != nil {
				return stats, err
			}

			stats.AttestationTimeTotal += attestationTime
			if isDelayed {
				stats.AttestationDelayCount++
			}
		}
	}

	if err := scanner.Err(); err != nil {
		slog.With("err", err).Error("error reading log file")
		return stats, err
	}

	return stats, nil
}

func fetchAttestationTime(attestationTimeLog string) (isDelayed bool, duration time.Duration, err error) {
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

		if attestationDuration > attestationDelay {
			isDelayed = true
		}
	}

	return
}
