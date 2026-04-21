package config

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/spec/phase0"
)

type Config struct {
	LogFilesDirectory string `mapstructure:"log-files-directory"`

	Blockchain string `mapstructure:"blockchain"`

	LogFormat string `mapstructure:"log-format"`

	AnalyzeCommitteeDuty             bool `mapstructure:"analyze-committee-duty"`
	AnalyzeProposerDuty              bool `mapstructure:"analyze-proposer-duty"`
	AnalyzeAggregatorDuty            bool `mapstructure:"analyze-aggregator-duty"`
	AnalyzeSyncCommitteeContribution bool `mapstructure:"analyze-sync-committee-contribution-duty"`

	DutyID     string      `mapstructure:"duty-id"`
	TargetSlot phase0.Slot `mapstructure:"target-slot"`
}

func (c *Config) Validate() error {
	if c.LogFilesDirectory == "" {
		return fmt.Errorf("❕ 'log-files-directory' was not specified")
	}
	if c.Blockchain == "" {
		return fmt.Errorf("❕ 'blockchain' was not specified")
	}
	if c.LogFormat == "" {
		return fmt.Errorf("❕ 'log-format' was not specified")
	}
	if c.DutyID == "" && c.TargetSlot == 0 {
		return fmt.Errorf("❕ at least one of `duty-id`, 'target-slot' must be specified")
	}

	if !c.AnalyzeCommitteeDuty && !c.AnalyzeProposerDuty && !c.AnalyzeAggregatorDuty && !c.AnalyzeSyncCommitteeContribution {
		return fmt.Errorf("❕ must specify at least 1 duty analyzer")
	}

	return nil
}
