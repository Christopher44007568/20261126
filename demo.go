package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

const (
	fileNormalFrame = "normal_frame.png"
	fileStaleFrame  = "stale_binding_frame.png"
	fileOverlay     = "version_source_overlay.png"
	fileReport      = "descriptor_report.json"
)

type ImageSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type Bounds struct {
	MinX  int  `json:"min_x"`
	MinY  int  `json:"min_y"`
	MaxX  int  `json:"max_x"`
	MaxY  int  `json:"max_y"`
	Valid bool `json:"valid"`
}

type FrameDifference struct {
	ChangedPixels int     `json:"changed_pixels"`
	TotalPixels   int     `json:"total_pixels"`
	Percent       float64 `json:"percent"`
	Threshold     int     `json:"threshold"`
	Bounds        Bounds  `json:"bounds"`
}

type CameraReport struct {
	Version         CameraVersion `json:"version"`
	Label           string        `json:"label"`
	Matrix          Mat3          `json:"matrix"`
	ProjectedCenter Vec2          `json:"projected_center"`
}

type BindingReport struct {
	Material          string            `json:"material"`
	DescriptorVersion DescriptorVersion `json:"descriptor_version"`
	Camera            CameraReport      `json:"camera"`
	TextureVersion    TextureVersion    `json:"texture_version"`
	TextureLabel      string            `json:"texture_label"`
	TextureHash       string            `json:"texture_hash"`
	Stale             bool              `json:"stale"`
	Invalidations     []string          `json:"invalidations"`
	CameraAlive       bool              `json:"camera_alive"`
	CameraReferences  int               `json:"camera_references"`
	TextureReferences int               `json:"texture_references"`
}

type LifetimeReport struct {
	RetainedCameras  []uint64        `json:"retained_cameras"`
	RetainedTextures []uint64        `json:"retained_textures"`
	GCCollected      []CameraVersion `json:"gc_collected_cameras"`
	PinnedCamera     uint64          `json:"pinned_camera"`
	PinnedCameraRefs int             `json:"pinned_camera_references"`
	TextureRefs      int             `json:"texture_references"`
}

type DemoReport struct {
	Schema            int             `json:"schema"`
	Canvas            ImageSize       `json:"canvas"`
	TextureVersion    TextureVersion  `json:"texture_version"`
	TextureLabel      string          `json:"texture_label"`
	TextureHash       string          `json:"texture_hash"`
	SameTexturePixels bool            `json:"same_texture_pixels"`
	NormalBinding     BindingReport   `json:"normal_binding"`
	StaleBinding      BindingReport   `json:"stale_binding"`
	ProjectionDelta   float64         `json:"projection_delta_pixels"`
	FrameDifference   FrameDifference `json:"frame_difference"`
	Lifetime          LifetimeReport  `json:"lifetime"`
	Files             []string        `json:"files"`
}

type DemoResult struct {
	Normal  *image.RGBA
	Stale   *image.RGBA
	Overlay *image.RGBA
	Report  DemoReport
}

func diffFrames(a, b *image.RGBA, threshold int) FrameDifference {
	bounds := a.Bounds().Intersect(b.Bounds())
	out := FrameDifference{
		TotalPixels: bounds.Dx() * bounds.Dy(),
		Threshold:   threshold,
	}
	minX, minY := math.MaxInt, math.MaxInt
	maxX, maxY := math.MinInt, math.MinInt
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			ai, bi := a.PixOffset(x, y), b.PixOffset(x, y)
			d := absDiff(a.Pix[ai], b.Pix[bi]) + absDiff(a.Pix[ai+1], b.Pix[bi+1]) + absDiff(a.Pix[ai+2], b.Pix[bi+2])
			if d < threshold {
				continue
			}
			out.ChangedPixels++
			minX, minY = min(minX, x), min(minY, y)
			maxX, maxY = max(maxX, x), max(maxY, y)
		}
	}
	if out.ChangedPixels > 0 {
		out.Bounds = Bounds{MinX: minX, MinY: minY, MaxX: maxX, MaxY: maxY, Valid: true}
	}
	if out.TotalPixels > 0 {
		out.Percent = 100 * float64(out.ChangedPixels) / float64(out.TotalPixels)
	}
	return out
}

func absDiff(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}

func writePNG(path string, img *image.RGBA) error {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func cameraReport(resource CameraResource) CameraReport {
	center, _ := resource.Matrix.Transform(Vec2{})
	return CameraReport{
		Version:         resource.Version,
		Label:           resource.Label,
		Matrix:          resource.Matrix,
		ProjectedCenter: center,
	}
}

func bindingReport(material *Material, cameras *CameraPool, textures *TextureStore) (BindingReport, error) {
	snapshot, err := material.Binding()
	if err != nil {
		return BindingReport{}, err
	}
	camera, ok := cameras.Get(snapshot.CameraVersion)
	if !ok {
		return BindingReport{}, fmt.Errorf("missing camera v%d for report", snapshot.CameraVersion)
	}
	names := snapshot.InvalidationNames()
	if names == nil {
		names = []string{}
	}
	return BindingReport{
		Material:          material.ID,
		DescriptorVersion: snapshot.DescriptorVersion,
		Camera:            cameraReport(camera),
		TextureVersion:    snapshot.TextureVersion,
		TextureLabel:      snapshot.TextureLabel,
		TextureHash:       snapshot.TextureHash,
		Stale:             snapshot.Stale,
		Invalidations:     names,
		CameraAlive:       cameras.Alive(snapshot.CameraVersion),
		CameraReferences:  cameras.RefCount(snapshot.CameraVersion),
		TextureReferences: textures.RefCount(snapshot.TextureVersion),
	}, nil
}

func demoCameraV2() Mat3 {
	return CameraMatrix(418, 278, 286, 222, -0.062, 0.022)
}

func demoCameraV3() Mat3 {
	return CameraMatrix(520, 308, 320, 252, 0.048, 0.014)
}

// BuildDemo constructs the version history, renders the live and stale
// materials, and captures the lifetime audit while both materials are alive.
func BuildDemo() (*DemoResult, error) {
	cameras := NewCameraPool(CameraMatrix(472, 294, 300, 236, -0.018, 0.018), "CAMERA.V1.INITIAL")
	textures := NewTextureStore()
	texture := textures.Install(MakeMaterialTexture(256), "MATERIAL.TEX.V1")
	system := NewDescriptorSystem(cameras, textures)

	previous := system.UpdateCamera(demoCameraV2(), "CAMERA.V2.PREVIOUS")
	staleMaterial, err := system.Bind("MATERIAL.STALE.SNAPSHOT", previous.Version, texture.Version)
	if err != nil {
		return nil, err
	}
	defer system.Release(staleMaterial)

	current := system.UpdateCamera(demoCameraV3(), "CAMERA.V3.CURRENT")
	normalMaterial, err := system.Bind("MATERIAL.LIVE.SNAPSHOT", current.Version, texture.Version)
	if err != nil {
		return nil, err
	}
	defer system.Release(normalMaterial)

	removedCameras, _ := system.GC()
	staleSnapshot, err := staleMaterial.Binding()
	if err != nil {
		return nil, err
	}
	normalSnapshot, err := normalMaterial.Binding()
	if err != nil {
		return nil, err
	}
	if !staleSnapshot.Stale || !staleSnapshot.Invalidations.Has(InvalidateCameraChanged) {
		return nil, fmt.Errorf("stale material did not receive camera invalidation: %s", staleSnapshot.Invalidations)
	}
	if normalSnapshot.Stale {
		return nil, fmt.Errorf("current material unexpectedly stale: %s", normalSnapshot.Invalidations)
	}
	if staleSnapshot.TextureVersion != normalSnapshot.TextureVersion ||
		staleSnapshot.TextureHash != normalSnapshot.TextureHash ||
		staleSnapshot.TextureHash != texture.Hash {
		return nil, fmt.Errorf("normal and stale frames do not share the exact texture revision")
	}

	staleBinding, err := system.Resolve(staleMaterial)
	if err != nil {
		return nil, err
	}
	normalBinding, err := system.Resolve(normalMaterial)
	if err != nil {
		return nil, err
	}
	if !cameras.Alive(staleSnapshot.CameraVersion) {
		return nil, fmt.Errorf("stale camera v%d was evicted while its material was active", staleSnapshot.CameraVersion)
	}

	normalFrame, err := RenderFrame(normalBinding)
	if err != nil {
		return nil, err
	}
	staleFrame, err := RenderFrame(staleBinding)
	if err != nil {
		return nil, err
	}
	overlay, err := BuildVersionOverlay(normalBinding, staleBinding, normalFrame)
	if err != nil {
		return nil, err
	}

	staleReport, err := bindingReport(staleMaterial, cameras, textures)
	if err != nil {
		return nil, err
	}
	normalReport, err := bindingReport(normalMaterial, cameras, textures)
	if err != nil {
		return nil, err
	}
	staleCenter, _ := staleBinding.Camera.Matrix.Transform(Vec2{})
	normalCenter, _ := normalBinding.Camera.Matrix.Transform(Vec2{})
	delta := normalCenter.sub(staleCenter).length()

	report := DemoReport{
		Schema:            1,
		Canvas:            ImageSize{Width: canvasW, Height: canvasH},
		TextureVersion:    texture.Version,
		TextureLabel:      texture.Label,
		TextureHash:       texture.Hash,
		SameTexturePixels: true,
		NormalBinding:     normalReport,
		StaleBinding:      staleReport,
		ProjectionDelta:   delta,
		FrameDifference:   diffFrames(normalFrame, staleFrame, 30),
		Lifetime: LifetimeReport{
			RetainedCameras:  cameras.RetainedVersions(),
			RetainedTextures: textures.RetainedVersions(),
			GCCollected:      removedCameras,
			PinnedCamera:     uint64(staleSnapshot.CameraVersion),
			PinnedCameraRefs: cameras.RefCount(staleSnapshot.CameraVersion),
			TextureRefs:      textures.RefCount(texture.Version),
		},
		Files: []string{fileNormalFrame, fileStaleFrame, fileOverlay, fileReport},
	}

	result := &DemoResult{
		Normal:  normalFrame,
		Stale:   staleFrame,
		Overlay: overlay,
		Report:  report,
	}

	return result, nil
}

// RunDemo writes the three required frames and a deterministic audit report.
func RunDemo(outDir string) (*DemoResult, error) {
	result, err := BuildDemo()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	if err := writePNG(filepath.Join(outDir, fileNormalFrame), result.Normal); err != nil {
		return nil, err
	}
	if err := writePNG(filepath.Join(outDir, fileStaleFrame), result.Stale); err != nil {
		return nil, err
	}
	if err := writePNG(filepath.Join(outDir, fileOverlay), result.Overlay); err != nil {
		return nil, err
	}
	if err := writeJSON(filepath.Join(outDir, fileReport), result.Report); err != nil {
		return nil, err
	}
	return result, nil
}
