package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"dds-preview/internal/preview"
)

func cliFixture() []byte {
	header := make([]byte, 128)
	copy(header, "DDS ")
	put := func(off int, v uint32) { binary.LittleEndian.PutUint32(header[off:], v) }
	put(4, 124)
	put(8, 0x1|0x2|0x4|0x8|0x1000)
	put(12, 1)
	put(16, 1)
	put(20, 4)
	put(76, 32)
	put(80, 1|0x40)
	put(88, 32)
	put(92, 0xFF)
	put(96, 0xFF00)
	put(100, 0xFF0000)
	put(104, 0xFF000000)
	put(108, 0x1000)
	return append(header, 20, 40, 60, 255)
}

func TestCLIKeepsPreviousImagesOnTruncation(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.dds")
	bad := filepath.Join(dir, "bad.dds")
	out := filepath.Join(dir, "preview")
	valid := cliFixture()
	if err := os.WriteFile(good, valid, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bad, valid[:len(valid)-1], 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := run([]string{"-out", out, good, bad}, &stdout, &stderr); code != 1 {
		t.Fatalf("exit code = %d, want 1\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	sphere, err := os.ReadFile(filepath.Join(out, "sphere.ppm"))
	if err != nil {
		t.Fatalf("previous sphere missing: %v", err)
	}
	if !bytes.HasPrefix(sphere, []byte("P6\n1 1\n255\n")) {
		t.Fatalf("sphere header = %q", sphere)
	}
	reportBytes, err := os.ReadFile(filepath.Join(out, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var report preview.Result
	if err := json.Unmarshal(reportBytes, &report); err != nil {
		t.Fatal(err)
	}
	if report.OK || !report.KeptPrevious || len(report.Errors) != 1 {
		t.Fatalf("report = %+v", report)
	}
	if report.Errors[0].Field != "data.face[0].mip[0]" {
		t.Fatalf("error field = %q", report.Errors[0].Field)
	}
	if report.Current == nil || report.Current.Width != 1 {
		t.Fatalf("report should retain prior texture summary: %+v", report.Current)
	}
}
