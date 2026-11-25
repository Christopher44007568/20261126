package main

import "math"

type RuntimeNode struct {
	FileID  int32
	Bounds  AABB
	Left    *RuntimeNode
	Right   *RuntimeNode
	LeftID  int32
	RightID int32
	First   int
	Count   int
}

type RuntimeInstance struct {
	FileID   int32
	StableID uint32
	Name     string
	Depth    int
	MeshID   int32
	Local    Mat4
	World    Mat4
	WorldInv Mat4
	Parent   *RuntimeInstance
	Children []*RuntimeInstance
}

type RelocatedAS struct {
	Path           string
	Header         diskHeader
	Checksum       uint32
	Nodes          []RuntimeNode
	NodeByFileID   map[int32]*RuntimeNode
	Root           *RuntimeNode
	Triangles      []Triangle
	Order          []uint32
	Instances      []RuntimeInstance
	RootInstance   *RuntimeInstance
	DiskToOriginal []uint32
}

func RelocateAsset(file *AssetFile, path string) *RelocatedAS {
	rt := &RelocatedAS{
		Path:           path,
		Header:         file.Header,
		Checksum:       file.Checksum,
		Triangles:      file.Triangles,
		Order:          file.Order,
		DiskToOriginal: file.DiskToOriginal,
		NodeByFileID:   map[int32]*RuntimeNode{},
	}

	rt.Nodes = make([]RuntimeNode, len(file.Nodes))
	for i, rec := range file.Nodes {
		id := int32(i)
		rt.Nodes[i] = RuntimeNode{
			FileID:  id,
			Bounds:  rec.Bounds,
			LeftID:  rec.Left,
			RightID: rec.Right,
			First:   int(rec.First),
			Count:   int(rec.Count),
		}
		rt.NodeByFileID[id] = &rt.Nodes[i]
	}
	for i := range rt.Nodes {
		node := &rt.Nodes[i]
		if node.LeftID >= 0 {
			node.Left = rt.NodeByFileID[node.LeftID]
		}
		if node.RightID >= 0 {
			node.Right = rt.NodeByFileID[node.RightID]
		}
	}
	rt.Root = rt.NodeByFileID[file.Header.RootNode]

	rt.Instances = make([]RuntimeInstance, len(file.Instances))
	for i, rec := range file.Instances {
		rt.Instances[i] = RuntimeInstance{
			FileID:   int32(i),
			StableID: rec.StableID,
			Name:     file.InstanceNames[i],
			Depth:    int(rec.Depth),
			MeshID:   rec.MeshID,
			Local:    rec.Local,
		}
	}
	for i := range rt.Instances {
		inst := &rt.Instances[i]
		parentID := file.Instances[i].Parent
		if parentID >= 0 {
			parent := &rt.Instances[parentID]
			inst.Parent = parent
			parent.Children = append(parent.Children, inst)
		}
	}
	rt.RootInstance = &rt.Instances[file.Header.RootInstance]
	rt.RootInstance.UpdateRuntimeWorld(IdentityMat4(), 0)
	return rt
}

func (in *RuntimeInstance) UpdateRuntimeWorld(parent Mat4, depth int) {
	in.Depth = depth
	in.World = matMul(parent, in.Local)
	in.WorldInv = matInverse(in.World)
	for _, child := range in.Children {
		child.UpdateRuntimeWorld(in.World, depth+1)
	}
}

func IntersectRelocatedScene(as *RelocatedAS, ray Ray) Hit {
	if as == nil || as.Root == nil || as.RootInstance == nil {
		return Hit{}
	}
	return intersectRuntimeInstance(as.RootInstance, as, ray, nil)
}

func intersectRuntimeInstance(inst *RuntimeInstance, as *RelocatedAS, ray Ray, parentPath []uint32) Hit {
	path := append(append([]uint32{}, parentPath...), inst.StableID)
	var best Hit
	bestT := ray.TMax

	if inst.MeshID >= 0 {
		localRay := TransformRayToLocal(ray, inst.WorldInv)
		localRay.TMax = math.Inf(1)
		hit, ok := intersectRuntimeNode(as.Root, as, localRay)
		if ok {
			tri := as.Triangles[hit.Triangle]
			worldHit := LocalHitToWorld(ray, hit, tri, inst.World)
			if worldHit.T >= ray.TMin && worldHit.T <= bestT {
				worldHit.InstancePath = append([]uint32{}, path...)
				worldHit.Depth = len(path)
				worldHit.StorageTriangle = hit.Triangle
				best = worldHit
				bestT = worldHit.T
			}
		}
	}

	for _, child := range inst.Children {
		childRay := ray
		childRay.TMax = bestT
		childHit := intersectRuntimeInstance(child, as, childRay, path)
		if childHit.OK && childHit.T < bestT {
			best = childHit
			bestT = childHit.T
		}
	}
	return best
}

func intersectRuntimeNode(node *RuntimeNode, as *RelocatedAS, ray Ray) (TriHit, bool) {
	if node == nil {
		return TriHit{}, false
	}
	var best TriHit
	found := false
	bestT := ray.TMax
	stack := []*RuntimeNode{node}

	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if cur == nil {
			continue
		}
		if _, _, ok := cur.Bounds.Intersect(ray); !ok {
			continue
		}

		if cur.Left == nil && cur.Right == nil {
			for k := 0; k < cur.Count; k++ {
				orderPos := cur.First + k
				storageIndex := int(as.Order[orderPos])
				queryRay := ray
				queryRay.TMax = bestT
				hit, ok := IntersectTriangle(queryRay, as.Triangles[storageIndex], storageIndex)
				if ok && hit.T < bestT {
					best = hit
					bestT = hit.T
					found = true
				}
			}
			continue
		}
		if cur.Left != nil {
			stack = append(stack, cur.Left)
		}
		if cur.Right != nil {
			stack = append(stack, cur.Right)
		}
	}
	if found {
		best.T = bestT
	}
	return best, found
}
