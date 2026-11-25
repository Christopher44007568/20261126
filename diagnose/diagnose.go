// Package diagnose detects color-space tagging mistakes per mip level.
package diagnose

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"

	"texturelab/colorspace"
	"texturelab/material"
	"texturelab/texcontainer"
)

// DefaultThreshold is a clearly visible linear-space difference.
const DefaultThreshold = 0.01

// Location identifies the largest difference in one mip level.
type Location struct {
	X                 int        `json:"x"`
	Y                 int        `json:"y"`
	Channel           string     `json:"channel"`
	ByteOffset        int        `json:"byte_offset"`
	StoredRGBA        [4]int     `json:"stored_rgba"`
	CorrectLinear     [3]float64 `json:"correct_linear_rgb"`
	MislabeledLinear  [3]float64 `json:"mislabeled_linear_rgb"`
	ChannelDifference [3]float64 `json:"channel_difference"`
	MaxDelta          float64    `json:"max_delta"`
}

// BoundingBox is an inclusive changed-texel rectangle.
type BoundingBox struct {
	MinX int `json:"min_x"`
	MinY int `json:"min_y"`
	MaxX int `json:"max_x"`
	MaxY int `json:"max_y"`
}

// LevelReport contains deterministic metrics for one mip level.
type LevelReport struct {
	Level            int          `json:"level"`
	Width            int          `json:"width"`
	Height           int          `json:"height"`
	CorrectSpace     string       `json:"correct_space"`
	InterpretedSpace string       `json:"interpreted_space"`
	Detected         bool         `json:"detected"`
	MaxAbsDelta      float64      `json:"max_abs_delta"`
	MeanAbsRGB       float64      `json:"mean_abs_rgb"`
	MeanSignedRGB    [3]float64   `json:"mean_signed_rgb"`
	ChangedTexels    int          `json:"changed_texels"`
	TotalTexels      int          `json:"total_texels"`
	ChangedRatio     float64      `json:"changed_ratio"`
	ChangedBounds    *BoundingBox `json:"changed_bounds,omitempty"`
	MaxLocation      Location     `json:"max_location"`
	DeltaFingerprint string       `json:"delta_fingerprint_sha256"`
}

// Report is the complete per-level tagging diagnostic.
type Report struct {
	TextureName       string        `json:"texture_name"`
	CorrectSpace      string        `json:"correct_space"`
	InterpretedSpace  string        `json:"interpreted_space"`
	Threshold         float64       `json:"threshold"`
	Mismatch          bool          `json:"mismatch"`
	FirstFailingLevel int           `json:"first_failing_level"`
	WorstLevel        int           `json:"worst_level"`
	Levels            []LevelReport `json:"levels"`
}

// CompareInterpretations compares two pixel-identical containers whose markers
// may differ. The first argument supplies the expected material interpretation.
func CompareInterpretations(name string, correct, interpreted *texcontainer.Texture, threshold float64) (*Report, error) {
	if threshold <= 0 {
		threshold = DefaultThreshold
	}
	if correct == nil || interpreted == nil {
		return nil, errors.New("diagnose: nil texture")
	}
	if len(correct.Levels) != len(interpreted.Levels) {
		return nil, fmt.Errorf("diagnose: level count mismatch: %d != %d",
			len(correct.Levels), len(interpreted.Levels))
	}

	report := &Report{
		TextureName:       name,
		CorrectSpace:      correct.Space.String(),
		InterpretedSpace:  interpreted.Space.String(),
		Threshold:         threshold,
		FirstFailingLevel: -1,
		WorstLevel:        -1,
		Levels:            make([]LevelReport, 0, len(correct.Levels)),
	}

	for level := range correct.Levels {
		levelReport, err := compareLevel(level, correct, interpreted, threshold)
		if err != nil {
			return nil, err
		}
		report.Levels = append(report.Levels, levelReport)
		if levelReport.Detected && report.FirstFailingLevel == -1 {
			report.FirstFailingLevel = level
		}
		if levelReport.MaxAbsDelta > 0 && (report.WorstLevel == -1 ||
			levelReport.MaxAbsDelta > report.Levels[report.WorstLevel].MaxAbsDelta) {
			report.WorstLevel = level
		}
	}
	report.Mismatch = report.FirstFailingLevel != -1
	return report, nil
}

// DiagnoseNormalMap retags a linear normal map as sRGB and measures the stable
// wrong-material interpretation without modifying the supplied texture.
func DiagnoseNormalMap(name string, normal *texcontainer.Texture, threshold float64) (*Report, error) {
	if normal == nil {
		return nil, errors.New("diagnose: nil normal texture")
	}
	if normal.Space != colorspace.Linear {
		return nil, fmt.Errorf("diagnose: expected linear normal map, got %s", normal.Space)
	}
	return CompareInterpretations(name, normal, normal.Retag(colorspace.SRGB), threshold)
}

// LocateFirstMismatch returns an error carrying the first failing mip level.
func (r *Report) LocateFirstMismatch() error {
	if r == nil || !r.Mismatch || r.FirstFailingLevel < 0 {
		return nil
	}
	level := r.Levels[r.FirstFailingLevel]
	loc := level.MaxLocation
	return fmt.Errorf(
		"normal map is mislabeled %s instead of %s: stable color difference first located at mip level %d (%dx%d), texel (%d,%d), channel %s, max linear delta %.6f",
		r.InterpretedSpace, r.CorrectSpace, level.Level, level.Width, level.Height,
		loc.X, loc.Y, loc.Channel, loc.MaxDelta)
}

func compareLevel(index int, correct, interpreted *texcontainer.Texture, threshold float64) (LevelReport, error) {
	correctLevel := correct.Levels[index]
	wrongLevel := interpreted.Levels[index]
	if correctLevel.Width != wrongLevel.Width || correctLevel.Height != wrongLevel.Height {
		return LevelReport{}, fmt.Errorf("diagnose: level %d dimensions differ", index)
	}
	if !bytes.Equal(correctLevel.Pixels, wrongLevel.Pixels) {
		return LevelReport{}, fmt.Errorf("diagnose: level %d pixel payloads differ; retag diagnostics require identical bytes", index)
	}

	result := LevelReport{
		Level:            index,
		Width:            correctLevel.Width,
		Height:           correctLevel.Height,
		CorrectSpace:     correct.Space.String(),
		InterpretedSpace: interpreted.Space.String(),
		TotalTexels:      correctLevel.Width * correctLevel.Height,
	}
	hash := sha256.New()
	minX, minY := correctLevel.Width, correctLevel.Height
	maxX, maxY := -1, -1

	var absoluteSum float64
	var signedSum [3]float64
	for y := 0; y < correctLevel.Height; y++ {
		for x := 0; x < correctLevel.Width; x++ {
			correctSample, err := material.SampleLinear(correct, index, x, y)
			if err != nil {
				return LevelReport{}, err
			}
			wrongSample, err := material.SampleLinear(interpreted, index, x, y)
			if err != nil {
				return LevelReport{}, err
			}
			offset := (y*correctLevel.Width + x) * 4
			var differences [3]float64
			pixelMax := 0.0
			channel := 0
			for c := 0; c < 3; c++ {
				differences[c] = wrongSample[c] - correctSample[c]
				absoluteSum += math.Abs(differences[c])
				signedSum[c] += differences[c]
				if math.Abs(differences[c]) > pixelMax {
					pixelMax = math.Abs(differences[c])
					channel = c
				}
				quantized := uint16(math.Round(math.Abs(differences[c]) * 65535))
				var encoded [2]byte
				binary.LittleEndian.PutUint16(encoded[:], quantized)
				hash.Write(encoded[:])
			}

			if pixelMax > result.MaxAbsDelta {
				result.MaxAbsDelta = pixelMax
				channelName := []string{"R", "G", "B"}[channel]
				result.MaxLocation = Location{
					X: x, Y: y, Channel: channelName, ByteOffset: offset + channel,
					StoredRGBA: [4]int{
						int(correctLevel.Pixels[offset]), int(correctLevel.Pixels[offset+1]),
						int(correctLevel.Pixels[offset+2]), int(correctLevel.Pixels[offset+3]),
					},
					CorrectLinear:     [3]float64{correctSample[0], correctSample[1], correctSample[2]},
					MislabeledLinear:  [3]float64{wrongSample[0], wrongSample[1], wrongSample[2]},
					ChannelDifference: differences,
					MaxDelta:          pixelMax,
				}
			}
			if pixelMax >= threshold {
				result.ChangedTexels++
				if x < minX {
					minX = x
				}
				if y < minY {
					minY = y
				}
				if x > maxX {
					maxX = x
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}

	result.MeanAbsRGB = absoluteSum / float64(result.TotalTexels*3)
	for c := 0; c < 3; c++ {
		result.MeanSignedRGB[c] = signedSum[c] / float64(result.TotalTexels)
	}
	result.ChangedRatio = float64(result.ChangedTexels) / float64(result.TotalTexels)
	result.Detected = result.MaxAbsDelta >= threshold
	result.DeltaFingerprint = hex.EncodeToString(hash.Sum(nil))
	if result.ChangedTexels > 0 {
		result.ChangedBounds = &BoundingBox{MinX: minX, MinY: minY, MaxX: maxX, MaxY: maxY}
	}
	return result, nil
}
