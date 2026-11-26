# Descriptor 版本与陈旧相机绑定演示

纯 Go 标准库实现的软件渲染示例，用来复现并验证以下问题：材质的纹理版本
没有变化，但描述符仍绑定上一版相机矩阵，于是纹理内容正确、投影位置却错误。

## 运行

```sh
go run . -out artifacts
go test ./...
go test -race ./...
go vet ./...
```

程序没有第三方依赖，`go.mod` 只声明 Go 语言版本。PNG 使用 `image/png`，
JSON 使用 `encoding/json`，投影、纹理采样、文字和叠加标记均为内置实现。

## 生成文件

| 文件 | 内容 |
| --- | --- |
| `artifacts/normal_frame.png` | 当前相机 v3 + 纹理 v1 + 新描述符 v2 |
| `artifacts/stale_binding_frame.png` | 陈旧快照：相机 v2 + 同一纹理 v1 + 描述符 v1 |
| `artifacts/version_source_overlay.png` | 版本来源叠加：青色为当前相机，琥珀色为固定快照，品红连接对应顶点以显示投影差 |
| `artifacts/descriptor_report.json` | 描述符版本、失效标记、纹理哈希、投影差、引用计数和 GC 结果 |

两帧共享完全相同的纹理像素，但投影中心相差约 106 px。报告中的
`same_texture_pixels=true` 与 `projection_delta_pixels` 分别验证这两个条件。

## 核心设计

### 不可变版本

- `CameraVersion`、`TextureVersion`、`DescriptorVersion` 都是单调递增版本号。
- `CameraPool.Install` 和 `TextureStore.Install` 总是分配新条目，绝不原地覆盖旧矩阵或旧像素。
- 纹理发布时会复制源图像；调用方之后修改源图不会影响已经绑定的材质。

### 绑定快照

- `DescriptorSystem.Bind` 创建新的 `DescriptorRecord` 和 `BindingSnapshot`。
- 快照固定描述符版本、相机版本、纹理版本、相机矩阵和纹理哈希。
- 相机/纹理发布新版本后，只增加 `InvalidationFlags`，不会改写已有快照。
- `Resolve` 渲染前会核对快照矩阵与纹理哈希；若资源被错误覆盖，会返回
  `snapshot_mismatch`，而不是静默画出错误结果。

失效标记包括：

- `camera_changed`
- `texture_changed`
- `descriptor_stale`
- `snapshot_mismatch`

### 资源生命周期

- 材质在 `Bind` / `RefreshCamera` 时先获取新资源引用，再释放旧引用。
- 非当前相机只要仍被材质快照引用，就保持 resident；`GC` 不会回收它。
- 引用归零后，`GC` 才删除历史相机或纹理。
- 演示中 v1 相机在 GC 时被回收，v2 因陈旧材质引用而保留，v3 因是当前版本而保留。

### 演示序列

1. 发布相机 v1、纹理 v1。
2. 发布相机 v2，并把 `MATERIAL.STALE.SNAPSHOT` 绑定到 v2。
3. 发布相机 v3；旧描述符得到 `camera_changed + descriptor_stale`，但仍持有 v2 矩阵。
4. 新建 `MATERIAL.LIVE.SNAPSHOT`，绑定当前 v3 和同一纹理 v1。
5. 执行 GC，验证 v2 未因发布 v3 被覆盖或回收。
6. 用两个快照分别渲染，再合成版本来源叠加图。

## 代码入口

- `resources.go`：不可变资源池、引用计数与 GC。
- `descriptor.go`：描述符版本、快照、失效标记和重绑定。
- `render.go`：单应矩阵投影、纹理四边形光栅化。
- `overlay.go`：版本来源叠加图。
- `demo.go`：演示版本序列、产物与审计报告。
