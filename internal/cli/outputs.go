package cli

import (
	"fmt"
	"path/filepath"

	"mipwrap/internal/mipmap"
)

func writeImages(outDir string, wrap, naive, inside, normals []*mipmap.RGBImage) error {
	colorMontage := mipmap.BuildMipMontage("COLOR MIP CHAIN", mipmap.EncodedColorItems(wrap))
	if err := mipmap.WritePNG(filepath.Join(outDir, "color_mip_montage.png"), colorMontage); err != nil {
		return err
	}

	normalMontage := mipmap.BuildMipMontage("NORMAL MIP CHAIN", mipmap.EncodedNormalItems(normals))
	if err := mipmap.WritePNG(filepath.Join(outDir, "normal_mip_montage.png"), normalMontage); err != nil {
		return err
	}

	seamSheet := mipmap.BuildSeamSheet("SEAM DETAIL", []mipmap.SeamColumn{
		{Title: "NAIVE", Levels: naive},
		{Title: "INSIDE NORM", Levels: inside},
		{Title: "WRAP", Levels: wrap},
	})
	if err := mipmap.WritePNG(filepath.Join(outDir, "seam_zoom.png"), seamSheet); err != nil {
		return err
	}

	normalSeam := mipmap.BuildSeamSheet("NORMAL SEAMS", []mipmap.SeamColumn{
		{Title: "REBUILT", Levels: normals, AsNormal: true},
	})
	if err := mipmap.WritePNG(filepath.Join(outDir, "normal_seam_zoom.png"), normalSeam); err != nil {
		return err
	}

	probeBase := mipmap.NewRGBImage(64, 64)
	probeBase.FillGray(0.5)
	probeWrap, _ := mipmap.GenerateWrapMipChain(probeBase)
	probeNaive := mipmap.GenerateNaiveMipChain(probeBase)
	probeInside := mipmap.GenerateInsideMipChain(probeBase)
	edgeProbeSheet := mipmap.BuildSeamSheet("EDGE PROBE", []mipmap.SeamColumn{
		{Title: "NAIVE", Levels: probeNaive},
		{Title: "INSIDE NORM", Levels: probeInside},
		{Title: "WRAP", Levels: probeWrap},
	})
	if err := mipmap.WritePNG(filepath.Join(outDir, "edge_probe_zoom.png"), edgeProbeSheet); err != nil {
		return err
	}

	for index, level := range wrap {
		path := filepath.Join(outDir, "levels", fmt.Sprintf("color_L%02d_%dx%d.png", index, level.Width, level.Height))
		if err := mipmap.WritePNG(path, level.ToOutputRGBA()); err != nil {
			return err
		}
	}
	for index, normal := range normals {
		path := filepath.Join(outDir, "levels", fmt.Sprintf("normal_L%02d_%dx%d.png", index, normal.Width, normal.Height))
		if err := mipmap.WritePNG(path, normal.ToNormalRGBA()); err != nil {
			return err
		}
	}
	return nil
}
