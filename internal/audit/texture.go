package audit

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
)

type Pixel struct {
	R float32
	G float32
	B float32
	A float32
}

type Texture struct {
	Role               Role
	Path               string
	DeclaredColorSpace ColorSpace
	Channel            string
	Width              int
	Height             int
	SHA256             string
	Pixels             []Pixel
}

type MaterialTextures struct {
	Albedo    *Texture
	Normal    *Texture
	Roughness *Texture
}

func LoadMaterialTextures(manifest Manifest) (MaterialTextures, error) {
	albedo, err := loadTexture(RoleAlbedo, manifest.Textures.Albedo, manifest.TexturePath(manifest.Textures.Albedo))
	if err != nil {
		return MaterialTextures{}, err
	}
	normal, err := loadTexture(RoleNormal, manifest.Textures.Normal, manifest.TexturePath(manifest.Textures.Normal))
	if err != nil {
		return MaterialTextures{}, err
	}
	roughness, err := loadTexture(RoleRoughness, manifest.Textures.Roughness, manifest.TexturePath(manifest.Textures.Roughness))
	if err != nil {
		return MaterialTextures{}, err
	}
	return MaterialTextures{Albedo: albedo, Normal: normal, Roughness: roughness}, nil
}

func loadTexture(role Role, declaration TextureDeclaration, path string) (*Texture, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s texture %q: %w", role, path, err)
	}
	decoded, _, err := image.Decode(bytes.NewReader(contents))
	if err != nil {
		return nil, fmt.Errorf("decode %s texture %q (PNG and JPEG are supported): %w", role, path, err)
	}

	bounds := decoded.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("%s texture %q has empty bounds", role, path)
	}
	if width > 8192 || height > 8192 || width*height > 16_777_216 {
		return nil, fmt.Errorf("%s texture %q is too large (%dx%d)", role, path, width, height)
	}

	pixels := make([]Pixel, width*height)
	for row := 0; row < height; row++ {
		for column := 0; column < width; column++ {
			source := decoded.At(bounds.Min.X+column, bounds.Min.Y+row)
			converted := colorToPixel(source)
			pixels[row*width+column] = converted
		}
	}

	digest := sha256.Sum256(contents)
	return &Texture{
		Role:               role,
		Path:               path,
		DeclaredColorSpace: declaration.ColorSpace,
		Channel:            declaration.Channel,
		Width:              width,
		Height:             height,
		SHA256:             fmt.Sprintf("%x", digest),
		Pixels:             pixels,
	}, nil
}

// SampleAlbedo applies a transfer function only when the selected
// interpretation explicitly says sRGB. The caller chooses declared (correct)
// or forced-sRGB (audit comparison).
func (texture *Texture) SampleAlbedo(u, v float64, interpretation ColorSpace) Vec3 {
	encoded := texture.sampleRGB(u, v)
	if interpretation != ColorSpaceSRGB {
		return encoded
	}
	return Vec3{
		SRGBToLinear(encoded.X),
		SRGBToLinear(encoded.Y),
		SRGBToLinear(encoded.Z),
	}
}

// SampleNormal returns the stored map values unchanged. Normal data is linear
// by definition; running sRGB decoding here bends the surface and invents
// highlights.
func (texture *Texture) SampleNormal(u, v float64) Vec3 {
	return texture.sampleRGB(u, v)
}

// SampleRoughness returns stored linear data unchanged. It deliberately has no
// ColorSpace parameter so a display-color correction cannot alter gloss.
func (texture *Texture) SampleRoughness(u, v float64) float64 {
	if texture.Channel == "luma" {
		value := texture.sampleRGB(u, v)
		return clamp01(0.2126*value.X + 0.7152*value.Y + 0.0722*value.Z)
	}
	channel := channelOffset(texture.Channel)
	return clamp01(texture.sampleChannel(u, v, channel))
}

func colorToPixel(value color.Color) Pixel {
	converted := color.NRGBA64Model.Convert(value).(color.NRGBA64)
	return Pixel{
		R: float32(converted.R) / 65535,
		G: float32(converted.G) / 65535,
		B: float32(converted.B) / 65535,
		A: float32(converted.A) / 65535,
	}
}

func channelOffset(channel string) int {
	switch channel {
	case "g":
		return 1
	case "b":
		return 2
	case "a":
		return 3
	default:
		return 0
	}
}

func (texture *Texture) sampleChannel(u, v float64, channel int) float64 {
	row0, row1, column0, column1, vertical, horizontal := texture.sampleCoordinates(u, v)
	topLeft := texture.channelAt(column0, row0, channel)
	topRight := texture.channelAt(column1, row0, channel)
	bottomLeft := texture.channelAt(column0, row1, channel)
	bottomRight := texture.channelAt(column1, row1, channel)
	top := mix(topLeft, topRight, horizontal)
	bottom := mix(bottomLeft, bottomRight, horizontal)
	return mix(top, bottom, vertical)
}

func (texture *Texture) sampleRGB(u, v float64) Vec3 {
	row0, row1, column0, column1, vertical, horizontal := texture.sampleCoordinates(u, v)
	topLeft := texture.pixelAt(column0, row0)
	topRight := texture.pixelAt(column1, row0)
	bottomLeft := texture.pixelAt(column0, row1)
	bottomRight := texture.pixelAt(column1, row1)
	top := Pixel{
		R: float32(mix(float64(topLeft.R), float64(topRight.R), horizontal)),
		G: float32(mix(float64(topLeft.G), float64(topRight.G), horizontal)),
		B: float32(mix(float64(topLeft.B), float64(topRight.B), horizontal)),
	}
	bottom := Pixel{
		R: float32(mix(float64(bottomLeft.R), float64(bottomRight.R), horizontal)),
		G: float32(mix(float64(bottomLeft.G), float64(bottomRight.G), horizontal)),
		B: float32(mix(float64(bottomLeft.B), float64(bottomRight.B), horizontal)),
	}
	return Vec3{
		mix(float64(top.R), float64(bottom.R), vertical),
		mix(float64(top.G), float64(bottom.G), vertical),
		mix(float64(top.B), float64(bottom.B), vertical),
	}
}

func (texture *Texture) sampleCoordinates(u, v float64) (int, int, int, int, float64, float64) {
	u = u - math.Floor(u)
	v = clamp01(v)
	x := u * float64(texture.Width)
	y := v * float64(texture.Height-1)
	column0 := int(math.Floor(x)) % texture.Width
	row0 := int(math.Floor(y))
	column1 := (column0 + 1) % texture.Width
	row1 := row0 + 1
	if row1 >= texture.Height {
		row1 = texture.Height - 1
	}
	return row0, row1, column0, column1, y - math.Floor(y), x - math.Floor(x)
}

func (texture *Texture) pixelAt(column, row int) Pixel {
	return texture.Pixels[row*texture.Width+column]
}

func (texture *Texture) channelAt(column, row, channel int) float64 {
	pixel := texture.pixelAt(column, row)
	switch channel {
	case 1:
		return float64(pixel.G)
	case 2:
		return float64(pixel.B)
	case 3:
		return float64(pixel.A)
	default:
		return float64(pixel.R)
	}
}
