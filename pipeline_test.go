package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRunGeneratesArtifactsAndExactPPMRestore(t *testing.T) {
	out := t.TempDir()
	report, err := run(out)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !report.Pass {
		t.Fatalf("report has failed checks: %+v", report.Checks)
	}

	expected := []string{
		"variant_summer.ppm",
		"variant_winter.ppm",
		"variant_night.ppm",
		"variant_contact_sheet.ppm",
		"selection_path.ppm",
		"current_selection_path.ppm",
		"restored_summer.ppm",
		"override_restored_winter.ppm",
		"current_selection_path.txt",
		"current_selection_path.json",
		"variant_report.json",
	}
	for _, name := range expected {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Errorf("expected artifact %s: %v", name, err)
		}
	}

	summerBytes, err := os.ReadFile(filepath.Join(out, "variant_summer.ppm"))
	if err != nil {
		t.Fatal(err)
	}
	restoredBytes, err := os.ReadFile(filepath.Join(out, "restored_summer.ppm"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(summerBytes, restoredBytes) {
		t.Fatal("restored summer PPM bytes differ from original summer PPM")
	}

	winterBytes, err := os.ReadFile(filepath.Join(out, "variant_winter.ppm"))
	if err != nil {
		t.Fatal(err)
	}
	winterRestoredBytes, err := os.ReadFile(filepath.Join(out, "override_restored_winter.ppm"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(winterBytes, winterRestoredBytes) {
		t.Fatal("restored winter PPM bytes differ after child override removal")
	}

	summer, err := ReadPPM(bytes.NewReader(summerBytes))
	if err != nil {
		t.Fatal(err)
	}
	if summer.W != FrameW || summer.H != FrameH {
		t.Fatalf("summer dimensions = %dx%d", summer.W, summer.H)
	}

	sheetFile, err := os.Open(filepath.Join(out, "variant_contact_sheet.ppm"))
	if err != nil {
		t.Fatal(err)
	}
	defer sheetFile.Close()
	sheet, err := ReadPPM(sheetFile)
	if err != nil {
		t.Fatal(err)
	}
	if sheet.W != 1520 || sheet.H != 532 {
		t.Fatalf("contact sheet dimensions = %dx%d", sheet.W, sheet.H)
	}

	pathText, err := os.ReadFile(filepath.Join(out, "current_selection_path.txt"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"target:  " + LeftWindowPath,
		"active:  summer",
		"differentPixels=0",
		"/Stage/Asset",
	} {
		if !bytes.Contains(pathText, []byte(want)) {
			t.Errorf("selection path text missing %q\n%s", want, pathText)
		}
	}
}

func TestPPMRoundTripPreservesPixels(t *testing.T) {
	stage := BuildCabin()
	im, err := NewRenderer().Render(stage, RenderOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := im.WritePPM(&buf); err != nil {
		t.Fatal(err)
	}
	decoded, err := ReadPPM(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.PixelSHA256() != im.PixelSHA256() {
		t.Fatal("PPM round trip changed pixels")
	}
}
