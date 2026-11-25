package main

import (
	"flag"
	"fmt"
	"image"
	"log"
	"path/filepath"
)

type buildOutput struct {
	Audit        Audit
	ContactSheet string
	Images       []string
	Provenance   []string
}

func buildArtifacts(outDir string) (buildOutput, error) {
	scene := demoScene()
	output := buildOutput{ContactSheet: filepath.Join(outDir, "contact_sheet.png")}
	winnersByMode := make(map[string][]PixelWinner, len(RenderModes))
	images := make([]*image.RGBA, 0, len(RenderModes))

	for _, mode := range RenderModes {
		winners := resolveScene(scene, mode)
		winnersByMode[mode.ID] = winners
		img := renderImage(scene, winners)
		images = append(images, img)

		imagePath := filepath.Join(outDir, mode.ImageFile)
		if err := writePNG(imagePath, img); err != nil {
			return buildOutput{}, err
		}
		output.Images = append(output.Images, imagePath)

		provenancePath := filepath.Join(outDir, mode.ProvenanceFile)
		report := buildProvenance(scene, mode, winners)
		if err := writeJSON(provenancePath, report); err != nil {
			return buildOutput{}, err
		}
		output.Provenance = append(output.Provenance, provenancePath)
	}

	if err := writePNG(output.ContactSheet, contactSheet(images)); err != nil {
		return buildOutput{}, err
	}

	reference, ok := modeByID("after_override")
	if !ok {
		return buildOutput{}, fmt.Errorf("reference mode after_override is missing")
	}
	audit := auditRenders(reference, winnersByMode[reference.ID], RenderModes, winnersByMode)
	if err := validateAudit(audit); err != nil {
		return buildOutput{}, err
	}
	if err := writeJSON(filepath.Join(outDir, "audit.json"), audit); err != nil {
		return buildOutput{}, err
	}
	output.Audit = audit
	return output, nil
}

func main() {
	outDir := flag.String("out", "out", "directory for PNG renders, provenance, and audit")
	flag.Parse()

	output, err := buildArtifacts(*outDir)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("wrote %d renders, %d provenance grids, and 1 audit to %s\n", len(output.Images), len(output.Provenance), *outDir)
	fmt.Printf("contact sheet: %s\n", output.ContactSheet)
	for _, mode := range output.Audit.Modes {
		fmt.Printf("%-20s wrong-color pixels: %d\n", mode.Mode, mode.WrongColorPixels)
	}
}
