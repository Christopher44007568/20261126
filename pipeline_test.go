package main

import (
	"encoding/json"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"animationtiming/report"
)

func TestRunGeneratesSheetsAndReport(t *testing.T) {
	out := t.TempDir()
	if err := run(out); err != nil {
		t.Fatalf("run: %v", err)
	}

	files := []string{
		"01_normal_speed.png",
		"02_slow_motion.png",
		"03_interpolation.png",
		"animation_frame_sheets_all.png",
		"timeline_report.json",
		"index.html",
	}
	for _, name := range files {
		info, err := os.Stat(filepath.Join(out, name))
		if err != nil || info.Size() == 0 {
			t.Fatalf("artifact %s missing or empty: %v", name, err)
		}
	}

	data, err := os.ReadFile(filepath.Join(out, "timeline_report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var audit report.Report
	if err := json.Unmarshal(data, &audit); err != nil {
		t.Fatalf("decode report: %v", err)
	}

	if len(audit.Sheets) != 3 {
		t.Fatalf("sheets=%d want 3", len(audit.Sheets))
	}
	wantFrames := map[string]int{
		"01_normal_speed":  12,
		"02_slow_motion":   21,
		"03_interpolation": 12,
	}
	wantSkips := map[string][]string{
		"01_normal_speed":  {"WAVE_PEAK", "JUMP_PEAK"},
		"02_slow_motion":   {},
		"03_interpolation": {},
	}
	for _, sheet := range audit.Sheets {
		if len(sheet.Frames) != wantFrames[sheet.ID] {
			t.Fatalf("%s frames=%d want %d", sheet.ID, len(sheet.Frames), wantFrames[sheet.ID])
		}
		if !equalStrings(sheet.SkippedSignatureKeys, wantSkips[sheet.ID]) {
			t.Fatalf("%s skips=%v want %v", sheet.ID, sheet.SkippedSignatureKeys, wantSkips[sheet.ID])
		}
	}

	if len(audit.StaticPolicy.Nodes) != 2 {
		t.Fatalf("static checks=%d want beacon and buddy", len(audit.StaticPolicy.Nodes))
	}
	for _, node := range audit.StaticPolicy.Nodes {
		if node.ID != "beacon" && node.ID != "buddy" {
			t.Fatalf("unexpected static node %q", node.ID)
		}
		if node.FramesChecked != 45 || !node.IdenticalToBindPose {
			t.Fatalf("static invariant failed: %+v", node)
		}
	}
}

func TestOutOfRangeCardsHaveStatusBorderPixels(t *testing.T) {
	out := t.TempDir()
	if err := run(out); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(filepath.Join(out, "01_normal_speed.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}

	// Card #00 starts at (14,104); its thick border is amber.
	r, g, b, _ := img.At(16, 106).RGBA()
	if uint8(r>>8) != 0xd9 || uint8(g>>8) != 0x94 || uint8(b>>8) != 0x00 {
		t.Fatalf("before border = #%02x%02x%02x, want d99400", uint8(r>>8), uint8(g>>8), uint8(b>>8))
	}

	// Card #10 is column 2, row 2: x=14+2*(320+12)=678, y=104+2*(336+12)=800.
	r, g, b, _ = img.At(680, 802).RGBA()
	if uint8(r>>8) != 0xd3 || uint8(g>>8) != 0x3b || uint8(b>>8) != 0x3b {
		t.Fatalf("after border = #%02x%02x%02x, want d33b3b", uint8(r>>8), uint8(g>>8), uint8(b>>8))
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
