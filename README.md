# Go 最近命中 / 任意命中射线查询示例

纯 Go 标准库实现，不依赖任何第三方包。同一个不可变、带内容版本的场景上运行两类查询：

- 主射线使用最近命中（closest hit）生成 `outputs/depth.png`。
- 主光阴影射线和环境遮蔽射线使用第一次任意命中（first any-hit）生成
  `outputs/anyhit_shadow.png`；图中灰度是主光可见性与半球 AO 可见性的乘积。
- 可见表面着色与遮蔽系数合成 `outputs/composite.png`。

一次 `Render` 调用产生同一 `Frame`，三张图和 `outputs/manifest.json` 都记录相同的
`scene_version` / `scene_hash`。清单还包含三张 PNG 的 SHA-256，避免图片来自不同场景版本。

## 运行

```bash
go test ./...
go run ./cmd/rayao
```

常用参数：

```bash
go run ./cmd/rayao -width 320 -height 240 -ao-samples 48 -out outputs
```

默认输出：

- `outputs/depth.png`：8 位灰度主交点深度；白色为近，黑色为远或未命中，度量范围见清单。
- `outputs/anyhit_shadow.png`：第一次任意命中阴影/AO 图；白色表示射线未被阻挡，黑色表示主光被阻挡或 AO 采样被阻挡。
- `outputs/composite.png`：材质、半球光照和遮蔽合成结果。
- `outputs/manifest.json`：同一场景版本、尺寸、射线统计和图片校验值。

渲染按像素使用固定 Hash/Hammersley 采样；并行只分配行，不使用共享随机源，因此重复运行的
PNG 字节哈希一致。

## 统一查询 API

所有射线都走 `query.go` 的同一套查询结构：

```go
result := scene.Query(rayao.DefaultRayQuery(ray))
result.Closest   // 最近交点
result.HasClosest
result.FirstAny  // 确定性场景顺序中的第一个任意命中
result.HasAny

hit, ok := scene.ClosestHit(query)
witness, blocked := scene.AnyHit(query)
```

`RayQuery.IgnorePrimitive` 可排除产生次级射线的主交点三角形。显式候选列表可来自 BVH；
如果重叠节点把同一个三角形重复送入查询，`QueryCandidates` 会按图元 ID 去重并记录
`DuplicateCandidates`。

## 边界情况测试

`query_test.go` 固定了题目要求的三类射线起点/重复命中语义：

- `TestOriginInsideClosedSurfaceHitsExit`：起点在封闭立方体内部，双面求交返回正向出口，
  并把着色法线翻向射线一侧。
- `TestPrimaryAndSecondaryCanHitSameTriangle`：主射线和从交点重建的次级射线命中同一个三角形；
  原始零下限查询会在 `t=0` 再见该三角形，设置 `IgnorePrimitive` 后消除自遮蔽。
- `TestZeroDistanceStartIsAccepted`：`MinT == 0` 时接受精确零距交点。
- `TestDuplicateTriangleCandidateCountsOnce`：同一三角形作为重复候选进入同一查询时只计算一次。

`render_test.go` 验证三张图尺寸一致、共享同一场景哈希，并且写出的 PNG 可重新解码、
校验值与清单一致。

## 代码结构

- `vec3.go`：向量运算。
- `ray.go`：射线区间与合法性。
- `triangle.go` / `primitives.go`：三角形、命中记录、立方体/盒体构造。
- `scene.go`：不可变场景快照和确定性内容哈希。
- `query.go`：Möller–Trumbore 求交、最近命中、任意命中和候选去重。
- `camera.go`：透视相机和主射线。
- `shade.go`：确定性半球采样、任意命中 AO 和材质着色。
- `render.go` / `output.go`：并行帧渲染、PNG/JSON 输出。
- `cmd/rayao/main.go`：命令行入口。
