package workflow

import (
	"image"
	"math"

	"texturelab/material"
	"texturelab/texcontainer"
)

func imagesFromTexture(texture *texcontainer.Texture) []*image.RGBA {
	images := make([]*image.RGBA, 0, len(texture.Levels))
	for _, level := range texture.Levels {
		img := image.NewRGBA(image.Rect(0, 0, level.Width, level.Height))
		copy(img.Pix, level.Pixels)
		images = append(images, img)
	}
	return images
}

func deltaImages(correct, interpreted *texcontainer.Texture, amplification float64) ([]*image.RGBA, error) {
	images := make([]*image.RGBA, 0, len(correct.Levels))
	for levelIndex := range correct.Levels {
		level := correct.Levels[levelIndex]
		img := image.NewRGBA(image.Rect(0, 0, level.Width, level.Height))
		for y := 0; y < level.Height; y++ {
			for x := 0; x < level.Width; x++ {
				correctSample, err := material.SampleLinear(correct, levelIndex, x, y)
				if err != nil {
					return nil, err
				}
				wrongSample, err := material.SampleLinear(interpreted, levelIndex, x, y)
				if err != nil {
					return nil, err
				}
				offset := (y*level.Width + x) * 4
				for channel := 0; channel < 3; channel++ {
					delta := math.Abs(wrongSample[channel] - correctSample[channel])
					intensity := math.Round(math.Min(1, delta*amplification) * 255)
					img.Pix[offset+channel] = uint8(intensity)
				}
				img.Pix[offset+3] = 255
			}
		}
		images = append(images, img)
	}
	return images, nil
}
