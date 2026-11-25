# Go 棋盘格相机标定与重投影诊断

纯 Go 标准库实现，零第三方依赖、零图形 SDK。流程包含原生分辨率角点检测、
棋盘格编号、相机内参/畸变/位姿 LM 拟合、反标定投影、去畸变与度量正视渲染、
残差热图，以及“上一张合格结果”保护。

## 运行

```sh
go run .
go test ./...
```

默认生成 960×720 的合成棋盘格帧并写入 `artifacts/`；核心图同时输出 PNG 和纯像素 PPM。检测和输出都保持
960×720，不使用金字塔或缩小整图来换取更高的“看起来完整”的检出率。

## 主要图像

- `artifacts/original_grid.png`：原始分辨率棋盘格、绿色检测网格、黄色拟合重投影点和误差线。
- `artifacts/corrected_grid.png`：按标定参数反向采样得到的去畸变图，青色为校正后角点网格。
- `artifacts/rectified_metric_grid.png`：利用相机和位姿反投影到棋盘平面的度量正视图。
- `artifacts/residual_heatmap.png`：合格参数的角点残差插值热图，固定 2.5 px 色标。
- `artifacts/edge_missing_original_grid.png`：右侧边角被遮挡的漏检诊断；红圈/红十字是网格模型预测但未检出的位置。
- `artifacts/reprojection_distortion_error.png`：将畸变系数置零后的径向残差。
- `artifacts/reprojection_focal_error.png`：焦距放偏 16% 后，由主点向外扩散的尺度误差。
- `artifacts/reprojection_principal_error.png`：主点平移后近似同向的偏移场。

参数扰动的 RMSE、最大误差和热图色标记录在
`artifacts/parameter_effects.json`；完整检测点、标定参数和哈希校验记录在
`artifacts/calibration_report.json`。

## 错误参数回退

`Keeper` 只接受相机参数有限、内参范围合理且重投影 RMSE 不超过阈值（默认 2 px）
的结果。焦距为负、主点为 NaN、RMSE 过大或扰动参数都会被拒绝：

- 不覆盖 `original_grid.png`、`corrected_grid.png`、`residual_heatmap.png`。
- 事件追加到 `artifacts/fallback_events.jsonl`。
- 拒绝详情写入 `artifacts/rejected_update.json`。
- 最近一次合格参数写入 `artifacts/last_good_calibration.json`。
- `calibration_report.json` 的 `previous_result_retained=true` 和前后 SHA-256
  证明正式图像未被坏参数覆盖。

## 代码结构

- `corners.go`：灰度高斯、Harris 响应、NMS、亚像素角点和四象限棋盘角过滤。
- `detect.go`、`gridgrow.go`、`gridmap.go`：网格扩张、多项式投影预测、编号对齐和缺失点输出。
- `camera.go`：针孔 + 径向/切向畸变、投影、去畸变和残差。
- `calibrate.go`：数值雅可比与 Levenberg–Marquardt 联合拟合。
- `synth.go`：原生分辨率合成棋盘格，仅用于生成可复现演示数据。
- `render_grid.go`、`heatmap.go`、`canvas.go`：网格、反标定渲染和热图。
- `store.go`：合格性校验、原子 JSON/PNG 写入和上一张可用结果保留。

当前合成演示的标定 RMSE 约 0.13 px；测试覆盖角点召回、边角漏检、
参数恢复、扰动误差和坏参数回退。
