package report

import (
	"os"
	"sync"

	"github.com/aquasecurity/table"
	"github.com/ssvlabs/ssv-benchmark/internal/platform/metric"
)

var headers = []string{"Group Name", "Metric Name", "Value", "Health", "Severity"}

type Record struct {
	GroupName  metric.Group
	MetricName metric.Name
	Value      string
	Health     metric.HealthStatus
	Severity   metric.SeverityLevel
}

type Report struct {
	t     *table.Table
	mutex sync.Mutex
}

func New() *Report {
	t := table.New(os.Stdout)

	t.SetHeaders(headers...)

	var alignments []table.Alignment
	for i := 0; i < len(headers); i++ {
		alignments = append(alignments, table.AlignCenter)
	}
	t.SetAlignment(alignments...)

	return &Report{
		t: t,
	}
}

func (r *Report) AddRecord(metric Record) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.t.AddRow(
		string(metric.GroupName),
		string(metric.MetricName),
		metric.Value,
		string(metric.Health),
		string(metric.Severity),
	)
}

func (r *Report) Render() {
	r.t.Render()
}
