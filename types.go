package main

type Vec3 struct {
	X float64
	Y float64
	Z float64
}

type RGB struct {
	R float64
	G float64
	B float64
}

type Layer struct {
	Name          string
	Min           Vec3
	Max           Vec3
	Transmittance RGB
	Tint          RGB
}

type Ray struct {
	Origin    Vec3
	Direction Vec3
}

func (r Ray) At(t float64) Vec3 {
	return Vec3{
		X: r.Origin.X + r.Direction.X*t,
		Y: r.Origin.Y + r.Direction.Y*t,
		Z: r.Origin.Z + r.Direction.Z*t,
	}
}

type Segment struct {
	Start  Vec3
	End    Vec3
	T0     float64
	T1     float64
	Reason string
}

type Hit struct {
	LayerIndex    int
	LayerName     string
	Entry         Vec3
	Exit          Vec3
	T0            float64
	T1            float64
	Transmittance RGB
	EnergyBefore  RGB
	EnergyAfter   RGB
	OutOfRange    bool
}

type TraceResult struct {
	Hits            []Hit
	Energy          RGB
	Complete        bool
	Unprocessed     *Segment
	RemainingLayers []string
}
