# 20261126

Repository initialized for 20261126.

## Transparent multi-hit query

This Go program traces a ray through three axis-aligned transparent slabs in order:
glass, colored water, and mist. Each slab counts as one hit. Every hit stores
entry and exit positions, transmittance, energy before the hit, accumulated
energy after the hit, and an out-of-range flag.

When the hit budget is smaller than the number of intersecting layers, the
query does not return a background color early. It returns the processed hits,
marks `Complete=false`, and keeps an explicit unprocessed segment covering the
remaining ray interval, including the names of remaining layers.

Run the demo:

```sh
go run .
```

By default it writes full-budget, two-hit-budget, and invalid-energy examples
to `artifacts/render`. Each example produces:

- `*_side.png`: side view with layer geometry, ray hits, and remaining energy.
- `*_camera.png`: camera view with per-hit energy and unprocessed-segment overlay.
- `*_layer_order.png`: ordered layer diagram, including budget truncation.
- `*_transmission.png`: cumulative RGB transmission chart.
- `*_range_mask.png`: red mask for NaN, infinity, negative, or greater-than-one energy.

Use a custom budget or output directory:

```sh
go run . -budget 1 -out artifacts/custom
```

Validate the implementation:

```sh
go test ./...
go vet ./...
```

The implementation uses only the Go standard library.
