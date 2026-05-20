package bin

type executableBundle struct {
	Executable   []byte
	SupportFiles map[string][]byte
}

type ErrUnsupportedPlatform struct {
	GOOS   string
	GOARCH string
}

func (e ErrUnsupportedPlatform) Error() string {
	return "unsupported platform: " + e.GOOS + "/" + e.GOARCH
}
