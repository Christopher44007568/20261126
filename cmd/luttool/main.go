// Command luttool parses custom .cube 3D LUTs and renders comparison
// sheets: identity, channel-swap and non-uniform-domain groups, plus
// interpolation slice grids. Standard library only.
//
// Usage:
//
//	luttool demo [outdir]               render the three comparison groups
//	luttool apply <in.cube> <out.png>   apply one LUT to the test chart
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "demo":
		dir := "out"
		if len(os.Args) > 2 {
			dir = os.Args[2]
		}
		err = runDemo(dir)
	case "apply":
		if len(os.Args) != 4 {
			usage()
			os.Exit(2)
		}
		err = runApply(os.Args[2], os.Args[3])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "luttool:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage:
  luttool demo [outdir]               render comparison groups + slices
  luttool apply <in.cube> <out.png>   apply one LUT to the test chart
On a format error, apply reports the file line and keeps the previous
out.png untouched.`)
}
