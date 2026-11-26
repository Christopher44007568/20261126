# channelpack

纯 Go 标准库实现的材质通道打包对比程序：生成 albedo、切线空间法线、
粗糙度和环境遮蔽（AO）四张 8-bit 贴图，比较三种打包布局，并输出材质球、
通道分解拼图和逐通道误差热力图。

## 运行

```sh
cd channelpack
go run .
```

默认把 PNG 和 `report.txt` 写入仓库根的 `artifacts/out`（即默认的 `../artifacts/out`）。可用参数：

```sh
go run . -out /tmp/out -size 256 -sphere 360
```

## 三种布局

- `separate`：四张独立 RGBA 贴图，标量复制到 RGB，未用的 alpha 填 255。
- `packed`：两张 RGBA。贴图 1 为 `RGB=albedo, A=AO`；贴图 2 为
  `RGB=normal XYZ, A=roughness`。八个源通道均保持独立 8-bit 槽位。
- `compact`：两张 RGBA。粗糙度与 AO 各压成 4-bit 后共用贴图 1 的 alpha
  （高/低 nibble）；albedo 与法线的每个字节仍是完整独立的 8-bit 槽位。

`decodeMisaligned` 故意用错误的槽位表解码 packed 贴图，模拟通道绑定错位，
让材质球出现非法颜色和破碎光照，并在误差图中暴露差异。

## 正确性保证

程序在主流程内执行断言，失败时以非零状态退出：

- `separate` 与 `packed`：所有八个通道逐像素字节一致，法线最大夹角为 0。
- 解码重建的 albedo RGB 与原图逐像素一致（三种布局均为 `max_abs=0`）。
- `compact` 的 4-bit 量化只作用于粗糙度/AO，误差不超过 8 个色阶；
  albedo 与法线三通道仍为 `max_abs=0`，法线最大夹角为 0。
- 独立性测试：只扰动粗糙度和 AO 后重新编码，解码出的 albedo 与法线字节
  完全不变，证明标量通道的量化无法影响法线方向或颜色。
- 错位解码必须检测到大量差异，确保该演示确实触发了槽位绑定错误。

## 输出文件

- `source_*.png`：四张源贴图。
- `<layout>_textureN.png`：各布局实际打包后的 RGBA 贴图。
- `channels_<layout>.png`：每张打包贴图的 R/G/B/A 分解拼图。
- `error_<layout>.png`：八通道误差热力图（差值放大 8 倍）。
- `sphere_*.png`：使用解码贴图着色的材质球。
- `comparison_source_packed_misaligned.png`：源、正确打包、错位三球对比。
- `report.txt`：逐通道 `max_abs`、差异像素数、RMSE 与法线角度误差。
