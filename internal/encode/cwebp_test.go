package encode

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/zoulux/img2webp/internal/analyze"
	"github.com/zoulux/img2webp/internal/strategy"
)

func TestBuildArgsOmitsNearLosslessWhenZero(t *testing.T) {
	c := strategy.Candidate{Kind: analyze.KindGraphic, Quality: 92, Method: 6, NearLossless: 0, AlphaQuality: 90}
	args := buildArgs("/tmp/in.png", "/tmp/out.webp", c)
	mustNotContain(t, args, "-near_lossless")
}

func TestBuildArgsIncludesLosslessWhenEnabled(t *testing.T) {
	c := strategy.Candidate{Kind: analyze.KindGraphic, Quality: 92, Method: 6, Lossless: true, AlphaQuality: 90}
	args := buildArgs("/tmp/in.png", "/tmp/out.webp", c)
	mustContain(t, args, "-lossless")
}

func TestRunCWebPIncludesCommandOutputOnFailure(t *testing.T) {
	binary := writeFailingScript(t, "stderr-text", "stdout-text")
	c := strategy.Candidate{Quality: 75, Method: 4, AlphaQuality: 100}

	err := RunCWebP(context.Background(), binary, "/tmp/in.png", "/tmp/out.webp", c)
	if err == nil {
		t.Fatal("expected command failure")
	}
	msg := err.Error()
	for _, want := range []string{"stderr-text", "stdout-text"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q does not contain %q", msg, want)
		}
	}
}

func TestRunCWebPCancelsWithContext(t *testing.T) {
	binary := writeSleepingScript(t, 5*time.Second)
	c := strategy.Candidate{Quality: 75, Method: 4, AlphaQuality: 100}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := RunCWebP(ctx, binary, "/tmp/in.png", "/tmp/out.webp", c)
	if err == nil {
		t.Fatal("expected cancellation error")
	}
	if !strings.Contains(err.Error(), context.Canceled.Error()) {
		t.Fatalf("error %q does not contain %q", err, context.Canceled)
	}
}

func mustContain(t *testing.T, args []string, want string) {
	t.Helper()
	for _, arg := range args {
		if arg == want {
			return
		}
	}
	t.Fatalf("args %v do not contain %q", args, want)
}

func mustNotContain(t *testing.T, args []string, want string) {
	t.Helper()
	for _, arg := range args {
		if arg == want {
			t.Fatalf("args %v unexpectedly contain %q", args, want)
		}
	}
}

func writeFailingScript(t *testing.T, stderrText, stdoutText string) string {
	t.Helper()
	path := t.TempDir() + "/fail.sh"
	content := "#!/bin/sh\nprintf '%s\\n' '" + stdoutText + "'\nprintf '%s\\n' '" + stderrText + "' >&2\nexit 1\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return path
}

func writeSleepingScript(t *testing.T, d time.Duration) string {
	t.Helper()
	path := t.TempDir() + "/sleep.sh"
	content := "#!/bin/sh\nsleep " + strings.TrimSuffix(d.String(), "s") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return path
}
