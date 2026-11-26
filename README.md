# varianttree：三变体选择、父子传播与逐像素恢复

纯 Go 标准库实现，外部依赖为零。示例资产是同一棵场景树中的湖边小屋，包含
`summer`、`winter`、`night` 三套 variant；几何节点不变，切换 variant 时会重新
解析并绑定真实 `Material`（着色器、颜色、绘制模式），而不是只替换材质名字符串。

## 运行

```sh
go test -race ./...
go run . artifacts
```

`go.mod` 没有任何 `require`。所有图像均为零依赖写出的二进制 P6 PPM。

## 产物

运行后写入 `artifacts/`：

- `variant_summer.ppm`、`variant_winter.ppm`、`variant_night.ppm`：三套独立变体，均为 `480x360`
- `variant_contact_sheet.ppm`：三变体横向 PPM 拼图，`1520x532`
- `selection_path.ppm`：父子传播演示帧；根节点选 `winter`，`WindowBank` 覆盖为 `night`
- `current_selection_path.ppm`：恢复到原始状态后的当前选择路径标注帧
- `current_selection_path.txt` / `current_selection_path.json`：当前选择路径、完整传播链和传播演示路径
- `override_restored_winter.ppm`：删除 `WindowBank=night` 覆盖后恢复为继承的冬季
- `restored_summer.ppm`：切回原始 `summer` 后的帧
- `rebind_repaired_summer.ppm`：故意只改材质名导致绑定失效，再重新绑定后的修复帧
- `variant_report.json`：检查项、输出文件 SHA-256、绑定解析和像素哈希

## 选择与传播规则

variant set 名称为 `ambience`。可以在任意节点编写选择：

```text
/Stage/Asset                         = winter
/Stage/Asset/Cabin/WindowBank        = night   # 临时覆盖
```

解析 `/Stage/Asset/Cabin/WindowBank/Left` 时，按 root -> target 顺序应用路径上的
意见，离目标越近的意见越晚生效：

```text
/Stage                                     inherit summer
/Stage/Asset                               AUTHOR  winter
/Stage/Asset/Cabin                         inherit winter
/Stage/Asset/Cabin/WindowBank              AUTHOR  night
/Stage/Asset/Cabin/WindowBank/Left         inherit night
```

因此 `WindowBank` 的覆盖会继续传播给 `Left`、`Right`，而 `Roof` 等兄弟节点仍继承
根资产的 `winter`。

## 绑定而不是只改名字

每个节点保存 `ResolvedBinding`，其中包含材质 ID、shader、生效 variant、意见来源，
以及指向材质注册表中真实资源的 `Material *Material` 指针。

`Stage.SelectVariant` / `Stage.Override` 后会立即执行 `Rebind`：

1. 沿父子链解析最终 variant；
2. 从节点的 `VariantMaterials[variant][slot]` 取得材质 ID；
3. 在 `Materials` 注册表中查找真实 `*Material`；
4. 写入 shader、来源路径和资源指针；
5. 渲染前 `ValidateBindings` 再次确认 ID、shader、指针一致。

`ChangeMaterialNameOnly` 故意复现反模式：只把 `MaterialID` 改成冬季材质并清空
`Material` 指针。此时校验和渲染都会失败，直到重新 `Rebind`。

## 覆盖恢复与逐像素一致

`Override` 返回 `SelectionToken`，记录该路径原先是否存在意见以及原值。恢复时：

- 原来有意见：写回原值；
- 原来无意见：删除该覆盖，而不是错误地写成某个默认 variant；
- 随后重新绑定整棵树。

渲染器每次都创建全新的 `Image`，不缓存或复用上一帧像素。报告中的实际自检结果为：

```text
override -> inherited winter: identical=true differentPixels=0
switch back to summer:       identical=true differentPixels=0
name-only then rebind:       identical=true differentPixels=0
```

端到端测试还直接比较 `variant_summer.ppm` 与 `restored_summer.ppm` 的文件字节，
并解码 PPM 校验像素哈希，因此“逐像素恢复”同时由内存比较、PPM 往返和文件字节
三层验证固定。

## 代码结构

- `variant.go`：场景节点、variant 解析、父子传播、覆盖令牌、绑定校验
- `scene.go` / `scene_materials.go`：小屋资产层级、三套材质和逐槽位绑定表
- `render.go` / `patterns.go`：零依赖渲染器、着色器模式和路径高亮
- `image.go` / `ppm.go` / `font.go`：RGBA 画布、P6 PPM 读写和位图字体
- `contact.go`：三变体拼图与路径标注
- `pipeline.go` / `report.go` / `main.go`：演示编排、报告和 CLI
- `variant_test.go` / `pipeline_test.go`：传播、恢复、负向绑定和产物测试
