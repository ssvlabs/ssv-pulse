package commit

import (
	"encoding/json"

	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser"
)

type commitLogEntry struct {
	Timestamp parser.MultiFormatTime `json:"T"`
	// Round is deprecated by https://github.com/ssvlabs/ssv/pull/2453#discussion_r2287196265 but kept for now
	// for backward-compatibility, use QBFTRound instead (we can remove Round later).
	Round         uint8             `json:"round"`
	QBFTRound     uint8             `json:"qbft_round"`
	DutyID        string            `json:"duty_id"`
	Message       string            `json:"M"`
	CommitSigners []parser.SignerID `json:"commit_signers"` //NOTE: This array always contains 1 item
}

func (p *commitLogEntry) UnmarshalJSON(data []byte) error {
	type Entry commitLogEntry

	alias := &struct {
		CommitSignersDash []parser.SignerID `json:"commit-signers"`
		*Entry
	}{
		Entry: (*Entry)(p),
	}

	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	if alias.CommitSignersDash != nil {
		p.CommitSigners = alias.CommitSignersDash
	}

	return nil
}
