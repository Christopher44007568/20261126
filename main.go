package main

import (
	"fmt"
	"os"
)

func main() {
	outDir := "artifacts"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}

	report, err := run(outDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "varianttree: %v\n", err)
		if report != nil {
			for _, check := range report.Checks {
				status := "PASS"
				if !check.Pass {
					status = "FAIL"
				}
				fmt.Fprintf(os.Stderr, "[%s] %s — %s\n", status, check.Name, check.Detail)
			}
		}
		os.Exit(1)
	}

	fmt.Println("varianttree: all variant propagation and pixel-restore checks passed")
	for _, check := range report.Checks {
		fmt.Printf("[PASS] %s — %s\n", check.Name, check.Detail)
	}
	fmt.Printf("artifacts written to %s\n", report.OutputDirectory)
	fmt.Println("contact sheet: " + report.OutputDirectory + "/variant_contact_sheet.ppm")
	fmt.Println("selection path: " + report.OutputDirectory + "/current_selection_path.txt")
}
