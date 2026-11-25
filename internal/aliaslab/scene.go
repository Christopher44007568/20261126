package aliaslab

const (
	ResSky    ResourceID = "sky"
	ResSun    ResourceID = "sun"
	ResClouds ResourceID = "clouds"
	ResLit    ResourceID = "lit"
	ResGraded ResourceID = "graded"
	ResFinal  ResourceID = "final"
)

// NewShowcase builds the six-pass frame graph used by all three executions.
func NewShowcase(width, height int) (*Graph, error) {
	resources := []Resource{
		{ID: ResSky, Width: width, Height: height, Format: "RGB8"},
		{ID: ResSun, Width: width, Height: height, Format: "RGB8"},
		{ID: ResClouds, Width: width, Height: height, Format: "RGB8"},
		{ID: ResLit, Width: width, Height: height, Format: "RGB8"},
		{ID: ResGraded, Width: width, Height: height, Format: "RGB8"},
		{ID: ResFinal, Width: width, Height: height, Format: "RGB8"},
	}
	passes := []Pass{
		{
			ID: "sky-fill", Name: "Sky gradient",
			Writes: []ResourceID{ResSky},
			Apply: func(map[ResourceID]*Image) *Image {
				return skyFill(width, height)
			},
		},
		{
			ID: "sun-glow", Name: "Sun and halo",
			Reads: []ResourceID{ResSky}, Writes: []ResourceID{ResSun},
			Apply: sunGlow,
		},
		{
			ID: "clouds", Name: "Cloud composite",
			Reads: []ResourceID{ResSky, ResSun}, Writes: []ResourceID{ResClouds},
			Apply: cloudComposite,
		},
		{
			ID: "atmosphere", Name: "Atmosphere light",
			Reads: []ResourceID{ResClouds}, Writes: []ResourceID{ResLit},
			Apply: atmosphereLight,
		},
		{
			ID: "grade", Name: "Color grade",
			Reads: []ResourceID{ResLit}, Writes: []ResourceID{ResGraded},
			Apply: colorGrade,
		},
		{
			ID: "vignette", Name: "Vignette",
			Reads: []ResourceID{ResGraded}, Writes: []ResourceID{ResFinal},
			Apply: vignette,
		},
	}
	return NewGraph(width, height, resources, passes)
}
