package operator

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser"
	"github.com/ssvlabs/ssv-pulse/internal/platform/array"
)

const (
	proposalMsg      = "📢 got proposal, broadcasting prepare message"
	savedInstanceMsg = "💾 saved instance upon decided"

	parserName = "operator"
)

type (
	Stats struct {
		Owner    parser.SignerID
		Clusters map[parser.SignerID][][]parser.SignerID
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
			Clusters: make(map[parser.SignerID][][]uint32),
		}
		clusters [][]parser.SignerID
	)

	for scanner.Scan() {
		line := scanner.Text()
		var entry logEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return stats, err
		}

		if strings.Contains(entry.Message, proposalMsg) {
			if len(entry.PrepareSigners) != 1 {
				errMsg := fmt.Sprintf(
					"the log message contained unexpected number of items. Could not determine the owner correctly. File name: %s", s.logFile.Name(),
				)
				return stats, errors.New(errMsg)
			}
			stats.Owner = entry.PrepareSigners[0]
		}

		if entry.DutyID != "" && strings.HasPrefix(entry.DutyID, "COMMITTEE") {
			newCluster, err := extractClusterIDs(entry.DutyID)
			if err != nil {
				slog.
					With("entry", entry).
					Warn("error extracting cluster IDs from the log entry")
				continue
			}
			var isUniqueCluster bool
			if len(clusters) == 0 {
				isUniqueCluster = true
			} else {
				isUniqueCluster = true
				for _, cluster := range clusters {
					if array.SameMembers(newCluster, cluster) {
						isUniqueCluster = false
						break
					}
				}
			}
			if isUniqueCluster {
				clusters = append(clusters, newCluster)
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

	stats.Clusters[stats.Owner] = clusters

	return stats, nil
}

func extractClusterIDs(input string) ([]parser.SignerID, error) {
	re := regexp.MustCompile(`COMMITTEE-([\d_]+)-`)
	matches := re.FindStringSubmatch(input)
	if len(matches) < 2 {
		return nil, errors.New("failed to find committee IDs in the string")
	}

	ids := strings.Split(matches[1], "_")
	var result []parser.SignerID
	for _, id := range ids {
		num, err := strconv.ParseUint(id, 10, 32)
		if err != nil {
			return nil, err
		}
		result = append(result, parser.SignerID(num))
	}

	return result, nil
}
