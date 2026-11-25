package main

import (
	"fmt"
	"path/filepath"
)

var (
	black  = RGB8{0, 0, 0}
	white8 = RGB8{245, 245, 245}
	gray8  = RGB8{82, 88, 96}
	red8   = RGB8{210, 55, 45}
	green8 = RGB8{55, 165, 80}
	blue8  = RGB8{55, 95, 210}
	gold8  = RGB8{220, 170, 45}
)

func writeImagePPMs(outputDir string, frames []*Image, report RunReport) error {
	framesDir := filepath.Join(outputDir, "ppm", "frames")
	beforeAfterDir := filepath.Join(outputDir, "ppm", "before_after")
	if err := mkdirAll(framesDir, beforeAfterDir); err != nil {
		return err
	}

	for i, frame := range frames {
		path := filepath.Join(framesDir, fmt.Sprintf("frame_%02d_input.ppm", i))
		if err := writePPM(path, linearToCanvas(frame)); err != nil {
			return err
		}

		corrected := []*Image{
			report.Frames[i].CorrectedGray,
			report.Frames[i].CorrectedWhite,
			report.Frames[i].CorrectedSmooth,
		}
		for j, strategy := range []string{"gray_world", "white_point", "temporal_smooth"} {
			path := filepath.Join(framesDir, fmt.Sprintf("frame_%02d_%s.ppm", i, strategy))
			if err := writePPM(path, linearToCanvas(corrected[j])); err != nil {
				return err
			}
		}
	}

	representative := []int{0, 9, 12, 23, 26, 39}
	for _, index := range representative {
		path := filepath.Join(beforeAfterDir, fmt.Sprintf("frame_%02d_gray_world.ppm", index))
		if err := writePPM(path, sideBySide(frames[index], report.Frames[index].CorrectedGray, "INPUT", "GRAY WORLD")); err != nil {
			return err
		}
		path = filepath.Join(beforeAfterDir, fmt.Sprintf("frame_%02d_white_point.ppm", index))
		if err := writePPM(path, sideBySide(frames[index], report.Frames[index].CorrectedWhite, "INPUT", "WHITE POINT")); err != nil {
			return err
		}
		path = filepath.Join(beforeAfterDir, fmt.Sprintf("frame_%02d_temporal_smooth.ppm", index))
		if err := writePPM(path, sideBySide(frames[index], report.Frames[index].CorrectedSmooth, "INPUT", "TEMPORAL SMOOTH")); err != nil {
			return err
		}
	}

	stable := []int{9, 23, 39}
	for _, index := range stable {
		path := filepath.Join(beforeAfterDir, fmt.Sprintf("stable_frame_%02d_overview.ppm", index))
		if err := writePPM(path, strategyOverview(frames[index], report.Frames[index])); err != nil {
			return err
		}
	}
	return nil
}

func sideBySide(before, after *Image, leftTitle, rightTitle string) *Canvas {
	canvas := NewCanvas(before.Width*2+18, before.Height+28)
	canvas.Fill(RGB8{238, 238, 232})
	drawText(canvas, 8, 5, leftTitle, black)
	drawText(canvas, before.Width+16, 5, rightTitle, black)
	drawImage(canvas, before, 0, 20)
	drawImage(canvas, after, before.Width+18, 20)
	drawRect(canvas, 0, 20, before.Width, before.Height, gray8)
	drawRect(canvas, before.Width+18, 20, after.Width, after.Height, gray8)
	return canvas
}

func strategyOverview(input *Image, frame FrameReport) *Canvas {
	images := []*Image{input, frame.CorrectedGray, frame.CorrectedWhite, frame.CorrectedSmooth}
	titles := []string{"INPUT", "GRAY WORLD", "WHITE POINT", "TEMPORAL SMOOTH"}
	canvas := NewCanvas(input.Width*2+18, input.Height*2+48)
	canvas.Fill(RGB8{238, 238, 232})
	positions := [][2]int{{0, 20}, {input.Width + 18, 20}, {0, input.Height + 44}, {input.Width + 18, input.Height + 44}}
	for i := range images {
		drawImage(canvas, images[i], positions[i][0], positions[i][1])
		drawText(canvas, positions[i][0]+4, positions[i][1]-12, titles[i], black)
		drawRect(canvas, positions[i][0], positions[i][1], input.Width, input.Height, gray8)
	}
	return canvas
}

func drawImage(canvas *Canvas, frame *Image, offsetX, offsetY int) {
	source := linearToCanvas(frame)
	for y := 0; y < source.Height; y++ {
		for x := 0; x < source.Width; x++ {
			canvas.Set(offsetX+x, offsetY+y, source.At(x, y))
		}
	}
}

func mkdirAll(paths ...string) error {
	for _, path := range paths {
		if err := osMkdirAll(path); err != nil {
			return err
		}
	}
	return nil
}
