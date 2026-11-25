package main

import (
	"bytes"
	"testing"
)

func TestWritePPMHeaderAndPayload(t *testing.T) {
	c := NewCanvas(2, 2)
	c.setRGB(0, 0, 255, 0, 0)
	c.setRGB(1, 1, 0, 255, 0)
	var buf bytes.Buffer
	if err := c.WritePPM(&buf); err != nil {
		t.Fatal(err)
	}
	header := "P6\n2 2\n255\n"
	data := buf.Bytes()
	if !bytes.HasPrefix(data, []byte(header)) {
		t.Fatalf("missing P6 header: %q", data[:min(len(data), 32)])
	}
	wantLen := len(header) + 2*2*3
	if len(data) != wantLen {
		t.Fatalf("PPM length = %d, want %d", len(data), wantLen)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
