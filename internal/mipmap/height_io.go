package mipmap

import "image"

func HeightFromLuminance(img *RGBImage) *HeightField {
	height := NewHeightField(img.Width, img.Height)
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			r, g, b := img.RGB(x, y)
			height.Set(x, y, Luminance([3]float64{r, g, b}))
		}
	}
	return height
}

func HeightFromImage(src image.Image) *HeightField {
	return HeightFromLuminance(FromImage(src))
}
