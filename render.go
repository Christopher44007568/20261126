package rayao

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"runtime"
	"sync"
)

// Options controls the deterministic three-pass frame renderer.
type Options struct {
	Width         int
	Height        int
	AOSamples     int
	AORadius      float64
	DepthNear     float64
	DepthFar      float64
	SurfaceOffset float64
	SecondaryMinT float64
}

// DefaultOptions is tuned for the built-in demo scene scale.
func DefaultOptions() Options {
	return Options{
		Width:         320,
		Height:        240,
		AOSamples:     48,
		AORadius:      1.45,
		DepthNear:     2.1,
		DepthFar:      8.5,
		SurfaceOffset: 0.0025,
		SecondaryMinT: 1e-5,
	}
}

// Stats records the shared query workload used for a frame.
type Stats struct {
	Pixels         int `json:"pixels"`
	PrimaryQueries int `json:"primary_queries"`
	PrimaryHits    int `json:"primary_hits"`
	KeyQueries     int `json:"key_shadow_queries"`
	KeyAnyHits     int `json:"key_shadow_any_hits"`
	AmbientQueries int `json:"ambient_queries"`
	AmbientAnyHits int `json:"ambient_any_hits"`
}

// Frame is one versioned scene rendered into the three requested maps.
type Frame struct {
	Width        int
	Height       int
	SceneName    string
	SceneVersion string
	SceneHash    string
	Depth        *image.Gray
	AnyHitShadow *image.Gray
	Composite    *image.RGBA
	DepthNear    float64
	DepthFar     float64
	Stats        Stats
}

// Render evaluates primary closest-hit rays and secondary any-hit ambient
// queries against one pinned scene snapshot.
func Render(scene *Scene, camera Camera, opts Options) (*Frame, error) {
	if scene == nil {
		return nil, fmt.Errorf("rayao: nil scene")
	}
	pinned := scene.Snapshot()
	opts = normalizeOptions(opts)

	frame := &Frame{
		Width:        opts.Width,
		Height:       opts.Height,
		SceneName:    pinned.Name,
		SceneVersion: pinned.Version,
		SceneHash:    pinned.ContentHash,
		Depth:        image.NewGray(image.Rect(0, 0, opts.Width, opts.Height)),
		AnyHitShadow: image.NewGray(image.Rect(0, 0, opts.Width, opts.Height)),
		Composite:    image.NewRGBA(image.Rect(0, 0, opts.Width, opts.Height)),
		DepthNear:    opts.DepthNear,
		DepthFar:     opts.DepthFar,
		Stats:        Stats{},
	}

	workers := runtime.GOMAXPROCS(0)
	if workers > opts.Height {
		workers = opts.Height
	}
	if workers < 1 {
		workers = 1
	}
	rows := make(chan int, workers)
	var wg sync.WaitGroup
	var statsMu sync.Mutex
	var total Stats
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var local Stats
			for y := range rows {
				for x := 0; x < opts.Width; x++ {
					renderPixel(pinned, camera, opts, frame, x, y, &local)
				}
			}
			statsMu.Lock()
			total.Add(local)
			statsMu.Unlock()
		}()
	}
	for y := 0; y < opts.Height; y++ {
		rows <- y
	}
	close(rows)
	wg.Wait()
	frame.Stats = total
	return frame, nil
}

// Add accumulates worker-local statistics.
func (s *Stats) Add(other Stats) {
	s.Pixels += other.Pixels
	s.PrimaryQueries += other.PrimaryQueries
	s.PrimaryHits += other.PrimaryHits
	s.KeyQueries += other.KeyQueries
	s.KeyAnyHits += other.KeyAnyHits
	s.AmbientQueries += other.AmbientQueries
	s.AmbientAnyHits += other.AmbientAnyHits
}

func renderPixel(scene *Scene, camera Camera, opts Options, frame *Frame, x, y int, stats *Stats) {
	stats.Pixels++
	stats.PrimaryQueries++
	primaryRay := camera.PrimaryRay(x, y, opts.Width, opts.Height)
	primary := scene.Query(DefaultRayQuery(primaryRay))
	if primary.HasClosest {
		stats.PrimaryHits++
	}

	depthValue := encodeDepth(primary.Closest.T, primary.HasClosest, opts)
	frame.Depth.SetGray(x, y, color.Gray{Y: depthValue})

	visibility := 1.0
	shadeColor := background(primaryRay.Direction)
	if primary.HasClosest {
		var ambientHits, ambientRays, keyHits, keyRays int
		visibility, ambientHits, ambientRays = ambientVisibility(scene, primary.Closest, opts, x, y)
		keyVisibility, keyHits, keyRays := keyLightVisibility(scene, primary.Closest, opts)
		visibility *= keyVisibility
		stats.AmbientQueries += ambientRays
		stats.AmbientAnyHits += ambientHits
		stats.KeyQueries += keyRays
		stats.KeyAnyHits += keyHits
		shadeColor = shadeHit(scene, primary.Closest, visibility, primaryRay.Direction, opts)
	}

	shadowGray := uint8(math.Round(clamp01(visibility) * 255))
	frame.AnyHitShadow.SetGray(x, y, color.Gray{Y: shadowGray})
	frame.Composite.SetRGBA(x, y, toRGBA(shadeColor))
}

func normalizeOptions(opts Options) Options {
	d := DefaultOptions()
	if opts.Width <= 0 {
		opts.Width = d.Width
	}
	if opts.Height <= 0 {
		opts.Height = d.Height
	}
	if opts.AOSamples <= 0 {
		opts.AOSamples = d.AOSamples
	}
	if opts.AORadius == 0 {
		opts.AORadius = d.AORadius
	}
	if opts.DepthNear <= 0 {
		opts.DepthNear = d.DepthNear
	}
	if opts.DepthFar <= opts.DepthNear {
		opts.DepthFar = opts.DepthNear + d.DepthFar - d.DepthNear
	}
	if opts.SurfaceOffset <= 0 {
		opts.SurfaceOffset = d.SurfaceOffset
	}
	if opts.SecondaryMinT <= 0 {
		opts.SecondaryMinT = d.SecondaryMinT
	}
	return opts
}

func encodeDepth(t float64, hit bool, opts Options) uint8 {
	if !hit || math.IsInf(t, 1) || math.IsNaN(t) {
		return 0
	}
	linear := clamp01((t - opts.DepthNear) / (opts.DepthFar - opts.DepthNear))
	// White is near; black is far or miss. The manifest records the metric range.
	return uint8(math.Round((1.0 - linear) * 255.0))
}

func background(direction Vec3) Vec3 {
	sky := Vec3{X: 0.72, Y: 0.82, Z: 0.92}
	ground := Vec3{X: 0.20, Y: 0.22, Z: 0.25}
	blend := clamp01(direction.Y*0.5 + 0.5)
	return ground.Lerp(sky, blend)
}

func toRGBA(c Vec3) color.RGBA {
	r := uint8(math.Round(255 * math.Pow(clamp01(c.X), 1.0/2.2)))
	g := uint8(math.Round(255 * math.Pow(clamp01(c.Y), 1.0/2.2)))
	b := uint8(math.Round(255 * math.Pow(clamp01(c.Z), 1.0/2.2)))
	return color.RGBA{R: r, G: g, B: b, A: 255}
}
