package main

import "fmt"

type NodeDump struct {
	ID           int32  `json:"id"`
	Address      string `json:"address"`
	LeftID       int32  `json:"left_id"`
	RightID      int32  `json:"right_id"`
	LeftAddress  string `json:"left_address,omitempty"`
	RightAddress string `json:"right_address,omitempty"`
	Bounds       AABB   `json:"bounds"`
	First        int    `json:"first"`
	Count        int    `json:"count"`
	OrderIndices []int  `json:"order_indices,omitempty"`
}

type InstanceDump struct {
	FileID   int32   `json:"file_id"`
	StableID uint32  `json:"stable_id"`
	Name     string  `json:"name"`
	Address  string  `json:"address"`
	ParentID int32   `json:"parent_id"`
	Depth    int     `json:"depth"`
	MeshID   int32   `json:"mesh_id"`
	ChildIDs []int32 `json:"child_file_ids"`
	World    Mat4    `json:"world_matrix"`
}

type StructureDump struct {
	Kind                 string           `json:"kind"`
	DiskFormat           string           `json:"disk_format,omitempty"`
	DiskPath             string           `json:"disk_path,omitempty"`
	Checksum             string           `json:"checksum,omitempty"`
	RootID               int32            `json:"root_id"`
	RootAddress          string           `json:"root_address"`
	NodeCount            int              `json:"node_count"`
	TriangleCount        int              `json:"triangle_count"`
	TriangleStorageOrder []int            `json:"triangle_storage_original_ids"`
	BVHOrder             []int            `json:"bvh_order,omitempty"`
	RelocationTable      map[int32]string `json:"relocation_table,omitempty"`
	Nodes                []NodeDump       `json:"nodes"`
	Instances            []InstanceDump   `json:"instances"`
}

func DumpOriginal(scene *DemoScene) StructureDump {
	nodes := CollectPointerNodes(scene.BVH.Root)
	dump := StructureDump{
		Kind:                 "pointer-bvh/original-addresses",
		RootID:               0,
		RootAddress:          fmt.Sprintf("%p", scene.BVH.Root),
		NodeCount:            len(nodes),
		TriangleCount:        len(scene.Triangles),
		TriangleStorageOrder: make([]int, len(scene.Triangles)),
		BVHOrder:             append([]int{}, scene.BVH.Order...),
		Nodes:                make([]NodeDump, 0, len(nodes)),
	}
	for i := range dump.TriangleStorageOrder {
		dump.TriangleStorageOrder[i] = i
	}

	for i, node := range nodes {
		nd := NodeDump{
			ID:      int32(i),
			Address: fmt.Sprintf("%p", node),
			LeftID:  -1, RightID: -1,
			Bounds: node.Bounds, First: node.First, Count: node.Count,
		}
		if node.Left != nil {
			nd.LeftID = int32(node.Left.SerializedID)
			nd.LeftAddress = fmt.Sprintf("%p", node.Left)
		}
		if node.Right != nil {
			nd.RightID = int32(node.Right.SerializedID)
			nd.RightAddress = fmt.Sprintf("%p", node.Right)
		}
		if node.Left == nil && node.Right == nil {
			nd.OrderIndices = make([]int, node.Count)
			for k := 0; k < node.Count; k++ {
				nd.OrderIndices[k] = scene.BVH.Order[node.First+k]
			}
		}
		dump.Nodes = append(dump.Nodes, nd)
	}
	dump.Instances = dumpOriginalInstances(scene.Root)
	return dump
}

func dumpOriginalInstances(root *Instance) []InstanceDump {
	var out []InstanceDump
	var walk func(inst *Instance, parent int32)
	walk = func(inst *Instance, parent int32) {
		childIDs := make([]int32, 0, len(inst.Children))
		for range inst.Children {
			// File IDs are assigned by the same preorder serializer. Fill
			// after collecting via a second stable-ID map if needed.
			childIDs = append(childIDs, -1)
		}
		out = append(out, InstanceDump{
			StableID: inst.ID, Name: inst.Name, Address: fmt.Sprintf("%p", inst),
			ParentID: parent, Depth: inst.Depth, MeshID: inst.MeshID,
			ChildIDs: childIDs, World: inst.World,
		})
		myIndex := int32(len(out) - 1)
		for _, child := range inst.Children {
			walk(child, myIndex)
		}
	}
	walk(root, -1)

	idToFile := map[uint32]int32{}
	for i := range out {
		idToFile[out[i].StableID] = int32(i)
	}
	for i := range out {
		inst := findOriginalByStableID(root, out[i].StableID)
		if inst == nil {
			continue
		}
		out[i].ChildIDs = make([]int32, 0, len(inst.Children))
		for _, child := range inst.Children {
			out[i].ChildIDs = append(out[i].ChildIDs, idToFile[child.ID])
		}
	}
	return out
}

func findOriginalByStableID(root *Instance, id uint32) *Instance {
	return FindInstance(root, id)
}

func DumpRelocated(rt *RelocatedAS) StructureDump {
	dump := StructureDump{
		Kind:                 "relocated-flat-bvh/runtime-pointers",
		DiskFormat:           assetMagic + "/v1",
		DiskPath:             rt.Path,
		Checksum:             fmt.Sprintf("%08x", rt.Checksum),
		RootID:               rt.Root.FileID,
		RootAddress:          fmt.Sprintf("%p", rt.Root),
		NodeCount:            len(rt.Nodes),
		TriangleCount:        len(rt.Triangles),
		TriangleStorageOrder: make([]int, len(rt.Triangles)),
		BVHOrder:             make([]int, len(rt.Order)),
		RelocationTable:      map[int32]string{},
		Nodes:                make([]NodeDump, 0, len(rt.Nodes)),
		Instances:            make([]InstanceDump, 0, len(rt.Instances)),
	}
	for i, tri := range rt.Triangles {
		dump.TriangleStorageOrder[i] = tri.OriginalIndex
	}
	for i, idx := range rt.Order {
		dump.BVHOrder[i] = int(idx)
	}
	for i := range rt.Nodes {
		node := &rt.Nodes[i]
		dump.RelocationTable[node.FileID] = fmt.Sprintf("%p", node)
		nd := NodeDump{
			ID: node.FileID, Address: fmt.Sprintf("%p", node),
			LeftID: node.LeftID, RightID: node.RightID,
			Bounds: node.Bounds, First: node.First, Count: node.Count,
		}
		if node.Left != nil {
			nd.LeftAddress = fmt.Sprintf("%p", node.Left)
		}
		if node.Right != nil {
			nd.RightAddress = fmt.Sprintf("%p", node.Right)
		}
		if node.Left == nil && node.Right == nil {
			nd.OrderIndices = make([]int, node.Count)
			for k := 0; k < node.Count; k++ {
				nd.OrderIndices[k] = int(rt.Order[node.First+k])
			}
		}
		dump.Nodes = append(dump.Nodes, nd)
	}
	for i := range rt.Instances {
		inst := &rt.Instances[i]
		childIDs := make([]int32, 0, len(inst.Children))
		for _, child := range inst.Children {
			childIDs = append(childIDs, child.FileID)
		}
		parentID := int32(-1)
		if inst.Parent != nil {
			parentID = inst.Parent.FileID
		}
		dump.Instances = append(dump.Instances, InstanceDump{
			FileID: inst.FileID, StableID: inst.StableID, Name: inst.Name,
			Address: fmt.Sprintf("%p", inst), ParentID: parentID,
			Depth: inst.Depth, MeshID: inst.MeshID, ChildIDs: childIDs,
			World: inst.World,
		})
	}
	return dump
}
