# Shadow Ray Flags Demo

Pure-Go (standard library only) comparison of how three shadow-ray policies
change occlusion for back-facing sheets, double-sided sheets and closed solids.

## Scene

All three maps trace the same directional shadow rays from a receiver plane
(`y=0`) toward one light direction (`internal/shadow/scene.go`):

| Object | Material | Expected behavior |
| --- | --- | --- |
| `ss_down` | single-sided, front face toward receiver | opaque shadow in every mode |
| `ss_up` | single-sided, back face toward receiver | never casts a solid shadow |
| `ds_sheet` | double-sided, back face toward receiver | shadow only when back faces are accepted |
| `cube` | closed, translucent solid (`Transmittance=0.6`) | one energy event per solid crossing |

## Ray flags

1. `BackfaceCull` (`shadow_backface_cull.png`): only front-face intersections
   count. The back of the double-sided sheet is rejected, so it stays lit.
2. `DoubleSidedHit` (`shadow_double_sided.png`): materials marked
   `DoubleSided` accept their back faces; single-sided materials still reject
   theirs. Newly accepted back faces are colored red.
3. `ReflectionClosure` (`shadow_reflection_closure.png`): same acceptance
   policy, but closure materials merge entry/exit hits into one per-solid
   event. The merged exit hit is colored amber instead of red.

## Color legend

Grayscale is remaining light energy (white = lit, black = fully blocked).

| Color | Meaning |
| --- | --- |
| red | extra occluder accepted only after back-face culling is relaxed |
| blue | geometric back-face hit on a single-sided sheet; shadow ignored |
| amber | real exit hit on a closed solid; energy already applied at entry |

## Correctness rules

- A single-sided sheet's back face never blocks energy under any flag. The
  blue footprint proves the hit exists but contributes zero occlusion.
- A closed solid applies its transmittance once per crossing even with
  double-sided hits enabled. The demo prints both the correct result
  (`0.60`) and the naive per-triangle double count (`0.36 = 0.6 * 0.6`).

## Run

```sh
go test ./...
go run ./cmd/shadowflags
```

PNG output is written to `artifacts/` using only `image/png`.
