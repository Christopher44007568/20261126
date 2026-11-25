# Soft-body BVH refit / rebuild comparison

纯 Go（仅标准库）的软体角色 BVH 对比示例。角色由三角扇组成的人形网格构成，
动画从明显弯曲的绑定姿态回到伸展姿态。最终帧分别用快速 refit、逐帧 rebuild
和暴力求交渲染，三分结果逐像素对齐。

## 运行

```bash
go run . -out artifacts -frames 36 -width 360
go test ./...
```

生成的图像与报告：

- `artifacts/refit.ppm`：仅快速 refit 的最终帧。叶节点包围盒按质心平移，
  大变形后旋转几何会跑出旧盒子，可看到漏击与误击。
- `artifacts/rebuild.ppm`：最终帧按当前顶点重建 BVH，结果与暴力求交一致。
- `artifacts/brute.ppm`：逐三角形暴力求交，作为像素真值。
- `artifacts/adaptive.ppm`：生产路径。逐帧 refit 并检查包围盒预算，
  超预算或整帧为空时自动重建。
- `artifacts/report.json`：耗时、重建帧、最终包围盒指标和逐像素差异计数。

PPM 为二进制 `P6`，顶部左侧色块图例：

- 红色：漏击。暴力求交命中，但当前模式返回背景。
- 蓝色：误击。模式命中错误三角形，或背景被误判为命中。

`report.json` 会分别给出 `MissedPixels`、`FalseHitPixels`、
`BackgroundFalseHit` 和 `WrongTriangle`。

## 实现要点

- `geometry.go`：二维向量、AABB、三角形和点包含测试。
- `bvh.go`：按最长轴中位数划分的 BVH；叶节点保存绑定姿态 AABB。
- `softbody.go`：软体人形与随高度变化的角度场。`Pose(0)` 为弯曲姿态，
  `Pose(1)` 为伸展姿态。
- `render.go`：正交相机、BVH/暴力遍历、覆盖率对比和误差着色。
- `simulation.go`：refit、rebuild、brute、adaptive 四种执行路径。

快速 refit 只将绑定姿态叶盒按当前叶质心平移。大幅旋转后，实际三角形可能
落到旧叶盒之外，因此会漏掉本应命中的像素；叶内三角形顺序和覆盖关系也可能
让最近命中变成错误三角形。这正好暴露“便宜 refit”在大变形下失效的边界。

adaptive 每次 refit 后重算当前叶盒，并检查：

- 单叶逃出率 `MaxLeafEscape > 5%`
- 根包围盒逃出率 `RootEscape > 2%`
- 叶盒面积恶化 `MaxAreaGrowth > 2.0`

任一条件超预算就立即按当前顶点重建。渲染后还会检查结果是否为空；若暴力真值
非空而 adaptive 整帧为空，会再强制重建，因此不会把空洞背景当成有效帧返回。

默认参数下，36 帧的 refit 更新耗时约为 rebuild 的 1/80；adaptive 只在预算
超限的帧重建，最终图与 brute 一致。
