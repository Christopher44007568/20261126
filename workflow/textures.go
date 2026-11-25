package workflow

import (
	"fmt"
	"image"

	"texturelab/colorspace"
	"texturelab/mipchain"
	"texturelab/texcontainer"
)

func buildAlbedo() (*texcontainer.Texture, []*image.RGBA, error) {
	source := mipchain.Albedo(MipSizes[0])
	images, err := mipchain.BuildChain(source, len(MipSizes), colorspace.SRGB, false)
	if err != nil {
		return nil, nil, err
	}
	if err := verifySizes(images); err != nil {
		return nil, nil, err
	}
	levels, err := mipchain.ToContainerLevels(images)
	if err != nil {
		return nil, nil, err
	}
	texture := &texcontainer.Texture{
		Format: texcontainer.FormatRGBA8,
		Space:  colorspace.SRGB,
		Levels: levels,
		Metadata: map[string]string{
			"asset":      "procedural-albedo",
			"role":       "albedo",
			"colorspace": colorspace.SRGB.String(),
			"mip_sizes":  "512x512,256x256,128x128",
			"mip_filter": "2x2-linear-light-box",
			"generator":  "texturelab/workflow",
		},
	}
	if err := texture.ValidateMipChain(); err != nil {
		return nil, nil, err
	}
	return texture, images, nil
}

func buildNormal() (*texcontainer.Texture, []*image.RGBA, error) {
	source := mipchain.NormalMap(MipSizes[0], 3.2)
	images, err := mipchain.BuildChain(source, len(MipSizes), colorspace.Linear, true)
	if err != nil {
		return nil, nil, err
	}
	if err := verifySizes(images); err != nil {
		return nil, nil, err
	}
	levels, err := mipchain.ToContainerLevels(images)
	if err != nil {
		return nil, nil, err
	}
	texture := &texcontainer.Texture{
		Format: texcontainer.FormatRGBA8,
		Space:  colorspace.Linear,
		Levels: levels,
		Metadata: map[string]string{
			"asset":           "procedural-normal",
			"role":            "normal-map",
			"colorspace":      colorspace.Linear.String(),
			"data_class":      "vector-data",
			"normal_encoding": "rgb_xyz_minus1_to_1",
			"mip_sizes":       "512x512,256x256,128x128",
			"mip_filter":      "2x2-vector-box-renormalize",
			"warning":         "normal maps must not use an sRGB transfer marker",
			"generator":       "texturelab/workflow",
		},
	}
	if err := texture.ValidateMipChain(); err != nil {
		return nil, nil, err
	}
	return texture, images, nil
}

func verifySizes(images []*image.RGBA) error {
	if len(images) != len(MipSizes) {
		return fmt.Errorf("workflow: got %d mip images, want %d", len(images), len(MipSizes))
	}
	for i, expected := range MipSizes {
		bounds := images[i].Bounds()
		if bounds.Dx() != expected || bounds.Dy() != expected {
			return fmt.Errorf("workflow: level %d is %dx%d, want %dx%d",
				i, bounds.Dx(), bounds.Dy(), expected, expected)
		}
	}
	return nil
}
