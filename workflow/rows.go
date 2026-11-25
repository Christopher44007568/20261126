package workflow

import (
	"path/filepath"

	"texturelab/atlas"
	"texturelab/catalog"
	"texturelab/diagnose"
	"texturelab/material"
	"texturelab/texcontainer"
)

type rowAsset struct {
	asset   string
	role    string
	texture *texcontainer.Texture
	info    *texcontainer.Info
	sheet   *atlas.Sheet
}

type rowInputs struct {
	paths      Files
	albedo     rowAsset
	normal     rowAsset
	mislabeled rowAsset
	diagnostic *diagnose.Report
}

func buildRows(input rowInputs) ([]catalog.Row, error) {
	rows := make([]catalog.Row, 0, 9)

	albedoRows, err := rowsForAsset(input, input.albedo,
		filepath.Base(input.paths.AlbedoContainer), filepath.Base(input.paths.AlbedoAtlas), false)
	if err != nil {
		return nil, err
	}
	normalRows, err := rowsForAsset(input, input.normal,
		filepath.Base(input.paths.NormalContainer), filepath.Base(input.paths.NormalAtlas), true)
	if err != nil {
		return nil, err
	}
	mislabeledRows, err := rowsForAsset(input, input.mislabeled,
		filepath.Base(input.paths.MislabeledNormalContainer), filepath.Base(input.paths.NormalAtlas), true)
	if err != nil {
		return nil, err
	}
	rows = append(rows, albedoRows...)
	rows = append(rows, normalRows...)
	rows = append(rows, mislabeledRows...)
	return rows, nil
}

func rowsForAsset(input rowInputs, asset rowAsset, containerFile, atlasFile string, attachDiagnostic bool) ([]catalog.Row, error) {
	rows := make([]catalog.Row, 0, len(asset.texture.Levels))
	for i, level := range asset.texture.Levels {
		fingerprint, err := material.LinearFingerprint(asset.texture, i)
		if err != nil {
			return nil, err
		}
		infoLevel := asset.info.Levels[i]
		cell := asset.sheet.Cells[i]
		row := catalog.Row{
			Asset:            asset.asset,
			Role:             asset.role,
			ContainerFile:    containerFile,
			ColorSpaceMarker: asset.texture.Space.String(),
			Level:            i,
			Width:            level.Width,
			Height:           level.Height,
			PixelBytes:       len(level.Pixels),
			ContainerOffset:  infoLevel.AbsoluteOffset,
			ContainerSize:    infoLevel.Size,
			CRC32:            infoLevel.CRC32,
			PayloadSHA256:    hashBytes(level.Pixels),
			LinearSHA256:     fingerprint,
			AtlasFile:        atlasFile,
			AtlasX:           cell.X,
			AtlasY:           cell.Y,
			AtlasWidth:       cell.Width,
			AtlasHeight:      cell.Height,
		}
		if attachDiagnostic && input.diagnostic != nil && i < len(input.diagnostic.Levels) {
			diagnosticLevel := input.diagnostic.Levels[i]
			row.DiagnosticMax = diagnosticLevel.MaxAbsDelta
			row.DiagnosticMean = diagnosticLevel.MeanAbsRGB
			row.DiagnosticLocated = diagnosticLevel.Detected
		}
		rows = append(rows, row)
	}
	return rows, nil
}
