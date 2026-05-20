//go:build !(darwin && arm64) && !(darwin && amd64) && !(linux && amd64) && !(linux && arm64) && !(windows && amd64)

package bin

func embeddedCWebP(goos, goarch string) (executableBundle, error) {
	return executableBundle{}, ErrUnsupportedPlatform{GOOS: goos, GOARCH: goarch}
}
