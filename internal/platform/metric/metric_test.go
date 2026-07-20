package metric

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGivenNoDataPointsWhenEvaluateMetricThenHealthy(t *testing.T) {
	bm := Base[int]{}

	health, severities := bm.EvaluateMetric()

	assert.Equal(t, Healthy, health)
	assert.Empty(t, severities)
}

func TestGivenDataPointWithoutViolationWhenEvaluateMetricThenHealthyWithNoneSeverity(t *testing.T) {
	bm := Base[int]{
		HealthConditions: []HealthCondition[int]{
			{Name: "Latency", Threshold: 100, Operator: OperatorGreaterThanOrEqual, Severity: SeverityHigh},
		},
	}

	bm.AddDataPoint(map[string]int{"Latency": 10})

	health, severities := bm.EvaluateMetric()

	assert.Equal(t, Healthy, health)
	assert.Equal(t, map[string]SeverityLevel{"Latency": SeverityNone}, severities)
}

func TestGivenViolatingDataPointWhenEvaluateMetricThenUnhealthyWithSeverity(t *testing.T) {
	bm := Base[int]{
		HealthConditions: []HealthCondition[int]{
			{Name: "Latency", Threshold: 100, Operator: OperatorGreaterThanOrEqual, Severity: SeverityHigh},
		},
	}

	bm.AddDataPoint(map[string]int{"Latency": 150})

	health, severities := bm.EvaluateMetric()

	assert.Equal(t, Unhealthy, health)
	assert.Equal(t, map[string]SeverityLevel{"Latency": SeverityHigh}, severities)
}

func TestGivenEscalatingViolationsWhenEvaluateMetricThenKeepsWorstSeverityEverSeen(t *testing.T) {
	bm := Base[int]{
		HealthConditions: []HealthCondition[int]{
			{Name: "Peers", Threshold: 40, Operator: OperatorLessThanOrEqual, Severity: SeverityLow},
			{Name: "Peers", Threshold: 20, Operator: OperatorLessThanOrEqual, Severity: SeverityMedium},
			{Name: "Peers", Threshold: 5, Operator: OperatorLessThanOrEqual, Severity: SeverityHigh},
		},
	}

	bm.AddDataPoint(map[string]int{"Peers": 30})  // Low
	bm.AddDataPoint(map[string]int{"Peers": 3})   // High
	bm.AddDataPoint(map[string]int{"Peers": 100}) // healthy again

	health, severities := bm.EvaluateMetric()

	// Matches the old full-history-rescan semantics: once the worst severity
	// for a measurement is observed, it is never downgraded by a later,
	// healthier reading — the report reflects the worst state seen over the
	// whole run.
	assert.Equal(t, Unhealthy, health)
	assert.Equal(t, map[string]SeverityLevel{"Peers": SeverityHigh}, severities)
}

func TestGivenMultipleMeasurementNamesWhenEvaluateMetricThenTracksIndependently(t *testing.T) {
	bm := Base[int]{
		HealthConditions: []HealthCondition[int]{
			{Name: "A", Threshold: 10, Operator: OperatorGreaterThanOrEqual, Severity: SeverityHigh},
			{Name: "B", Threshold: 10, Operator: OperatorGreaterThanOrEqual, Severity: SeverityLow},
		},
	}

	bm.AddDataPoint(map[string]int{"A": 100, "B": 1})

	health, severities := bm.EvaluateMetric()

	assert.Equal(t, Unhealthy, health)
	assert.Equal(t, map[string]SeverityLevel{"A": SeverityHigh, "B": SeverityNone}, severities)
}

func TestGivenMultipleDataPointsWhenLastValueThenReturnsMostRecent(t *testing.T) {
	bm := Base[int]{}

	bm.AddDataPoint(map[string]int{"X": 1})
	bm.AddDataPoint(map[string]int{"X": 2})
	bm.AddDataPoint(map[string]int{"X": 3})

	assert.Equal(t, 3, bm.LastValue("X"))
}

func TestGivenNoDataPointsWhenLastValueThenReturnsZeroValue(t *testing.T) {
	bm := Base[int]{}

	assert.Equal(t, 0, bm.LastValue("X"))
}
