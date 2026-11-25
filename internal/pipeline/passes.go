package pipeline

import "math"

// 管线中流转的具名资源。
const (
	ResDepth      = "depth"       // 单通道：几何深度
	ResShadow     = "shadow"      // 单通道：阴影遮蔽，0=全影 1=无影
	ResColor      = "color"       // RGB：未着色的物体颜色
	ResLit        = "lit"         // RGB：光照合成结果
	ResFinalColor = "final_color" // RGB：后处理后的最终颜色
	ResPPMColor   = "ppm_color"   // RGB：输出 Pass 拷贝，供落盘使用
)

func insideObject(x, y, w, h int) bool {
	fx := float64(x)/float64(w-1) - 0.5
	fy := float64(y)/float64(h-1) - 0.5
	// 一个斜椭圆“物体”，深度在中心最近。
	return fx*fx/0.10+fy*fy/0.22 <= 1.0
}

// GeometryPass 写入深度缓冲。
func GeometryPass() *Pass {
	return &Pass{
		Name:   "geometry",
		Writes: []string{ResDepth},
		Fn: func(frame *Frame) error {
			depth := frame.Ensure(ResDepth, 1)
			for y := 0; y < frame.Height; y++ {
				for x := 0; x < frame.Width; x++ {
					fx := float64(x)/float64(frame.Width-1) - 0.5
					fy := float64(y)/float64(frame.Height-1) - 0.5
					// 背景是远离相机的平面；物体区域按到中心距离变近。
					d := 0.9
					if insideObject(x, y, frame.Width, frame.Height) {
						d = 0.35 + 0.4*math.Hypot(fx/0.32, fy/0.47)
					}
					*depth.At(x, y, 0) = d
				}
			}
			return nil
		},
	}
}

// ShadowPass 读取深度，写入阴影遮蔽。深度越深越接近阴影区。
func ShadowPass() *Pass {
	return &Pass{
		Name:   "shadow",
		Reads:  []string{ResDepth},
		Writes: []string{ResShadow},
		Fn: func(frame *Frame) error {
			depth, err := frame.Get(ResDepth)
			if err != nil {
				return err
			}
			shadow := frame.Ensure(ResShadow, 1)
			for y := 0; y < frame.Height; y++ {
				for x := 0; x < frame.Width; x++ {
					d := depth.Get(x, y, 0)
					s := clamp01((d - 0.3) / 0.55)
					*shadow.At(x, y, 0) = 0.25 + 0.75*s
				}
			}
			return nil
		},
	}
}

// ColorPass 写入未着色颜色：天空渐变 + 橙色物体。
func ColorPass() *Pass {
	return &Pass{
		Name:   "color",
		Writes: []string{ResColor},
		Fn: func(frame *Frame) error {
			color := frame.Ensure(ResColor, 3)
			for y := 0; y < frame.Height; y++ {
				for x := 0; x < frame.Width; x++ {
					t := float64(y) / float64(frame.Height-1)
					if insideObject(x, y, frame.Width, frame.Height) {
						*color.At(x, y, 0) = 0.95
						*color.At(x, y, 1) = 0.42
						*color.At(x, y, 2) = 0.12
						continue
					}
					*color.At(x, y, 0) = 0.12 + 0.10*(1-t)
					*color.At(x, y, 1) = 0.25 + 0.20*(1-t)
					*color.At(x, y, 2) = 0.55 + 0.30*(1-t)
				}
			}
			return nil
		},
	}
}

// LightingPass 同时消费颜色与阴影：环境光 + 方向光 x 遮蔽。
func LightingPass() *Pass {
	return &Pass{
		Name:   "lighting",
		Reads:  []string{ResColor, ResShadow},
		Writes: []string{ResLit},
		Fn: func(frame *Frame) error {
			color, err := frame.Get(ResColor)
			if err != nil {
				return err
			}
			shadow, err := frame.Get(ResShadow)
			if err != nil {
				return err
			}
			lit := frame.Ensure(ResLit, 3)
			for y := 0; y < frame.Height; y++ {
				for x := 0; x < frame.Width; x++ {
					s := shadow.Get(x, y, 0)
					intensity := 0.35 + 0.65*s
					for ch := 0; ch < 3; ch++ {
						v := color.Get(x, y, ch) * intensity
						*lit.At(x, y, ch) = clamp01(v)
					}
				}
			}
			return nil
		},
	}
}

// PostProcessPass 只消费光照颜色：gamma、对比度和暗角。
func PostProcessPass() *Pass {
	return &Pass{
		Name:   "postprocess",
		Reads:  []string{ResLit},
		Writes: []string{ResFinalColor},
		Fn: func(frame *Frame) error {
			lit, err := frame.Get(ResLit)
			if err != nil {
				return err
			}
			out := frame.Ensure(ResFinalColor, 3)
			for y := 0; y < frame.Height; y++ {
				for x := 0; x < frame.Width; x++ {
					fx := float64(x)/float64(frame.Width-1) - 0.5
					fy := float64(y)/float64(frame.Height-1) - 0.5
					vignette := 1.0 - 0.35*math.Hypot(fx, fy)
					for ch := 0; ch < 3; ch++ {
						v := lit.Get(x, y, ch)
						v = math.Pow(v, 1.0/2.2)
						v = (v-0.5)*1.15 + 0.5
						*out.At(x, y, ch) = clamp01(v * vignette)
					}
				}
			}
			return nil
		},
	}
}

// OutputPass 把最终颜色拷贝为可落盘缓冲，是写文件前的最后一道 Pass。
func OutputPass() *Pass {
	return &Pass{
		Name:   "output",
		Reads:  []string{ResFinalColor},
		Writes: []string{ResPPMColor},
		Fn: func(frame *Frame) error {
			src, err := frame.Get(ResFinalColor)
			if err != nil {
				return err
			}
			dst := frame.Ensure(ResPPMColor, 3)
			copy(dst.Data, src.Data)
			return nil
		},
	}
}
