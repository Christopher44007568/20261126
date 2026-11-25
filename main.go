package main

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	outputDir := "output"
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}

	scene := referenceScene()
	referenceImage := encodeNormalized(scene, 0, fullWell)
	if err := writePNG(filepath.Join(outputDir, "reference_clean.png"), referenceImage); err != nil {
		return err
	}

	textureRect := imageRect{x: 414, y: 86, width: 150, height: 138}
	flatRect := imageRect{x: 36, y: 330, width: 145, height: 105}
	var metricRows []map[string]float64

	for _, setup := range exposures {
		r := newRNG(20261126 + int64(setup.iso))
		captured := capture(scene, r, setup)
		denoised := denoiseCapture(captured.raw, setup)

		noisyImage := encodeNormalized(captured.raw, 0, fullWell)
		denoisedImage := encodeNormalized(denoised, 0, fullWell)
		if err := writePNG(filepath.Join(outputDir, setup.output), noisyImage); err != nil {
			return err
		}
		if err := writePNG(filepath.Join(outputDir, "denoised_"+setup.name+".png"), denoisedImage); err != nil {
			return err
		}

		metrics := textureRetention(captured.raw, denoised, captured.clean, textureRect, flatRect)
		metrics["iso"] = float64(setup.iso)
		metricRows = append(metricRows, metrics)

		if setup.name == "low" {
			if err := writeComponentSheet(outputDir, captured, setup); err != nil {
				return err
			}
			if err := writeDetailComparison(outputDir, referenceImage, noisyImage, denoisedImage, textureRect); err != nil {
				return err
			}
		}
	}

	if err := writeExposureContact(outputDir, referenceImage); err != nil {
		return err
	}

	fmt.Println("ISO  texture-retention  flat-noise-reduction")
	for _, metrics := range metricRows {
		fmt.Printf("%4.0f  %16.2f%%  %20.2f%%\n",
			metrics["iso"], metrics["textureRetention"]*100, metrics["flatNoiseReduction"]*100)
	}
	fmt.Println("PNG outputs written to", outputDir)
	return nil
}

func writeComponentSheet(outputDir string, captured *capturedImage, setup exposure) error {
	components := []struct {
		label string
		image image.Image
		file  string
	}{
		{"PHOTON SHOT NOISE", encodeNormalized(signedComponent(captured.shotNoise, 42), -255, 255), "component_photon_shot.png"},
		{"READ NOISE ISO " + fmt.Sprint(setup.iso), encodeNormalized(signedComponent(captured.readNoise, setup.read*2), -255, 255), "component_read_noise.png"},
		{"FIXED PATTERN NOISE", encodeNormalized(fixedComponent(captured.patternNoise, setup.fpn*2.4), 0, 255), "component_fixed_pattern.png"},
	}

	for _, component := range components {
		if err := writePNG(filepath.Join(outputDir, component.file), component.image); err != nil {
			return err
		}
	}

	sheet := image.NewNRGBA(image.Rect(0, 0, sceneWidth*3+32, sceneHeight+44))
	fillRect(sheet, sheet.Bounds(), color.NRGBA{R: 12, G: 14, B: 18, A: 255})
	for i, component := range components {
		copyScaled(sheet, component.image, 8+i*(sceneWidth+8), 30, 1)
		drawText(sheet, 14+i*(sceneWidth+8), 10, component.label)
	}
	return writePNG(filepath.Join(outputDir, "sheet_noise_components.png"), sheet)
}

func writeDetailComparison(outputDir string, reference, noisy, denoised image.Image, rect imageRect) error {
	cropRect := image.Rect(rect.x, rect.y, rect.x+rect.width, rect.y+rect.height)
	referenceCrop := cropImage(reference, cropRect)
	noisyCrop := cropImage(noisy, cropRect)
	denoisedCrop := cropImage(denoised, cropRect)

	width := rect.width*3*3 + 32
	height := rect.height*3 + 44
	sheet := image.NewNRGBA(image.Rect(0, 0, width, height))
	fillRect(sheet, sheet.Bounds(), color.NRGBA{R: 12, G: 14, B: 18, A: 255})
	copyScaled(sheet, referenceCrop, 8, 34, 3)
	copyScaled(sheet, noisyCrop, 16+rect.width*3, 34, 3)
	copyScaled(sheet, denoisedCrop, 24+rect.width*6, 34, 3)
	drawText(sheet, 14, 10, "REFERENCE TEXTURE")
	drawText(sheet, 22+rect.width*3, 10, "NOISY LOW EXPOSURE")
	drawText(sheet, 30+rect.width*6, 10, "DENOISED TEXTURE")
	return writePNG(filepath.Join(outputDir, "detail_reference_vs_denoised.png"), sheet)
}

func writeExposureContact(outputDir string, reference image.Image) error {
	positions := []struct {
		file  string
		label string
	}{
		{"reference_clean.png", "CLEAN REFERENCE"},
		{"exposure_low.png", "LOW EXPOSURE ISO 3200"},
		{"exposure_middle.png", "MIDDLE EXPOSURE ISO 800"},
		{"exposure_high.png", "HIGH EXPOSURE ISO 200"},
	}
	width := sceneWidth*2 + 24
	height := sceneHeight*2 + 68
	sheet := image.NewNRGBA(image.Rect(0, 0, width, height))
	fillRect(sheet, sheet.Bounds(), color.NRGBA{R: 12, G: 14, B: 18, A: 255})
	for i, position := range positions {
		loaded, err := readPNG(filepath.Join(outputDir, position.file))
		if err != nil {
			return err
		}
		x := 8 + (i%2)*(sceneWidth+8)
		y := 30 + (i/2)*(sceneHeight+24)
		copyScaled(sheet, loaded, x, y, 1)
		drawText(sheet, x+6, y-18, position.label)
	}
	return writePNG(filepath.Join(outputDir, "sheet_exposures.png"), sheet)
}

func readPNG(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return pngDecode(file)
}
