# AE dark-to-window lab

纯 Go、零外部依赖的自动曝光时间响应实验。程序程序化渲染一个 HDR 室内场景：镜头从有书架、门洞和植物的暗部转向带窗棂、楼宇、树枝和天空高光的窗口，在 10 秒、30 FPS 下比较三种 AE 时间响应。

## 运行

```bash
go run . -out artifacts -jpeg-quality 72
go test ./...
```

只使用 Go 标准库；`go.mod` 没有任何第三方 require。

## 三种响应

- `rapid_breathing`：高增益欠阻尼对照，转向后有明显过冲和来回收敛。
- `fast_smooth`：推荐曲线；起步受 EV 步进限制，接近目标时提高阻尼，快速且无明显过冲。
- `gentle_slow`：过阻尼慢响应；无过冲，但窗口曝光锁定更晚。

## 导出物

运行后在 `artifacts/` 下生成：

- `frames/<profile>/frame_0001.jpg` ... `frame_0300.jpg`：每种响应 300 张连续帧缩略图，共 900 张，480×270。
- `exposure_values.csv`：逐帧测光、目标 EV、实际 EV、线性增益、速度、帧间亮度和 ROI 细节指标。
- `overshoot_curves.csv`：宽表格式的三响应实际 EV、相对最终曝光的有符号过冲量和收敛百分比。
- `ae_metrics.csv`：稳定时间、最大过冲、零穿越次数、EV/亮度跳变、窗口边缘/裁切、暗部均值/标准差。
- `exposure_curves.svg` / `exposure_curves.png`：曝光值曲线，虚线为测光目标。
- `overshoot_curves.svg` / `overshoot_curves.png`：过冲与收敛曲线。
- `contact_sheet_<profile>.jpg`：每 10 帧取样的连续过程总览。
- `all_frames_sheet_<profile>.jpg`：该响应全部 300 帧的顺序拼图。
- `report.html`：曲线、拼图、关键帧和指标汇总。

## 验收口径

推荐档 `fast_smooth` 的 EV 步进被限制在 `0.225 EV/frame` 内，实测最大输出亮度跳变低于 `0.022`；最终窗口 ROI 高光灯裁切约 6%，窗口边缘能量约 `0.051`，暗部 ROI 平均亮度约 `0.23`、标准差约 `0.098`，因此窗格/楼宇和室内暗部层次可同时辨认。完整最新数值以 `artifacts/ae_metrics.csv` 为准。
