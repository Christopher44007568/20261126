package aliaslab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunExperimentWritesThreeVerifiedPPMs(t *testing.T) {
	dir := t.TempDir()
	report, err := RunExperiment(dir, 112, 63)
	if err != nil {
		t.Fatal(err)
	}
	if !report.allAssertionsPassed() {
		t.Fatalf("experiment assertions failed: %+v", report.Assertions)
	}
	if report.ValidReuse.SlotCount >= report.NoReuse.SlotCount {
		t.Fatalf("valid plan did not reduce slots: %d vs %d",
			report.ValidReuse.SlotCount, report.NoReuse.SlotCount)
	}
	if !report.ValidReuse.ReferenceMatch || !report.NoReuse.ReferenceMatch {
		t.Fatal("reference/valid exact-match flags are not both true")
	}
	if report.ErrorReuse.ReferenceMatch {
		t.Fatal("error image must not match the reference")
	}

	noBytes := readFile(t, filepath.Join(dir, "no_reuse.ppm"))
	validBytes := readFile(t, filepath.Join(dir, "valid_reuse.ppm"))
	errorBytes := readFile(t, filepath.Join(dir, "error_reuse.ppm"))
	if !bytes.Equal(noBytes, validBytes) {
		t.Fatal("valid_reuse.ppm is not byte-for-byte identical to no_reuse.ppm")
	}
	if bytes.Equal(noBytes, errorBytes) {
		t.Fatal("error_reuse.ppm unexpectedly equals the reference")
	}
	header := fmt.Sprintf("P6\n%d %d\n255\n", 112, 63)
	for name, data := range map[string][]byte{
		"no_reuse.ppm": noBytes, "valid_reuse.ppm": validBytes, "error_reuse.ppm": errorBytes,
	} {
		if !strings.HasPrefix(string(data[:len(header)]), header) {
			t.Fatalf("%s has wrong P6 header", name)
		}
		if want := len(header) + 112*63*3; len(data) != want {
			t.Fatalf("%s length=%d, want %d", name, len(data), want)
		}
	}

	for _, name := range []string{
		"intervals.log", "aliases.log", "barriers.log", "summary.txt", "result.json",
	} {
		if len(readFile(t, filepath.Join(dir, name))) == 0 {
			t.Fatalf("%s is empty", name)
		}
	}
	barrierLog := string(readFile(t, filepath.Join(dir, "barriers.log")))
	if !strings.Contains(barrierLog, "ALIAS_WAR") {
		t.Fatal("barrier log has no legal alias transition")
	}
	if !strings.Contains(barrierLog, "ALIAS_RAW") {
		t.Fatal("barrier log has no stale-read alias hazard")
	}

	var decoded ExperimentReport
	if err := json.Unmarshal(readFile(t, filepath.Join(dir, "result.json")), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ErrorReuse.Contamination == nil ||
		decoded.ErrorReuse.Contamination.RegionPixels <= 0 {
		t.Fatal("result.json has no first contamination metadata")
	}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
