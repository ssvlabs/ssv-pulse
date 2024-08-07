package log_analyzer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"time"
)

type LogAnalyzer struct {
	logFile *os.File
}

// LogEntry represents a single log entry
type LogEntry struct {
	Level         string    `json:"L"`
	Timestamp     time.Time `json:"T"`
	Component     string    `json:"N"`
	Message       string    `json:"M"`
	Pubkey        string    `json:"pubkey"`
	Role          string    `json:"role"`
	DutyID        string    `json:"duty_id"`
	Height        int       `json:"height"`
	Round         int       `json:"round"`
	CommitSigners []int     `json:"commit-signers"`
	Root          string    `json:"root"`
}

// SignerStats keeps track of signer's score and total delay
type SignerStats struct {
	Signer int
	Score  int
	Delay  time.Duration
}

// SignerPerformance keeps track of signer's performance
type SignerPerformance struct {
	Signer   int
	Earliest time.Time
}

// Scores for ranks
var rankScores = []int{5, 4, 3, 2, 1, 0}

func (r *LogAnalyzer) New(logFilePath string) (*LogAnalyzer, error) {
	file, err := os.Open(logFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}
	return &LogAnalyzer{
		logFile: file,
	}, nil
}

func (r *LogAnalyzer) Analize() error {

	defer r.logFile.Close()

	scanner := bufio.NewScanner(r.logFile)
	dutySignerTimes := make(map[string]map[int]time.Time)
	signerStats := make(map[int]SignerStats)

	for scanner.Scan() {
		var entry LogEntry
		line := scanner.Text()
		err := json.Unmarshal([]byte(line), &entry)
		if err != nil {
			log.Printf("failed to parse log entry: %v", err)
			continue
		}

		// Consider only logs with round 1
		if entry.Message == "📬 got commit message" && entry.Round == 1 {
			dutyID := entry.DutyID
			if _, exists := dutySignerTimes[dutyID]; !exists {
				dutySignerTimes[dutyID] = make(map[int]time.Time)
			}
			for _, signer := range entry.CommitSigners {
				// Record the earliest time for each signer
				if existingTime, exists := dutySignerTimes[dutyID][signer]; !exists || entry.Timestamp.Before(existingTime) {
					dutySignerTimes[dutyID][signer] = entry.Timestamp
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("error reading log file: %v", err)
	}

	// Calculate scores and delays
	for _, signers := range dutySignerTimes {
		performances := []SignerPerformance{}
		for signer, earliestTime := range signers {
			performances = append(performances, SignerPerformance{
				Signer:   signer,
				Earliest: earliestTime,
			})
		}

		// Sort by earliest time, the earlier the better
		sort.Slice(performances, func(i, j int) bool {
			return performances[i].Earliest.Before(performances[j].Earliest)
		})

		// Assign scores and calculate delays
		if len(performances) > 0 {
			firstTime := performances[0].Earliest
			for rank, performance := range performances {
				if rank < len(rankScores) {
					signerStats[performance.Signer] = SignerStats{
						Signer: performance.Signer,
						Score:  signerStats[performance.Signer].Score + rankScores[rank],
						Delay:  signerStats[performance.Signer].Delay + performance.Earliest.Sub(firstTime),
					}
				}
			}
		}
	}

	// Collect stats into a slice for sorting
	sortedSigners := make([]SignerStats, 0, len(signerStats))
	for _, stats := range signerStats {
		sortedSigners = append(sortedSigners, stats)
	}

	// Sort signers by score in descending order
	sort.Slice(sortedSigners, func(i, j int) bool {
		return sortedSigners[i].Score > sortedSigners[j].Score
	})

	// Output scores and delays per signer
	fmt.Println("Total Scores and Delays per Signer:")
	for _, stats := range sortedSigners {
		fmt.Printf("Signer: %d, Score: %d, Total Delay: %d seconds\n", stats.Signer, stats.Score, int(stats.Delay.Seconds()))
	}
	return nil
}
