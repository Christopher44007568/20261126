# lazyscene：子树 payload 按需加载演示

纯 Go（仅标准库，外部依赖为零）实现的场景树渲染演示，覆盖远端 payload
的**按需加载、缓存、卸载**，并输出未加载 / 加载中 / 完整三帧 PPM 图像，
每个叶子子树都叠加其当前加载状态色。

## 运行

```sh
go run . artifacts
```

产物（512x512，二进制 P6 PPM）：

- `artifacts/frame1_unloaded.ppm`：先卸载全部远端 payload 后再渲染，明确显示尚未加载内容
- `artifacts/frame2_loading.ppm`：按需请求后捕获的中间态（T1 已完整，T2-T4 加载中）
- `artifacts/frame3_loaded.ppm`：所有请求结算、命中缓存后的完整画面

运行结束会打印自检报告；测试：

```sh
go test -race ./...
```

## 三态与状态色

| 状态 | 含义 | 叠加/描边色 | 画面表现 |
| --- | --- | --- | --- |
| `UNLOADED` | 缓存中无 payload | 红 | 斜纹占位 + `NOT LOADED` |
| `LOADING` | 远端请求在途 | 琥珀 | 扫描线 + 进度槽 + `LOADING` |
| `LOADED` | payload 已缓存 | 绿 | 确定性彩色单元网格 + 种子标识 |

## 代码结构

- `scene.go`：两级场景树（root -> group -> 4 个叶子子树）与 2x2 布局
- `loader.go`：加载状态机、异步按需加载、缓存与卸载（含在途请求失效）
- `payload.go`：确定性 LCG 与“远端” payload 生成
- `renderer.go` / `renderer_full.go`：三态渲染与状态色叠加
- `image.go` / `font.go`：零依赖 RGBA 画布、alpha 混合、P6 PPM 写出、位图字体
- `main.go`：三帧编排与“切回已卸载区域”验证
- `verify.go`：像素哈希断言

## 为什么切回已卸载区域不会复用旧像素

1. `Loader.Unload/UnloadAll` 立即清空缓存并递增 `generation`，使任何在途结果失效，
   状态回到 `UNLOADED`。
2. `Renderer` 不持有任何像素缓存，每次 `Render` 都通过 `NewImage` 分配一张清零画布；
   已卸载子树只能走占位渲染路径，物理上没有旧像素可复用。
3. `verify.go` 对完整帧与“卸载后切回”帧的每个子树区域计算 FNV-1a 哈希并断言不同，
   同时断言切回帧与首次未加载帧的占位像素完全一致。
