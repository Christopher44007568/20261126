package main

type InstanceStatus int

const (
	StatusNormal InstanceStatus = iota
	StatusStaleBounds
	StatusMissedUpdate
)

type Instance struct {
	ID           int
	Name         string
	BLAS         *BLAS
	Color        Vec3
	Status       InstanceStatus
	World        Mat4
	InvWorld     Mat4
	Bounds       AABB
	ActualBounds AABB
	LastFrame    int
}

func NewInstance(id int, name string, blas *BLAS, world Mat4, color Vec3, status InstanceStatus) *Instance {
	in := &Instance{
		ID:       id,
		Name:     name,
		BLAS:     blas,
		Color:    color,
		Status:   status,
		World:    world,
		InvWorld: RigidInverse(world),
	}
	in.ActualBounds = TransformAABB(blas.Bounds, world)
	in.Bounds = in.ActualBounds
	return in
}

// SetTransform changes the instance transform and computes its true bounds,
// but does not publish those bounds to the TLAS leaf.
func (in *Instance) SetTransform(world Mat4, frame int) {
	in.World = world
	in.InvWorld = RigidInverse(world)
	in.ActualBounds = TransformAABB(in.BLAS.Bounds, world)
	in.LastFrame = frame
}

// CommitBounds publishes the actual transformed BLAS bounds to the TLAS.
func (in *Instance) CommitBounds() {
	in.Bounds = in.ActualBounds
}

func (in *Instance) Intersect(worldRay Ray, maxT float64) (triID int, u, v, t float64, ok bool) {
	localOrigin := in.InvWorld.Point(worldRay.Origin)
	localDir := in.InvWorld.Direction(worldRay.Dir)
	localRay := NewRay(localOrigin, localDir)
	t, triID, u, v, ok = in.BLAS.Intersect(localRay, maxT)
	return triID, u, v, t, ok
}

type TLASNode struct {
	Bounds   AABB
	Left     int
	Right    int
	Instance int
}

type TLAS struct {
	Instances  []*Instance
	Nodes      []TLASNode
	Root       int
	RefitCount int
}

func BuildTLAS(instances []*Instance) *TLAS {
	t := &TLAS{Instances: instances}
	order := make([]int, len(instances))
	for i := range instances {
		order[i] = i
	}

	var build func(start, end int) int
	build = func(start, end int) int {
		bounds := EmptyAABB()
		for at := start; at < end; at++ {
			bounds = bounds.Union(instances[order[at]].Bounds)
		}
		nodeID := len(t.Nodes)
		t.Nodes = append(t.Nodes, TLASNode{Bounds: bounds, Left: -1, Right: -1, Instance: -1})

		if end-start == 1 {
			t.Nodes[nodeID].Instance = order[start]
			return nodeID
		}

		centerBounds := EmptyAABB()
		for at := start; at < end; at++ {
			centerBounds.Add(instances[order[at]].Bounds.Centroid())
		}
		extent := V(
			centerBounds.Max.X-centerBounds.Min.X,
			centerBounds.Max.Y-centerBounds.Min.Y,
			centerBounds.Max.Z-centerBounds.Min.Z,
		)
		axis := 0
		if extent.Y > extent.X && extent.Y >= extent.Z {
			axis = 1
		} else if extent.Z > extent.X && extent.Z > extent.Y {
			axis = 2
		}

		// A tiny insertion sort is enough for this small character TLAS.
		for i := start + 1; i < end; i++ {
			for j := i; j > start; j-- {
				a := instances[order[j-1]].Bounds.Centroid()
				b := instances[order[j]].Bounds.Centroid()
				less := false
				switch axis {
				case 1:
					less = b.Y < a.Y
				case 2:
					less = b.Z < a.Z
				default:
					less = b.X < a.X
				}
				if !less {
					break
				}
				order[j-1], order[j] = order[j], order[j-1]
			}
		}

		mid := (start + end) / 2
		left := build(start, mid)
		right := build(mid, end)
		t.Nodes[nodeID].Left = left
		t.Nodes[nodeID].Right = right
		return nodeID
	}

	t.Root = build(0, len(instances))
	return t
}

// UpdateTransform performs the correct instance update and publishes bounds.
func (t *TLAS) UpdateTransform(id int, world Mat4, frame int) {
	in := t.Instances[id]
	in.SetTransform(world, frame)
	in.CommitBounds()
}

// UpdateTransformWithStaleBounds reproduces the optimization bug: the matrix
// changes, but the old TLAS AABB is deliberately left in place.
func (t *TLAS) UpdateTransformWithStaleBounds(id int, world Mat4, frame int) {
	t.Instances[id].SetTransform(world, frame)
}

// SkipUpdate leaves both matrix and registered bounds at the previous pose.
func (t *TLAS) SkipUpdate(id int) {
	_ = t.Instances[id]
}

func (t *TLAS) Refit() {
	var refit func(nodeID int) AABB
	refit = func(nodeID int) AABB {
		node := &t.Nodes[nodeID]
		if node.Instance >= 0 {
			node.Bounds = t.Instances[node.Instance].Bounds
			return node.Bounds
		}
		left := refit(node.Left)
		right := refit(node.Right)
		node.Bounds = left.Union(right)
		return node.Bounds
	}
	refit(t.Root)
	t.RefitCount++
}

func (t *TLAS) Intersect(ray Ray, maxT float64) (instanceID, triID int, u, v, hitT float64, ok bool) {
	hitT = maxT
	var stack [64]int
	sp := 0
	stack[sp] = t.Root
	sp++

	for sp > 0 {
		sp--
		nodeID := stack[sp]
		node := &t.Nodes[nodeID]
		if _, hit := node.Bounds.Intersect(ray, hitT); !hit {
			continue
		}

		if node.Instance >= 0 {
			in := t.Instances[node.Instance]
			localRay := NewRay(
				in.InvWorld.Point(ray.Origin),
				in.InvWorld.Direction(ray.Dir),
			)
			dist, tri, hu, hv, hit := in.BLAS.Intersect(localRay, hitT)
			if hit {
				hitT = dist
				instanceID = node.Instance
				triID = tri
				u, v = hu, hv
				ok = true
			}
			continue
		}

		stack[sp] = node.Left
		sp++
		stack[sp] = node.Right
		sp++
	}
	return instanceID, triID, u, v, hitT, ok
}
