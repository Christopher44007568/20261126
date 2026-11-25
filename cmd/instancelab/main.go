// Command instancelab generates prototype/instance audit diagrams and runs
// the isolation, deletion and geometry checks using the Go standard library
// only.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"instancelab/internal/lab"
	"instancelab/internal/render"
)

const editColor = "#FF2E93"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "generate":
		err = generateCmd(os.Args[2:])
	case "verify":
		err = verifyCmd()
	case "serve":
		err = serveCmd(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		log.Fatalf("instancelab: %v", err)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: instancelab [generate -out DIR | verify | serve -addr ADDR]")
}

type auditBundle struct {
	Schema          string                    `json:"schema"`
	Counts          map[string]int            `json:"counts"`
	ColorIsolation  *lab.ColorIsolationReport `json:"color_isolation"`
	DeletionSafety  *lab.DeletionSafetyReport `json:"deletion_safety"`
	GeometrySync    *lab.GeometrySyncReport   `json:"geometry_sync"`
	SharedMapIssues []lab.SharedMapIssue      `json:"shared_map_issues"`
	AllPassed       bool                      `json:"all_core_checks_passed"`
}

func runAudit() (*lab.Document, *auditBundle, *lab.SharedMapFault, *lab.WritebackFault, error) {
	doc := lab.BuildScenario()
	meta := doc.Meta()

	color, err := lab.CheckColorIsolation(doc, "I-007", "body", editColor)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// Use a separate document for destructive checks so the generated diagrams
	// still show the full 128-instance scenario.
	deleteDoc := lab.BuildScenario()
	deletion, err := lab.CheckDeletionSafety(deleteDoc, "I-128", "I-001")
	if err != nil {
		return nil, nil, nil, nil, err
	}
	geometry, err := lab.CheckGeometrySync(lab.BuildScenario(), "I-001", 24, 18)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	sharedFault, err := lab.SimulateSharedMapFault("I-037", "body", editColor)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	writebackFault, err := lab.SimulateWritebackFault("I-004", "body", editColor)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	visible := 0
	for _, in := range doc.Instances() {
		if !in.Hidden {
			visible++
		}
	}
	bundle := &auditBundle{
		Schema: "instancelab.audit/v1",
		Counts: map[string]int{
			"prototypes":        len(doc.Prototypes()),
			"instances":         doc.InstanceCount(),
			"visible_instances": visible,
			"color_overrides":   meta.ColorOverrides,
			"hidden_instances":  meta.HiddenInstances,
			"hidden_nodes":      meta.HiddenNodes,
		},
		ColorIsolation:  color,
		DeletionSafety:  deletion,
		GeometrySync:    geometry,
		SharedMapIssues: lab.DetectSharedColorMaps(doc),
		AllPassed:       color.Pass && deletion.Pass && geometry.Pass,
	}
	return doc, bundle, sharedFault, writebackFault, nil
}

func generateCmd(args []string) error {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	out := fs.String("out", "artifacts/generated", "output directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		return err
	}

	doc, bundle, sharedFault, writebackFault, err := runAudit()
	if err != nil {
		return err
	}
	proto, _ := doc.Prototype(lab.NewBeaconCardPrototype().ID)

	if err := writeSVG(filepath.Join(*out, "prototype_diagram.svg"),
		func(f *os.File) error { return render.WritePrototypeDiagram(f, proto) }); err != nil {
		return err
	}
	if err := writeSVG(filepath.Join(*out, "instances_overview.svg"),
		func(f *os.File) error { return render.WriteInstancesOverview(f, doc) }); err != nil {
		return err
	}
	if err := writeSVG(filepath.Join(*out, "shared_color_error.svg"),
		func(f *os.File) error { return render.WriteSharedColorError(f, doc, sharedFault) }); err != nil {
		return err
	}
	if err := writeSVG(filepath.Join(*out, "prototype_writeback_error.svg"),
		func(f *os.File) error { return render.WriteWritebackError(f, doc, writebackFault) }); err != nil {
		return err
	}
	if err := writeSVG(filepath.Join(*out, "fixed_color_isolation.svg"),
		func(f *os.File) error { return render.WriteFixedIsolation(f, doc, bundle.ColorIsolation) }); err != nil {
		return err
	}
	if err := writeSVG(filepath.Join(*out, "bounds_outline_sync.svg"),
		func(f *os.File) error {
			return render.WriteBoundsOutlineSync(f, lab.BuildScenario(), "I-001", 24, 18)
		}); err != nil {
		return err
	}

	if err := writeJSON(filepath.Join(*out, "audit.json"), bundle); err != nil {
		return err
	}
	faults := map[string]any{
		"shared_map_fault":          sharedFault,
		"prototype_writeback_fault": writebackFault,
	}
	if err := writeJSON(filepath.Join(*out, "faults.json"), faults); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(*out, "report.md"),
		[]byte(buildReport(bundle, sharedFault, writebackFault)), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(*out, "index.html"),
		[]byte(buildIndex()), 0o644); err != nil {
		return err
	}

	fmt.Printf("generated 6 SVG diagrams + audit.json/faults.json/report.md/index.html in %s\n", *out)
	fmt.Printf("core checks passed: %v\n", bundle.AllPassed)
	return nil
}

func verifyCmd() error {
	_, bundle, sharedFault, writebackFault, err := runAudit()
	if err != nil {
		return err
	}
	data, _ := json.MarshalIndent(bundle, "", "  ")
	fmt.Println(string(data))
	if !bundle.AllPassed {
		return fmt.Errorf("core checks failed")
	}
	if len(bundle.SharedMapIssues) != 0 {
		return fmt.Errorf("unexpected shared color maps in correct scenario")
	}
	if !sharedFault.AuditDetectable {
		return fmt.Errorf("shared-map fault not detectable by audit")
	}
	if len(writebackFault.UnexpectedInstanceIDs) == 0 {
		return fmt.Errorf("writeback fault produced no collateral changes")
	}
	fmt.Println("verify: PASS (isolation, deletion, geometry, fault detection)")
	return nil
}

func serveCmd(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", "127.0.0.1:8090", "listen address")
	dir := fs.String("dir", "artifacts/generated", "directory to serve")
	if err := fs.Parse(args); err != nil {
		return err
	}
	http.Handle("/", http.FileServer(http.Dir(*dir)))
	fmt.Printf("serving %s at http://%s\n", *dir, *addr)
	return http.ListenAndServe(*addr, nil)
}

func writeSVG(path string, fn func(*os.File) error) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return fn(f)
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
