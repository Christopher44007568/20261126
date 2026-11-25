package main

// Frame 尺寸与子树布局常量。
const (
	FrameW = 512
	FrameH = 512
	Margin = 10
	Gutter = 8
	TileW  = (FrameW - 2*Margin - Gutter) / 2 // 242
	TileH  = 242
	TopPad = 18 // 顶部全局标题区
)

// Node 是场景树节点。只有叶子节点持有需要远端加载的 payload；
// 内部节点仅表达子树归属关系。
type Node struct {
	ID       string
	Label    string
	Seed     uint32
	Rect     Rect
	Children []*Node
}

// Rect 为轴对齐矩形。
type Rect struct{ X, Y, W, H int }

func (n *Node) IsLeaf() bool { return len(n.Children) == 0 }

// Leaves 按稳定顺序返回全部叶子子树。
func (n *Node) Leaves() []*Node {
	var out []*Node
	var walk func(*Node)
	walk = func(node *Node) {
		if node.IsLeaf() {
			out = append(out, node)
			return
		}
		for _, c := range node.Children {
			walk(c)
		}
	}
	walk(n)
	return out
}

// BuildScene 构造两级场景树：root -> group -> 4 个叶子子树，
// 叶子按 2x2 网格摆放。
func BuildScene() *Node {
	positions := []Rect{
		{X: Margin, Y: TopPad, W: TileW, H: TileH},
		{X: Margin + TileW + Gutter, Y: TopPad, W: TileW, H: TileH},
		{X: Margin, Y: TopPad + TileH + Gutter, W: TileW, H: TileH},
		{X: Margin + TileW + Gutter, Y: TopPad + TileH + Gutter, W: TileW, H: TileH},
	}
	ids := []string{"T1", "T2", "T3", "T4"}
	var seeds uint32 = 100

	group := &Node{ID: "G1", Label: "QUAD GROUP"}
	root := &Node{ID: "ROOT", Label: "SCENE ROOT", Children: []*Node{group}}
	for i, id := range ids {
		leaf := &Node{
			ID:    id,
			Label: "SUBTREE " + id,
			Seed:  seeds + uint32(i)*7,
			Rect:  positions[i],
		}
		group.Children = append(group.Children, leaf)
	}
	return root
}
