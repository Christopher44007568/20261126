package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ColorSpace string

const (
	ColorSpaceSRGB   ColorSpace = "srgb"
	ColorSpaceLinear ColorSpace = "linear"
)

type Role string

const (
	RoleAlbedo    Role = "albedo"
	RoleNormal    Role = "normal"
	RoleRoughness Role = "roughness"
)

type RenderConfig struct {
	Width    int     `json:"width"`
	Height   int     `json:"height"`
	Samples  int     `json:"samples"`
	Exposure float64 `json:"exposure"`
}

type MaterialConfig struct {
	Metallic       float64 `json:"metallic"`
	Specular       float64 `json:"specular"`
	NormalStrength float64 `json:"normal_strength"`
}

type LightConfig struct {
	AzimuthDegrees   float64 `json:"azimuth_degrees"`
	ElevationDegrees float64 `json:"elevation_degrees"`
	Intensity        float64 `json:"intensity"`
	Ambient          float64 `json:"ambient"`
}

type TextureDeclaration struct {
	Path       string     `json:"path"`
	ColorSpace ColorSpace `json:"color_space"`
	Channel    string     `json:"channel,omitempty"`
}

type TextureSet struct {
	Albedo    TextureDeclaration `json:"albedo"`
	Normal    TextureDeclaration `json:"normal"`
	Roughness TextureDeclaration `json:"roughness"`
}

type Manifest struct {
	Title    string         `json:"title,omitempty"`
	Render   RenderConfig   `json:"render"`
	Material MaterialConfig `json:"material"`
	Light    LightConfig    `json:"light"`
	Textures TextureSet     `json:"textures"`

	BaseDir string `json:"-"`
}

func defaultManifest() Manifest {
	return Manifest{
		Render: RenderConfig{
			Width:    720,
			Height:   720,
			Samples:  2,
			Exposure: 1,
		},
		Material: MaterialConfig{
			Specular:       0.5,
			NormalStrength: 1,
		},
		Light: LightConfig{
			AzimuthDegrees:   -35,
			ElevationDegrees: 42,
			Intensity:        4.2,
			Ambient:          0.10,
		},
	}
}

// LoadManifest uses only the explicit role and color_space declarations. It
// never infers either from the texture file name or extension.
func LoadManifest(path string) (Manifest, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read manifest: %w", err)
	}

	manifest := defaultManifest()
	decoder := json.NewDecoder(strings.NewReader(string(contents)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode manifest: %w", err)
	}

	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("resolve manifest path: %w", err)
	}
	manifest.BaseDir = filepath.Dir(absolutePath)
	applyDeclarationDefaults(&manifest)
	if err := manifest.Validate(); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func applyDeclarationDefaults(manifest *Manifest) {
	manifest.Textures.Albedo.ColorSpace = normalizeColorSpace(manifest.Textures.Albedo.ColorSpace)
	manifest.Textures.Normal.ColorSpace = normalizeColorSpace(manifest.Textures.Normal.ColorSpace)
	manifest.Textures.Roughness.ColorSpace = normalizeColorSpace(manifest.Textures.Roughness.ColorSpace)
	manifest.Textures.Albedo.Channel = normalizeChannel(manifest.Textures.Albedo.Channel)
	manifest.Textures.Normal.Channel = normalizeChannel(manifest.Textures.Normal.Channel)
	manifest.Textures.Roughness.Channel = normalizeChannel(manifest.Textures.Roughness.Channel)
	if manifest.Textures.Albedo.Channel == "" {
		manifest.Textures.Albedo.Channel = "rgb"
	}
	if manifest.Textures.Normal.Channel == "" {
		manifest.Textures.Normal.Channel = "rgb"
	}
	if manifest.Textures.Roughness.Channel == "" {
		manifest.Textures.Roughness.Channel = "r"
	}
}

func normalizeColorSpace(colorSpace ColorSpace) ColorSpace {
	return ColorSpace(strings.ToLower(strings.TrimSpace(string(colorSpace))))
}

func normalizeChannel(channel string) string {
	return strings.ToLower(strings.TrimSpace(channel))
}

func (manifest Manifest) Validate() error {
	if manifest.Render.Width <= 0 || manifest.Render.Height <= 0 {
		return fmt.Errorf("render width and height must be positive")
	}
	if manifest.Render.Width*manifest.Render.Height > 16_777_216 {
		return fmt.Errorf("render dimensions are too large (maximum 16,777,216 pixels)")
	}
	if manifest.Render.Samples < 1 || manifest.Render.Samples > 4 {
		return fmt.Errorf("render samples must be between 1 and 4")
	}
	if manifest.Render.Exposure <= 0 {
		return fmt.Errorf("render exposure must be positive")
	}
	if manifest.Material.Metallic < 0 || manifest.Material.Metallic > 1 {
		return fmt.Errorf("material metallic must be between 0 and 1")
	}
	if manifest.Material.Specular < 0 || manifest.Material.Specular > 1 {
		return fmt.Errorf("material specular must be between 0 and 1")
	}
	if manifest.Material.NormalStrength < 0 || manifest.Material.NormalStrength > 4 {
		return fmt.Errorf("material normal_strength must be between 0 and 4")
	}
	if manifest.Light.Intensity < 0 || manifest.Light.Ambient < 0 {
		return fmt.Errorf("light intensity and ambient must not be negative")
	}

	declarations := []struct {
		role        Role
		declaration TextureDeclaration
	}{
		{RoleAlbedo, manifest.Textures.Albedo},
		{RoleNormal, manifest.Textures.Normal},
		{RoleRoughness, manifest.Textures.Roughness},
	}
	for _, entry := range declarations {
		if strings.TrimSpace(entry.declaration.Path) == "" {
			return fmt.Errorf("%s texture path is required", entry.role)
		}
		switch entry.declaration.ColorSpace {
		case ColorSpaceSRGB, ColorSpaceLinear:
		default:
			return fmt.Errorf("%s texture has unsupported color_space %q (use srgb or linear)", entry.role, entry.declaration.ColorSpace)
		}
		if entry.role != RoleAlbedo && entry.declaration.ColorSpace != ColorSpaceLinear {
			return fmt.Errorf("%s is data and must be declared linear; sRGB correction would create false shading", entry.role)
		}
		if err := validateChannel(entry.role, entry.declaration.Channel); err != nil {
			return err
		}
	}
	return nil
}

func validateChannel(role Role, channel string) error {
	switch role {
	case RoleAlbedo, RoleNormal:
		if channel != "rgb" {
			return fmt.Errorf("%s texture channel must be rgb", role)
		}
	case RoleRoughness:
		switch channel {
		case "r", "g", "b", "a", "luma":
		default:
			return fmt.Errorf("roughness channel must be r, g, b, a, or luma")
		}
	}
	return nil
}

func (manifest Manifest) TexturePath(declaration TextureDeclaration) string {
	if filepath.IsAbs(declaration.Path) {
		return declaration.Path
	}
	return filepath.Join(manifest.BaseDir, declaration.Path)
}
