package main

// seedShowroom returns the pristine "showroom" reference dataset.
//
// It is the canonical fixture that gets materialised under fixtures/showroom
// and copied to the working tree artifacts/work/showroom before every proof
// run. Keeping it in code makes `go run .` self-contained with zero external
// dependencies and no requirement that the JSON files already exist on disk.
func seedShowroom() map[string]string {
	return map[string]string{
		"instances/live.json": `{
  "kind": "scene",
  "name": "Showroom / Referenced",
  "references": {
    "model": "../models/live.json"
  }
}
`,
		"instances/archive.json": `{
  "kind": "scene",
  "name": "Showroom / Independent Copy",
  "references": {
    "model": "../models/archive.json"
  }
}
`,
		"models/live.json": `{
  "kind": "model",
  "name": "Orbit Display",
  "references": {
    "material": "../materials/live.json",
    "geometry": "../geometry/showcase.json",
    "camera": "../cameras/hero.json"
  }
}
`,
		"models/archive.json": `{
  "kind": "model",
  "name": "Orbit Display",
  "references": {
    "material": "../materials/archive.json",
    "geometry": "../geometry/showcase.json",
    "camera": "../cameras/hero.json"
  }
}
`,
		"materials/live.json": `{
  "kind": "material",
  "name": "Coral Dusk",
  "baseColor": "#F06A5D",
  "accentColor": "#FFB547",
  "inkColor": "#10212B"
}
`,
		"materials/archive.json": `{
  "kind": "material",
  "name": "Coral Dusk",
  "baseColor": "#F06A5D",
  "accentColor": "#FFB547",
  "inkColor": "#10212B"
}
`,
		"geometry/showcase.json": `{
  "kind": "geometry",
  "name": "Orbit Showcase",
  "panelWidth": 292,
  "panelHeight": 166,
  "cornerRadius": 30
}
`,
		"cameras/hero.json": `{
  "kind": "camera",
  "name": "Hero 35",
  "zoom": 1.08,
  "offsetX": 0,
  "offsetY": -8
}
`,
	}
}
