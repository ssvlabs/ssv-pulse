package metric

type (
	Group         string
	Name          string
	HealthStatus  string
	SeverityLevel string

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

	Healthy   HealthStatus = "Healthy✅"
	Unhealthy HealthStatus = "Unhealthy⚠️"

	SeverityNone   SeverityLevel = "None"
	SeverityLow    SeverityLevel = "Low"
	SeverityMedium SeverityLevel = "Medium"
	SeverityHigh   SeverityLevel = "High"
)
