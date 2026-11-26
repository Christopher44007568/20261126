# USDA-style layer stack (Go)

仅使用 Go 标准库实现一个自定义 USDA 子集解析、层叠合成和 PPM 渲染器。

## 运行

```bash
go run .
go test ./...
```

程序每次运行都会重写 `layers/` 与 `artifacts/` 中的确定性产物：

- `layers/01_base.usda` ... `layers/04_spot.usda`：四个原始 USDA 层。
- `artifacts/merged_scene.usda`：合并后的场景，每个颜色属性旁标注最强 opinion 来源。
- `artifacts/final.ppm`：最终 P6 PPM 画面。
- `artifacts/opinion_sources.txt` / `.json`：所有属性的 winner 及强弱 opinion 栈。
- `artifacts/scene_forward_order.ppm` / `scene_reverse_order.ppm`：正序、逆序加载结果。
- `artifacts/permutation_check.txt`：全部 24 种加载顺序的哈希校验。

## 合成规则

1. 颜色 opinion 只按层内声明的 `layerStrength` 比较，较大的层获胜。
2. 加载顺序不参与 opinion 排序；同强度时按层 ID 字典序，同层内后出现的行获胜。
3. `def`、`over`、`class` 使用相同的相对子路径拼接规则。
4. 嵌套的局部 `over` 只向既有路径添加 opinion，不会移动或复制子节点路径。

核心样例 `/Stage/StackProbe` 同时收到四层颜色 opinion：
`L4-spot(40) > L3-cool(30) > L2-warm(25) > L1-base(10)`，最终 winner 恒为
`L4-spot`，并在合并场景和画面中标注其来源文件与行号。
