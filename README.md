# 20261126

Repository initialized for 20261126.

## displaysim

Pure-Go (standard library only) simulation of three monitors that differ in
brightness, white point and color gamut, rendering one shared scene so it
stays acceptable on every device rather than only on one.

### The three panels (left / middle / right)

| Panel | Gamut | White point | Peak | Transfer |
| --- | --- | --- | --- | --- |
| SDR wide gamut | Display P3 | D65 | 200 nits | sRGB EOTF |
| PQ HDR | Rec.2020 | D93 | 1000 nits | SMPTE ST 2084 (PQ) |
| Low brightness | sRGB | 5000 K | 80 nits | sRGB EOTF |

### Pipeline

1. One master scene is rendered once in absolute CIE XYZ (`Y` in cd/m^2).
2. A single shared exposure (`-ev`, in EV stops) is applied once; the same
   exposed buffer is handed to every display. Switching displays never
   rebuilds or re-exposes the scene.
3. Per display: Bradford (von Kries) white-point adaptation to its own white,
   XYZ -> display-linear RGB via gamut primaries, extended-Reinhard tone
   mapping against display peak, then the native sRGB or PQ transfer
   function.
4. Every panel is soft-proofed back to an sRGB/D65 canvas so all three can be
   compared side by side, with a common 200-nit reference and HDR shoulder.

### Clipping mask

Pixels that cannot be represented are flagged from two independent causes:
out-of-gamut negative channels, and residual over-range after tone mapping.
Each panel shows the mask as a magenta overlay and as a dedicated
`CLIP MASK` inset, plus the measured clip percentage in its footer. The dim
display clips the most and the HDR display the least.

### Run

```sh
go run . -out artifacts/displays.png
go run . -ev 1 -out artifacts/brighter.png
go test ./...
```

No third-party packages are imported; even the 5x7 bitmap font is local.
