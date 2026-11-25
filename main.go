package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"dds-preview/internal/preview"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	outDir := "preview"
	outExplicit := false
	quiet := false
	var inputs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-out" || arg == "--out":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "dds-preview: -out requires a directory")
				return 2
			}
			i++
			outDir = args[i]
			outExplicit = true
		case strings.HasPrefix(arg, "-out="):
			outDir = strings.TrimPrefix(arg, "-out=")
			outExplicit = true
		case strings.HasPrefix(arg, "--out="):
			outDir = strings.TrimPrefix(arg, "--out=")
			outExplicit = true
		case arg == "-q" || arg == "--quiet":
			quiet = true
		case arg == "-h" || arg == "--help":
			usage(stdout)
			return 0
		case strings.HasPrefix(arg, "-"):
			fmt.Fprintf(stderr, "dds-preview: unknown option %q\n", arg)
			return 2
		default:
			inputs = append(inputs, arg)
		}
	}
	outDir = strings.TrimSpace(outDir)
	if outDir == "" {
		fmt.Fprintln(stderr, "dds-preview: output directory is empty")
		return 2
	}
	if len(inputs) == 2 && !outExplicit && !looksLikeDDS(inputs[1]) {
		outDir = inputs[1]
		inputs = inputs[:1]
	}
	if len(inputs) == 0 {
		usage(stderr)
		return 2
	}

	p := preview.New()
	failed := false
	for _, input := range inputs {
		res := p.LoadFile(input)
		if res.OK {
			tex, _ := p.Current()
			if err := preview.WritePreview(tex, res, outDir); err != nil {
				res.OK = false
				res.OutputError = err.Error()
				fmt.Fprintf(stderr, "dds-preview: output failed: %v\n", err)
				failed = true
			}
		}
		if err := preview.WriteReport(res, outDir); err != nil {
			fmt.Fprintf(stderr, "dds-preview: cannot write report: %v\n", err)
			failed = true
		}
		if res.OK {
			if !quiet {
				printSuccess(stdout, res)
			}
		} else if !quiet || len(res.Errors) > 0 {
			printFailure(stderr, res)
			failed = true
		}
	}
	if failed {
		return 1
	}
	return 0
}

func looksLikeDDS(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".dds")
}

func printSuccess(w io.Writer, res *preview.Result) {
	if res.Current == nil {
		return
	}
	s := res.Current
	fmt.Fprintf(w, "%s: %dx%d, %d face(s), %d mip(s), %s\n",
		res.Source, s.Width, s.Height, s.FaceCount, s.MipCount, s.Format)
	fmt.Fprintf(w, "  wrote %d preview image(s) and report.json\n", len(res.Artifacts))
}

func printFailure(w io.Writer, res *preview.Result) {
	if res.OutputError != "" {
		fmt.Fprintf(w, "%s: preview output failed\n", res.Source)
	} else {
		fmt.Fprintf(w, "%s: invalid DDS; no texture committed\n", res.Source)
	}
	if res.KeptPrevious {
		fmt.Fprintf(w, "  kept previous texture from %s\n", res.PreviousSource)
	}
	for _, e := range res.Errors {
		offset := ""
		if e.Offset >= 0 {
			offset = fmt.Sprintf(" @%d", e.Offset)
		}
		fmt.Fprintf(w, "  %s%s: %s", e.Field, offset, e.Message)
		if e.Expected != "" || e.Actual != "" {
			fmt.Fprintf(w, " (expected %s, got %s)", e.Expected, e.Actual)
		}
		fmt.Fprintln(w)
	}
	if res.OutputError != "" {
		fmt.Fprintf(w, "  output: %s\n", res.OutputError)
	}
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "usage: dds-preview [-out DIR] <file.dds> [more.dds ...]")
	fmt.Fprintln(w, "       dds-preview <file.dds> <output-dir>")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Outputs one PPM per face/mip, an original-size sphere.ppm,")
	fmt.Fprintln(w, "and report.json with field-level diagnostics.")
}
