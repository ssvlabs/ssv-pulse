package metric

type Group string

const (
	ConsensusGroup      Group = "Consensus"
	ExecutionGroup      Group = "Execution"
	SSVGroup            Group = "SSV"
	InfrastructureGroup Group = "Infrastructure"
)

type GroupResult struct {
	ViewResult string
	Health     HealthStatus
	Severity   map[string]SeverityLevel
}
