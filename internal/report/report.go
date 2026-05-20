package report

type Status string

const (
	StatusSuccess Status = "success"
	StatusSkipped Status = "skipped"
	StatusFailed  Status = "failed"
)

type Result struct {
	InputPath   string
	OutputPath  string
	Status      Status
	SourceBytes int64
	OutputBytes int64
	Message     string
}

type Summary struct {
	TotalFiles  int
	Success     int
	Skipped     int
	Failed      int
	SourceBytes int64
	OutputBytes int64
}

func (s *Summary) Add(r Result) {
	s.TotalFiles++
	s.SourceBytes += r.SourceBytes
	s.OutputBytes += r.OutputBytes

	switch r.Status {
	case StatusSuccess:
		s.Success++
	case StatusSkipped:
		s.Skipped++
	case StatusFailed:
		s.Failed++
	}
}
