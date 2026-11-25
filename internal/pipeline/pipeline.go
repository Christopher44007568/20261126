package pipeline

import (
	"image"
	"math"

	"aelab/internal/ae"
	"aelab/internal/render"
	"aelab/internal/scene"
)

const (
	TransitionStart = scene.PanStart
	TransitionEnd   = scene.PanStart + scene.PanDuration
	targetBias      = 0.18
)

type FrameRecord struct {
	ProfileID       string
	ProfileLabel    string
	Frame           int
	TimeS           float64
	PanX            int
	MeterLum        float64
	LogLum          float64
	Percentile95    float64
	TargetEV        float64
	ActualEV        float64
	Gain            float64
	VelocityEVPerS  float64
	EVStep          float64
	TrackingErrorEV float64
	ReferenceEV     float64
	OvershootEV     float64
	CompletionPct   float64
	MeanLuma        float64
	LumaDelta       float64
	WindowMean      float64
	WindowStd       float64
	WindowEdge      float64
	WindowClipHigh  float64
	ShadowMean      float64
	ShadowStd       float64
	ShadowEdge      float64
	ShadowClipLow   float64
}

type Metrics struct {
	InitialEV            float64
	FinalTargetEV        float64
	FinalActualEV        float64
	SettleTimeS          float64
	MaxOvershootEV       float64
	MaxSignedErrorEV     float64
	MaxAbsDeviationEV    float64
	ZeroCrossings        int
	MaxEVStep            float64
	MaxLumaJump          float64
	MeanAbsLumaJump      float64
	LumaReversals        int
	WindowMean           float64
	WindowEdge           float64
	WindowClipHigh       float64
	ShadowMean           float64
	ShadowStd            float64
	ShadowEdge           float64
	PassesNoFlash        bool
	PassesSmoothConverge bool
}

type ProfileResult struct {
	Profile ae.Profile
	Records []FrameRecord
	Metrics Metrics
}

type Result struct {
	Width    int
	Height   int
	FPS      int
	Duration float64
	Profiles []ProfileResult
}

type FrameSink func(profileID string, frameNumber int, img *image.RGBA) error

func Run(world *scene.HDR, sink FrameSink) (*Result, error) {
	controllers := make([]*ae.Controller, len(ae.Profiles))
	results := make([]ProfileResult, len(ae.Profiles))
	for i, profile := range ae.Profiles {
		controllers[i] = ae.NewController(profile)
		results[i] = ProfileResult{Profile: profile, Records: make([]FrameRecord, 0, scene.FrameCount)}
	}

	initialFrame := scene.CameraFrame(world, 0)
	initialMeter := ae.Measure(initialFrame)
	initialEV := ae.TargetEV(initialMeter) - targetBias
	for _, controller := range controllers {
		controller.Reset(initialEV)
	}

	dt := 1.0 / float64(scene.FPS)
	for frame := 0; frame < scene.FrameCount; frame++ {
		timeS := float64(frame) / float64(scene.FPS)
		hdrFrame := scene.CameraFrame(world, timeS)
		metering := ae.Measure(hdrFrame)
		targetEV := ae.TargetEV(metering) - targetBias

		for i, controller := range controllers {
			previousEV := controller.EV
			actualEV := controller.Update(targetEV, dt)
			img := render.ToRGBA(hdrFrame, ae.Gain(actualEV))
			if sink != nil {
				if err := sink(results[i].Profile.ID, frame+1, img); err != nil {
					return nil, err
				}
			}

			meanLuma := render.MeanLuma(img)
			windowStats := render.AnalyzeROI(img, image.Rect(scene.WindowROI.X0, scene.WindowROI.Y0, scene.WindowROI.X1, scene.WindowROI.Y1))
			shadowStats := render.AnalyzeROI(img, image.Rect(scene.ShadowROI.X0, scene.ShadowROI.Y0, scene.ShadowROI.X1, scene.ShadowROI.Y1))
			lumaDelta := 0.0
			if len(results[i].Records) > 0 {
				lumaDelta = meanLuma - results[i].Records[len(results[i].Records)-1].MeanLuma
			}

			record := FrameRecord{
				ProfileID: results[i].Profile.ID, ProfileLabel: results[i].Profile.Label,
				Frame: frame + 1, TimeS: timeS, PanX: scene.PanX(timeS),
				MeterLum: metering.GeometricLum, LogLum: math.Log(math.Max(metering.GeometricLum, 1e-6)), Percentile95: metering.Percentile95,
				TargetEV: targetEV, ActualEV: actualEV, Gain: ae.Gain(actualEV),
				VelocityEVPerS: controller.Velocity, EVStep: actualEV - previousEV,
				TrackingErrorEV: actualEV - targetEV, MeanLuma: meanLuma, LumaDelta: lumaDelta,
				WindowMean: windowStats.Mean, WindowStd: windowStats.StdDev, WindowEdge: windowStats.EdgeEnergy, WindowClipHigh: windowStats.ClipHigh,
				ShadowMean: shadowStats.Mean, ShadowStd: shadowStats.StdDev, ShadowEdge: shadowStats.EdgeEnergy, ShadowClipLow: shadowStats.ClipLow,
			}
			results[i].Records = append(results[i].Records, record)
		}
	}

	finalTarget := results[0].Records[len(results[0].Records)-1].TargetEV
	for i := range results {
		finalizeRecords(results[i].Records, initialEV, finalTarget)
		results[i].Metrics = calculateMetrics(results[i], initialEV, finalTarget)
	}

	return &Result{Width: scene.FrameWidth, Height: scene.FrameHeight, FPS: scene.FPS, Duration: scene.Duration, Profiles: results}, nil
}

func finalizeRecords(records []FrameRecord, initialEV, finalTarget float64) {
	span := initialEV - finalTarget
	for i := range records {
		r := &records[i]
		r.ReferenceEV = finalTarget
		r.OvershootEV = finalTarget - r.ActualEV
		if math.Abs(span) > 1e-6 {
			r.CompletionPct = (initialEV - r.ActualEV) / span * 100
		}
	}
}
