# 20261126

Repository initialized for 20261126.

## luttool — custom 3D LUT viewer (Go, stdlib only)

Parses Adobe `.cube` 3D LUTs and renders comparison sheets showing what
goes wrong when `DOMAIN_MIN/MAX`, the unit scale, or the channel/storage
order are misread. No external graphics libraries (`image/png` only).

```
go run ./cmd/luttool demo out              # three comparison groups + slices
go run ./cmd/luttool apply in.cube out.png # apply one LUT to the test chart
```

- `out/compare_groups.png` — three rows: identity LUT (source | applied |
  amplified diff), channel-swap LUT (source | correct order | wrong storage
  order), non-uniform-domain LUT (source | domain honored | domain ignored).
- `out/interp_slices.png` — per-LUT slice grids: one tile per blue lattice
  node plus one fractional-blue slice through the trilinear interpolator.
- Format errors keep the last valid `out.png` (atomic write) and report the
  offending line, e.g. `testdata/bad.cube:7: bad number "oops" in data row`.

Layout: `lut/` parser + trilinear core, `render/` charts and slice grids,
`cmd/luttool/` CLI, `testdata/` sample good/bad `.cube` files.
