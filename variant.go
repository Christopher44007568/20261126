package main

import (
	"errors"
	"fmt"
	"sort"
)

// Variant 是同一资产的三套视图。
type Variant string

const (
	VariantSummer Variant = "summer"
	VariantWinter Variant = "winter"
	VariantNight  Variant = "night"
)

func AllVariants() []Variant {
	return []Variant{VariantSummer, VariantWinter, VariantNight}
}

func (v Variant) Valid() bool {
	switch v {
	case VariantSummer, VariantWinter, VariantNight:
		return true
	default:
		return false
	}
}

func (v Variant) Label() string {
	switch v {
	case VariantSummer:
		return "SUMMER"
	case VariantWinter:
		return "WINTER"
	case VariantNight:
		return "NIGHT"
	default:
		return string(v)
	}
}

// Material 是真正参与渲染的资源，而不只是一个材质名字符串。
type Material struct {
	ID     string `json:"id"`
	Shader string `json:"shader"`
	Fill   RGB    `json:"fill"`
	Accent RGB    `json:"accent"`
	Glow   RGB    `json:"glow"`
}

// ResolvedBinding 是 variant 选择后解析出的槽位绑定。
// Material 指针必须非空；只修改 RequestedName 不会产生合法绑定。
type ResolvedBinding struct {
	Slot       string    `json:"slot"`
	MaterialID string    `json:"materialId"`
	Shader     string    `json:"shader"`
	Variant    Variant   `json:"variant"`
	SourcePath string    `json:"sourcePath"`
	Material   *Material `json:"-"`
}

func (b ResolvedBinding) Resolved() bool {
	return b.Material != nil && b.Material.ID == b.MaterialID && b.Shader == b.Material.Shader
}

// Node 是场景树节点。VariantMaterials 为每个 variant 的每个槽位给出材质 ID。
type Node struct {
	Name             string                        `json:"name"`
	Path             string                        `json:"path"`
	Shape            Shape                         `json:"shape"`
	Slots            []string                      `json:"slots"`
	VariantMaterials map[Variant]map[string]string `json:"-"`
	Children         []*Node                       `json:"children,omitempty"`
	Bindings         map[string]ResolvedBinding    `json:"bindings,omitempty"`
	Parent           *Node                         `json:"-"`
	order            int                           `json:"-"`
}

// Stage 保存资产树、材质注册表和在任意节点上编写的 variant 选择。
type Stage struct {
	Name         string               `json:"name"`
	VariantSet   string               `json:"variantSet"`
	Default      Variant              `json:"defaultVariant"`
	Root         *Node                `json:"root"`
	Nodes        map[string]*Node     `json:"-"`
	Materials    map[string]*Material `json:"-"`
	selections   map[string]Variant
	orderedNodes []*Node
}

func NewStage(name, variantSet string, defaultVariant Variant) *Stage {
	return &Stage{
		Name:       name,
		VariantSet: variantSet,
		Default:    defaultVariant,
		Nodes:      map[string]*Node{},
		Materials:  map[string]*Material{},
		selections: map[string]Variant{},
	}
}

func (s *Stage) AddMaterial(m Material) *Material {
	mat := m
	s.Materials[mat.ID] = &mat
	return &mat
}

// AddNode 挂载节点。材质表必须覆盖该节点声明的所有 slot 和三套 variant。
func (s *Stage) AddNode(parent *Node, name string, shape Shape, slots []string, materials map[Variant]map[string]string) (*Node, error) {
	if name == "" {
		return nil, errors.New("node name must not be empty")
	}
	path := "/" + name
	if parent != nil {
		if parent.Path == "/" {
			path = "/" + name
		} else {
			path = parent.Path + "/" + name
		}
	}
	if _, exists := s.Nodes[path]; exists {
		return nil, fmt.Errorf("duplicate node path %s", path)
	}
	slotsCopy := append([]string(nil), slots...)
	node := &Node{
		Name:             name,
		Path:             path,
		Shape:            shape,
		Slots:            slotsCopy,
		VariantMaterials: materials,
		Bindings:         map[string]ResolvedBinding{},
		Parent:           parent,
		order:            len(s.orderedNodes),
	}
	if parent == nil {
		if s.Root != nil {
			return nil, fmt.Errorf("root already exists: %s", s.Root.Path)
		}
		s.Root = node
	} else {
		parent.Children = append(parent.Children, node)
	}
	s.Nodes[path] = node
	s.orderedNodes = append(s.orderedNodes, node)
	return node, nil
}

func (s *Stage) MustNode(path string) *Node {
	n, ok := s.Nodes[path]
	if !ok {
		panic("unknown node path: " + path)
	}
	return n
}

func (s *Stage) Node(path string) (*Node, error) {
	n, ok := s.Nodes[path]
	if !ok {
		return nil, fmt.Errorf("unknown node path %q", path)
	}
	return n, nil
}

// SelectionStep 记录路径上每一级看到的选择来源。
type SelectionStep struct {
	Path            string  `json:"path"`
	Authored        bool    `json:"authored"`
	AuthoredVariant Variant `json:"authoredVariant,omitempty"`
	Active          Variant `json:"activeVariant"`
	SourcePath      string  `json:"sourcePath"`
}

// Resolution 是某个节点最终生效的 variant 和完整传播路径。
type Resolution struct {
	TargetPath string          `json:"targetPath"`
	VariantSet string          `json:"variantSet"`
	Variant    Variant         `json:"variant"`
	SourcePath string          `json:"sourcePath"`
	Steps      []SelectionStep `json:"steps"`
}

func (s *Stage) ancestorChain(path string) ([]*Node, error) {
	target, err := s.Node(path)
	if err != nil {
		return nil, err
	}
	var chain []*Node
	for n := target; n != nil; n = n.Parent {
		chain = append(chain, n)
	}
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	return chain, nil
}

// Resolve 按 root -> target 顺序应用编写意见；离 target 越近的意见越晚生效。
func (s *Stage) Resolve(path string) (Resolution, error) {
	chain, err := s.ancestorChain(path)
	if err != nil {
		return Resolution{}, err
	}
	active := s.Default
	source := "(default)"
	steps := make([]SelectionStep, 0, len(chain))
	for _, n := range chain {
		authored, ok := s.selections[n.Path]
		step := SelectionStep{Path: n.Path, Authored: ok, Active: active, SourcePath: source}
		if ok {
			active = authored
			source = n.Path
			step.AuthoredVariant = authored
		}
		step.Active = active
		step.SourcePath = source
		steps = append(steps, step)
	}
	return Resolution{
		TargetPath: path,
		VariantSet: s.VariantSet,
		Variant:    active,
		SourcePath: source,
		Steps:      steps,
	}, nil
}

func (s *Stage) SetSelection(path string, v Variant) error {
	if _, err := s.Node(path); err != nil {
		return err
	}
	if !v.Valid() {
		return fmt.Errorf("invalid variant %q", v)
	}
	s.selections[path] = v
	return nil
}

func (s *Stage) ClearSelection(path string) error {
	if _, err := s.Node(path); err != nil {
		return err
	}
	delete(s.selections, path)
	return nil
}

// SelectVariant 是正式切换入口：写选择并立即重新绑定所有槽位。
func (s *Stage) SelectVariant(path string, v Variant) error {
	if err := s.SetSelection(path, v); err != nil {
		return err
	}
	return s.Rebind()
}

// SelectionToken 保存覆盖前的意见，保证“恢复”不依赖猜测默认值。
type SelectionToken struct {
	Path     string  `json:"path"`
	Previous Variant `json:"previous,omitempty"`
	Had      bool    `json:"had"`
}

// Override 在某一级临时覆盖，并返回可精确恢复原意见的令牌。
func (s *Stage) Override(path string, v Variant) (SelectionToken, error) {
	previous, had := s.selections[path]
	token := SelectionToken{Path: path, Previous: previous, Had: had}
	if err := s.SelectVariant(path, v); err != nil {
		return SelectionToken{}, err
	}
	return token, nil
}

// Restore 恢复令牌记录的精确状态：原来没有意见就删除，而不是写成默认值。
func (s *Stage) Restore(token SelectionToken) error {
	if token.Had {
		s.selections[token.Path] = token.Previous
	} else {
		delete(s.selections, token.Path)
	}
	return s.Rebind()
}

func (s *Stage) SnapshotSelections() map[string]Variant {
	out := make(map[string]Variant, len(s.selections))
	for k, v := range s.selections {
		out[k] = v
	}
	return out
}

func (s *Stage) RestoreSelections(snapshot map[string]Variant) error {
	s.selections = make(map[string]Variant, len(snapshot))
	for k, v := range snapshot {
		if _, err := s.Node(k); err != nil {
			return err
		}
		if !v.Valid() {
			return fmt.Errorf("invalid snapshot variant %q", v)
		}
		s.selections[k] = v
	}
	return s.Rebind()
}

// Rebind 根据传播后的 variant 查找真实 Material 指针并写入每个槽位。
func (s *Stage) Rebind() error {
	for _, n := range s.orderedNodes {
		resolution, err := s.Resolve(n.Path)
		if err != nil {
			return err
		}
		for _, slot := range n.Slots {
			byVariant, ok := n.VariantMaterials[resolution.Variant]
			if !ok {
				return fmt.Errorf("node %s missing variant table %q", n.Path, resolution.Variant)
			}
			materialID, ok := byVariant[slot]
			if !ok || materialID == "" {
				return fmt.Errorf("node %s variant %q missing slot %q", n.Path, resolution.Variant, slot)
			}
			mat, ok := s.Materials[materialID]
			if !ok {
				return fmt.Errorf("node %s slot %q references unregistered material %q", n.Path, slot, materialID)
			}
			n.Bindings[slot] = ResolvedBinding{
				Slot:       slot,
				MaterialID: materialID,
				Shader:     mat.Shader,
				Variant:    resolution.Variant,
				SourcePath: resolution.SourcePath,
				Material:   mat,
			}
		}
	}
	return nil
}

// ValidateBindings 防止“名字改了但指针仍旧/为空”的伪切换进入渲染。
func (s *Stage) ValidateBindings() error {
	var errs []error
	for _, n := range s.orderedNodes {
		for _, slot := range n.Slots {
			b, ok := n.Bindings[slot]
			if !ok {
				errs = append(errs, fmt.Errorf("%s slot %s has no binding", n.Path, slot))
				continue
			}
			if !b.Resolved() {
				errs = append(errs, fmt.Errorf("%s slot %s is name-only or stale: %+v", n.Path, slot, b))
			}
		}
	}
	return errors.Join(errs...)
}

// ChangeMaterialNameOnly 故意复现错误做法：只改材质名并清空真实绑定。
// 后续 ValidateBindings/Render 必须失败，直到 Rebind 重新绑定。
func (s *Stage) ChangeMaterialNameOnly(path, slot, materialID string) error {
	n, err := s.Node(path)
	if err != nil {
		return err
	}
	b := n.Bindings[slot]
	b.Slot = slot
	b.MaterialID = materialID
	b.Material = nil
	b.Shader = ""
	n.Bindings[slot] = b
	return nil
}

func SortedSelectionKeys(m map[string]Variant) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
