package analyzer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ssvlabs/ssv-benchmark/internal/utils"
)

type LogAnalyzer struct {
	logFile *os.File
	cluster []uint64
}

// LogEntry represents a single log entry
type LogEntry struct {
	Level           string    `json:"L"`
	Timestamp       time.Time `json:"T"`
	Component       string    `json:"N"`
	Message         string    `json:"M"`
	Pubkey          string    `json:"pubkey"`
	Role            string    `json:"role"`
	DutyID          string    `json:"duty_id"`
	Height          int       `json:"height"`
	Round           int       `json:"round"`
	CommitSigners   []int     `json:"commit-signers"`
	Root            string    `json:"root"`
	AttestationTime string    `json:"attestation_data_time"`
	Leader          int       `json:"leader"`
	PrepareSigners  []int     `json:"prepare-signers"`
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

type ProposeSignerStats struct {
	Signer          int
	Count           int
	AverageDelay    int64
	HighestDelay    time.Duration
	MoreSecondDelay int
}

// Scores for ranks
var rankScores = []int{5, 4, 3, 2, 1, 0}

func New(logFilePath string, cluster []string) (*LogAnalyzer, error) {
	file, err := os.Open(logFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}
	ids, err := utils.StingSliceToUintArray(cluster)
	if err != nil {
		return nil, err
	}
	return &LogAnalyzer{
		logFile: file,
		cluster: ids,
	}, nil
}

func (r *LogAnalyzer) AnalyzeConsensus() error {
	defer r.logFile.Close()

	scanner := bufio.NewScanner(r.logFile)
	dutySignerTimes := make(map[string]map[int]time.Time)
	signerStats := make(map[int]SignerStats)
	var attestationTimeCount uint64
	var attestationTimeTotal uint64
	var attestationTimeAverage uint64
	attestationDelayCount := 0

	// Calculate propose-prepare times
	proposeStats := make(map[int]ProposeSignerStats)
	prepareSignerTimes := make(map[string]map[int]time.Duration)
	leaderProposeTime := make(map[string]time.Time)
	for scanner.Scan() {
		var entry LogEntry
		line := scanner.Text()
		err := json.Unmarshal([]byte(line), &entry)
		if err != nil {
			log.Printf("failed to parse log entry: %v", err)
			continue
		}

		// Check attestation_data_time
		if strings.Contains(entry.Message, "starting QBFT instance") {
			var t float64
			t, err = strconv.ParseFloat(strings.Replace(entry.AttestationTime, "ms", "", 2), 64)
			// try parse mircosec
			if err != nil {
				t, _ = strconv.ParseFloat(strings.Replace(entry.AttestationTime, "µs", "", 2), 64)
				t = t / 1000
			}
			if t != 0 {
				attestationTimeCount = attestationTimeCount + 1
				attestationTimeTotal = attestationTimeTotal + uint64(t)
				attestationTimeAverage = attestationTimeTotal / attestationTimeCount
				if uint64(t) > 1000 {
					attestationDelayCount++
				}
			}
		}

		// Check leader propose message
		if strings.Contains(entry.Message, "leader broadcasting proposal message") {
			leaderProposeTime[entry.DutyID] = entry.Timestamp
		}

		if strings.Contains(entry.Message, "got prepare message") && entry.Round == 1 {
			dutyID := entry.DutyID
			if leaderProposeTime, exist := leaderProposeTime[dutyID]; exist {
				// Check if signer among provided at CI
				for _, signer := range entry.PrepareSigners {
					for _, ID := range r.cluster {
						if uint64(signer) == ID {
							if _, exists := prepareSignerTimes[dutyID]; !exists {
								prepareSignerTimes[dutyID] = make(map[int]time.Duration)
							}
							// Record the earliest time for each signer
							if _, exists := prepareSignerTimes[dutyID][signer]; !exists || entry.Timestamp.After(leaderProposeTime) {
								prepareSignerTimes[dutyID][signer] = entry.Timestamp.Sub(leaderProposeTime)
							}
						}
					}
				}
			}
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

	// Calculate propose time delays
	for _, ID := range r.cluster {
		var prepareMessageCount int
		var prepareMessageCountMoreSecond int
		var averageTimePrepareMessage int64
		var totalTimePrepareMessage time.Duration
		var highestTimePrepareMessage time.Duration
		for _, signers := range prepareSignerTimes {
			for signer, delay := range signers {
				if signer == int(ID) {
					prepareMessageCount = prepareMessageCount + 1
					totalTimePrepareMessage = totalTimePrepareMessage + delay
					if highestTimePrepareMessage < delay {
						highestTimePrepareMessage = delay
					}
					if delay > time.Second {
						prepareMessageCountMoreSecond = prepareMessageCountMoreSecond + 1
					}
				}

			}
		}
		averageTimePrepareMessage = totalTimePrepareMessage.Milliseconds() / int64(prepareMessageCount)
		proposeStats[int(ID)] = ProposeSignerStats{
			Signer:          int(ID),
			Count:           prepareMessageCount,
			AverageDelay:    averageTimePrepareMessage,
			HighestDelay:    highestTimePrepareMessage,
			MoreSecondDelay: prepareMessageCountMoreSecond,
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
	fmt.Printf("attestation data time average: %dms\n", attestationTimeAverage)
	fmt.Printf("attestation data time > 1 sec: %d/%d\n", attestationDelayCount, attestationTimeCount)

	for _, ID := range r.cluster {
		fmt.Printf("ID: %d \n", ID)
		for _, stats := range sortedSigners {
			if stats.Signer == int(ID) {
				fmt.Printf("Score: %d\n", stats.Score)
				fmt.Printf("Total Delay: %d seconds\n", int(stats.Delay.Seconds()))
			}
		}
		fmt.Printf("Average time to receive prepare message when leader (ms): %d\n", proposeStats[int(ID)].AverageDelay)
		fmt.Printf("Highest time to receive prepare message when leader (ms): %d\n", proposeStats[int(ID)].HighestDelay.Milliseconds())
		fmt.Printf("Time to receive prepare message when leader > 1 sec: %d/%d\n", proposeStats[int(ID)].MoreSecondDelay, proposeStats[int(ID)].Count)
	}
	return nil
}
