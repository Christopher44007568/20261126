package main

import (
	"path/filepath"
	"strings"
)

type checkpoint struct {
	Name  string
	Frame int
}

var checkpoints = []checkpoint{
	{Name: "LOW", Frame: 0},
	{Name: "MEDIUM", Frame: 3},
	{Name: "HIGH", Frame: 10},
}

var strategyOrder = []Strategy{StrategyFull, StrategyChunk, StrategyScreen}

type GeneratedArtifacts struct {
	Paths       []string
	Report      Report
	Comparison  map[string]*Image
	FrameImages map[string]map[Strategy]*Image
}

func Generate(outDir string) (*GeneratedArtifacts, error) {
	source := NewTextureSource()
	art := &GeneratedArtifacts{
		Comparison:  make(map[string]*Image),
		FrameImages: make(map[string]map[Strategy]*Image),
	}
	report := NewReport()

	for _, detail := range checkpoints {
		art.FrameImages[detail.Name] = make(map[Strategy]*Image)
	}

	for _, kind := range strategyOrder {
		loader := NewLoader(kind, source)
		strategyReport := StrategyReport{Name: string(kind), Title: strategyTitle(kind)}

		for frame := 0; frame <= 10; frame++ {
			cam := CameraAt(float64(frame))
			vp := mainViewport()
			loader.Plan(cam, vp)

			if name, ok := checkpointNameAt(frame); ok {
				rendered := RenderFrame(loader, cam, name)
				lower := strings.ToLower(name)
				path := filepath.Join(outDir, string(kind)+"_"+lower+".ppm")
				if err := WritePPMFile(path, rendered.Image); err != nil {
					return nil, err
				}
				art.Paths = append(art.Paths, path)
				art.FrameImages[name][kind] = rendered.Image
				strategyReport.Frames = append(strategyReport.Frames, rendered.Metrics)
			}

			loader.Transfer(cam)
		}

		report.Strategies = append(report.Strategies, strategyReport)
	}

	for _, detail := range checkpoints {
		sheet := NewImage(frameW*len(strategyOrder), frameH, colorBackground)
		for i, kind := range strategyOrder {
			art.FrameImages[detail.Name][kind].BlitTo(sheet, i*frameW, 0)
		}
		lower := strings.ToLower(detail.Name)
		for _, filename := range []string{lower + ".ppm", "comparison_" + lower + ".ppm"} {
			path := filepath.Join(outDir, filename)
			if err := WritePPMFile(path, sheet); err != nil {
				return nil, err
			}
			art.Paths = append(art.Paths, path)
		}
		art.Comparison[detail.Name] = sheet
	}

	verification := VerifyContinuity(source)
	report.Verification = verification
	if err := WriteReports(outDir, report); err != nil {
		return nil, err
	}
	art.Paths = append(art.Paths,
		filepath.Join(outDir, "streaming_report.json"),
		filepath.Join(outDir, "streaming_report.md"),
	)
	art.Report = report
	return art, nil
}

func checkpointNameAt(frame int) (string, bool) {
	for _, cp := range checkpoints {
		if cp.Frame == frame {
			return cp.Name, true
		}
	}
	return "", false
}

func (im *Image) BlitTo(dst *Image, dx, dy int) {
	for y := 0; y < im.H; y++ {
		srcStart := y * im.W * 4
		dstY := dy + y
		if dstY < 0 || dstY >= dst.H {
			continue
		}
		for x := 0; x < im.W; x++ {
			dstX := dx + x
			if dstX < 0 || dstX >= dst.W {
				continue
			}
			s := srcStart + x*4
			d := (dstY*dst.W + dstX) * 4
			copy(dst.Pix[d:d+4], im.Pix[s:s+4])
		}
	}
}
