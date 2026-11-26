# 20261126

Repository initialized for 20261126.

## Software Visibility Buffer

The Go package in this repository implements a CPU visibility-buffer pass using
only the Go standard library. The first pass stores only a triangle ID and
depth at every MSAA sample. Material IDs, instance data, and normals are
reconstructed later through a lookup from triangle ID, so the high-frequency
visibility buffer does not need material or normal channels.

### Pipeline

1. `Render` rasterizes triangles into `VisibilityBuffer.Samples`.
2. Each covered sample performs an alpha coverage test when `Coverage` is set.
3. Surviving samples use per-sample depth testing; triangle submission order
   does not determine the result. Equal depth has a deterministic lower-ID
   tie break.
4. `BuildFramebuffer` resolves every pixel and produces:
   - `GeometryID`: nearest visible triangle ID.
   - `InstanceID`: instance recovered from that triangle ID.
   - `MaterialID`: material recovered from that triangle ID.
   - `Normal`: normalized normal interpolated at the selected sample.
   - `EdgeDefect`: `1` when MSAA samples in a pixel do not all contain the
     same triangle ID, including covered/empty edge pixels; otherwise `0`.

### Coordinates And Transparency

Positions use screen-space pixel coordinates. Pixel `(x, y)` covers the
continuous square from `(x, y)` to `(x+1, y+1)`, and `Position.Z` is the
depth compared with smaller values winning.

`Coverage` models alpha-tested transparent leaves. Interpolated coverage below
`AlphaThreshold` discards that sample before depth testing; passing samples
remain opaque and participate normally in depth testing. Transparent blending
is intentionally outside the single-visibility-sample model.

Triangle IDs must be globally unique and nonzero. Instance and material IDs may
be reused by multiple triangles because all reconstruction starts from the
triangle ID.

Run the tests with:

```sh
go test ./...
```
