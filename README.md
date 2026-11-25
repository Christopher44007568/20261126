# 20261126

Repository initialized for 20261126.

# 不可变 BLAS 与可更新 TLAS 角色演示

纯 Go 标准库实现的小型软件光线追踪场景。角色的每个部件网格只构建一次
BLAS/BVH；四帧之间复用这些不可变 BLAS，只更新 TLAS 中的实例矩阵、叶节点
包围盒并自底向上重拟合节点包围盒。

## 运行

```sh
go run .
go test ./...
```

默认生成：

- `artifacts/pose_0.ppm` 到 `artifacts/pose_3.ppm`：四个连续姿态。
- `artifacts/report.json`：BLAS/TLAS 数量、输出路径、重拟合次数和耗时。

可使用 `go run . --out /tmp/poses --width 640 --height 400` 调整输出位置或尺寸。

## 高亮含义

- 琥珀色：`left_arm`。每帧都更新了实例矩阵，但故意不发布新的 TLAS 叶包围盒；
  琥珀色线框是残留在 TLAS 中的旧 AABB，当前手臂仍以琥珀色实体标出。
- 品红色：`right_leg`。步态更新被故意跳过，因此实例矩阵和包围盒都停在第一帧。

相机、灯光、地面和采样规则在四帧间完全相同；除角色运动和用于诊断的高亮外，
没有其他场景变化。`TestStaleBoundsCanHideMovedInstance` 验证旧包围盒会让
TLAS 错过新位置，而直接查询共享 BLAS 仍能命中。
