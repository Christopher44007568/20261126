# texturestream：父级回退的流式纹理对比

纯 Go 标准库实现，不使用图形 SDK、CGO 或第三方依赖。程序用确定性的 512×512
程序化纹理、四叉树 mip 分块和连续移动相机，比较三种加载策略，并输出低 / 中 /
高细节检查点的二进制 P6 PPM。

## 运行

```sh
go test ./...
go run . -out artifacts
```

运行后生成：

- `artifacts/low.ppm`
- `artifacts/medium.ppm`
- `artifacts/high.ppm`

以及相同内容的描述性文件名：

- `artifacts/comparison_low.ppm`
- `artifacts/comparison_medium.ppm`
- `artifacts/comparison_high.ppm`

这三张是低、中、高细节的三联对比帧，每帧从左到右依次为 `full`、
`chunked`、`screen-error`。另外还有 9 张单策略帧：

```text
artifacts/{full,chunked,screen-error}_{low,medium,high}.ppm
```

机器和人工可读报告分别是 `artifacts/streaming_report.json` 与
`artifacts/streaming_report.md`。

## 三种策略

| 策略 | 请求方式 | 模拟结果 |
| --- | --- | --- |
| `full` | 等待完整高分辨率 payload | 传输量固定；完成前只显示根父级和琥珀加载斜纹，完成后整图变清晰 |
| `chunked` | 固定行主序请求全部 8×8 L3 分块 | 相同预算下可能先加载远处或低优先级块，镜头热点仍缺块 |
| `screen-error` | 按投影纹素尺寸选 LOD，优先可见且误差最大的节点 | 相同或更少字节下更早覆盖当前镜头所需区域 |

屏幕误差用节点纹素在屏幕上的直径近似：

```text
texelPixels = cameraScale / (64 × 2^lod)
idealLOD = 满足 texelPixels <= 3px 的最粗 LOD
screenError = max(0, texelPixels - 3px)
```

调度器每模拟帧只有 16 KiB 预算，即 4 个 4 KiB 分块；完整 payload 为
128 KiB。所有随机/时序均由固定相机动线和排序规则确定，PPM 可重复生成。

## 加载状态叠色

叠色分两层：底色表示当前实际采样到的回退层级，琥珀斜纹表示目标块仍在加载。

| 颜色 | 状态 | 含义 |
| --- | --- | --- |
| 绿色 `#22C55E` | `EXACT` | 目标 LOD 或更细块已驻留 |
| 青色 `#38BDF8` | `COARSE PARENT` | 目标块未到，正在使用中间 mip 父级 |
| 靛蓝 `#818CF8` | `ROOT PARENT` | 所有更细祖先都未到，使用已提交根纹理 |
| 琥珀斜纹 `#F59E0B` | `LOADING` | 目标 payload / chunk 在途或排队 |

## 为什么不会空白或黑白闪烁

- `TextureSource` 预生成高分辨率图，并用 box downsample 生成真实 mip；根节点不是黑/白占位图，而是整幅纹理的缩小颜色平均。
- `TileStore` 初始只把 L0 根纹理标记为 resident。渲染每个 UV 时从目标 LOD 向根逐级解析，因此任何未到达子块都有不透明、同 UV 的父级可采样。
- 子块到达只替换该 UV 解析到的 tile，不改变相机、四边形、UV 或 coverage mask。测试逐像素比较加载前后的几何遮罩。
- 状态叠色是半透明叠加，不会先把网格清成纯黑或纯白。自检遍历 3 个策略 × 11 个连续相机帧，断言空白像素、纯黑/纯白像素均为 0，并限制帧均亮度跳变。

## 当前检查点指标

高细节帧中：

- `full`：128 KiB，目标覆盖 100%，平均屏幕误差 0。
- `chunked`：160 KiB，目标覆盖约 72.5%，平均屏幕误差约 4.55px。
- `screen-error`：100 KiB，目标覆盖 100%，平均屏幕误差 0。

中细节帧在三者都传输 48 KiB 时，SSE 已达到约 90% 目标覆盖，剩余区域由青色中间父级和琥珀斜纹覆盖；全量与固定分块仍主要落在靛蓝根父级。

## 代码结构

- `texture.go`：程序化 512² 纹理、box mip、四叉树节点、tile store 与祖先解析。
- `camera.go`：连续相机路径、UV 投影、屏幕误差和理想 LOD。
- `loader.go`：三种调度策略、预算传输、优先级、请求状态。
- `render.go`：逐像素软件采样、状态叠色、HUD/图例和 `FrameMetrics`。
- `pipeline.go`：低/中/高检查点编排、单帧与三联 PPM 输出。
- `verify.go`：连续镜头空白、黑白闪烁、亮度跳变和几何遮罩校验。
- `font.go`、`image.go`：内置位图字体、RGBA 画布和 P6 PPM 编码。
- `*_test.go`：PPM 编码、回退非空、指标关系、确定性和连续性测试。

PPM 可用 GIMP、ImageMagick、FFmpeg 或 Netpbm 查看，例如：

```sh
convert artifacts/comparison_high.ppm comparison-high.png
```
