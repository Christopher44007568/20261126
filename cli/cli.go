// Package cli implements the texcontainer command-line interface.
package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"texturelab/texcontainer"
	"texturelab/workflow"
)

// Run executes the CLI and returns a process-style exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stdout)
		return 0
	}
	switch args[0] {
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0
	case "demo":
		return runDemo(args[1:], stdout, stderr)
	case "inspect":
		return runInspect(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, `texcontainer - KTX-style texture mip container tool

Usage:
  texcontainer demo [-out artifacts/generated]
  texcontainer inspect [--json] <file.ktxc>

Commands:
  demo     Generate 512/256/128 albedo and normal mip chains, atlases,
           metadata tables, round-trip proof and sRGB mislabel diagnostic.
  inspect  Parse and print a TXCON1 container header and level index.

The implementation uses only Go and its standard image/PNG support; it does
not link OpenGL, Vulkan, DirectX, Metal or another graphics SDK.`)
}

func runDemo(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("demo", flag.ContinueOnError)
	flags.SetOutput(stderr)
	outDir := flags.String("out", "artifacts/generated", "directory for generated containers and reports")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	result, err := workflow.Run(*outDir)
	if err != nil {
		fmt.Fprintf(stderr, "demo failed: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "generated texture pipeline in %s\n", result.OutDir)
	fmt.Fprintf(stdout, "mip metadata: %s, %s, %s\n",
		result.Files.MetadataCSV, result.Files.MetadataJSON, result.Files.MetadataMarkdown)
	fmt.Fprintf(stdout, "reload color shift: none (no_color_shift=%t)\n", result.Roundtrip.NoColorShift)
	if result.Diagnostic != nil {
		fmt.Fprintf(stdout, "normal/sRGB mismatch: detected=%t first_level=%d worst_level=%d\n",
			result.Diagnostic.Mismatch, result.Diagnostic.FirstFailingLevel, result.Diagnostic.WorstLevel)
		if locationErr := result.Diagnostic.LocateFirstMismatch(); locationErr != nil {
			fmt.Fprintf(stdout, "location: %s\n", locationErr.Error())
		}
		fmt.Fprintf(stdout, "diagnostic report: %s\n", result.Files.DiagnosticMarkdown)
	}
	return 0
}

func runInspect(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("inspect", flag.ContinueOnError)
	flags.SetOutput(stderr)
	asJSON := flags.Bool("json", false, "print container info as JSON")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: texcontainer inspect [--json] <file.ktxc>")
		return 2
	}

	data, err := os.ReadFile(flags.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "inspect failed: %v\n", err)
		return 1
	}
	info, err := texcontainer.Inspect(data)
	if err != nil {
		fmt.Fprintf(stderr, "inspect failed: %v\n", err)
		return 1
	}

	if *asJSON {
		encoded, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "encode info failed: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, string(encoded))
		return 0
	}

	fmt.Fprintf(stdout, "TXCON1 version=%d format=%d color_space=%s base=%dx%d levels=%d file=%d bytes\n",
		info.Version, info.Format, info.ColorSpace, info.BaseWidth, info.BaseHeight,
		info.LevelCount, info.FileSize)
	for _, level := range info.Levels {
		fmt.Fprintf(stdout, "  level=%d size=%dx%d offset=%d length=%d crc32=%s\n",
			level.Level, level.Width, level.Height, level.AbsoluteOffset, level.Size, level.CRC32)
	}
	return 0
}
