package report

import (
	"fmt"
	"io"
)

func PrintSummary(w io.Writer, s Summary) {
	fmt.Fprintf(
		w,
		"files=%d success=%d skipped=%d failed=%d source=%d output=%d\n",
		s.TotalFiles,
		s.Success,
		s.Skipped,
		s.Failed,
		s.SourceBytes,
		s.OutputBytes,
	)
}
