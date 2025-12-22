package duties

import (
	"fmt"
	"math"
	"os"
	"path"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/sanity-io/litter"
)

type analyzer interface {
	Analyze(logFilePath string, dutyID string, targetSlot phase0.Slot) error
}

func Analyze(a analyzer, dir string, files []os.DirEntry, dutyID string, targetSlot phase0.Slot) error {
	for _, file := range files {
		filePath := path.Join(dir, file.Name())

		fileSizeMB := 0.0
		stat, err := os.Stat(filePath)
		if err != nil {
			return fmt.Errorf("fetch file info for %s: %w", file.Name(), err)
		}
		fileSizeMB = float64(stat.Size()) / (1024 * 1024)

		fmt.Println()
		fmt.Println(fmt.Sprintf("⏳⏳⏳ analyzing log file (size=%s): %s", litter.Sdump(math.Round(fileSizeMB)), file.Name()))

		err = a.Analyze(filePath, dutyID, targetSlot)
		if err != nil {
			return fmt.Errorf("commitee: analyze file: %w", err)
		}
	}

	return nil
}
