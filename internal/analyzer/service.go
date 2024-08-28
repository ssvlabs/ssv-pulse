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
	logFile   *os.File
	operators []uint64
	cluster   bool
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

func New(logFilePath string, operators []string, cluster bool) (*LogAnalyzer, error) {
	file, err := os.Open(logFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}
	ids, err := utils.StingSliceToUintArray(operators)
	if err != nil {
		return nil, err
	}
	return &LogAnalyzer{
		logFile:   file,
		operators: ids,
		cluster:   cluster,
	}, nil
}

func (r *LogAnalyzer) AnalyzeConsensus() error {
	defer r.logFile.Close()

	scanner := bufio.NewScanner(r.logFile)
	commitSignerTimes := make(map[string]map[int]time.Time)
	var attestationTimeCount uint64
	var attestationTimeTotal uint64
	var attestationTimeAverage uint64
	attestationDelayCount := 0

	// Calculate propose-prepare times
	prepareSignerTimes := make(map[string]map[int]time.Time)
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
			// Check if signer among provided at CI
			for _, signer := range entry.PrepareSigners {
				writeTimeStamps(entry, prepareSignerTimes, entry.DutyID, signer)
			}
		}

		// Consider only logs with round 1
		if strings.Contains(entry.Message, "got commit message") && entry.Round == 1 {
			for _, signer := range entry.CommitSigners {
				writeTimeStamps(entry, commitSignerTimes, entry.DutyID, signer)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("error reading log file: %v", err)
	}
	// Calculate commit scores and delays
	commitStats := r.calcCommitTimes(commitSignerTimes)
	// Calculate propose delays
	proposeStats := r.calcPrepareTimes(leaderProposeTime, prepareSignerTimes)
	// Output scores and delays per signer
	fmt.Printf("attestation data time average: %dms\n", attestationTimeAverage)
	fmt.Printf("attestation data time > 1 sec: %d/%d\n", attestationDelayCount, attestationTimeCount)
	if r.cluster && len(commitStats) == 0 || len(proposeStats) == 0 {
		fmt.Printf("Cluster was not found in logs, try without cluster flag... \n")
	}
	for _, stats := range commitStats {
		fmt.Printf("ID: %d \n", stats.Signer)
		fmt.Printf("Score: %d\n", stats.Score)
		fmt.Printf("Total Delay: %d seconds\n", int(stats.Delay.Seconds()))
		fmt.Printf("Average time to receive prepare message when leader (ms): %d\n", proposeStats[stats.Signer].AverageDelay)
		fmt.Printf("Highest time to receive prepare message when leader (ms): %d\n", proposeStats[stats.Signer].HighestDelay.Milliseconds())
		fmt.Printf("Time to receive prepare message when leader > 1 sec: %d/%d\n\n", proposeStats[stats.Signer].MoreSecondDelay, proposeStats[stats.Signer].Count)
	}

	return nil
}

func writeTimeStamps(entry LogEntry, signerTimes map[string]map[int]time.Time, dutyID string, signer int) {
	if _, exists := signerTimes[dutyID]; !exists {
		signerTimes[dutyID] = make(map[int]time.Time)
	}
	// Record the earliest time for each signer
	if existingTime, exists := signerTimes[dutyID][signer]; !exists || entry.Timestamp.Before(existingTime) {
		signerTimes[dutyID][signer] = entry.Timestamp
	}
}

func isCluster(operators []uint64, signers map[int]time.Time) bool {
	if len(operators) != 4 || len(operators) != 7 || len(operators) != 10 || len(operators) != 13 {
		return false
	}
	if len(operators) != len(signers) {
		return false
	}
	for _, id := range operators {
		if _, exist := signers[int(id)]; !exist {
			return false
		}
	}
	return true
}

func (r *LogAnalyzer) calcCommitTimes(commitSignerTimes map[string]map[int]time.Time) []SignerStats {
	signerStats := make(map[int]SignerStats)
	for _, signers := range commitSignerTimes {
		if r.cluster && len(r.operators) != 0 {
			if !isCluster(r.operators, signers) {
				continue
			}
		}
		var performances []SignerPerformance
		for signer, earliestTime := range signers {
			if len(r.operators) != 0 {
				var ok bool
				for _, ID := range r.operators {
					if signer == int(ID) {
						ok = true
					}
				}
				if !ok {
					continue
				}
			}
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
	return sortedSigners
}

func (r *LogAnalyzer) calcPrepareTimes(leaderProposeTime map[string]time.Time, prepareSignerTimes map[string]map[int]time.Time) map[int]ProposeSignerStats {
	proposeStats := make(map[int]ProposeSignerStats)

	var prepareMessageCount int
	var prepareMessageCountMoreSecond int
	var averageTimePrepareMessage int64
	var totalTimePrepareMessage time.Duration
	var highestTimePrepareMessage time.Duration

	for dutyID, signers := range prepareSignerTimes {
		leaderProposeMessageTime, exist := leaderProposeTime[dutyID]
		if !exist {
			continue
		}
		if r.cluster && len(r.operators) != 0 {
			if !isCluster(r.operators, signers) {
				continue
			}
		}
		for signer, prepareMessageTimeStamp := range signers {
			if len(r.operators) != 0 {
				var ok bool
				for _, ID := range r.operators {
					if signer == int(ID) {
						ok = true
					}
				}
				if !ok {
					continue
				}
			}
			if prepareMessageTimeStamp.Before(leaderProposeMessageTime) {
				log.Println("error: got prepare message before leader propose message")
				break
			}
			delay := prepareMessageTimeStamp.Sub(leaderProposeMessageTime)
			prepareMessageCount = prepareMessageCount + 1
			totalTimePrepareMessage = totalTimePrepareMessage + delay
			if highestTimePrepareMessage < delay {
				highestTimePrepareMessage = delay
			}
			if delay > time.Second {
				prepareMessageCountMoreSecond = prepareMessageCountMoreSecond + 1
			}
			averageTimePrepareMessage = totalTimePrepareMessage.Milliseconds() / int64(prepareMessageCount)
			proposeStats[signer] = ProposeSignerStats{
				Signer:          signer,
				Count:           prepareMessageCount,
				AverageDelay:    averageTimePrepareMessage,
				HighestDelay:    highestTimePrepareMessage,
				MoreSecondDelay: prepareMessageCountMoreSecond,
			}
		}
	}
	return proposeStats
}
