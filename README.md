# 自建场景的 Object ID → Hit Group 映射

这是一个仅使用 Go 标准库的确定性光线追踪示例。场景几何只保存
`ObjectID`，材质与发光行为必须经过 `ObjectMapping` 解析到命中组：

```text
ObjectID -> HitGroupID -> MaterialID -> shading
```

因此石墙不会直接携带材质，灯板也不会被硬编码成“唯一光源”。交换一条
映射会同时改变材质编号图、命中组图和最终着色，并且能被逐像素诊断定位。

## 运行

```bash
go run .
# 或
go run . -out result -width 960 -height 600
go test ./...
```

默认输出到 `out/`。所有输出在相同尺寸和参数下完全确定，没有随机采样，
也没有第三方包。

## 输出

| 文件 | 内容 |
| --- | --- |
| `material_index.png` | 每个初命中像素的材质编号色块 |
| `hit_groups.png` | 每个初命中像素的命中组色块 |
| `shaded.png` | 正确映射下的最终着色图 |
| `broken_material_index.png` | 故意打乱一条映射后的材质编号图 |
| `broken_hit_groups.png` | 故意打乱一条映射后的命中组图 |
| `broken_shaded.png` | 故意打乱一条映射后的最终着色图 |
| `mapping_diff.png` | 差异图；红色为组/材质不同，白色方框标出首个错误像素 |
| `report.json` | 对象映射、组/材质调色板、唯一改动及首错误详情 |

`report.json` 同时包含 `reference_object_mappings` 和
`broken_object_mappings`，便于直接比较。PNG 中每个色块对应的编号记录在
`hit_groups` 与 `materials` 的 `palette_color` 字段中。

## 场景映射

| Object ID | 对象 | 正确命中组 | 材质编号 |
| ---: | --- | --- | ---: |
| 1 | `floor` | `floor` | 3 |
| 2 | `back_stone_wall` | `stone` | 1 |
| 3 | `left_stone_wall` | `stone` | 1 |
| 4 | `glass_panel` | `glass` | 2 |
| 5 | `wall_area_light` | `area_light` | 6 |
| 6 | `ceramic_sphere` | `ceramic` | 4 |
| 7 | `metal_sphere` | `metal` | 5 |
| 8 | `stone_plinth` | `stone` | 1 |

灯板是否为光源由映射后的材质 `Emission` 决定。`gatherLights` 从映射后的
发光几何生成光源，因此把灯板错映射到石头会移除对应照明；把普通几何映射
到灯组则会错误地产生光源。

## 单条错映射自检

程序每次运行都会执行以下自检：

1. 以正确映射渲染参考图。
2. 只把 `back_stone_wall` 映射到 `glass`。
3. 按行优先扫描初命中记录与最终颜色，找到第一个不同的像素。
4. 用参考命中确认该像素的对象、正确命中组和材质；再用错映射结果报告实际
   命中组和材质。

默认参数下，报告中的结果形如：

```json
{
  "first_error": {
    "found": true,
    "x": 692,
    "y": 16,
    "object_id": 2,
    "object_name": "back_stone_wall",
    "expected_hit_group_id": 1,
    "expected_hit_group": "stone",
    "actual_hit_group_id": 2,
    "actual_hit_group": "glass",
    "expected_material_id": 1,
    "actual_material_id": 2,
    "difference": ["hit_group", "material_index", "shaded_color"]
  }
}
```

坐标从 0 开始、按 `y` 再 `x` 的行优先顺序扫描。即使着色偶然接近，命中组
或材质编号不同也会被判定为首错误；定位信息始终来自参考图的真实对象。

## 代码结构

- `scene.go` / `geometry.go`：几何、相机与场景对象。
- `mapping.go`：对象 ID、命中组、材质编号及唯一映射入口。
- `render.go` / `shading.go`：初命中、材质着色、阴影与反射/透射。
- `diagnostic.go`：首差异像素定位。
- `output.go` / `report.go`：三张映射/着色图与 JSON 报告。

测试覆盖映射完整性、单条改动、灯光组驱动照明、首错误行优先定位，以及
CLI 的全部 PNG/JSON 产物。
