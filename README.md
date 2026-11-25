# 光线微分驱动的反射 LOD 选择 (raydiff)

纯 Go（仅标准库）实现的微分光线追踪器。次级反射光线比主光线更粗糙，
本项目通过传播光线微分（ray differentials）为主射线与次级反射射线分别
计算纹理足迹并选择 mip 层级，避免远处金属反射因沿用主射线足迹而产生
噪点/闪烁。

## 输出

运行 `go run .` 在 `out/` 生成三张图：

- `beauty.png`：主色图，含地面着色与金属 Fresnel 反射。
- `reflection.png`：反射图，仅展示每条反射射线带回的颜色。
- `lod.png`：纹理层级色块图，每个 mip 层级映射为一种颜色，小数层级
  在相邻颜色间线性混合，因此可直接观察层级是否连续切换。

`go run . -naive` 额外生成 `*_naive.png`，强制次级射线复用主射线 LOD，
用于对比演示远处金属的错误采样（球体退化为均匀低层级色块）。

## 算法

### 1. 主射线微分

针孔相机所有像素共享原点，故 `dO=0`；方向对像素坐标求导
（`internal/camera.go`）：

```
D     = fwd + tan(fov/2) * (sx*right + sy*up)
dD/dx = tan(fov/2) * (2/width)  * right
dD/dy = tan(fov/2) * (-2/height) * up
```

### 2. 求交点与足迹

对球面/平面解析求交并对曲面方程求导得到 `dt`，再由
`dP = dO + t*dD + D*dt` 得到像素在表面的世界空间足迹，同时给出
法线微分 `dN`（球面 `dN=dP/R`，平面为 0）。见 `internal/geometry.go`。

### 3. 反射微分与粗糙度展宽

反射方向 `R=reflect(D,n)` 的微分（Igehy 1999，`internal/reflection.go`）：

```
dn = dD·n + D·dN
dR = dD - 2*(dn*n + (D·n)*dN)
```

粗糙度按 GGX `alpha=roughness^2` 将反射方向微分沿切线方向展宽
（展宽系数 `1+64*alpha^2`），镜面（roughness=0）系数为 1，保证随
粗糙度连续变化；展宽量做上限钳制防止失控。

### 4. 基于足迹的 LOD

将世界足迹投影到纹理 UV，取 texel 足迹较长边
`rho=max(|dUVdx|,|dUVdy|)`，`LOD=clamp(log2(rho), 0, maxMip)`，
并做三线性过滤（`internal/texture.go`）。

屏幕边缘/掠射角导致的发散通过两道保护处理：UV 微分中的 NaN/Inf 归零，
足迹幅度钳制到 `2^13` texel；LOD 再钳制到 `[0, maxMip]`，因此边缘
不会跳到非法层级。

## 目录

- `internal/vec.go` 向量数学
- `internal/camera.go` 主射线及其微分
- `internal/ray.go` 光线结构与足迹传递
- `internal/geometry.go` 球/平面微分求交
- `internal/reflection.go` 反射微分与粗糙度展宽
- `internal/texture.go` mip 链与足迹 LOD 采样
- `internal/procedural.go` 棋盘格/天空纹理与 UV 微分
- `internal/renderer.go` 场景追踪、递归反射与缓冲输出
- `internal/shading.go` Fresnel、光照、LOD 调色板

## 验证

```
go test ./...
```

覆盖主射线微分、足迹随距离增大、镜面守恒、粗糙度展宽、LOD 单调连续、
边缘发散钳制、屏幕同表面连续性等性质。
