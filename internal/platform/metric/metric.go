package metric

import (
	"maps"
	"sync"

	"golang.org/x/exp/constraints"
)

type (
	Metricable interface {
		constraints.Integer | constraints.Float | ~string
	}

	Base[T Metricable] struct {
		Name             string
		HealthConditions []HealthCondition[T]

		mu            sync.Mutex
		lastValues    map[string]T
		overallHealth HealthStatus
		maxSeverities map[string]SeverityLevel
	}
)

func (bm *Base[T]) GetName() string {
	return bm.Name
}

// AddDataPoint records the latest values for the given measurements and
// incrementally updates health evaluation state. It intentionally does not
// retain historical values: callers that need whole-run aggregates (e.g.
// percentiles) must track that themselves (see Histogram).
func (bm *Base[T]) AddDataPoint(values map[string]T) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if bm.lastValues == nil {
		bm.lastValues = make(map[string]T)
	}
	if bm.maxSeverities == nil {
		bm.maxSeverities = make(map[string]SeverityLevel)
		bm.overallHealth = Healthy
	}

	for name, value := range values {
		bm.lastValues[name] = value

		if _, ok := bm.maxSeverities[name]; !ok {
			bm.maxSeverities[name] = SeverityNone
		}

		for _, condition := range bm.HealthConditions {
			if condition.Name == name && condition.Evaluate(value) {
				bm.overallHealth = Unhealthy
				if CompareSeverities(condition.Severity, bm.maxSeverities[name]) > 0 {
					bm.maxSeverities[name] = condition.Severity
				}
			}
		}
	}
}

// LastValue returns the most recently recorded value for the given
// measurement name.
func (bm *Base[T]) LastValue(name string) T {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	return bm.lastValues[name]
}

func (bm *Base[T]) EvaluateMetric() (HealthStatus, map[string]SeverityLevel) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	overallHealth := bm.overallHealth
	if overallHealth == "" {
		overallHealth = Healthy
	}

	maxSeverities := make(map[string]SeverityLevel, len(bm.maxSeverities))
	maps.Copy(maxSeverities, bm.maxSeverities)

	return overallHealth, maxSeverities
}
