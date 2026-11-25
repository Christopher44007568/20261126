# hotreload

Zero-dependency Go scene-graph hot-reload demo. Only the Go standard
library is used (PNG encoding included); there are no external modules.

## Scene graph

```
rgb.payload -> rgb.layer -> rgb.grade ─────────────┐
geo.asset -> geo.layer -> surface -> composite     │
                     ├-> outline -> composite       ├-> composite
material -> surface ──────────────────────────────┘
background ───────────────────────────────────────-> composite
```

Every renderable node owns a content-addressed `CacheEntry` keyed by an FNV
hash of its inputs. On reload the whole graph is re-walked in topological
order:

- unchanged keys are retained and reported in `cache_hits` /
  `cache_retained`;
- changed keys invalidate the old entry, recompute the node, and are
  reported in `cache_invalidated` / `recomputed_nodes`;
- recomputation is reachable only inside the changed subtree, and an
  invariant panics if a node outside it ever changes key;
- a broken RGB payload stores an explicit error entry, so an old successful
  frame can never be served as a cache hit over the new failure;
- geometry replacement invalidates the geometry, surface, outline and
  composite nodes, and detaches materials absent from the new asset's
  compatibility list instead of re-binding them.

## Scenarios

`cmd/hotreload` runs four scenarios, each writing `*_before.png`,
`*_after.png`, `*_diff.png` and a JSON report:

- `rgb` — hot reload of the RGB layer; only the RGB subtree recomputes.
- `geometry` — rectangle replaced by a triangle with a compatible material.
- `material_detached` — same geometry swap but the old stripe material stays
  detached and the surface renders an explicit danger fallback.
- `rgb_error` — truncated RGBA payload demonstrating error propagation.

Diff images desaturate unchanged pixels, paint changed pixels magenta, and
draw a yellow bounding box around the changed region. The pixel threshold is
an Euclidean RGB distance of 28, matching the JSON `threshold` field.

## Run

```sh
go test ./...
go run ./cmd/hotreload ./output
```

Outputs land in `./output` (or the directory passed as the first argument),
including a combined `reports.json`.
