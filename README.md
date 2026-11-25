# 20261126

Repository initialized for 20261126.

## metering/ — 测光与曝光模拟（Go，零外部依赖）

先测光再曝光：对"暗室 + 窗口高光 + 移动主体"的合成场景按时间序列模拟
三种测光模式，并附一个移动点测光用于验证曝光的时间稳定性。

- 全画面平均测光（`fullframe`）
- 中央重点测光（`center`，高斯权重）
- 高光保护测光（`highlight`，加权 90% 亮度分位对齐到 0.85，
  并相对中央重点 EV 钳制最多 2 档欠曝，主体不会被压成黑块）
- 移动点测光（`spot`，演示用）：EMA + 每帧最大 0.35 EV 斜率限制，
  测光点移动时曝光不会每帧乱跳

运行：

```sh
cd metering && go run .
```

每帧每种模式输出到 `metering/out/<mode>/`：

- `frame_XX_photo.png` — 曝光后的照片
- `frame_XX_meter.png` — 测光区域权重覆盖图（红色越亮权重越高，
  点测光带青色十字标记）
- `frame_XX_hist.png` — 亮度直方图（绿线为 18% 中灰）

控制台同时打印各模式原始/平滑 EV 对照表与最大帧间跳变。

测试：`cd metering && go test ./...`
