# TEXPATH

本地贴图搜索与回退演示程序。使用 Go 标准库实现，运行时零外部依赖；资产、缓存冲突和占位贴图都由程序在本机生成。

当贴图在全部目录中缺失时，程序返回可见的棋盘格占位，而不是黑色纹理。场景光照、背景、基座和其他材质继续正常渲染。

## 运行

需要 Go 1.24 或更高版本。

```bash
go run .
```

打开 <http://localhost:8080>。可用 `ADDR=:9000 go run .` 修改监听地址。

首次启动会自动补齐演示贴图：

- `assets/textures/`：本地资产，最高优先级
- `.cache/textures/`：运行时缓存，第二优先级
- `assets/fallback/`：最终回退，第三优先级

## 搜索规则

搜索顺序由 `texturepaths.json` 的 `searchOrder` 决定，默认顺序为：

```text
assets/textures -> .cache/textures -> assets/fallback
```

对每个文件名，程序按优先级依次探测，最终只采用第一个存在的普通文件。后续目录中的同名文件会进入 `conflicts` 列表，界面同时显示：

- 最终采用来源
- 被抑制的同名候选
- 每个目录的完整搜索轨迹
- 程序生成的棋盘格回退来源

可用 `TEXPATH_CONFIG=/path/to/config.json` 指定另一份配置。配置缺失时使用内置的相同默认值。

## 界面内容

- 搜索顺序轨道：显示每个根路径的命中数、被跳过候选和优先级状态
- 三组贴图状态总览：本地资产、运行时缓存、最终回退（含占位）
- 实时材质场景：三种材质同场渲染，缺失槽位显示棋盘格
- 贴图状态明细：9 个槽位的缩略图、文件名、采用路径和冲突提示
- 来源检查器：点击任意槽位查看完整 `trace`、最终目录和同名候选

## API

```text
GET /api/scene                完整场景、三组统计、材质与解析结果
GET /api/resolve?name=...     解析单个文件名并返回全部候选
GET /texture/{root}/{file}    读取最终采用的实际贴图
GET /texture/_placeholder.png 程序生成的棋盘格
GET /healthz                  健康检查
```

## 验证

```bash
go test ./...
go vet ./...
```

测试覆盖优先级选择、同名冲突、缺失占位、路径穿越拒绝、三组统计聚合和默认配置回退。
