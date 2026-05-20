//go:build darwin && amd64

package bin

import "embed"

//go:embed assets/darwin-amd64/cwebp
var darwinAmd64 embed.FS

func embeddedCWebP(goos, goarch string) (executableBundle, error) {
	if goos != "darwin" || goarch != "amd64" {
		return executableBundle{}, ErrUnsupportedPlatform{GOOS: goos, GOARCH: goarch}
	}
	return readDarwinAmd64Bundle()
}

func readDarwinAmd64Bundle() (executableBundle, error) {
	binBlob, err := darwinAmd64.ReadFile("assets/darwin-amd64/cwebp")
	if err != nil {
		return executableBundle{}, err
	}
	return executableBundle{Executable: binBlob, SupportFiles: nil}, nil
}
