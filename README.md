# 动画时间采样与重定时（Go 零依赖）

播放帧率固定为 `6 FPS`，但动画样本时间是不规则的（`0.25s` 到 `1.50s` 内共 10 个关键姿态）。
当“样本时间轴”和“播放帧时钟”不同步时，整帧阶跃读取会跳过 `WAVE_PEAK(0.38s)` 与
`JUMP_PEAK(1.04s)` 两个招牌姿态。

本仓库用纯 Go 标准库实现：

- 每个节点独立持有自己的时间样本轨道和绑定姿态；
- 播放时钟按帧号生成时间，再通过 `source = offset + rate * playback` 重定时；
- 支持 `step_hold`（整帧保持）和 `linear`（相邻样本线性插值）；
- 明确标记 `before_sample_range`、`in_range`、`after_sample_range` 与
  `no_samples_static`；
- 没有时间样本的节点永远求值为自身 `bind_pose`，不会按位置、名称或邻近关系借用
  `hero` 的样本；
- 使用标准库 `image/png` 输出三张帧拼图、总拼图、JSON 逐帧报告和中文 HTML 索引。

## 运行

```sh
go run .
go test ./...
```

默认写入 `artifacts/golang/`，避免覆盖仓库根目录中已有的高清中文版拼图基线：

| Go 生成文件 | 内容 |
|---|---|
| `artifacts/golang/01_normal_speed.png` | 原速 `1x`、`6 FPS`、`step_hold`，12 帧；跳过两个招牌姿态 |
| `artifacts/golang/02_slow_motion.png` | 慢放 `0.5x` 重定时、`step_hold`，21 帧；所有关键姿态均被整帧保持 |
| `artifacts/golang/03_interpolation.png` | 原速 `1x`、线性括号插值，12 帧；不跳变，括号覆盖全部关键样本 |
| `artifacts/golang/animation_frame_sheets_all.png` | 三组拼图的纵向总览 |
| `artifacts/golang/timeline_report.json` | 45 帧的时间、状态、括号索引、插值系数和静态节点校验 |
| `artifacts/golang/index.html` | 中文读图说明与三组图片索引 |

也可以指定输出目录：

```sh
go run . /tmp/timing-sheets
```

## 时序模型

第 `i` 个播放帧的时间为：

```text
playback_time = start_time + i / playback_fps
source_time   = retime.offset + retime.rate * playback_time
```

样本读取规则：

- `source_time < first.time`：状态为 `before_sample_range`，钳制到首个样本，卡片用琥珀色边框；
- `source_time > last.time`：状态为 `after_sample_range`，钳制到末个样本，卡片用红色边框；
- 区间内：用二分查找得到 `[lo, hi]` 括号和 `alpha`；
  - `step_hold` 使用 `samples[lo].pose`，因此没有落在播放帧上的关键姿态会被跳过；
  - `linear` 使用 `LerpPose(samples[lo].pose, samples[hi].pose, alpha)`，混合帧的
    `shown_key = -1`，但报告记录其括号是否覆盖某个关键样本。

帧数由样本区间和尾部越界帧决定：

```text
last_output_frame = ceil(last_sample_time / rate * playback_fps)
frame_count       = last_output_frame + 1 + tail_frames
```

因此原速/插值组为 `9 + 1 + 2 = 12` 帧，慢放组为 `18 + 1 + 2 = 21` 帧，三组共 `45` 帧。

## 静态节点隔离

`beacon` 和 `buddy` 的 `samples` 为空。对任意时间和任意读取模式，求值结果都必须满足：

```text
status    = no_samples_static
pose      = node.bind_pose
lo = hi = shown_key = -1
```

`timeline.StaticPolicy` 会跨三组共 45 帧检查这两个节点。端到端测试还会验证 JSON 中
`frames_checked = 45`、`identical_to_bind_pose = true`。

## 代码结构

- `timeline/types.go`：姿态、样本、节点、重定时和报告数据结构；
- `timeline/sampler.go`：轨道预处理、二分采样、钳制、阶跃/线性求值和帧时钟；
- `timeline/analysis.go`：关键姿态覆盖率、跳过列表和静态节点不变量；
- `timeline/demo_scene.json` / `timeline/demo.go`：嵌入的不规则样本 fixture；
- `raster/`：纯标准库 RGB 画布、图元和位图字体；
- `render/`：角色、`beacon`、`buddy`、时间轴标记、状态边框和拼图排版；
- `report/`：JSON/HTML 输出；
- `main.go`：生成全部产物并强制校验关键结论；
- `pipeline_test.go`：帧数、跳过项、静态不变量、产物存在性和越界边框像素测试。

## 依赖

`go.mod` 没有任何 require，仅使用 Go 标准库（`encoding/json`、`image/png`、`math`、
`sort`、`embed` 等）。
