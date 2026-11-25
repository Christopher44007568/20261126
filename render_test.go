package rayao

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestRenderMapsSharePinnedSceneVersion(t *testing.T) {
	scene, camera, options := NewDemoScene()
	options.Width = 48
	options.Height = 36
	options.AOSamples = 4

	frame, err := Render(scene, camera, options)
	if err != nil {
		t.Fatal(err)
	}
	if frame.SceneVersion != scene.Version || frame.SceneHash != scene.ContentHash {
		t.Fatalf("frame version %s/%s does not match scene %s/%s",
			frame.SceneVersion, frame.SceneHash, scene.Version, scene.ContentHash)
	}

	want := frame.Depth.Bounds()
	if frame.AnyHitShadow.Bounds() != want || frame.Composite.Bounds() != want {
		t.Fatalf("bounds differ: depth=%v shadow=%v composite=%v",
			frame.Depth.Bounds(), frame.AnyHitShadow.Bounds(), frame.Composite.Bounds())
	}
	if frame.Stats.PrimaryQueries != options.Width*options.Height {
		t.Fatalf("primary queries = %d, want %d", frame.Stats.PrimaryQueries, options.Width*options.Height)
	}
	if frame.Stats.PrimaryHits < options.Width*options.Height/2 {
		t.Fatalf("only %d primary hits in demo scene", frame.Stats.PrimaryHits)
	}
	if frame.Stats.AmbientQueries != frame.Stats.PrimaryHits*options.AOSamples {
		t.Fatalf("AO queries=%d, want hits*samples", frame.Stats.AmbientQueries)
	}
	if frame.Stats.KeyQueries < 0 || frame.Stats.KeyQueries > frame.Stats.PrimaryHits {
		t.Fatalf("key queries = %d outside [0,%d]", frame.Stats.KeyQueries, frame.Stats.PrimaryHits)
	}
	if frame.Stats.KeyAnyHits > frame.Stats.KeyQueries {
		t.Fatalf("key hits %d exceed queries %d", frame.Stats.KeyAnyHits, frame.Stats.KeyQueries)
	}
}

func TestWriteOutputsRoundTripAndManifestVersion(t *testing.T) {
	scene, camera, options := NewDemoScene()
	options.Width = 32
	options.Height = 24
	options.AOSamples = 2
	frame, err := Render(scene, camera, options)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	manifest, err := WriteOutputs(frame, dir)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.SceneVersion != scene.Version || manifest.SceneHash != scene.ContentHash {
		t.Fatalf("manifest version mismatch: %+v", manifest)
	}

	for _, name := range []string{DepthPNGFilename, AnyHitShadowFilename, CompositePNGFilename} {
		path := filepath.Join(dir, name)
		file, err := os.Open(path)
		if err != nil {
			t.Fatalf("open %s: %v", name, err)
		}
		cfg, err := png.DecodeConfig(file)
		_ = file.Close()
		if err != nil {
			t.Fatalf("decode config %s: %v", name, err)
		}
		if cfg.Width != options.Width || cfg.Height != options.Height {
			t.Fatalf("%s size=%dx%d want %dx%d", name, cfg.Width, cfg.Height, options.Width, options.Height)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		if got := hex.EncodeToString(sum[:]); got != manifest.SHA256[name] {
			t.Fatalf("%s checksum %s != manifest %s", name, got, manifest.SHA256[name])
		}
	}

	data, err := os.ReadFile(filepath.Join(dir, ManifestFilename))
	if err != nil {
		t.Fatal(err)
	}
	var loaded Manifest
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}
	if loaded.SceneHash != scene.ContentHash {
		t.Fatalf("loaded hash = %q, want %q", loaded.SceneHash, scene.ContentHash)
	}
}

func TestSnapshotPreservesVersionAndCopiesGeometry(t *testing.T) {
	scene, _, _ := NewDemoScene()
	snapshot := scene.Snapshot()
	if snapshot.Version != scene.Version || snapshot.ContentHash != scene.ContentHash {
		t.Fatal("snapshot changed version")
	}
	if &snapshot.Triangles[0] == &scene.Triangles[0] {
		t.Fatal("snapshot did not copy triangle slice")
	}
}
