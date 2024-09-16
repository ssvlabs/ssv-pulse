package commit

import (
	"encoding/json"
	"time"

	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser"
)

type commitLogEntry struct {
	Timestamp     time.Time         `json:"T"`
	Round         uint8             `json:"round"`
	DutyID        string            `json:"duty_id"`
	Message       string            `json:"M"`
	CommitSigners []parser.SignerID `json:"commit_signers"`
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