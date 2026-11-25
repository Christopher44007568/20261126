package main

import (
	"fmt"
	"os"

	"mipwrap/internal/cli"
)

func main() {
	if err := cli.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "mipwrap: %v\n", err)
		os.Exit(1)
	}
}
