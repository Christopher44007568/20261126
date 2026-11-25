package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"os"
)

const assetMagic = "ACCREL01"
const assetVersion uint16 = 1

type diskHeader struct {
	Magic            [8]byte
	Version          uint16
	HeaderSize       uint16
	RootNode         int32
	RootInstance     int32
	NodeCount        uint32
	TriangleCount    uint32
	OrderCount       uint32
	InstanceCount    uint32
	PermutationCount uint32
	Flags            uint32
	PayloadCRC       uint32
}

type DiskNode struct {
	Bounds AABB
	Left   int32
	Right  int32
	First  uint32
	Count  uint32
}

type DiskTriangle struct {
	A             Vec3
	B             Vec3
	C             Vec3
	Normal        Vec3
	Color         RGB
	SurfaceID     uint64
	OriginalIndex int32
	Flags         uint32
}

type DiskInstance struct {
	StableID   uint32
	Parent     int32
	Depth      uint32
	MeshID     int32
	ChildCount uint32
	Local      Mat4
}

type AssetFile struct {
	Header         diskHeader
	Nodes          []DiskNode
	Triangles      []Triangle
	Order          []uint32
	Instances      []DiskInstance
	InstanceNames  []string
	DiskToOriginal []uint32
	Checksum       uint32
}

type flattenInstance struct {
	record DiskInstance
	name   string
}

func SaveAsset(path string, scene *DemoScene) (*AssetFile, error) {
	if scene == nil || scene.BVH == nil || scene.BVH.Root == nil {
		return nil, fmt.Errorf("cannot save empty acceleration structure")
	}

	pointerNodes := CollectPointerNodes(scene.BVH.Root)
	nodes := make([]DiskNode, len(pointerNodes))
	for i, node := range pointerNodes {
		rec := DiskNode{
			Bounds: node.Bounds,
			Left:   -1,
			Right:  -1,
			First:  uint32(node.First),
			Count:  uint32(node.Count),
		}
		if node.Left != nil {
			rec.Left = int32(node.Left.SerializedID)
		}
		if node.Right != nil {
			rec.Right = int32(node.Right.SerializedID)
		}
		nodes[i] = rec
	}

	triangleCount := len(scene.Triangles)
	diskTriangles := make([]Triangle, triangleCount)
	diskToOriginal := make([]uint32, triangleCount)
	originalToDisk := make([]uint32, triangleCount)
	for diskPos := 0; diskPos < triangleCount; diskPos++ {
		originalIndex := triangleCount - 1 - diskPos
		diskToOriginal[diskPos] = uint32(originalIndex)
		originalToDisk[originalIndex] = uint32(diskPos)
		diskTriangles[diskPos] = scene.Triangles[originalIndex]
	}

	diskOrder := make([]uint32, len(scene.BVH.Order))
	for i, originalTri := range scene.BVH.Order {
		diskOrder[i] = originalToDisk[originalTri]
	}

	flatInstances := flattenInstances(scene.Root)
	instances := make([]DiskInstance, len(flatInstances))
	names := make([]string, len(flatInstances))
	for i, item := range flatInstances {
		instances[i] = item.record
		names[i] = item.name
	}

	header := diskHeader{
		Version:          assetVersion,
		HeaderSize:       uint16(binary.Size(diskHeader{})),
		RootNode:         0,
		RootInstance:     0,
		NodeCount:        uint32(len(nodes)),
		TriangleCount:    uint32(triangleCount),
		OrderCount:       uint32(len(diskOrder)),
		InstanceCount:    uint32(len(instances)),
		PermutationCount: uint32(len(diskToOriginal)),
		Flags:            1,
	}
	copy(header.Magic[:], assetMagic)

	var body bytes.Buffer
	if err := binary.Write(&body, binary.LittleEndian, nodes); err != nil {
		return nil, err
	}
	for _, tri := range diskTriangles {
		rec := diskTriangleFromRuntime(tri)
		if err := binary.Write(&body, binary.LittleEndian, rec); err != nil {
			return nil, err
		}
	}
	if err := binary.Write(&body, binary.LittleEndian, diskOrder); err != nil {
		return nil, err
	}
	for i, inst := range instances {
		if err := binary.Write(&body, binary.LittleEndian, inst); err != nil {
			return nil, err
		}
		name := names[i]
		if len(name) > 65535 {
			name = name[:65535]
		}
		if err := binary.Write(&body, binary.LittleEndian, uint16(len(name))); err != nil {
			return nil, err
		}
		if _, err := body.WriteString(name); err != nil {
			return nil, err
		}
	}
	if err := binary.Write(&body, binary.LittleEndian, diskToOriginal); err != nil {
		return nil, err
	}

	header.PayloadCRC = crc32.ChecksumIEEE(body.Bytes())
	var file bytes.Buffer
	if err := binary.Write(&file, binary.LittleEndian, header); err != nil {
		return nil, err
	}
	if _, err := file.Write(body.Bytes()); err != nil {
		return nil, err
	}
	checksum := crc32.ChecksumIEEE(file.Bytes())
	if err := binary.Write(&file, binary.LittleEndian, checksum); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, file.Bytes(), 0o644); err != nil {
		return nil, err
	}

	return &AssetFile{
		Header:         header,
		Nodes:          nodes,
		Triangles:      diskTriangles,
		Order:          diskOrder,
		Instances:      instances,
		InstanceNames:  names,
		DiskToOriginal: diskToOriginal,
		Checksum:       checksum,
	}, nil
}

func flattenInstances(root *Instance) []flattenInstance {
	idToFile := map[uint32]int32{}
	var out []flattenInstance
	var walk func(inst *Instance, parent int32)
	walk = func(inst *Instance, parent int32) {
		fileID := int32(len(out))
		idToFile[inst.ID] = fileID
		childCount := uint32(len(inst.Children))
		out = append(out, flattenInstance{
			name: inst.Name,
			record: DiskInstance{
				StableID:   inst.ID,
				Parent:     parent,
				Depth:      uint32(inst.Depth),
				MeshID:     inst.MeshID,
				ChildCount: childCount,
				Local:      inst.Local,
			},
		})
		for _, child := range inst.Children {
			walk(child, fileID)
		}
	}
	walk(root, -1)
	return out
}

func diskTriangleFromRuntime(t Triangle) DiskTriangle {
	flags := uint32(0)
	if t.Degenerate {
		flags |= 1
	}
	return DiskTriangle{
		A:             t.V[0],
		B:             t.V[1],
		C:             t.V[2],
		Normal:        t.Normal,
		Color:         t.Color,
		SurfaceID:     t.SurfaceID,
		OriginalIndex: int32(t.OriginalIndex),
		Flags:         flags,
	}
}

func (d DiskTriangle) Runtime() Triangle {
	return Triangle{
		V:             [3]Vec3{d.A, d.B, d.C},
		Normal:        d.Normal,
		Color:         d.Color,
		SurfaceID:     d.SurfaceID,
		OriginalIndex: int(d.OriginalIndex),
		Degenerate:    d.Flags&1 != 0,
	}
}
