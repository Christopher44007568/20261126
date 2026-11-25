package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"math"
	"os"
	"path/filepath"
	"time"
)

type UpdateEvent struct {
	Sequence           int       `json:"sequence"`
	Generation         int       `json:"generation"`
	Accepted           bool      `json:"accepted"`
	Reason             string    `json:"reason,omitempty"`
	RMSE               *float64  `json:"rmse_px"`
	RetainedResult     bool      `json:"retained_previous_good"`
	RetainedGeneration int       `json:"retained_generation,omitempty"`
	FileHashes         []string  `json:"file_hashes,omitempty"`
	At                 time.Time `json:"at"`
}

type Keeper struct {
	Dir        string
	MaxRMSE    float64
	LastGood   *CalibrationResult
	Generation int
	Sequence   int
}

func NewKeeper(dir string, maxRMSE float64) *Keeper {
	if maxRMSE <= 0 {
		maxRMSE = 2.0
	}
	return &Keeper{Dir: dir, MaxRMSE: maxRMSE}
}

func (k *Keeper) LoadLastGood() error {
	path := filepath.Join(k.Dir, "last_good_calibration.json")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var saved struct {
		Generation int `json:"generation"`
		CalibrationResult
	}
	if err := json.Unmarshal(data, &saved); err != nil {
		return err
	}
	if err := saved.Camera.Valid(); err != nil || saved.RMSE > k.MaxRMSE {
		return fmt.Errorf("saved result is unusable: %w", err)
	}
	k.LastGood = &saved.CalibrationResult
	k.Generation = saved.Generation
	if data, err := os.ReadFile(filepath.Join(k.Dir, "fallback_events.jsonl")); err == nil {
		for _, b := range data {
			if b == '\n' {
				k.Sequence++
			}
		}
	}
	return nil
}

func (k *Keeper) Publish(result CalibrationResult, files map[string]*image.RGBA) (*UpdateEvent, error) {
	k.Sequence++
	var rmseValue *float64
	if !math.IsNaN(result.RMSE) && !math.IsInf(result.RMSE, 0) {
		rmseValue = &result.RMSE
	}
	event := UpdateEvent{Sequence: k.Sequence, RMSE: rmseValue, At: time.Now().UTC()}
	if reason := k.invalidReason(result); reason != "" {
		event.Accepted = false
		event.Reason = reason
		event.RetainedResult = k.LastGood != nil
		if k.LastGood != nil {
			event.RetainedGeneration = k.Generation
			event.FileHashes = k.hashCanonicalFiles()
		}
		_ = appendJSONL(filepath.Join(k.Dir, "fallback_events.jsonl"), event)
		safeResult := sanitizeResultForJSON(result)
		_ = SaveJSON(filepath.Join(k.Dir, "rejected_update.json"), map[string]any{
			"event":          event,
			"candidate":      safeResult,
			"last_good_used": k.LastGood,
		})
		return &event, errors.New(reason)
	}

	nextGeneration := k.Generation + 1
	for name, img := range files {
		if img == nil {
			continue
		}
		if err := SavePNG(filepath.Join(k.Dir, name), img); err != nil {
			return nil, err
		}
	}
	payload := struct {
		Generation int `json:"generation"`
		CalibrationResult
	}{Generation: nextGeneration, CalibrationResult: result}
	if err := SaveJSON(filepath.Join(k.Dir, "last_good_calibration.json"), payload); err != nil {
		return nil, err
	}
	if err := SaveJSON(filepath.Join(k.Dir, "calibration.json"), payload); err != nil {
		return nil, err
	}
	k.LastGood = &result
	k.Generation = nextGeneration
	event.Accepted = true
	event.Generation = nextGeneration
	_ = appendJSONL(filepath.Join(k.Dir, "fallback_events.jsonl"), event)
	_ = SaveJSON(filepath.Join(k.Dir, "keeper_state.json"), map[string]any{
		"generation": k.Generation, "last_good_rmse": result.RMSE, "sequence": k.Sequence,
	})
	return &event, nil
}

func sanitizeResultForJSON(result CalibrationResult) CalibrationResult {
	safe := result
	if math.IsNaN(safe.RMSE) || math.IsInf(safe.RMSE, 0) {
		safe.RMSE = -1
	}
	cam := safe.Camera
	clean := func(v float64) float64 {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0
		}
		return v
	}
	cam.Fx, cam.Fy, cam.Cx, cam.Cy = clean(cam.Fx), clean(cam.Fy), clean(cam.Cx), clean(cam.Cy)
	cam.K1, cam.K2, cam.K3 = clean(cam.K1), clean(cam.K2), clean(cam.K3)
	cam.P1, cam.P2 = clean(cam.P1), clean(cam.P2)
	safe.Camera = cam
	return safe
}

func (k *Keeper) invalidReason(r CalibrationResult) string {
	if r.Points < 24 {
		return "too few calibration points"
	}
	if math.IsNaN(r.RMSE) || math.IsInf(r.RMSE, 0) || r.RMSE > k.MaxRMSE {
		return fmt.Sprintf("reprojection RMSE %.3f exceeds acceptance threshold %.3f", r.RMSE, k.MaxRMSE)
	}
	if err := r.Camera.Valid(); err != nil {
		return err.Error()
	}
	return ""
}

func (k *Keeper) hashCanonicalFiles() []string {
	names := []string{
		"original_grid.png", "corrected_grid.png", "rectified_metric_grid.png",
		"residual_heatmap.png", "edge_missing_original_grid.png",
	}
	var out []string
	for _, name := range names {
		path := filepath.Join(k.Dir, name)
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		h := sha256.New()
		_, _ = io.Copy(h, f)
		_ = f.Close()
		out = append(out, name+":"+hex.EncodeToString(h.Sum(nil)))
	}
	return out
}

func SaveJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func appendJSONL(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(data, '\n'))
	return err
}
