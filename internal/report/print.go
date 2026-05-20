package report

import (
	"fmt"
	"io"
)

func PrintSummary(w io.Writer, s Summary) {
	saved := s.SourceBytes - s.OutputBytes
	ratio := 0.0
	if s.SourceBytes > 0 {
		ratio = float64(saved) / float64(s.SourceBytes) * 100
	}
	fmt.Fprintf(
		w,
		"files=%d success=%d skipped=%d failed=%d saved=%d (%.1f%%)\n",
		s.TotalFiles,
		s.Success,
		s.Skipped,
		s.Failed,
		saved,
		ratio,
	)
}
