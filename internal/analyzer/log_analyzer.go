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

	"github.com/jedib0t/go-pretty/v6/table"

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

type Result struct {
	ID                     uint64
	Score                  uint64
	TotalDelay             time.Duration
	AttestationTimeAverage time.Duration
	AttestationDelayCount  int
	AttestationTotalCount  uint64
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

func (r *LogAnalyzer) AnalyzeConsensus() (*Result, error) {
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
			log.Printf("error decoding: %v", err)
			if e, ok := err.(*json.SyntaxError); ok {
				log.Printf("syntax error at byte offset %d", e.Offset)
			}
			return nil, err
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
			writeTimeStamps(entry, prepareSignerTimes, entry.DutyID, entry.PrepareSigners[0])
		}

		// Consider only logs with round 1
		if strings.Contains(entry.Message, "got commit message") && entry.Round == 1 {
			writeTimeStamps(entry, commitSignerTimes, entry.DutyID, entry.CommitSigners[0])
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("error reading log file: %v", err)
	}
	// Calculate commit scores and delays
	commitStats := r.calcCommitTimes(commitSignerTimes)
	// Calculate propose delays
	proposeStats := r.calcPrepareTimes(leaderProposeTime, prepareSignerTimes)
	// collect all IDs
	IDs := collectIDs(commitStats, proposeStats)
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Operator", "Beacon Time: avg", "Beacon Time: > 1sec", "Score", "Commit: Total Delay", "Prepare: avg", "Prepare: highest", "Prepare: > 1sec"})
	for _, ID := range IDs {
		fmt.Printf("ID: %d \n", ID)
		score := 0
		totalDelay := 0
		for _, commitStat := range commitStats {
			if commitStat.Signer == int(ID) {
				score = commitStat.Score
				totalDelay = int(commitStat.Delay.Seconds())
			}
		}
		t.AppendRows([]table.Row{
			{
				ID,
				attestationTimeAverage,
				strconv.Itoa(attestationDelayCount) + "/" + strconv.Itoa(int(attestationTimeCount)),
				score,
				totalDelay,
				proposeStats[int(ID)].AverageDelay,
				proposeStats[int(ID)].HighestDelay.Milliseconds(),
				strconv.Itoa(proposeStats[int(ID)].MoreSecondDelay) + "/" + strconv.Itoa(proposeStats[int(ID)].Count),
			},
		})
		t.AppendSeparator()
	}
	t.Render()
	return nil, nil
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

	if len(operators) < 4 || len(operators) > 13 || len(operators)%2 != 1 {
		return false
	}

	if len(operators) != len(signers) {
		return false
	}

	for _, id := range operators {
		_, exist := signers[int(id)]
		if !exist {
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

	prepareMessageCount := make(map[int]int, 0)
	prepareMessageCountMoreSecond := make(map[int]int, 0)
	averageTimePrepareMessage := make(map[int]int64, 0)
	totalTimePrepareMessage := make(map[int]time.Duration, 0)
	highestTimePrepareMessage := make(map[int]time.Duration, 0)

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
			prepareMessageCount[signer] = prepareMessageCount[signer] + 1
			totalTimePrepareMessage[signer] = totalTimePrepareMessage[signer] + delay
			if highestTimePrepareMessage[signer] < delay {
				highestTimePrepareMessage[signer] = delay
			}
			if delay > time.Second {
				prepareMessageCountMoreSecond[signer] = prepareMessageCountMoreSecond[signer] + 1
			}
			averageTimePrepareMessage[signer] = totalTimePrepareMessage[signer].Milliseconds() / int64(prepareMessageCount[signer])
			proposeStats[signer] = ProposeSignerStats{
				Signer:          signer,
				Count:           prepareMessageCount[signer],
				AverageDelay:    averageTimePrepareMessage[signer],
				HighestDelay:    highestTimePrepareMessage[signer],
				MoreSecondDelay: prepareMessageCountMoreSecond[signer],
			}
		}
	}
	return proposeStats
}

func collectIDs(commitStats []SignerStats, proposeStats map[int]ProposeSignerStats) []uint64 {
	var IDs []uint64
	tmpIDs := make(map[uint64]bool, 0)
	for _, commitStat := range commitStats {
		if _, exist := tmpIDs[uint64(commitStat.Signer)]; !exist {
			tmpIDs[uint64(commitStat.Signer)] = true
		}
	}
	for _, proposeStat := range proposeStats {
		if _, exist := tmpIDs[uint64(proposeStat.Signer)]; !exist {
			tmpIDs[uint64(proposeStat.Signer)] = true
		}
	}
	for ID, _ := range tmpIDs {
		IDs = append(IDs, ID)
	}
	sort.Slice(IDs, func(i, j int) bool {
		return IDs[i] < IDs[j]
	})
	return IDs
}
