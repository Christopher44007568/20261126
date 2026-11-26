package audit

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
)

const demoManifestName = "manifest.json"

// WriteDemoAssets creates a complete scene whose three input files have
// deliberately uninformative names. Only manifest.json declares their roles.
func WriteDemoAssets(outputDir string) (string, error) {
	if err := os.MkdirAll(filepath.Join(outputDir, "inputs"), 0o755); err != nil {
		return "", fmt.Errorf("create demo input directory: %w", err)
	}

	const size = 512
	albedo := generateDemoAlbedo(size)
	normal := generateDemoNormal(size)
	roughness := generateDemoRoughness(size)
	files := []struct {
		name    string
		picture *Image
	}{
		{"surface-a.png", albedo},
		{"surface-b.png", normal},
		{"surface-c.png", roughness},
	}
	for _, file := range files {
		path := filepath.Join(outputDir, "inputs", file.name)
		if err := writePNG(path, file.picture); err != nil {
			return "", err
		}
	}

	manifest := defaultManifest()
	manifest.Title = "Linear albedo audit / uninformative input filenames"
	manifest.Render = RenderConfig{Width: 640, Height: 640, Samples: 3, Exposure: 1}
	manifest.Material = MaterialConfig{Metallic: 0, Specular: 0.5, NormalStrength: 1}
	manifest.Light = LightConfig{
		AzimuthDegrees:   -35,
		ElevationDegrees: 42,
		Intensity:        4.2,
		Ambient:          0.10,
	}
	manifest.Textures = TextureSet{
		Albedo:    TextureDeclaration{Path: "inputs/surface-a.png", ColorSpace: ColorSpaceLinear, Channel: "rgb"},
		Normal:    TextureDeclaration{Path: "inputs/surface-b.png", ColorSpace: ColorSpaceLinear, Channel: "rgb"},
		Roughness: TextureDeclaration{Path: "inputs/surface-c.png", ColorSpace: ColorSpaceLinear, Channel: "r"},
	}

	manifestPath := filepath.Join(outputDir, demoManifestName)
	contents, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode demo manifest: %w", err)
	}
	if err := os.WriteFile(manifestPath, append(contents, '\n'), 0o644); err != nil {
		return "", fmt.Errorf("write demo manifest: %w", err)
	}
	return manifestPath, nil
}

func generateDemoAlbedo(size int) *Image {
	picture := NewImage(size, size)
	colors := []Vec3{
		{0.58, 0.16, 0.045},
		{0.022, 0.20, 0.22},
		{0.62, 0.43, 0.16},
		{0.15, 0.035, 0.24},
		{0.42, 0.36, 0.31},
		{0.018, 0.10, 0.13},
	}
	for row := 0; row < size; row++ {
		v := (float64(row) + 0.5) / float64(size)
		for column := 0; column < size; column++ {
			u := (float64(column) + 0.5) / float64(size)
			tileU := int(u * 6)
			tileV := int(v * 5)
			base := colors[(tileU*3+tileV*2)%len(colors)]

			ring := 0.5 + 0.5*math.Sin(2*math.Pi*(u*3+v*2.2))
			accent := Vec3{0.65, 0.58, 0.46}
			base = blendColor(base, accent, 0.16*ring)

			stripe := math.Sin(2 * math.Pi * (u*18 + v*5))
			if stripe > 0.72 {
				base = base.Scale(0.52)
			}
			if stripe < -0.93 {
				base = base.Add(Vec3{0.08, 0.075, 0.06})
			}

			edge := math.Min(math.Min(u, 1-u), math.Min(v, 1-v))
			if edge < 0.022 {
				base = base.Scale(0.18)
			}
			picture.Set(column, row, base.Clamp01())
		}
	}
	return picture
}

func generateDemoNormal(size int) *Image {
	picture := NewImage(size, size)
	for row := 0; row < size; row++ {
		v := (float64(row) + 0.5) / float64(size)
		for column := 0; column < size; column++ {
			u := (float64(column) + 0.5) / float64(size)
			const epsilon = 1.0 / 2048
			du := (demoHeight(u+epsilon, v) - demoHeight(u-epsilon, v)) / (2 * epsilon)
			dv := (demoHeight(u, v+epsilon) - demoHeight(u, v-epsilon)) / (2 * epsilon)
			normalX := clamp(-du*0.018, -0.95, 0.95)
			normalY := clamp(-dv*0.018, -0.95, 0.95)
			normalZ := math.Sqrt(math.Max(0.04, 1-normalX*normalX-normalY*normalY))
			picture.Set(column, row, Vec3{
				0.5 + 0.5*normalX,
				0.5 + 0.5*normalY,
				0.5 + 0.5*normalZ,
			})
		}
	}
	return picture
}

func demoHeight(u, v float64) float64 {
	wave := math.Sin(2*math.Pi*(5*u+1.3*v)) * math.Sin(2*math.Pi*4*v)
	fine := 0.35 * math.Cos(2*math.Pi*13*u) * math.Sin(2*math.Pi*9*v)
	dent := 0.0
	for row := 0; row < 4; row++ {
		for column := 0; column < 4; column++ {
			centerU := (float64(column) + 0.5) / 4
			centerV := (float64(row) + 0.5) / 4
			du := u - centerU
			dv := v - centerV
			dent -= 0.75 * math.Exp(-(du*du+dv*dv)/0.0018)
		}
	}
	return wave + fine + dent
}

func generateDemoRoughness(size int) *Image {
	picture := NewImage(size, size)
	for row := 0; row < size; row++ {
		v := (float64(row) + 0.5) / float64(size)
		for column := 0; column < size; column++ {
			u := (float64(column) + 0.5) / float64(size)
			value := 0.20 + 0.10*(0.5+0.5*math.Sin(2*math.Pi*(u*2.2+v*3.1)))
			for patch := 0; patch < 5; patch++ {
				centerU := 0.16 + 0.68*math.Mod(float64(patch)*0.271, 1)
				centerV := 0.18 + 0.64*math.Mod(float64(patch)*0.413, 1)
				du := u - centerU
				dv := v - centerV
				blob := math.Exp(-(du*du + dv*dv) / 0.010)
				value += 0.52 * blob
			}
			value = clamp(value, 0.12, 0.82)
			picture.Set(column, row, Vec3{value, value, value})
		}
	}
	return picture
}
