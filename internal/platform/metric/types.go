package metric

type (
	Group  string
	Name   string
	Result struct {
		Value    []byte
		Health   HealthStatus
		Severity SeverityLevel
	}
)

const (
	ConsensusGroup      Group = "Consensus"
	ExecutionGroup      Group = "Execution"
	SSVGroup            Group = "SSV"
	InfrastructureGroup Group = "Infrastructure"
)
