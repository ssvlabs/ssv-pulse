package operator

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"

	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser"
	"github.com/ssvlabs/ssv-pulse/internal/ssv"
)

const (
	proposalMsg      = "📢 got proposal, broadcasting prepare message"
	savedInstanceMsg = "💾 saved instance upon decided"
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
	scanner := bufio.NewScanner(s.logFile)
	var (
		stats Stats = Stats{
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

		if strings.Contains(entry.Message, savedInstanceMsg) {
			if ssv.IsValidClusterSize(entry.Signers) {
				//verify we store only distinct arrays
				if len(clusters) > 0 {
					var isUniqueArray bool
					for _, cluster := range clusters {
						if len(cluster) == len(entry.Signers) {
							for _, signerID := range entry.Signers {
								if !slices.Contains(cluster, signerID) {
									isUniqueArray = true
								}
							}
						}
					}
					if isUniqueArray {
						clusters = append(clusters, entry.Signers)
					}
				} else {
					clusters = append(clusters, entry.Signers)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		slog.With("err", err).Error("error reading log file")
		return stats, err
	}

	stats.Clusters[stats.Owner] = clusters

	return stats, nil
}
