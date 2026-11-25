package main

import (
	"fmt"
	"os"

	"deshake/internal/geom"
	"deshake/internal/stab"
	"deshake/internal/synth"
)

const (
	frameW  = 480
	frameH  = 360
	nFrames = 30
	outDir  = "out"
)

type rowMetrics struct {
	Frame                                   int
	Hole2D, Hole3D                          float64
	JellyNoneRMSE, Jelly2DRMSE, Jelly3DRMSE float64
	JellyNonePP, Jelly2DPP, Jelly3DPP       float64
	Bound3DRMSE, Bound3DPP                  float64
	Edge2D, Edge3D                          stab.Margin
	Box2D, Box3D                            stab.Rect
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "deshake:", err)
		os.Exit(1)
	}
}

func run() error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	cam := geom.Camera{
		W: frameW, H: frameH,
		Fx: 430, Fy: 430,
		Cx: frameW / 2.0, Cy: frameH / 2.0,
		Depth: 700,
	}
	sc := synth.New()
	motion := synth.NewMotion(cam, nFrames)

	all := make([]rowMetrics, 0, nFrames)
	var res2d, res3d []*stab.FrameResult
	frames := make([]*synth.Frame, 0, nFrames)

	for fi := 0; fi < nFrames; fi++ {
		f := motion.Capture(sc, fi)
		frames = append(frames, f)

		r2 := stab.Stabilize2D(cam, f)
		r3 := stab.Stabilize3D(cam, f)
		res2d = append(res2d, r2)
		res3d = append(res3d, r3)

		jNoneRMSE, jNonePP := stab.JellyWaviness(cam, f, "none")
		j2RMSE, j2PP := stab.JellyWaviness(cam, f, "2d")
		j3RMSE, j3PP := stab.JellyWaviness(cam, f, "3d")
		bRMSE, bPP := stab.BoundaryWaviness(r3)

		all = append(all, rowMetrics{
			Frame:  fi,
			Hole2D: stab.HoleFraction(r2), Hole3D: stab.HoleFraction(r3),
			JellyNoneRMSE: jNoneRMSE, Jelly2DRMSE: j2RMSE, Jelly3DRMSE: j3RMSE,
			JellyNonePP: jNonePP, Jelly2DPP: j2PP, Jelly3DPP: j3PP,
			Bound3DRMSE: bRMSE, Bound3DPP: bPP,
			Edge2D: r2.Edge, Edge3D: r3.Edge,
			Box2D: r2.Box, Box3D: r3.Box,
		})
		fmt.Printf("frame %02d  hole 2D=%5.2f%% 3D=%5.2f%%  jellyRMSE none=%.2f 2D=%.2f 3D=%.2f px\n",
			fi, 100*stab.HoleFraction(r2), 100*stab.HoleFraction(r3),
			jNoneRMSE, j2RMSE, j3RMSE)
	}

	global2D := intersectBoxes(extractBoxes(res2d))
	global3D := intersectBoxes(extractBoxes(res3d))
	if err := writeMetrics(all, global2D, global3D); err != nil {
		return err
	}
	if err := writeTriptyches(frames, res2d, res3d, all, global2D, global3D); err != nil {
		return err
	}
	if err := writeMarginChart(all, res3d); err != nil {
		return err
	}
	fmt.Println("global safe crop 2D:", global2D, " 3D:", global3D)
	return nil
}
