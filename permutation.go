package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

type permutationResult struct {
	report        []byte
	forwardMerged []byte
	forwardPPM    []byte
	reverseMerged []byte
	reversePPM    []byte
	scene         *Scene
	allIdentical  bool
}

func runPermutationCheck(files []string) (*permutationResult, error) {
	refMerged, refPPM, scene, err := composeOrder(files)
	if err != nil {
		return nil, err
	}
	refMergedHash := hashBytes(refMerged)
	refPPMHash := hashBytes(refPPM)

	reverse := append([]string(nil), files...)
	for i, j := 0, len(reverse)-1; i < j; i, j = i+1, j-1 {
		reverse[i], reverse[j] = reverse[j], reverse[i]
	}
	reverseMerged, reversePPM, _, err := composeOrder(reverse)
	if err != nil {
		return nil, err
	}
	result := &permutationResult{
		forwardMerged: refMerged,
		forwardPPM:    refPPM,
		reverseMerged: reverseMerged,
		reversePPM:    reversePPM,
		scene:         scene,
		allIdentical:  true,
	}

	var b strings.Builder
	b.WriteString("LOAD-ORDER PERMUTATION CHECK\n")
	b.WriteString("==================================================\n")
	fmt.Fprintf(&b, "reference merged sha256: %s\n", refMergedHash)
	fmt.Fprintf(&b, "reference ppm    sha256: %s\n\n", refPPMHash)

	orders := permutations(files)
	for i, order := range orders {
		merged, ppm, _, err := composeOrder(order)
		if err != nil {
			return nil, err
		}
		mergedHash := hashBytes(merged)
		ppmHash := hashBytes(ppm)
		identical := mergedHash == refMergedHash && ppmHash == refPPMHash
		if !identical {
			result.allIdentical = false
		}
		status := "DIFFERENT"
		if identical {
			status = "IDENTICAL"
		}
		fmt.Fprintf(&b, "%02d [%s] load=%s merged=%s ppm=%s\n",
			i+1, status, orderNames(order), mergedHash[:12], ppmHash[:12])
	}
	fmt.Fprintf(&b, "\nRESULT: %s\n", permutationVerdict(result.allIdentical, len(orders)))
	result.report = []byte(b.String())
	return result, nil
}

func composeOrder(files []string) ([]byte, []byte, *Scene, error) {
	layers, err := LoadLayerFiles(files)
	if err != nil {
		return nil, nil, nil, err
	}
	scene := Compose(layers)
	return RenderMergedScene(scene), RenderScene(scene).PPM(), scene, nil
}

func permutations(items []string) [][]string {
	var out [][]string
	var walk func(int)
	work := append([]string(nil), items...)
	walk = func(index int) {
		if index == len(work) {
			out = append(out, append([]string(nil), work...))
			return
		}
		for i := index; i < len(work); i++ {
			work[index], work[i] = work[i], work[index]
			walk(index + 1)
			work[index], work[i] = work[i], work[index]
		}
	}
	walk(0)
	return out
}

func orderNames(files []string) string {
	names := make([]string, len(files))
	for i, file := range files {
		if idx := strings.LastIndex(file, "/"); idx >= 0 {
			names[i] = file[idx+1:]
		} else {
			names[i] = file
		}
	}
	return strings.Join(names, " -> ")
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func permutationVerdict(identical bool, count int) string {
	if identical {
		return fmt.Sprintf("all %d load orders produce the same merged scene and the same PPM bytes.", count)
	}
	return "at least one load order changed the merged scene or PPM bytes."
}
