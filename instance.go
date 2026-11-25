package main

type Instance struct {
	ID       uint32
	Name     string
	Depth    int
	Local    Mat4
	World    Mat4
	WorldInv Mat4
	MeshID   int32
	Children []*Instance
}

func NewGroup(id uint32, name string, local Mat4) *Instance {
	return &Instance{ID: id, Name: name, Local: local, MeshID: -1}
}

func NewMeshInstance(id uint32, name string, meshID int32, local Mat4) *Instance {
	return &Instance{ID: id, Name: name, Local: local, MeshID: meshID}
}

func (in *Instance) Add(child *Instance) *Instance {
	in.Children = append(in.Children, child)
	return child
}

func (in *Instance) UpdateWorld(parent Mat4, depth int) {
	in.Depth = depth
	in.World = matMul(parent, in.Local)
	in.WorldInv = matInverse(in.World)
	for _, child := range in.Children {
		child.UpdateWorld(in.World, depth+1)
	}
}

func TransformRayToLocal(ray Ray, worldInv Mat4) Ray {
	origin := matPoint(worldInv, ray.Origin)
	rawDir := matDir(worldInv, ray.Dir)
	scale := rawDir.Len()
	dir := rawDir.Normalized()
	if scale == 0 {
		scale = 1
	}
	return Ray{
		Origin: origin,
		Dir:    dir,
		TMin:   ray.TMin * scale,
		TMax:   ray.TMax * scale,
	}
}

func LocalHitToWorld(ray Ray, localHit TriHit, localTri Triangle, world Mat4) Hit {
	point := matPoint(world, localHit.Point)
	normal := matDir(world, localHit.Normal).Normalized()
	worldT := point.Sub(ray.Origin).Dot(ray.Dir)
	return Hit{
		OK:               true,
		T:                worldT,
		Point:            point,
		Normal:           normal,
		BaryU:            localHit.BaryU,
		BaryV:            localHit.BaryV,
		SurfaceID:        localTri.SurfaceID,
		OriginalTriangle: localTri.OriginalIndex,
		StorageTriangle:  localHit.Triangle,
		Color:            localTri.Color,
		LocalPoint:       localHit.Point,
	}
}
