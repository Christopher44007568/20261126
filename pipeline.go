package main

import "path/filepath"

// BuildManifest runs the full reference-chain proof and returns the manifest
// skeleton. Image metadata is filled in by main after the PNGs are rendered.
//
// The same working tree is moved through three explicit states so every error
// string and chain is produced by the real resolver rather than hard coded:
//
//	before  - pristine fixture, all references resolve
//	after   - live material updated through the chain (contract-invalid save)
//	missing - live geometry reference re-pointed at an absent asset
func (a *App) BuildManifest() (*Manifest, error) {
	if err := a.SeedFixture(); err != nil {
		return nil, err
	}
	if err := a.ResetWork(); err != nil {
		return nil, err
	}

	liveScene := "instances/live.json"
	archiveScene := "instances/archive.json"

	before := Stage{
		Label:   "before reference",
		Live:    a.resolveScene(liveScene),
		Archive: a.resolveScene(archiveScene),
	}

	liveMatChain := before.Live.Material.Chain
	liveMatSource := before.Live.Material.Source
	liveMatFinal := before.Live.Material.FinalPath

	if err := a.commitUpdatedMaterial(); err != nil {
		return nil, err
	}

	after := Stage{
		Label:   "after reference",
		Live:    a.resolveScene(liveScene),
		Archive: a.resolveScene(archiveScene),
	}

	if _, err := a.setModelReference("models/live.json", "geometry", "../geometry/showcase_missing.json"); err != nil {
		return nil, err
	}

	missing := Stage{
		Label:   "missing asset placeholder",
		Live:    a.resolveScene(liveScene),
		Archive: a.resolveScene(archiveScene),
	}

	missingGeo := missing.Live.Geometry

	changes := []Change{
		{
			Kind:        "material-field",
			Asset:       "materials/live.json",
			ChangedPath: liveMatFinal,
			Field:       "baseColor",
			Before:      "#F06A5D",
			After:       "#6C7BFF",
			FinalPath:   after.Live.Material.FinalPath,
			Source:      liveMatSource,
			Chain:       cloneHops(liveMatChain),
		},
		{
			Kind:        "reference-change",
			Asset:       "models/live.json",
			ChangedPath: filepath.Join(a.WorkDir, "models", "live.json"),
			Field:       "references.geometry",
			Before:      "../geometry/showcase.json",
			After:       "../geometry/showcase_missing.json",
			FinalPath:   missingGeo.FinalPath,
			Source:      missingGeo.Source,
			Chain:       cloneHops(missingGeo.Chain),
		},
	}

	impact := Impact{
		OnlyReferencingPathsUpdate: true,
		UpdatedScene:               liveScene,
		UpdatedSource:              before.Live.Material.Source,
		UnchangedScene:             archiveScene,
		UnchangedSource:            before.Archive.Material.Source,
	}

	m := &Manifest{
		Fixture:      a.FixtureDir,
		WorkDir:      a.WorkDir,
		Before:       before,
		After:        after,
		Missing:      missing,
		Changes:      changes,
		Impact:       impact,
		ManifestPath: filepath.Join(a.ArtDir, "manifest.json"),
	}
	return m, nil
}

func cloneHops(in []Hop) []Hop {
	out := make([]Hop, len(in))
	copy(out, in)
	return out
}
