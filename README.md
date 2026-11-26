# 10-bit / 12-bit 渐变管线到 8-bit 显示

纯 Go 标准库实现的位深对比实验。三种场景共用同一份浮点源、同一几何和同一条色相路径，只在浮点源之后分叉为 8-bit 直出、10-bit 主数据、12-bit 主数据，再降到 8-bit 显示。

![三场景位深拼图](artifacts/bit_gradient_puzzle.png)

## 结果

- 暗部阶梯：中性亮度从 8-bit code 1 到 56，叠加硬阶梯边沿刻度与理想亮度轨迹。
- 彩色坡：HSV 色相从 300° 连续升到 660°，跨越 360°/0° 红点；切换位深只改量化，不重置色相。
- 高光截断：暖色 HDR 曝光坡按 R 58% → G 70% → B 90% 顺序截断，洋红斜线标记截断区。
- 量化误差：每个面板底部为逐列平均绝对误差热条，右下角附 0–0.75 code 色标。
- 可见层次：4×4 Bayer 抖动把 10/12-bit 保留的小数 8-bit 坐标变成空间密度，而不是伪造额外 8-bit 码字。

亚码梯子直接展示关键差异：

![亚码抖动梯子](artifacts/subcode_dither_ladder.png)

同一对相邻 8-bit code 之间，12-bit 可以形成 16 个不同 Bayer 密度，10-bit 约 4–5 个，8-bit 直出只有 2 个。下方 8 px 是实际 8-bit 颜色，肉眼接近纯色；差异确实发生在空间平均，而不是 8-bit 硬码数量。

## 实现

`internal/render/color.go` 是核心管线：

1. 同一浮点源采样，不受位深影响。
2. `Quantize` 将每通道舍入到 `(1<<bits)-1` 级。
3. 10/12-bit 主数据乘 255 后保留小数部分。
4. 4×4 有序 Bayer 阈值决定向上或向下取整，最终仍是 8-bit 码字。
5. 8-bit 直出不做补偿，只取最近整数码。

`master_10bit_16bit.png` 和 `master_12bit_16bit.png` 是真正的 16-bit PNG，用 16-bit 通道承载精确的 10/12-bit 整数层级。三张 8-bit sheet 和总拼图都由同一源函数重新采样。

色相连续性由彩色坡单独统计：`metrics.json` 中 `hue_switches` 比较 8↔10、10↔12、8↔12 的逐像素色相，最大差异低于 1°。暗部为中性色，无有效色相；高光包含正确截断到白色，因此色相指标标记为 `hue_applicable: false`，避免把消色截断误算成色相跳变。

## 运行

```bash
go run . -out artifacts
go test ./...
```

无第三方依赖，`go.mod` 只声明模块本身。

输出：

- `bit_gradient_puzzle.png`：3 场景 × 3 管线的总拼图。
- `sheet_8bit.png` / `sheet_10bit_to_8bit.png` / `sheet_12bit_to_8bit.png`：几何完全一致的可切换 sheet。
- `subcode_dither_ladder.png`：相邻 8-bit 码之间的亚码密度梯子。
- `master_10bit_16bit.png` / `master_12bit_16bit.png`：16-bit 主数据。
- `metrics.json`：误差、色相、截断位置、硬色数与梯子密度。
- `index.html`：本地总览与位深切换页。

## 读图注意

8-bit 显示每通道最多 256 个硬码字，这是物理上限。10/12-bit 不会突破它；收益是更准确的舍入、更多可区分的 Bayer 相位，以及 4×4 块平均后更小的误差。逐像素误差会因为抖动在相邻码之间切换而略升，这是空间平均的代价，应结合 `dither_block_rmse_code` 和亚码梯子一起看。
