# 对比度检测自动对焦（Go，无图形 SDK）

这个仓库实现一个可嵌入的对比度检测 AF 引擎，并用确定性合成场景生成调试产物：

- 中心 ROI 的 Tenengrad 对比度清晰度检测；
- 从最近端点开始的单向粗步进搜索；
- 粗峰附近只走一遍的细步进精修；
- 2 帧确认后锁定，锁定后使用 18% 回差和连续弱帧计数避免拉风箱；
- 主体运动门控，主体进入画面后才开始搜索；
- 十帧缩略图、清晰度曲线、焦点位置 HUD 和 JSON 遥测。

仅使用 Go 标准库，不接相机或图形 SDK。镜头通过整数位置抽象，调用方负责执行马达命令并采集下一帧。

## 快速运行

```bash
go test ./...
go run ./cmd/afdemo -out artifacts -frames 160
```

演示输出：

```text
locked=43 true_depth=44 frame=61 moves=22 reversals=2 after_lock=0
```

即：第 61 帧锁定到位置 43（合成主体真值 44），搜索期间发生 2 次局部方向变化，锁定后移动次数为 0。

## 诊断产物

- `artifacts/ten_frames.png`：5x2 接触表，包含十帧缩略图、绿色 ROI、帧号、焦点位置、清晰度和位置条。
- `artifacts/sharpness_curve.png`：清晰度-焦点位置曲线，区分粗扫、细扫、确认点，并绘制锁定位置、回差带和位置覆盖。
- `artifacts/focus_overlay.png`：最终锁定帧的 ROI、近/远焦位置标尺和 AF HUD。
- `artifacts/af_report.json`：配置、最终快照、搜索样本和生成信息。

![十帧缩略图](artifacts/ten_frames.png)

![清晰度曲线](artifacts/sharpness_curve.png)

![焦点位置覆盖](artifacts/focus_overlay.png)

## 算法流程

1. **清晰度检测**：把 ROI 转成亮度，计算 Sobel 梯度平方均值。纹理越清晰，分数越高；同时输出亮度方差作为对比度辅助量。
2. **主体门控**：比较相邻帧 ROI 亮度差。连续 `PresenceFrames` 帧超过阈值才认为主体进入；安静 `AbsenceFrames` 帧后才释放。
3. **粗扫**：从离当前位置最近的端点开始，沿单一方向按 `CoarseStep` 前进。连续两个采样明显低于当前峰值后停止，不进行全程往返扫描。
4. **细扫**：以粗峰为基准，覆盖峰值前半个粗步长到峰值后一个粗步长；沿粗扫反方向只走一遍，`FineStep` 为 1。
5. **带确认的锁定**：移动到细扫最优点后连续确认 `LockConfirmFrames` 帧，再进入锁定。
6. **回差保持**：锁定后分数高于 `lockScore * (1 - LockHysteresis)` 时不动作。只有连续 `UnlockFrames` 帧低于回差下限，才允许重新捕获；稳定主体不会持续拉风箱。

## 摄像头接入

核心接口只依赖 `image.Image` 和当前位置：

```go
engine, err := autofocus.New(autofocus.DefaultConfig())
if err != nil {
    return err
}

position := motor.Position()
for {
    frame := camera.Capture()
    decision := engine.Update(frame, position)
    if decision.MoveTo >= 0 {
        motor.MoveTo(decision.MoveTo)
        position = decision.MoveTo
    }
}
```

`SettleFrames` 用于丢弃马达到位后的机械/曝光稳定帧。当前演示内置了基于运动的主体门控；若上层已有人形/目标检测器，可调用 `engine.NotifySubject(true/false)` 注入外部判断，避免静态主体因缺少帧差而无法启动搜索。

## 目录

- `autofocus/`：清晰度指标、主体门控、搜索状态机、回差锁定。
- `sim/`：无外部依赖的合成场景和深度相关散焦。
- `report/`：PNG 接触表、曲线、HUD 和文件输出。
- `cmd/afdemo/`：端到端演示入口。

`go test ./...` 覆盖峰值位置、主体门控、锁定精度、锁定后零移动和诊断画布尺寸。
