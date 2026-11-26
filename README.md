# 20261126 — hdrlab

三帧包围曝光（欠曝 / 正常 / 过曝）对齐并合成 HDR 预览的纯 Go 工具，
仅使用标准库，不依赖任何图形 SDK。输入输出均为 PPM（同时另存 PNG 便于查看）。

## 管线

1. **合成包围曝光**：`scene.go` 生成带高光细节（广告牌板条/方块/暗条）与
   运动物体（汽车）的线性辐射场景，按 ±2EV 渲染三帧，并施加手持微位移
   （欠曝 +3/-2，过曝 -3/+2）与传感器噪声。
2. **对齐**：`align.go` 在 ±5px 内做全搜索，以曝光补偿后亮度差的截尾均值
   为得分，运动物体不会带偏估计。
3. **HDR 合成**：`merge.go` 使用 Debevec 帽式权重加权融合；以参考帧为锚
   做去鬼影——非参考帧与参考帧辐射值差异超阈即剔除该帧（饱和通道按
   下界处理），运动物体只保留参考帧位置，不会拖出多重边缘。
4. **可视化**：测光 12×8 网格（EV/裁剪/阴影）、对齐误差 jet 色图、
   鬼影簇品红标记、位移矢量插图、对比表。

## 运行

```sh
go run ./cmd/hdrlab artifacts   # 生成全部产物到 artifacts/
go test ./...                   # 端到端验收测试
```

产物：`bracket_{under,normal,over}`、`direct_average`、`aligned_average`、
`hdr_preview`、`hdr_metering`、`hdr_overlay`、`alignment_error`、
`comparison_sheet`（均为 .ppm + .png）以及 `hdr_report.json`
（对齐真值对比、测光矩阵、高光边缘能量、运动边缘计数与验收结论）。
