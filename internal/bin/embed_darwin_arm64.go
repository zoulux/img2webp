//go:build darwin && arm64

package bin

import "embed"

//go:embed assets/darwin-arm64/cwebp
var darwinArm64 embed.FS

func embeddedCWebP(goos, goarch string) (executableBundle, error) {
	if goos != "darwin" || goarch != "arm64" {
		return executableBundle{}, ErrUnsupportedPlatform{GOOS: goos, GOARCH: goarch}
	}
	return readDarwinArm64Bundle()
}

func readDarwinArm64Bundle() (executableBundle, error) {
	binBlob, err := darwinArm64.ReadFile("assets/darwin-arm64/cwebp")
	if err != nil {
		return executableBundle{}, err
	}
	return executableBundle{Executable: binBlob, SupportFiles: nil}, nil
}
