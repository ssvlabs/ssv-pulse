package prepare

import (
	"encoding/json"

	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser"
)

type prepareLogEntry struct {
	Timestamp parser.MultiFormatTime `json:"T"`
	// Round is deprecated by https://github.com/ssvlabs/ssv/pull/2453#discussion_r2287196265 but kept for now
	//for backward-compatibility, use QBFTRound instead (we can remove Round later).
	Round          uint8             `json:"round"`
	QBFTRound      uint8             `json:"qbft_round"`
	DutyID         string            `json:"duty_id"`
	Message        string            `json:"M"`
	PrepareSigners []parser.SignerID `json:"prepare_signers"`
}

func (p *prepareLogEntry) UnmarshalJSON(data []byte) error {
	type Alias prepareLogEntry

	alias := &struct {
		PrepareSignersDash []parser.SignerID `json:"prepare-signers"`
		*Alias
	}{
		Alias: (*Alias)(p),
	}

	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	if alias.PrepareSignersDash != nil {
		p.PrepareSigners = alias.PrepareSignersDash
	}

	return nil
}
