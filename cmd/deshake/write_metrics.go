package main

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"

	"deshake/internal/stab"
)

func writeMetrics(all []rowMetrics, g2, g3 stab.Rect) error {
	cf, err := os.Create(filepath.Join(outDir, "metrics.csv"))
	if err != nil {
		return err
	}
	defer cf.Close()
	w := csv.NewWriter(cf)
	header := []string{
		"frame", "hole_2d_pct", "hole_3d_pct",
		"jelly_none_rmse_px", "jelly_2d_rmse_px", "jelly_3d_rmse_px",
		"jelly_none_pp_px", "jelly_2d_pp_px", "jelly_3d_pp_px",
		"bound3d_wav_rmse_px", "bound3d_wav_pp_px",
		"m2d_top", "m2d_bottom", "m2d_left", "m2d_right",
		"m3d_top", "m3d_bottom", "m3d_left", "m3d_right",
	}
	if err := w.Write(header); err != nil {
		return err
	}
	for _, m := range all {
		rec := []string{
			strconv.Itoa(m.Frame),
			f4(100 * m.Hole2D), f4(100 * m.Hole3D),
			f4(m.JellyNoneRMSE), f4(m.Jelly2DRMSE), f4(m.Jelly3DRMSE),
			f4(m.JellyNonePP), f4(m.Jelly2DPP), f4(m.Jelly3DPP),
			f4(m.Bound3DRMSE), f4(m.Bound3DPP),
			f4(m.Edge2D.Top), f4(m.Edge2D.Bottom), f4(m.Edge2D.Left), f4(m.Edge2D.Right),
			f4(m.Edge3D.Top), f4(m.Edge3D.Bottom), f4(m.Edge3D.Left), f4(m.Edge3D.Right),
		}
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}

	summary := map[string]any{
		"frames":          len(all),
		"global_crop_2d":  g2,
		"global_crop_3d":  g3,
		"mean":            meanBlock(all),
		"fast_turn_frame": selectTurn(all),
	}
	jf, err := os.Create(filepath.Join(outDir, "summary.json"))
	if err != nil {
		return err
	}
	defer jf.Close()
	enc := json.NewEncoder(jf)
	enc.SetIndent("", "  ")
	return enc.Encode(summary)
}

func meanBlock(all []rowMetrics) map[string]float64 {
	var h2, h3, jn, j2, j3, b3 float64
	for _, m := range all {
		h2 += m.Hole2D
		h3 += m.Hole3D
		jn += m.JellyNoneRMSE
		j2 += m.Jelly2DRMSE
		j3 += m.Jelly3DRMSE
		b3 += m.Bound3DRMSE
	}
	n := float64(len(all))
	return map[string]float64{
		"hole_2d_pct":         100 * h2 / n,
		"hole_3d_pct":         100 * h3 / n,
		"jelly_none_rmse_px":  jn / n,
		"jelly_2d_rmse_px":    j2 / n,
		"jelly_3d_rmse_px":    j3 / n,
		"bound3d_wav_rmse_px": b3 / n,
	}
}

func selectTurn(all []rowMetrics) int {
	idx, maxV := 0, -1.0
	for _, m := range all {
		if m.Bound3DPP > maxV {
			maxV, idx = m.Bound3DPP, m.Frame
		}
	}
	return idx
}

func f4(v float64) string { return strconv.FormatFloat(v, 'f', 3, 64) }
