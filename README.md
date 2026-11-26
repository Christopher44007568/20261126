# 20261126 — 命名空间迁移：reference / 材质绑定 / 相机目标保持与红框审计

纯 Go 标准库、**零外部依赖**的可运行演示。把一个 USD 风格的小场景从旧命名空间
`/Show/OldNamespace` 迁移到新路径 `/Show/Shot042/Set`，并做到：

- 路径/父子感知的正确迁移后，`reference`、`rel material:binding`、`camera.target`
  仍然解析到真实节点（有单元测试锁定）。
- 渲染迁移前、迁移后两张 PPM：`artifacts/migration_before.ppm`、
  `artifacts/migration_after.ppm`，并用纯红 `(255,0,0)` 框标出**断链对象**与
  **保留的默认值**。
- 额外给出反例图 `artifacts/naive_string_replace_after.ppm`：当只做字面字符串替换、
  没有沿父子链更新子孙路径时，报告与图片都会给出每条链接的**具体旧路径**
  （`old_path`）和残留的旧父子路径（`stale_parent_child_path`）。

## 运行

```sh
go run .          # 重新生成 artifacts/ 下的 PPM/PNG/JSON/MD
go test ./...     # 路径重写、迁移不变量、具体旧路径、红框像素与端到端产物
```

只使用标准库（`image/png`、`encoding/json`、`math` 等）；PPM 为手写二进制 P6，
PNG 仅作方便预览的副产品。默认写入 `artifacts/`，也可 `go run . /other/dir`。

## 产物

| 文件 | 说明 |
|---|---|
| `artifacts/migration_before.ppm`（+ `.png`） | **必需 PPM ①**：迁移前（旧命名空间）场景与链接审计 |
| `artifacts/migration_after.ppm`（+ `.png`） | **必需 PPM ②**：正确迁移后，reference/材质/相机目标保持 |
| `artifacts/naive_string_replace_after.ppm` / `.png` | 只替换字符串导致父子路径失配的反例图 |
| `artifacts/red_boxes.json` | 每个红框的像素坐标、类别、`link_id` 与 `old_path` |
| `artifacts/migration_report.md` / `.json` | 完整文字/结构化审计报告（含具体旧路径表） |

## 场景里的链接（旧路径即具体旧路径）

| ID | 类型 | 持有对象 | 旧路径目标 | 正确迁移后 |
|---|---|---|---|---|
| R1 | reference | `…/Environment/Chair` | `/Show/OldNamespace/Assets/ChairRig` | 解析成功 |
| M1 | 材质绑定 | `…/Environment/Table` | `/Show/OldNamespace/Materials/RedPlastic` | 解析成功 |
| M2 | 材质绑定 | `…/Props/Teapot` | `/Show/OldNamespace/Materials/BlueMetal` | 解析成功 |
| C1 | 相机目标 | `…/Cameras/HeroCam` | `/Show/OldNamespace/Rig/FocusTarget` | 解析成功 |
| R2 | reference | `…/Environment/Crate` | `/Show/OldNamespace/Assets/MissingCrateRig` | 仍断链（刻意边界用例，红框） |
| M3 | 材质绑定 | `…/Props/Lamp` | `/Show/OldNamespace/Materials/GhostGlass` | 失效→保留默认材质 `DefaultSurface #B9BCC6`（红框） |

R2/M3 在迁移前就不存在目标，正确迁移不会“编造”链接，而是持续用红框标出，
避免把问题静默洗掉。

## 正确迁移 vs 只替换字符串

- `paths.RewritePrefix` 按**路径组件边界**重写：目标等于旧根或位于旧根之下才替换，
  因此 `/Show/OldNamespaceArchive` 这类同前缀旁支不会被误伤（见 `paths/path_test.go`）。
- `migrate.MigrateCorrect` 遍历全部 Prim 路径与三类链接，子孙父子路径一起重根，
  `migrate.CheckParentChild` 校验 `child.Path == parent.Path + "/" + Base(child.Path)`。
- `migrate.MigrateStringOnly` 复现反馈中的错误做法：只 `strings.ReplaceAll` 链接字符串
  并改根名，子孙节点仍停在旧父子路径。`audit.Audit` 对每条断链反查出
  `stale_parent_child_path`，并始终携带 `old_path`（具体旧路径）。例如：

  ```
  M1 material_binding
    old_path  = /Show/OldNamespace/Materials/RedPlastic
    now       = /Show/Shot042/Set/Materials/RedPlastic   (字符串已替换)
    stale     = /Show/OldNamespace/Materials/RedPlastic (节点仍在此父子路径)
  ```

## 红框语义

`red_boxes.json` 中 `category`：

- `broken_link`：断链对象/状态徽标（如 R2，反例中还有 R1、C1）。
- `retained_default`：保留的默认值（M3 的 `DefaultSurface` 色卡；反例中 M1/M2 也退化为默认材质）。

`viz/poster_test.go` 与根目录 `pipeline_test.go` 会检查每个登记框的边框像素确实是
纯红 `(255,0,0)`，并端到端校验 P6 PPM、PNG 与报告中的具体旧路径。

## 代码结构

- `paths/`：组件感知的场景路径、前缀重写与裸字符串替换对照。
- `scene/`：场景/Prim/链接模型与固定 fixture、迁移映射。
- `migrate/`：正确迁移、字符串-only 反例、父子一致性检查。
- `audit/`：链接解析、`ok/broken/default_retained` 分类、旧路径与残留父子路径。
- `raster/`：零依赖 RGB 画布、图元、5×7 位图字体、P6 PPM 与标准库 PNG 输出。
- `viz/`：顶视场景图、链接弧线、红框与右侧路径台账海报。
- `report/`：JSON / Markdown / 红框清单。
- `main.go`：串联生成全部产物，并在核心不变量被破坏时以非零码退出。

Repository initialized for 20261126.
