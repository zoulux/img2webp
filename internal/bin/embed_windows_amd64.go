//go:build windows && amd64

package bin

import "embed"

//go:embed assets/windows-amd64/cwebp.exe
var windowsAmd64 embed.FS

func embeddedCWebP(goos, goarch string) (executableBundle, error) {
	if goos != "windows" || goarch != "amd64" {
		return executableBundle{}, ErrUnsupportedPlatform{GOOS: goos, GOARCH: goarch}
	}
	return readWindowsAmd64Bundle()
}

func readWindowsAmd64Bundle() (executableBundle, error) {
	binBlob, err := windowsAmd64.ReadFile("assets/windows-amd64/cwebp.exe")
	if err != nil {
		return executableBundle{}, err
	}
	return executableBundle{Executable: binBlob, SupportFiles: nil}, nil
}
