package main

// ImageInfo describes a rendered proof image.
type ImageInfo struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

// Hop is one edge of a resolved reference chain.
type Hop struct {
	From      string `json:"from"`
	Slot      string `json:"slot"`
	Reference string `json:"reference"`
	To        string `json:"to"`
	Exists    bool   `json:"exists"`
}

// SlotReport is the fully resolved provenance for one material/geometry/camera
// reference, including any resolution/validation error.
type SlotReport struct {
	Slot        string `json:"slot"`
	Reference   string `json:"reference"`
	Kind        string `json:"kind,omitempty"`
	FinalPath   string `json:"finalPath"`
	DisplayPath string `json:"displayPath"`
	Source      string `json:"source"`
	Chain       []Hop  `json:"chain"`
	Error       string `json:"error,omitempty"`
}

// SceneReport carries both the file locations and the three resolved slots
// for one instance scene.
type SceneReport struct {
	Name         string     `json:"name"`
	Root         string     `json:"root"`
	ScenePath    string     `json:"scenePath"`
	SceneDisplay string     `json:"sceneDisplay"`
	ModelPath    string     `json:"modelPath"`
	ModelDisplay string     `json:"modelDisplay"`
	Material     SlotReport `json:"material"`
	Geometry     SlotReport `json:"geometry"`
	Camera       SlotReport `json:"camera"`
}

// Stage is one rendered proof board plus its two scene reports.
type Stage struct {
	Label   string      `json:"label"`
	Image   ImageInfo   `json:"image"`
	Live    SceneReport `json:"live"`
	Archive SceneReport `json:"archive"`
}

// Change is a single edit and the final resolved path it lands on.
type Change struct {
	Kind        string `json:"kind"`
	Asset       string `json:"asset"`
	ChangedPath string `json:"changedPath"`
	Field       string `json:"field"`
	Before      string `json:"before"`
	After       string `json:"after"`
	FinalPath   string `json:"finalPath"`
	Source      string `json:"source"`
	Chain       []Hop  `json:"chain"`
}

// Impact states the answer to the reference-locality question.
type Impact struct {
	OnlyReferencingPathsUpdate bool   `json:"onlyReferencingPathsUpdate"`
	UpdatedScene               string `json:"updatedScene"`
	UpdatedSource              string `json:"updatedSource"`
	UnchangedScene             string `json:"unchangedScene"`
	UnchangedSource            string `json:"unchangedSource"`
}

// Manifest is the machine readable record emitted next to the three images.
type Manifest struct {
	Fixture      string   `json:"fixture"`
	WorkDir      string   `json:"workDir"`
	Before       Stage    `json:"before"`
	After        Stage    `json:"after"`
	Missing      Stage    `json:"missing"`
	Changes      []Change `json:"changes"`
	Impact       Impact   `json:"impact"`
	ManifestPath string   `json:"manifestPath"`
}

// refDoc is the small subset of a showroom JSON asset that the resolver reads.
type refDoc struct {
	Kind       string            `json:"kind"`
	Name       string            `json:"name"`
	References map[string]string `json:"references"`
}
