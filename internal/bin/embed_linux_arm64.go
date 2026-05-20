//go:build linux && arm64

package bin

import "embed"

//go:embed assets/linux-arm64/cwebp
var linuxArm64 embed.FS

func embeddedCWebP(goos, goarch string) (executableBundle, error) {
	if goos != "linux" || goarch != "arm64" {
		return executableBundle{}, ErrUnsupportedPlatform{GOOS: goos, GOARCH: goarch}
	}
	return readLinuxArm64Bundle()
}

func readLinuxArm64Bundle() (executableBundle, error) {
	binBlob, err := linuxArm64.ReadFile("assets/linux-arm64/cwebp")
	if err != nil {
		return executableBundle{}, err
	}
	return executableBundle{Executable: binBlob, SupportFiles: nil}, nil
}
