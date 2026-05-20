//go:build linux && amd64

package bin

import "embed"

//go:embed assets/linux-amd64/cwebp
var linuxAmd64 embed.FS

func embeddedCWebP(goos, goarch string) (executableBundle, error) {
	if goos != "linux" || goarch != "amd64" {
		return executableBundle{}, ErrUnsupportedPlatform{GOOS: goos, GOARCH: goarch}
	}
	return readLinuxAmd64Bundle()
}

func readLinuxAmd64Bundle() (executableBundle, error) {
	binBlob, err := linuxAmd64.ReadFile("assets/linux-amd64/cwebp")
	if err != nil {
		return executableBundle{}, err
	}
	return executableBundle{Executable: binBlob, SupportFiles: nil}, nil
}
