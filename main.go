package main

import (
	"flag"
	"fmt"
	"os"

	"bitgradient/internal/render"
)

func main() {
	out := flag.String("out", "artifacts", "output directory")
	flag.Parse()

	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "create output directory:", err)
		os.Exit(1)
	}
	files, err := render.Generate(*out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "generate:", err)
		os.Exit(1)
	}
	for _, name := range files {
		fmt.Println(name)
	}
}
