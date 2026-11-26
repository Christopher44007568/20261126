package main

import (
	"flag"
	"fmt"
	"image"
	"os"
	"path/filepath"
)

func main() {
	outDir := flag.String("out", filepath.Join("..", "artifacts", "out"), "output directory")
	mapSize := flag.Int("size", 256, "source map edge length in pixels")
	sphereSize := flag.Int("sphere", 360, "material ball edge length in pixels")
	flag.Parse()

	if err := run(*outDir, *mapSize, *sphereSize); err != nil {
		fmt.Fprintln(os.Stderr, "channelpack:", err)
		os.Exit(1)
	}
}

func run(outDir string, mapSize, sphereSize int) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	src := buildSource(mapSize, mapSize)
	if err := saveSourceMaps(outDir, src); err != nil {
		return err
	}
	if err := savePNG(filepath.Join(outDir, "sphere_source.png"), renderSphere(src, sphereSize)); err != nil {
		return err
	}

	layouts := allLayouts()
	reports := make([]LayoutReport, 0, len(layouts))
	sphereSource := renderSphere(src, sphereSize)
	var packedTex []*Texture

	for _, l := range layouts {
		texs := l.Encode(src)
		l.Textures = texs
		if l.Name == "packed" {
			packedTex = texs
		}
		for i, t := range texs {
			name := fmt.Sprintf("%s_texture%d.png", l.Name, i+1)
			if err := savePNG(filepath.Join(outDir, name), textureToImage(t, src.W, src.H)); err != nil {
				return err
			}
		}

		got := l.Decode(texs, src.W, src.H)
		if err := savePNG(filepath.Join(outDir, "sphere_"+l.Name+".png"), renderSphere(got, sphereSize)); err != nil {
			return err
		}
		if err := savePNG(filepath.Join(outDir, "channels_"+l.Name+".png"),
			buildDecomposition(l.Name, texs, src.W, src.H)); err != nil {
			return err
		}
		if err := savePNG(filepath.Join(outDir, "error_"+l.Name+".png"),
			buildErrorMap("ERROR MAP X8 - "+upper(l.Name), src, got, 8)); err != nil {
			return err
		}
		reports = append(reports, verifyLayout(l, src))
	}

	mis := decodeMisaligned(packedTex, src.W, src.H)
	misDiff := countAlbedoDifferences(src, mis)
	if err := savePNG(filepath.Join(outDir, "sphere_misaligned.png"), renderSphere(mis, sphereSize)); err != nil {
		return err
	}
	if err := savePNG(filepath.Join(outDir, "error_misaligned.png"),
		buildErrorMap("ERROR MAP X8 - MISALIGNED", src, mis, 8)); err != nil {
		return err
	}
	spherePacked := renderSphere(decodePacked(packedTex, src.W, src.H), sphereSize)
	sphereMis := renderSphere(mis, sphereSize)
	if err := savePNG(filepath.Join(outDir, "comparison_source_packed_misaligned.png"),
		buildComparison([]*image.NRGBA{sphereSource, spherePacked, sphereMis})); err != nil {
		return err
	}

	report := formatReport(reports, misDiff, mapSize*mapSize*3, mapSize, mapSize, sphereSize)
	if err := os.WriteFile(filepath.Join(outDir, "report.txt"), []byte(report), 0o644); err != nil {
		return err
	}
	fmt.Print(report)
	return assertRequirements(reports, misDiff)
}

func saveSourceMaps(outDir string, src *Maps) error {
	images := map[string]*image.NRGBA{
		"source_albedo.png":    planarToImage(src.Albedo, src.W, src.H, 3),
		"source_normal.png":    planarToImage(src.Normal, src.W, src.H, 3),
		"source_roughness.png": planarToImage(src.Roughness, src.W, src.H, 1),
		"source_occlusion.png": planarToImage(src.Occlusion, src.W, src.H, 1),
	}
	for name, img := range images {
		if err := savePNG(filepath.Join(outDir, name), img); err != nil {
			return err
		}
	}
	return nil
}

func assertRequirements(reports []LayoutReport, misDiff int) error {
	byName := map[string]LayoutReport{}
	for _, r := range reports {
		rr := r
		byName[rr.Name] = rr
	}
	sep, packed, compact := byName["separate"], byName["packed"], byName["compact"]
	var problems []string
	angleTolerance := 1e-4
	if !(sep.AlbedoExact && sep.NormalExact && sep.RoughExact && sep.AOExact) || sep.MaxAngle > angleTolerance {
		problems = append(problems, "separate layout must be byte-exact with zero angular error")
	}
	if !(packed.AlbedoExact && packed.NormalExact && packed.RoughExact && packed.AOExact) || packed.MaxAngle > angleTolerance {
		problems = append(problems, "packed layout must be byte-exact with zero angular error")
	}
	if !compact.AlbedoExact || !compact.NormalExact || compact.MaxAngle > angleTolerance {
		problems = append(problems, "compact scalar quantization must not affect albedo or normals")
	}
	if !(sep.Independence && packed.Independence && compact.Independence) {
		problems = append(problems, "normal/albedo decoding must be independent of scalar channels")
	}
	if compact.roughMax() > 8 || compact.aoMax() > 8 {
		problems = append(problems, "compact scalar error must stay within one 4-bit bin (8 levels)")
	}
	if misDiff == 0 {
		problems = append(problems, "misaligned decoder must demonstrate a detectable slot-binding error")
	}
	if len(problems) > 0 {
		return fmt.Errorf("%v", problems)
	}
	return nil
}

func (r LayoutReport) roughMax() int { return r.channelMax("ROUGH") }

func (r LayoutReport) aoMax() int { return r.channelMax("AO") }

func (r LayoutReport) channelMax(name string) int {
	for _, st := range r.Stats {
		if st.Name == name {
			return st.MaxAbs
		}
	}
	return -1
}
