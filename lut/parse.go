package lut

import (
	"bufio"
	"io"
	"strconv"
	"strings"
)

// Parse reads an Adobe .cube 3D LUT. Every format problem is reported
// as *Error carrying the 1-based source line number.
func Parse(r io.Reader) (*LUT, error) {
	l := &LUT{Min: [3]float64{0, 0, 0}, Max: [3]float64{1, 1, 1}}
	var data []float64
	sizeSeen := false
	dataStart := 0 // line where the first data row appeared, for errors

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		kw := strings.ToUpper(fields[0])
		switch kw {
		case "TITLE":
			l.Title = strings.Trim(strings.TrimPrefix(line, fields[0]), " \t\"")
		case "LUT_3D_SIZE":
			if sizeSeen {
				return nil, errf(lineNo, "duplicate LUT_3D_SIZE")
			}
			if len(fields) != 2 {
				return nil, errf(lineNo, "LUT_3D_SIZE expects 1 value, got %d", len(fields)-1)
			}
			n, err := strconv.Atoi(fields[1])
			if err != nil || n < 2 || n > 256 {
				return nil, errf(lineNo, "invalid LUT_3D_SIZE %q (want integer 2..256)", fields[1])
			}
			l.Size = n
			sizeSeen = true
		case "DOMAIN_MIN", "DOMAIN_MAX":
			if len(fields) != 4 {
				return nil, errf(lineNo, "%s expects 3 values, got %d", kw, len(fields)-1)
			}
			var v [3]float64
			for i := 0; i < 3; i++ {
				f, err := strconv.ParseFloat(fields[i+1], 64)
				if err != nil {
					return nil, errf(lineNo, "%s: bad number %q", kw, fields[i+1])
				}
				v[i] = f
			}
			if kw == "DOMAIN_MIN" {
				l.Min = v
			} else {
				l.Max = v
			}
			l.HasDom = true
		case "LUT_1D_SIZE", "LUT_3D_INPUT_RANGE":
			return nil, errf(lineNo, "unsupported keyword %s", kw)
		default:
			if !sizeSeen {
				return nil, errf(lineNo, "data row before LUT_3D_SIZE")
			}
			if len(fields) != 3 {
				return nil, errf(lineNo, "data row expects 3 values, got %d", len(fields))
			}
			if len(data) == 0 {
				dataStart = lineNo
			}
			for _, f := range fields {
				v, err := strconv.ParseFloat(f, 64)
				if err != nil {
					return nil, errf(lineNo, "bad number %q in data row", f)
				}
				data = append(data, v)
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, errf(lineNo, "read error: %v", err)
	}
	if !sizeSeen {
		return nil, errf(0, "missing LUT_3D_SIZE")
	}
	want := l.Size * l.Size * l.Size
	got := len(data) / 3
	if got != want {
		line := dataStart + got
		if got > want {
			line = dataStart + want
		}
		return nil, errf(line, "data rows = %d, want %d for size %d", got, want, l.Size)
	}
	for c := 0; c < 3; c++ {
		if l.Max[c] <= l.Min[c] {
			return nil, errf(0, "DOMAIN_MAX[%d] must exceed DOMAIN_MIN[%d] (%g <= %g)",
				c, c, l.Max[c], l.Min[c])
		}
	}
	l.Data = data
	return l, nil
}
