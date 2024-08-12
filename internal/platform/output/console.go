package output

import (
	"fmt"
	"os"

	"github.com/aquasecurity/table"
)

type Console struct {
	headers []string
	t       *table.Table
}

func NewConsole(headers []string) *Console {
	t := table.New(os.Stdout)

	var alignment []table.Alignment
	for i := 0; i < len(headers); i++ {
		alignment = append(alignment, table.AlignCenter)
	}

	t.SetAlignment(alignment...)
	t.SetHeaders(headers...)

	return &Console{
		headers: headers,
		t:       t,
	}
}

func (r *Console) Update(cols []string) {
	r.t.AddRow(cols...)

	r.clearScreen()
	r.t.Render()
}

func (Console) clearScreen() {
	fmt.Print("\033[H\033[2J")
}
