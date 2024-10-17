package operator

import (
	"encoding/json"

	"github.com/ssvlabs/ssv-pulse/internal/analyzer/parser"
)

type logEntry struct {
	Message        string            `json:"M"`
	PrepareSigners []parser.SignerID `json:"prepare_signers"`
	Signers        []parser.SignerID `json:"signers"`
}

func (p *logEntry) UnmarshalJSON(data []byte) error {
	type Alias logEntry

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
