# 20261126

## Shadow ray flag 对比演示

纯 Go（仅标准库）实现的阴影光线 flag 演示，渲染同一场景的三张阴影图：

- `artifacts/shadow_backface_cull.png`：背面剔除，仅正面命中遮挡。
- `artifacts/shadow_double_sided.png`：双面材质接受背面命中，单面背面仍忽略。
- `artifacts/shadow_reflection_closure.png`：反射闭包，封闭体进/出命中合并为一次能量事件。

颜色图例与设计细节见 `docs/DESIGN.md`。

```sh
go test ./...
go run ./cmd/shadowflags
```
