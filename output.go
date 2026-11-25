package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

func materialColor(id MaterialID) color.RGBA {
	switch id {
	case MaterialStone:
		return color.RGBA{150, 136, 115, 255}
	case MaterialGlass:
		return color.RGBA{67, 181, 224, 255}
	case MaterialFloor:
		return color.RGBA{198, 166, 105, 255}
	case MaterialCeramic:
		return color.RGBA{232, 75, 46, 255}
	case MaterialMetal:
		return color.RGBA{160, 176, 199, 255}
	case MaterialLight:
		return color.RGBA{255, 218, 93, 255}
	default:
		return color.RGBA{12, 13, 17, 255}
	}
}

func hitGroupColor(id HitGroupID) color.RGBA {
	switch id {
	case GroupStone:
		return color.RGBA{183, 104, 66, 255}
	case GroupGlass:
		return color.RGBA{30, 185, 224, 255}
	case GroupFloor:
		return color.RGBA{87, 170, 92, 255}
	case GroupCeramic:
		return color.RGBA{227, 61, 158, 255}
	case GroupMetal:
		return color.RGBA{91, 117, 228, 255}
	case GroupLight:
		return color.RGBA{250, 226, 64, 255}
	default:
		return color.RGBA{12, 13, 17, 255}
	}
}

func colorHex8(c color.RGBA) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

func linearToRGBA(value Vec3) color.RGBA {
	value = value.Clamp(0, 64)
	tone := func(channel float64) uint8 {
		mapped := channel / (1 + channel)
		gamma := math.Pow(mapped, 1/2.2)
		return uint8(math.Round(math.Max(0, math.Min(1, gamma)) * 255))
	}
	return color.RGBA{tone(value.X), tone(value.Y), tone(value.Z), 255}
}

func writeMaterialIndexPNG(path string, result *RenderResult) error {
	return writePNG(path, result.Width, result.Height, func(x, y int) color.RGBA {
		index := y*result.Width + x
		return materialColor(result.Pixels[index].MaterialID)
	})
}

func writeHitGroupPNG(path string, result *RenderResult) error {
	return writePNG(path, result.Width, result.Height, func(x, y int) color.RGBA {
		index := y*result.Width + x
		return hitGroupColor(result.Pixels[index].HitGroupID)
	})
}

func writeShadedPNG(path string, result *RenderResult) error {
	return writePNG(path, result.Width, result.Height, func(x, y int) color.RGBA {
		index := y*result.Width + x
		return linearToRGBA(result.Pixels[index].Color)
	})
}

func writeDiffPNG(path string, reference, broken *RenderResult, first FirstMismatch) error {
	if reference.Width != broken.Width || reference.Height != broken.Height {
		return fmt.Errorf("cannot diff different image sizes")
	}
	return writePNG(path, reference.Width, reference.Height, func(x, y int) color.RGBA {
		if first.Found && isMarkerPixel(x, y, first.X, first.Y, 9) {
			return color.RGBA{255, 255, 255, 255}
		}
		index := y*reference.Width + x
		want := reference.Pixels[index]
		got := broken.Pixels[index]
		if len(pixelDifferences(want, got)) == 0 {
			return color.RGBA{18, 19, 24, 255}
		}
		if want.HitGroupID != got.HitGroupID || want.MaterialID != got.MaterialID {
			return color.RGBA{255, 48, 44, 255}
		}
		return color.RGBA{255, 166, 32, 255}
	})
}

func isMarkerPixel(x, y, centerX, centerY, radius int) bool {
	dx := x - centerX
	dy := y - centerY
	if dx < -radius || dx > radius || dy < -radius || dy > radius {
		return false
	}
	return (dx == -radius || dx == radius) && dy >= -radius && dy <= radius ||
		(dy == -radius || dy == radius) && dx >= -radius && dx <= radius
}

func writePNG(path string, width, height int, pixelAt func(x, y int) color.RGBA) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, pixelAt(x, y))
		}
	}
	temp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*.tmp")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := encoder.Encode(temp, img); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempName, path)
}
