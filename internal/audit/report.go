package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type OutputPaths struct {
	Correct    string `json:"correct"`
	Misread    string `json:"forced_srgb_albedo"`
	Comparison string `json:"comparison"`
	ErrorMap   string `json:"local_error"`
	Report     string `json:"report"`
}

type TextureReport struct {
	Role               Role       `json:"role"`
	Path               string     `json:"path"`
	DeclaredColorSpace ColorSpace `json:"declared_color_space"`
	CorrectTransfer    string     `json:"correct_transfer"`
	MisreadTransfer    string     `json:"misread_transfer"`
	Channel            string     `json:"channel"`
	Width              int        `json:"width"`
	Height             int        `json:"height"`
	SHA256             string     `json:"sha256"`
}

type DataInvariant struct {
	NormalLinear                bool `json:"normal_linear"`
	RoughnessLinear             bool `json:"roughness_linear"`
	MisreadModifiedDataChannels bool `json:"misread_modified_data_channels"`
}

type AuditReport struct {
	Tool                string            `json:"tool"`
	Manifest            string            `json:"manifest"`
	Outputs             OutputPaths       `json:"outputs"`
	ColorSpacePolicy    string            `json:"color_space_policy"`
	AlbedoInterpretedAs ColorSpace        `json:"albedo_correct_interpretation"`
	MisreadAlbedoAs     ColorSpace        `json:"misread_albedo_interpretation"`
	MisreadHasEffect    bool              `json:"misread_has_effect"`
	ErrorGain           float64           `json:"local_error_gain"`
	Metrics             DifferenceMetrics `json:"metrics"`
	Textures            []TextureReport   `json:"textures"`
	Invariants          DataInvariant     `json:"invariants"`
}

func RunAudit(manifestPath, outputDir string, errorGain float64) (AuditReport, error) {
	if errorGain <= 0 {
		return AuditReport{}, fmt.Errorf("local error gain must be positive")
	}
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return AuditReport{}, err
	}
	textures, err := LoadMaterialTextures(manifest)
	if err != nil {
		return AuditReport{}, err
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return AuditReport{}, fmt.Errorf("create output directory: %w", err)
	}

	correct := Render(manifest, textures, RenderVariant{})
	misread := Render(manifest, textures, RenderVariant{ForceSRGBAlbedo: true})
	errorMap, metrics, err := BuildErrorMap(correct, misread, errorGain)
	if err != nil {
		return AuditReport{}, fmt.Errorf("build local error map: %w", err)
	}
	comparison, err := BuildComparison(
		correct,
		misread,
		"DECLARED ALBEDO",
		"FORCED SRGB ALBEDO",
	)
	if err != nil {
		return AuditReport{}, fmt.Errorf("build comparison: %w", err)
	}

	absoluteOutput, err := filepath.Abs(outputDir)
	if err != nil {
		return AuditReport{}, fmt.Errorf("resolve output directory: %w", err)
	}
	outputs := OutputPaths{
		Correct:    filepath.Join(absoluteOutput, "correct.png"),
		Misread:    filepath.Join(absoluteOutput, "forced-srgb-albedo.png"),
		Comparison: filepath.Join(absoluteOutput, "comparison.png"),
		ErrorMap:   filepath.Join(absoluteOutput, "local-error.png"),
		Report:     filepath.Join(absoluteOutput, "report.json"),
	}
	if err := writePNG(outputs.Correct, correct); err != nil {
		return AuditReport{}, err
	}
	if err := writePNG(outputs.Misread, misread); err != nil {
		return AuditReport{}, err
	}
	if err := writePNG(outputs.Comparison, comparison); err != nil {
		return AuditReport{}, err
	}
	if err := writePNG(outputs.ErrorMap, errorMap); err != nil {
		return AuditReport{}, err
	}

	report := AuditReport{
		Tool:                "texaudit",
		Manifest:            manifestPath,
		Outputs:             outputs,
		ColorSpacePolicy:    "only albedo is reinterpreted as sRGB; normal and roughness remain raw linear data",
		AlbedoInterpretedAs: textures.Albedo.DeclaredColorSpace,
		MisreadAlbedoAs:     ColorSpaceSRGB,
		MisreadHasEffect:    textures.Albedo.DeclaredColorSpace != ColorSpaceSRGB,
		ErrorGain:           errorGain,
		Metrics:             metrics,
		Textures: []TextureReport{
			textureReport(textures.Albedo, albedoTransfer(textures.Albedo.DeclaredColorSpace), "sRGB to linear (forced)"),
			textureReport(textures.Normal, "identity: linear data", "identity: linear data"),
			textureReport(textures.Roughness, "identity: linear data", "identity: linear data"),
		},
		Invariants: DataInvariant{
			NormalLinear:                true,
			RoughnessLinear:             true,
			MisreadModifiedDataChannels: false,
		},
	}
	if err := writeJSON(outputs.Report, report); err != nil {
		return AuditReport{}, err
	}
	return report, nil
}

func albedoTransfer(colorSpace ColorSpace) string {
	if colorSpace == ColorSpaceSRGB {
		return "sRGB to linear (declared)"
	}
	return "identity: declared linear"
}

func textureReport(texture *Texture, correctTransfer, misreadTransfer string) TextureReport {
	return TextureReport{
		Role:               texture.Role,
		Path:               texture.Path,
		DeclaredColorSpace: texture.DeclaredColorSpace,
		CorrectTransfer:    correctTransfer,
		MisreadTransfer:    misreadTransfer,
		Channel:            texture.Channel,
		Width:              texture.Width,
		Height:             texture.Height,
		SHA256:             texture.SHA256,
	}
}

func writePNG(path string, picture *Image) error {
	temporary := path + ".tmp"
	file, err := os.Create(temporary)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	if err := picture.WritePNG(file); err != nil {
		file.Close()
		os.Remove(temporary)
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := file.Close(); err != nil {
		os.Remove(temporary)
		return fmt.Errorf("close %s: %w", path, err)
	}
	if err := os.Rename(temporary, path); err != nil {
		os.Remove(temporary)
		return fmt.Errorf("install %s: %w", path, err)
	}
	return nil
}

func writeJSON(path string, value any) error {
	temporary := path + ".tmp"
	file, err := os.Create(temporary)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		file.Close()
		os.Remove(temporary)
		return fmt.Errorf("encode %s: %w", path, err)
	}
	if err := file.Close(); err != nil {
		os.Remove(temporary)
		return fmt.Errorf("close %s: %w", path, err)
	}
	if err := os.Rename(temporary, path); err != nil {
		os.Remove(temporary)
		return fmt.Errorf("install %s: %w", path, err)
	}
	return nil
}
