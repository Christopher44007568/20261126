# glasstrace

Pure-Go (standard library only) Whitted-style ray tracer demonstrating
**any-hit transparent occlusion queries**. Instead of truncating a shadow ray
at the first glass boundary (which makes colored objects behind glass vanish
inside hard shadows), every intersection event is walked front-to-back:

- an **opaque** surface hard-stops the ray and produces a solid shadow;
- a **glass** entry/exit pair is matched by volume identity and the optical
  path length drives Beer-Lambert spectral attenuation;
- clear glass (zero absorption) alters no visibility, so it cannot cast a
  solid shadow, while strong absorption restores near-opaque occlusion.

## Outputs per frame

Each frame writes three images (binary P6 PPM, no external dependencies):

- `frameNN_beauty.ppm` — full shading: Lambert lighting, Fresnel reflection,
  Snell refraction and in-glass absorption;
- `frameNN_shadow.ppm` — shadow visibility: white/gray reaches the light,
  black is a hard opaque shadow, gray encodes glass transmittance;
- `frameNN_transmission.ppm` — spectral color transmission through glass
  (white for clear glass, tinted where absorption applies, black where an
  opaque surface blocks the ray).

`stats.csv` records per-frame ray counters, including the number of
penetrating rays (glass-traversed plus fully clear rays that reach the
light), opaque-blocked rays and individual glass boundary crossings.

## Animation

`FrameParamsFor` cycles each frame through:

- slab geometric thickness (0.35 to 0.80);
- absorption tint hue across the spectrum;
- absorption strength from exactly zero (frame 0, perfectly transparent) to
  a peak where the most absorbed channel transmits under 2% (near opaque).

## Run

```sh
go run . -frames 6 -w 320 -h 240 -aa 2 -out artifacts
go test ./...
```

Flags: `-frames`, `-w`, `-h`, `-aa` (per-pixel supersampling grid), `-out`.

## Code map

- `vec.go` — vector math, Fresnel, refraction, Beer-Lambert;
- `scene.go` — primitives (sphere, ground, AABB glass slab) and the full
  front-to-back hit event list used by any-hit traversal;
- `occlusion.go` — the any-hit query and ray statistics;
- `render.go` — recursive Whitted camera path and opaque shading;
- `animation.go` — per-frame glass state and scene construction;
- `render_driver.go`, `main.go` — frame loop, image/CSV output;
- `image.go` — tone mapping and PPM writer.
