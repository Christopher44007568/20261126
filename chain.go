package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// slotDir maps a reference slot to the working-tree directory that owns it.
var slotDir = map[string]string{
	"material": "materials",
	"geometry": "geometry",
	"camera":   "cameras",
}

// resolveScene resolves the model reference and the three asset slots for one
// instance scene. It never stops at the first error: every slot gets a report
// so the board can show path provenance together with a readable failure.
func (a *App) resolveScene(sceneDisplay string) SceneReport {
	sceneAbs := a.workPath(sceneDisplay)
	sceneDoc, _, sceneErr := readDoc(sceneAbs)

	rep := SceneReport{
		Name:         sceneDoc.Name,
		Root:         a.WorkDir,
		ScenePath:    sceneAbs,
		SceneDisplay: filepath.ToSlash(sceneDisplay),
	}

	modelRef := ""
	if sceneDoc.References != nil {
		modelRef = sceneDoc.References["model"]
	}
	modelAbs := resolveRef(sceneAbs, modelRef)
	modelDisplay := a.display(modelAbs)
	rep.ModelPath = modelAbs
	rep.ModelDisplay = modelDisplay

	modelExists := fileExists(modelAbs)
	modelHop := Hop{
		From:      rep.SceneDisplay,
		Slot:      "model",
		Reference: modelRef,
		To:        modelDisplay,
		Exists:    modelExists,
	}

	modelDoc := refDoc{}
	if modelExists {
		modelDoc, _, _ = readDoc(modelAbs)
	}

	rep.Material = a.resolveSlot("material", modelAbs, modelDoc, modelHop)
	rep.Geometry = a.resolveSlot("geometry", modelAbs, modelDoc, modelHop)
	rep.Camera = a.resolveSlot("camera", modelAbs, modelDoc, modelHop)

	if sceneErr != nil && rep.Name == "" {
		rep.Name = "(unreadable scene)"
	}
	return rep
}

func (a *App) resolveSlot(slot string, modelAbs string, modelDoc refDoc, modelHop Hop) SlotReport {
	ref := ""
	if modelDoc.References != nil {
		ref = modelDoc.References[slot]
	}
	targetAbs := resolveRef(modelAbs, ref)
	targetDisplay := a.display(targetAbs)

	hop := Hop{
		From:      a.display(modelAbs),
		Slot:      slot,
		Reference: ref,
		To:        targetDisplay,
		Exists:    fileExists(targetAbs),
	}
	chain := []Hop{modelHop, hop}
	source := modelHop.To
	if source != "" {
		source = modelHop.From + " -> " + modelHop.To + " -> " + hop.To
	} else {
		source = modelHop.From + " -> " + hop.To
	}

	slotRep := SlotReport{
		Slot:        slot,
		Reference:   ref,
		FinalPath:   targetAbs,
		DisplayPath: targetDisplay,
		Source:      source,
		Chain:       chain,
	}

	body, err := os.ReadFile(targetAbs)
	if err != nil {
		slotRep.Error = fmt.Sprintf("read reference %q: %v", targetAbs, err)
		return slotRep
	}
	var doc refDoc
	if err := json.Unmarshal(body, &doc); err != nil {
		slotRep.Error = fmt.Sprintf("parse reference %q: %v", targetAbs, err)
		return slotRep
	}
	if strings.TrimSpace(doc.Kind) == "" {
		slotRep.Error = fmt.Sprintf("reference %q has no kind", targetAbs)
		return slotRep
	}
	slotRep.Kind = doc.Kind
	return slotRep
}

// resolveRef resolves a ".." relative reference against the file that declares
// it (not the process working directory).
func resolveRef(declaringFile, ref string) string {
	if ref == "" {
		return ""
	}
	if filepath.IsAbs(ref) {
		return filepath.Clean(ref)
	}
	return filepath.Clean(filepath.Join(filepath.Dir(declaringFile), filepath.FromSlash(ref)))
}

func (a *App) display(abs string) string {
	rel, err := filepath.Rel(a.WorkDir, abs)
	if err != nil {
		return filepath.ToSlash(abs)
	}
	return filepath.ToSlash(rel)
}

func fileExists(p string) bool {
	if p == "" {
		return false
	}
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}
