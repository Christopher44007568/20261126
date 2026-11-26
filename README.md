# VRAM budget acceleration-structure demo

纯 Go 标准库实现的确定性 BVH 预算降级示例。场景是一个径向多尖刺徽章：高模有
80 个角分段，压缩档使用 8 位量化顶点和紧凑 BVH，低模回退使用 16 分段粗模。
程序会逐像素比较各档与完整画面的覆盖率，并用高对比边框描出预算降级造成的
轮廓误差。

## 运行

```bash
go run .
go test ./...
```

默认输出到 `artifacts/`，也可以指定目录或分辨率：

```bash
go run . -out artifacts -width 480 -height 360 -detail 80 -low-detail 16
```

## PPM 产物

- `artifacts/full.ppm`：完整 BVH 与完整高模，绿色外框表示高预算档。
- `artifacts/compressed.ppm`：逐出完整节点后使用的量化网格和 16 字节紧凑 BVH
  节点，橙色外框表示压缩档；洋红/黄色描边是相对完整画面的轮廓偏差。
- `artifacts/fallback.ppm`：压缩结构仍超预算时，逐出高模和压缩节点，切换到
  低模 BVH，红色外框表示回退档；更粗的洋红/黄色描边标出缺失或多出的轮廓。
- `artifacts/restored.ppm`：预算恢复后重新构建的完整画面，仅作为额外验证产物。
  它与未加框的完整渲染逐像素一致，报告中记录相同 FNV 指纹。
- `artifacts/report.json`：三档预算、实际驻留字节、节点/三角形数量、差异像素、
  陈旧句柄拒绝结果、逐出次数、代际和恢复指纹。

PPM 使用二进制 `P6` 格式；可用 ImageMagick `convert`、`pnmtopng` 或支持 PPM 的
看图器打开。

## 三档预算

预算不是只按节点数估算，而是按显式 GPU 布局字节记账：

- 完整节点：40 B/节点，索引 4 B/项，三角形布局 40 B/个。
- 压缩节点：16 B/节点，`uint16` 索引 2 B/项，量化三角形 12 B/个。
- 低模包：复用完整 BVH 布局，但三角形和节点数量显著减少。

`NewBudgetPlan` 在测量三个包的实际大小后生成高、中、低三个限额。中档限额保证
压缩包放得下而完整包放不下；低档限额保证低模包放得下而压缩包放不下。每档切换
都先调用 `Package.Evict` 释放旧节点和几何引用，再构建新包。

## 过期节点防护

`Manager` 维护单调递增代际。每次预算切换或恢复重建都会产生新的 `Handle`：

- 旧句柄的代际与当前代际不一致时，`Manager.Render` 返回 `ErrStaleHandle`。
- 被逐出的 `Package` 标记为 `Evicted`，即使绕过管理器直接渲染也会拒绝。
- `RestoreFull` 总是逐出降级包并执行 `NewFullPackage`，不会复用降级前的旧
  `BVH.Nodes` 切片。

测试覆盖 BVH 与暴力求交一致性、量化 BVH 与量化网格一致性、三档预算大小关系、
旧句柄失效、低模轮廓误差、恢复图像逐像素相同以及 PPM 往返读写。
