# Octahedral Normal Compression

纯 Go 标准库实现的八面体法线压缩示例，不使用图形 SDK。每个法线只保存两个量化通道，第三分量由八面体折叠关系重建，并显式保留负 Z 切线空间方向。

## 运行

```bash
go run . -size 512 -bits 8 -out artifacts
go test ./...
```

`size` 控制方形预览边长，`bits` 控制每个存储通道的量化位数，范围为 1 到 16。

## 输出

- `artifacts/original_normal_sphere.png`：原始法线球，RGB 直接映射切线空间 XYZ。
- `artifacts/compressed_normal_sphere.png`：两通道量化后重建的法线球。
- `artifacts/angular_error_heatmap.png`：每个像素的角误差热图。
- `artifacts/lighting_comparison.png`：左原始、右压缩的 Blinn-Phong 光照对照。
- `artifacts/packed_octahedral_rg.png`：实际打包预览，只有 R/G 承载数据，B 始终为零。

512 像素、8-bit/通道的一次默认输出：

- 平均角误差约 `0.31540°`。
- 最大角误差约 `0.93593°`。
- 球体采样同时覆盖正 Z 与负 Z 切线空间半球。

## 编码与重建

编码先把单位法线投影到 L1 八面体：

```text
u = x / (|x| + |y| + |z|)
v = y / (|x| + |y| + |z|)
```

当 `z < 0` 时对 `(u, v)` 做八面体折叠。解码时先计算 `z = 1 - |u| - |v|`；如果 `z < 0`，必须通过折叠逆运算恢复 `x/y`，最后重新归一化。

着色器中对应的关键形式如下：

```glsl
float signNotZero(float v) {
    return v < 0.0 ? -1.0 : 1.0;
}

vec3 decodeOctahedral(vec2 e) {
    vec3 n = vec3(e.x, e.y, 1.0 - abs(e.x) - abs(e.y));
    if (n.z < 0.0) {
        n.xy = vec2(
            (1.0 - abs(n.y)) * signNotZero(n.x),
            (1.0 - abs(n.x)) * signNotZero(n.y)
        );
    }
    return normalize(n);
}
```

`signNotZero` 对零值返回正号，确保无符号边界不会随机翻转。负 Z 的符号由折叠关系恢复，不能写成 `z = abs(1 - abs(u) - abs(v))`；那会把背光表面错误地翻到正 Z，形成整块假高光。测试 `TestAbsoluteThirdComponentCreatesFalseLight` 明确覆盖这个错误。
