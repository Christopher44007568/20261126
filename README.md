# 射线参考与简化加速结构对比诊断

纯 Go 标准库实现的软件光线追踪诊断工程。它把逐三角形暴力参考与保守的
简化 BVH 并排渲染，输出同相机、同灯光的参考图、简化图、交点数图和误差热图，
并用逐像素与探针射线两级严格比较定位漏命中、假命中、错误三角形和交点缺失。

## 运行

```sh
go run .
go test -count=1 ./...
go vet ./...
```

常用参数：

```sh
go run . -out artifacts -width 480 -height 270 -probe-rays 1024
```

只使用 Go 标准库，`go.mod` 没有第三方依赖。

## 产物

默认写入 `artifacts/`：

- `reference.png`：三组输入纵向排列的暴力参考图。
- `simplified.png`：三组输入纵向排列的简化 BVH 渲染图。
- `intersection_count.png`：每条相机射线的有效三角形交点数。
- `traversal_tests.png`：简化 BVH 对每条射线执行的精确三角形求交次数。
- `error_heatmap.png`：零误差为深海军蓝；任何局部差异按热力色显示，结构错误直接记为最高误差。
- `error_mask.png`：分类遮罩，红色为漏命中，橙色为假命中，品红为错误三角形，青色为交点数不一致。
- `side_by_side.png`：3 行 4 列总览；行为擦边、平行、大三角形，列为参考、简化、交点数、热图。
- `index.html`：带说明、指标和局部错误包围盒的并排报告。
- `report.json`：机器可读的逐组指标、区域和前若干个差异。
- `suites/grazing/`、`suites/parallel/`、`suites/large_triangles/`：每组独立的四类图像、遮罩和并排图。
- `fault_injection/`：明确标注的负面对照，故意漏掉一个三角形，演示热图如何显示局部红色区域。

## 三组边界输入

- `grazing`：共享边、细长三角片、刀边和近边探针，覆盖精确过边、过顶点、内侧/外侧近边射线。
- `parallel`：沿纵深伸展的平行条带；探针包含平行于三角形边/平面、共面和带微小平面偏移的射线。
- `large_triangles`：三块占满屏幕区域的大三角形和前方局部小三角阵；另有 `1e6` 坐标尺度探针。

每组都比较：

1. 最近命中的三角形 ID 与 `t`；
2. 该射线的全部有效交点数量、ID 顺序和 `t`；
3. 最终 RGB 颜色；
4. 确定性边缘、平行、大尺度和随机探针射线。

因此即使平均颜色误差很低，只要某个像素或某条探针漏掉一个三角形，
`pass` 仍为 `false`，CLI 和 `go test` 都会失败。

## 实现约定

- `geometry.go`：float64 向量、保守 AABB、Möller–Trumbore 求交和边/顶点容差。
- `bvh.go`：最长轴中位数划分的简化 BVH；三角形包围盒按坐标尺度外扩。
- `scene.go`、`camera.go`：同一透视相机、灯光、材质和着色函数。
- `render_diag.go`、`diagnose.go`：逐像素双路追踪、分类、热图和局部区域统计。
- `raysuite.go`：擦边、平行、随机和极端尺度探针。
- `pipeline.go`：生成全部 PNG/JSON/HTML 产物。

加速结构只负责候选剔除；候选三角形仍调用与参考路径完全相同的
`Triangle.Intersect`，从根上避免两套求交公式产生细微不一致。诊断追踪不做
最近点提前裁剪，因此可以核对射线的完整交点列表，而不只是最终可见颜色。
