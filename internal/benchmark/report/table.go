package report

import (
	"os"
	"sync"
	"time"

	"github.com/aquasecurity/table"
	"github.com/ssvlabs/ssv-benchmark/internal/platform/metric"
)

type Record struct {
	GroupName  metric.Group
	MetricName metric.Name
	Value      string
	Timestamp  time.Time
}

type Report struct {
	t     *table.Table
	mutex sync.Mutex
}

func New() *Report {
	t := table.New(os.Stdout)

	t.SetHeaders("Group Name", "Metric Name", "Value", "Timestamp")
	t.SetAlignment(table.AlignCenter, table.AlignCenter, table.AlignCenter, table.AlignCenter)
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
		metric.Timestamp.Format("2006-01-02 15:04:05"),
	)
}

func (r *Report) Render() {
	r.t.Render()
}
