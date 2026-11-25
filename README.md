# 20261126

Repository initialized for 20261126.

## 加速结构写盘、重定位与重新求交

纯 Go 标准库实现的 BVH/实例树序列化演示。原始结构是带 Go 指针的
指针 BVH；写盘时把节点指针拍平成文件节点 ID，并故意把三角形存储顺序
整体逆序。重新加载后通过重定位表把文件 ID 解析到新的运行时节点地址。

### 运行

```sh
go run .
go test -count=1 ./...
```

可选参数：

```sh
go run . --out artifacts --width 640 --height 360 --random-rays 512
```

### 产物

- `artifacts/accel.asb`：自定义小端二进制加速结构，含魔数、版本、
  节点、三角形、重映射索引、深层实例树、排列表和 CRC32。
- `artifacts/original.ppm`：原始指针 BVH 渲染结果与叶包围盒。
- `artifacts/reloaded.ppm`：磁盘重载、重定位后的渲染结果与包围盒；
  当前确定性场景下与 `original.ppm` 逐字节相同。
- `artifacts/diff.ppm`：两路渲染的放大逐像素差分；正确往返时为全黑。
- `artifacts/bounds_mismatch.ppm`：错位包围盒彩色诊断图。
- `artifacts/structure_original.json`、`structure_reloaded.json`：
  新旧节点地址、子节点 ID、AABB、叶索引、实例路径和重定位表。
- `artifacts/validation.json`、`artifacts/report.txt`：结构、射线、
  CRC、像素差分和颜色像素统计。

### 重定位与校验

写盘链路为：

```text
*PointerNode -> DFS 文件节点 ID -> DiskNode.Left/Right
             -> 逆序三角形 + 新的 Order/排列表
```

重载链路为：

```text
CRC/魔数/长度校验 -> []RuntimeNode -> FileID -> *RuntimeNode
                  -> 重建实例父子指针 -> 重算 World/WorldInv
```

校验器会比较节点数量、AABB、父子拓扑、叶内三角形的稳定
`OriginalIndex`、实例路径和求交结果。三角形在文件中的位置虽然改变，
BVH 的 `Order` 也会同步重映射，因此不会用过时下标访问错误三角形。

### 边界与深层实例

场景中央的金色三角形是稳定目标表面 `surface_id=1001`。内置探针覆盖：

- 三条边的精确擦边射线、近边/近顶点射线和小角度掠射射线；
- 穿过点状、重复顶点和共线退化三角形的“门射线”，退化三角形必须被
  零面积检测拒绝，射线继续命中金色表面；
- 12 层变换组加叶节点共 15 层实例路径；每条局部探针都会用叶实例的
  `World` 矩阵变换成世界射线，验证深层实例仍落在同一金色表面。

另有确定性随机世界射线比较原始结构与重载结构的命中 ID、原始三角形 ID、
`t`、交点和法线。

### 错位包围盒颜色

`bounds_mismatch.ppm` 中：

- 绿色：重定位正确的 AABB；
- 红色：模拟旧地址/旧三角形下标导致的错位 AABB；
- 黄色：旧盒中心到正确盒中心的位移向量。

真实往返结果要求 AABB 错位为 0；红色盒只是主动注入的可视化故障样本，
并由 `TestInjectedBoundsMismatchIsDetected` 验证校验器能够捕获。

整个模块没有第三方依赖，`go.mod` 不需要任何 `require`。
