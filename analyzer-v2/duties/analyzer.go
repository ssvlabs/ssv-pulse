package duties

import (
	"fmt"
	"log/slog"
	"math"
	"os"
	"path"

	"github.com/attestantio/go-eth2-client/spec/phase0"
)

type analyzer interface {
	Analyze(logFilePath string, targetSlot phase0.Slot) error
}

func Analyze(a analyzer, dir string, files []os.DirEntry, targetSlot phase0.Slot) error {
	for _, file := range files {
		filePath := path.Join(dir, file.Name())

		fileSizeMB := 0.0
		stat, err := os.Stat(filePath)
		if err != nil {
			slog.With("err", err.Error()).Warn(fmt.Sprintf("error fetching `%s` file info, will try to read the file anyway", file.Name()))
		}
		if err == nil {
			fileSizeMB = float64(stat.Size()) / (1024 * 1024)
		}
		slog.
			With("file_size_megabytes", math.Round(fileSizeMB)).
			Info(fmt.Sprintf("⏳⏳⏳ analyzing log file: %s", file.Name()))

		err = a.Analyze(filePath, targetSlot)
		if err != nil {
			return fmt.Errorf("commitee: analyze file: %w", err)
		}
	}

	return nil
}
