# 20261126 — Scene Flattening Demo

A zero-dependency Go program that renders a layered scene graph,
flattens it into a self-contained scene, and proves the flattened
scene re-renders to the identical image.

## What it does

1. Renders the original hierarchical scene (layers, groups, nested
   parent/child transforms, hidden nodes, layer opacity, material
   overrides) to `out/original.png`.
2. Flattens the scene: transforms are baked into world-space geometry,
   material overrides are resolved into final materials, layer opacity
   is folded into alpha, and hidden shapes are kept with `Hidden=true`
   (nothing is dropped). The flat scene is saved to `out/flattened.json`.
3. Reloads the JSON and renders it independently to `out/flattened.png`.
4. Writes a per-pixel difference image to `out/diff.png` and fails if
   any pixel differs.

## Run

```sh
go run .          # writes out/{original,flattened,diff}.png + flattened.json
go test ./...     # locks the flatten invariants
```

Only the Go standard library is used (`image`, `image/png`,
`encoding/json`, `math`).
