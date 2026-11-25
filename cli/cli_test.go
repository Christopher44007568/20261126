package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestDemoAndInspectCommands(t *testing.T) {
	outDir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Run([]string{"demo", "--out", outDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("demo code=%d stderr=%s", code, stderr.String())
	}
	output := stdout.String()
	if !strings.Contains(output, "no_color_shift=true") {
		t.Fatalf("demo output lacks round-trip proof:\n%s", output)
	}
	if !strings.Contains(output, "first_level=0") {
		t.Fatalf("demo output lacks diagnostic level:\n%s", output)
	}

	container := filepath.Join(outDir, "normal_linear.ktxc")
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"inspect", container}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("inspect code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "color_space=linear") ||
		!strings.Contains(stdout.String(), "level=2 size=128x128") {
		t.Fatalf("inspect output missing expected data:\n%s", stdout.String())
	}
}

func TestUnknownCommandReturnsUsageExitCode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"bogus"}, &stdout, &stderr); code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
}
