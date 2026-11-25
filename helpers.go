package main

import (
	"fmt"
	"strings"

	"texpath/internal/texresolver"
	"texpath/internal/texturegen"
)

func texturegenEnsure() error {
	return texturegen.Ensure(".")
}

func checkerPNG() ([]byte, error) {
	return texturegen.CheckerboardPNG()
}

func describeOrder(roots []texresolver.Root) string {
	parts := make([]string, 0, len(roots))
	for i, root := range roots {
		parts = append(parts, fmt.Sprintf("%d:%s", i+1, root.Path))
	}
	return strings.Join(parts, " -> ")
}
