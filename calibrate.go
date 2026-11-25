package main

import (
	"errors"
	"math"
)

type CalibFrame struct {
	Object   []Vec3
	Observed []Vec2
	Pose     Pose
}

type CalibrationInput struct {
	Width  int
	Height int
	Frames []CalibFrame
}

type CalibrationResult struct {
	Camera     Camera  `json:"camera"`
	Poses      []Pose  `json:"-"`
	RMSE       float64 `json:"rmse_px"`
	Iterations int     `json:"iterations"`
	Points     int     `json:"points"`
}

func Calibrate(in CalibrationInput) (*CalibrationResult, error) {
	if len(in.Frames) < 2 {
		return nil, errors.New("at least two calibration frames are required")
	}
	for f := range in.Frames {
		if len(in.Frames[f].Object) < 12 || len(in.Frames[f].Object) != len(in.Frames[f].Observed) {
			return nil, errors.New("each frame needs matching object/observed points and at least 12 points")
		}
	}
	cam := Camera{
		Width: in.Width, Height: in.Height,
		Fx: float64(in.Width) * 1.05, Fy: float64(in.Width) * 1.05,
		Cx: float64(in.Width) / 2, Cy: float64(in.Height) / 2,
	}
	p := make([]float64, 8+6*len(in.Frames))
	p[0], p[1] = math.Log(cam.Fx), math.Log(cam.Fy)
	p[2], p[3] = cam.Cx, cam.Cy
	for f, frame := range in.Frames {
		spacing := medianObservedSpacing(frame.Observed)
		square := medianObjectSpacing(frame.Object)
		z := cam.Fx * square / math.Max(spacing, 1)
		rz := estimateInPlaneAngle(frame, square)
		r0 := RotationMatrix(Vec3{Z: rz})
		var ox, oy, ux, uy float64
		for i := range frame.Object {
			ox += frame.Object[i].X
			oy += frame.Object[i].Y
			ux += frame.Observed[i].X
			uy += frame.Observed[i].Y
		}
		n := float64(len(frame.Object))
		ox, oy, ux, uy = ox/n, oy/n, ux/n, uy/n
		mx := r0[0][0]*ox + r0[0][1]*oy
		my := r0[1][0]*ox + r0[1][1]*oy
		base := 8 + 6*f
		p[base+0], p[base+1], p[base+2] = 0, 0, rz
		p[base+3] = (ux-cam.Cx)*z/cam.Fx - mx
		p[base+4] = (uy-cam.Cy)*z/cam.Fy - my
		p[base+5] = z
	}

	objective := func(q []float64) ([]float64, Camera, []Pose) {
		c := camFromParams(q, in.Width, in.Height)
		poses := make([]Pose, len(in.Frames))
		r := make([]float64, 0, 2*countPoints(in.Frames))
		for f, frame := range in.Frames {
			base := 8 + 6*f
			rv := Vec3{X: q[base], Y: q[base+1], Z: q[base+2]}
			tv := Vec3{X: q[base+3], Y: q[base+4], Z: q[base+5]}
			poses[f] = PoseFromRV(rv, tv)
			for i, obj := range frame.Object {
				got := c.Project(obj, poses[f])
				r = append(r, frame.Observed[i].X-got.X, frame.Observed[i].Y-got.Y)
			}
		}
		return r, c, poses
	}

	residuals, _, _ := objective(p)
	cost := dot(residuals, residuals) / float64(len(residuals))
	lambda := 1e-3
	iter := 0
	for iter < 60 {
		jac := numericalJacobian(p, objective)
		jtj := make([][]float64, len(p))
		for i := range jtj {
			jtj[i] = make([]float64, len(p))
		}
		jtr := make([]float64, len(p))
		for row := range residuals {
			for i := range p {
				jtr[i] += jac[row][i] * residuals[row]
				for j := range p {
					jtj[i][j] += jac[row][i] * jac[row][j]
				}
			}
		}
		diag := make([]float64, len(p))
		for i := range p {
			diag[i] = math.Max(jtj[i][i], 1e-9)
		}
		var delta []float64
		for attempt := 0; attempt < 18; attempt++ {
			a := make([][]float64, len(p))
			rhs := make([]float64, len(p))
			for i := range p {
				a[i] = make([]float64, len(p))
				copy(a[i], jtj[i])
				a[i][i] += lambda * diag[i]
				rhs[i] = -jtr[i]
			}
			solved, err := SolveDense(a, rhs)
			if err == nil {
				delta = solved
				candidate := addScaled(p, delta, 1)
				nr, nc, nposes := objective(candidate)
				newCost := dot(nr, nr) / float64(len(nr))
				if !math.IsNaN(newCost) && !math.IsInf(newCost, 0) && newCost < cost {
					p = candidate
					residuals = nr
					cost = newCost
					lambda = math.Max(lambda/3, 1e-10)
					break
				}
				_ = nc
				_ = nposes
			}
			lambda *= 5
			if attempt == 17 {
				delta = nil
			}
		}
		iter++
		if delta == nil || norm(delta) < 1e-7 || math.Sqrt(cost) < 0.12 {
			break
		}
	}
	finalR, finalCam, poses := objective(p)
	rmse := math.Sqrt(dot(finalR, finalR) / float64(len(finalR)/2))
	if err := finalCam.Valid(); err != nil {
		return nil, err
	}
	return &CalibrationResult{Camera: finalCam, Poses: poses, RMSE: rmse, Iterations: iter, Points: len(finalR) / 2}, nil
}

func camFromParams(p []float64, w, h int) Camera {
	return Camera{
		Width: w, Height: h,
		Fx: math.Exp(p[0]), Fy: math.Exp(p[1]),
		Cx: p[2], Cy: p[3],
		K1: p[4], K2: p[5], P1: p[6], P2: p[7],
	}
}

func numericalJacobian(p []float64, objective func([]float64) ([]float64, Camera, []Pose)) [][]float64 {
	r0, _, _ := objective(p)
	j := make([][]float64, len(r0))
	for i := range j {
		j[i] = make([]float64, len(p))
	}
	for col := range p {
		eps := 1e-6 * math.Max(1, math.Abs(p[col]))
		plus := append([]float64{}, p...)
		minus := append([]float64{}, p...)
		plus[col] += eps
		minus[col] -= eps
		rp, _, _ := objective(plus)
		rm, _, _ := objective(minus)
		for row := range r0 {
			j[row][col] = (rp[row] - rm[row]) / (2 * eps)
		}
	}
	return j
}

func countPoints(frames []CalibFrame) int {
	n := 0
	for _, f := range frames {
		n += len(f.Object)
	}
	return n
}

func dot(a, b []float64) float64 {
	s := 0.0
	for i := range a {
		s += a[i] * b[i]
	}
	return s
}

func norm(a []float64) float64 { return math.Sqrt(dot(a, a)) }

func addScaled(a, b []float64, scale float64) []float64 {
	out := make([]float64, len(a))
	for i := range a {
		out[i] = a[i] + scale*b[i]
	}
	return out
}

func medianObservedSpacing(p []Vec2) float64 {
	var ds []float64
	for i := 1; i < len(p); i++ {
		ds = append(ds, math.Hypot(p[i].X-p[i-1].X, p[i].Y-p[i-1].Y))
	}
	return medianFloat(ds)
}

func medianObjectSpacing(p []Vec3) float64 {
	var ds []float64
	for i := 1; i < len(p); i++ {
		ds = append(ds, math.Hypot(p[i].X-p[i-1].X, p[i].Y-p[i-1].Y))
	}
	return medianFloat(ds)
}

func estimateInPlaneAngle(frame CalibFrame, square float64) float64 {
	var sinSum, cosSum float64
	found := false
	for i := 0; i < len(frame.Object); i++ {
		for j := i + 1; j < len(frame.Object); j++ {
			dx := frame.Object[j].X - frame.Object[i].X
			dy := frame.Object[j].Y - frame.Object[i].Y
			if math.Abs(dy) > 1e-6 || math.Abs(math.Abs(dx)-square) > 1e-6 {
				continue
			}
			ux := frame.Observed[j].X - frame.Observed[i].X
			uy := frame.Observed[j].Y - frame.Observed[i].Y
			if dx < 0 {
				ux, uy = -ux, -uy
			}
			length := math.Hypot(ux, uy)
			if length < 1 {
				continue
			}
			cosSum += ux / length
			sinSum += uy / length
			found = true
		}
	}
	if !found {
		return 0
	}
	return math.Atan2(sinSum, cosSum)
}

func medianFloat(xs []float64) float64 {
	if len(xs) == 0 {
		return 1
	}
	cp := append([]float64{}, xs...)
	for i := 1; i < len(cp); i++ {
		for j := i; j > 0 && cp[j] < cp[j-1]; j-- {
			cp[j], cp[j-1] = cp[j-1], cp[j]
		}
	}
	return cp[len(cp)/2]
}
