package operator

import (
	"bufio"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"strings"

	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser"
)

const (
	proposalMsg = "📢 got proposal, broadcasting prepare message"
)

type (
	logEntry struct {
		Message        string            `json:"M"`
		PrepareSigners []parser.SignerID `json:"prepare_signers"`
	}
	Stats struct {
		Owner parser.SignerID
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
		stats Stats
	)

	for scanner.Scan() {
		line := scanner.Text()
		var entry logEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return stats, err
		}

		if strings.Contains(entry.Message, proposalMsg) {
			if len(entry.PrepareSigners) != 1 {
				const errMsg = "the log message contained unexpected number of items. Could not determine the owner correctly"
				return stats, errors.New(errMsg)
			}
			stats.Owner = entry.PrepareSigners[0]
			return stats, nil
		}
	}

	if err := scanner.Err(); err != nil {
		slog.With("err", err).Error("error reading log file")
		return stats, err
	}

	return stats, nil
}
