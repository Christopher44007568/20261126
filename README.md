# Virtual Wall：Go 标准库虚拟纹理

这个仓库用纯 Go 标准库实现了一个 CPU/软件版虚拟纹理系统，不依赖图形 SDK 或第三方包。场景模拟相机贴近高倍率墙面：高分辨率 mip0 页面只在 UV 反馈发现可见时异步载入；视口移动后，新页未到达前使用最近的驻留父级 mip，而不是用相邻同层页面顶替。

## 运行

```sh
go run ./cmd/virtualwall
go test -race ./...
```

默认产物写入 `artifacts/virtualwall/`。直接打开：

- `artifacts/virtualwall/index.html`：所有视图和指标报告
- `artifacts/virtualwall/camera_view.png`：移动后父级 mip 回退的相机视图
- `artifacts/virtualwall/miss_mask.png`：缺页掩码，白色表示精确请求页未驻留
- `artifacts/virtualwall/resident_set.png`：各 mip 的驻留集合和物理槽位颜色
- `artifacts/virtualwall/uv_feedback.png`：低分辨率 UV/PageID 反馈缓冲
- `artifacts/virtualwall/metrics.svg`：缺页率随加载阶段下降的曲线
- `artifacts/virtualwall/report.json`：指标和逐帧内容校验结果

## 流水线

1. `vt/camera.go` 由针孔相机的 UV 足迹计算 `du/dx`、`dv/dy` 和整数 mip。
2. `BuildFeedback` 生成低分辨率 UV 反馈缓冲，并统计可见 `PageID`。
3. `vt/scheduler.go` 对反馈直方图去重，补齐每个缺页的父级链，按粗 mip 优先调度，并用代际集合拒绝旧视口的过期结果。
4. `vt/loader.go` 是有界队列和固定 worker 池；演示用程序化墙面模拟异步磁盘/纹理加载延迟。
5. `vt/table.go` 维护页表、物理槽位和 LRU 链表。当前可见父级链会在替换前 `Touch`，容量固定为 12 页。
6. `vt/renderer.go` 在一帧开始时取页表快照。`Resolve` 只返回精确页或其祖先；采样坐标被钳制在该页自身的 `64×64` 矩形内，因此不会读到相邻物理槽或相邻虚拟页。
7. `vt/verify.go` 对每个像素校验解析关系和颜色 oracle。父级采样只与完整纹理对应祖先页矩形内的边缘钳制双线性结果比较。

## 演示阶段

默认纹理为 `512×512`，页大小 `64×64`，每边 8 个 mip0 页，共 4 级 mip、85 个虚拟页；物理缓存只有 12 页。

相机视口对齐到 `3×2` 个 mip0 页面，加上直接父页、更高层祖先和根页，当前工作集恰好 12 页。跨墙移动后：

- `01_settled`：首视角 6 个高分辨率页全部到达，像素缺页率 `0`。
- `02_parent_fallback`：新视角 6 个 mip0 页都在异步队列中，5 个新祖先页已到达，所有像素由直接父级 mip 支撑，缺页率 `1`，无无页空洞。
- `03_partial`：故意只放行 2 个 mip0 加载，两块区域变清晰，缺页率降到 `2/3`。
- `04_complete`：全部可见 mip0 页到达，缺页率回到 `0`；LRU 已淘汰 11 个旧视角页面，缓存仍为 `12/12`。

## 正确性不变量

- 缺页时只能使用 `Requested` 本身或 `Requested` 的 mip 祖先，绝不使用同层邻居。
- 双线性采样的索引被钳制在解析页矩形内；物理槽之间有独立 `image.RGBA` 或暗色槽间边，不存在 atlas 邻页渗入。
- 异步结果携带调度代际；已经离开当前 `Wanted` 集合的旧页面即使晚到也不会安装进缓存，避免错误工作集驱逐。
- 缺页率下降只来自精确页实际安装。`Frame.Verify` 逐像素比较颜色与解析页 oracle；演示的四帧和 Go 测试中的端到端移动帧均通过校验。
