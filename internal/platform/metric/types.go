package metric

type (
	Group string
	Name  string
)

const (
	ConsensusGroup      Group = "Consensus"
	ExecutionGroup      Group = "Execution"
	SSVGroup            Group = "SSV"
	InfrastructureGroup Group = "Infrastructure"
)
