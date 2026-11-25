package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"mipwrap/internal/mipmap"
)

func Run(args []string) error {
	flags := flag.NewFlagSet("mipwrap", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	outDir := flags.String("out", "artifacts/mipwrap", "directory for PNG, CSV, JSON and Markdown outputs")
	input := flags.String("input", "", "optional color image (PNG/JPEG); default creates a diagnostic tile")
	heightInput := flags.String("height", "", "optional height image; defaults to luminance or procedural height")
	size := flags.Int("size", 256, "procedural texture edge size")
	normalStrength := flags.Float64("normal-strength", 2.4, "height-to-normal slope strength")
	linearLight := flags.Bool("linear-light", false, "average in linear light instead of gamma-encoded output RGB")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if *size < 2 {
		*size = 2
	}

	colorBase, heightBase, err := loadSources(*input, *heightInput, *size, *linearLight)
	if err != nil {
		return err
	}

	wrapChain, energyReports := mipmap.GenerateWrapMipChain(colorBase)
	naiveChain := mipmap.GenerateNaiveMipChain(colorBase)
	insideChain := mipmap.GenerateInsideMipChain(colorBase)
	heightChain := mipmap.GenerateHeightMipChain(heightBase)
	normalChain := mipmap.NormalMapMipChain(heightChain, *normalStrength)

	if err := writeImages(*outDir, wrapChain, naiveChain, insideChain, normalChain); err != nil {
		return err
	}

	report := buildReport(colorBase, *normalStrength, wrapChain, naiveChain, insideChain, energyReports)
	if err := mipmap.WriteJSON(filepath.Join(*outDir, "metrics.json"), report); err != nil {
		return err
	}
	if err := mipmap.WriteColorCSV(filepath.Join(*outDir, "average-color-error.csv"), report.ColorMetrics); err != nil {
		return err
	}
	if err := mipmap.WriteMarkdownReport(filepath.Join(*outDir, "report.md"), report); err != nil {
		return err
	}
	if err := mipmap.WriteErrorChart(filepath.Join(*outDir, "average-color-error.png"), report); err != nil {
		return err
	}

	fmt.Printf("wrote mipmap outputs to %s\n", *outDir)
	fmt.Printf("color space: %s\n", report.ColorSpace)
	fmt.Printf("wrap max average-color RMS error: %.3e\n", report.MaxWrapColorErrorWorking)
	fmt.Printf("constant edge max deviation: %.3e\n", report.MaxProbeEdgeDeviation)
	if !report.Pass {
		return fmt.Errorf("brightness conservation check failed")
	}
	return nil
}

func loadSources(input, heightInput string, size int, linearLight bool) (*mipmap.RGBImage, *mipmap.HeightField, error) {
	if input == "" {
		color, height := mipmap.NewDiagnosticTexture(size, linearLight)
		return color, height, nil
	}

	decoded, err := mipmap.ReadImage(input)
	if err != nil {
		return nil, nil, fmt.Errorf("read input: %w", err)
	}
	var color *mipmap.RGBImage
	if linearLight {
		color = mipmap.FromImageLinear(decoded)
	} else {
		color = mipmap.FromImageEncoded(decoded)
	}

	if heightInput != "" {
		heightImage, err := mipmap.ReadImage(heightInput)
		if err != nil {
			return nil, nil, fmt.Errorf("read height: %w", err)
		}
		return color, mipmap.HeightFromImage(heightImage), nil
	}
	return color, mipmap.HeightFromLuminance(color), nil
}
