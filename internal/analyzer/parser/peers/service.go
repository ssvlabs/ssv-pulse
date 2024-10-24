package peers

import (
	"bufio"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"regexp"
	"strings"

	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser"
	"github.com/ssvlabs/ssv-pulse/internal/platform/array"
)

const (
	scoredPeersMsg  = "scored peers"
	peerIdentityMsg = "peer identity"
	peerScoresMsg   = "peer scores"

	parserName = "peers"
)

type (
	subnetStats struct {
		Max    string `json:"max"`
		Min    string `json:"min"`
		Median string `json:"median"`
	}

	peer struct {
		Peer          string  `json:"Peer"`
		Score         float32 `json:"Score"`
		SharedSubnets uint16  `json:"SharedSubnets"`
	}

	logEntry struct {
		Message     string      `json:"M"`
		SubnetStats subnetStats `json:"subnet_stats,omitempty"`
		Peers       []peer      `json:"peers,omitempty"`
		NodeVersion string      `json:"node_version,omitempty"`
		SelfPeer    string      `json:"selfPeer,omitempty"`
	}

	Stats struct {
		PeerCountAvg          parser.Metric[float64]
		PeerSSVClientVersions []string
		PeerID                string
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
		stats                 Stats
		selfPeer              string
		peerCountTotal        uint32
		peerScoresRecordCount uint16
		nodeClientVersions    []string
	)

	for scanner.Scan() {
		var entry logEntry
		line := scanner.Text()

		if strings.Contains(line, scoredPeersMsg) || strings.Contains(line, peerIdentityMsg) || strings.Contains(line, peerScoresMsg) {
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				slog.
					With("err", err).
					With("line", line).
					Info("failed to unmarshal line. Skipping")
				continue
			}

			if strings.EqualFold(entry.Message, scoredPeersMsg) {
				peerCountTotal += uint32(len(entry.Peers))
				peerScoresRecordCount++
			}

			if strings.EqualFold(entry.Message, peerIdentityMsg) {
				re := regexp.MustCompile(`^v\d+\.\d+\.\d+`) //v2.0.0, v1.3.3, ...
				if re.MatchString(entry.NodeVersion) {
					extractedVersion := re.FindString(entry.NodeVersion)
					nodeClientVersions = append(nodeClientVersions, extractedVersion)
				}
			}

			if strings.EqualFold(entry.Message, peerScoresMsg) {
				if selfPeer == "" {
					selfPeer = entry.SelfPeer
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

	distinctClientVersions := array.CollectDistinct(nodeClientVersions)

	if peerScoresRecordCount != 0 {
		stats.PeerCountAvg = parser.Metric[float64]{
			Found: true,
			Value: float64(peerCountTotal) / float64(peerScoresRecordCount),
		}
	} else {
		stats.PeerCountAvg = parser.Metric[float64]{Found: false}
	}

	stats.PeerSSVClientVersions = distinctClientVersions
	stats.PeerID = selfPeer

	return stats, nil
}
