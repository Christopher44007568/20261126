package pipeline

import (
	"fmt"
)

// Pass 描述渲染管线中的一个渲染通道。
// Reads/Writes 用缓冲名声明资源依赖，Fn 是真正的渲染逻辑。
type Pass struct {
	Name   string
	Reads  []string
	Writes []string
	Fn     func(*Frame) error
}

// Pipeline 是一组按资源依赖组织的 Pass。
type Pipeline struct {
	name   string
	width  int
	height int
	passes []*Pass
	byName map[string]*Pass
}

// New 创建一条具名管线，帧尺寸为 width x height。
func New(name string, width, height int) *Pipeline {
	return &Pipeline{
		name:   name,
		width:  width,
		height: height,
		byName: make(map[string]*Pass),
	}
}

// Name 返回管线名。
func (p *Pipeline) Name() string { return p.name }

// Width 返回帧宽。
func (p *Pipeline) Width() int { return p.width }

// Height 返回帧高。
func (p *Pipeline) Height() int { return p.height }

// Add 注册一个 Pass，重名会被拒绝。
func (p *Pipeline) Add(pass *Pass) error {
	if pass == nil || pass.Name == "" {
		return fmt.Errorf("pipeline: pass must have a non-empty name")
	}
	if pass.Fn == nil {
		return fmt.Errorf("pipeline: pass %q has no render function", pass.Name)
	}
	if len(pass.Writes) == 0 {
		return fmt.Errorf("pipeline: pass %q must write at least one buffer", pass.Name)
	}
	if _, dup := p.byName[pass.Name]; dup {
		return fmt.Errorf("pipeline: duplicate pass name %q", pass.Name)
	}
	p.passes = append(p.passes, pass)
	p.byName[pass.Name] = pass
	return nil
}

// BuildGraph 做全部静态校验并构建 Pass 依赖图：
// 边 producer -> consumer 表示资源生产者必须先于消费者执行。
// 任何不合法输入都在这一步被拒绝，不会产生任何执行或画面。
func (p *Pipeline) BuildGraph() (*Graph, error) {
	writer := make(map[string]string)
	for _, pass := range p.passes {
		for _, res := range pass.Writes {
			if other, ok := writer[res]; ok {
				return nil, fmt.Errorf(
					"pipeline: buffer %q written by both %q and %q",
					res, other, pass.Name)
			}
			writer[res] = pass.Name
		}
	}
	for _, pass := range p.passes {
		for _, res := range pass.Reads {
			if _, ok := writer[res]; !ok {
				return nil, fmt.Errorf(
					"pipeline: pass %q reads buffer %q but no pass produces it",
					pass.Name, res)
			}
		}
	}

	// 节点顺序与 Pass 注册顺序一致，保证输出稳定可复现。
	g := NewGraph(p.passNames())
	for _, pass := range p.passes {
		for _, res := range pass.Reads {
			if err := g.AddEdge(writer[res], pass.Name); err != nil {
				return nil, err
			}
		}
	}
	return g, nil
}

func (p *Pipeline) passNames() []string {
	names := make([]string, 0, len(p.passes))
	for _, pass := range p.passes {
		names = append(names, pass.Name)
	}
	return names
}

// Plan 是通过全部校验后的可执行计划。
type Plan struct {
	pipeline *Pipeline
	graph    *Graph
	order    []string
}

// Graph 返回依赖图。
func (plan *Plan) Graph() *Graph { return plan.graph }

// Order 返回拓扑执行顺序。
func (plan *Plan) Order() []string {
	out := make([]string, len(plan.order))
	copy(out, plan.order)
	return out
}

// Plan 校验管线、检测循环并生成拓扑执行计划。
func (p *Pipeline) Plan() (*Plan, error) {
	g, err := p.BuildGraph()
	if err != nil {
		return nil, err
	}
	order, err := g.TopoSort()
	if err != nil {
		return nil, err
	}
	return &Plan{pipeline: p, graph: g, order: order}, nil
}

// Run 按拓扑顺序执行全部 Pass。
// 任一 Pass 失败立即停止并返回错误，调用方不会拿到“半张画面”的成功结果。
func (plan *Plan) Run(frame *Frame) error {
	for _, name := range plan.order {
		pass := plan.pipeline.byName[name]
		if err := pass.Fn(frame); err != nil {
			return fmt.Errorf("pipeline: pass %q failed: %w", name, err)
		}
	}
	return nil
}

// NewFrame 按管线尺寸创建一帧。
func (p *Pipeline) NewFrame() *Frame {
	return NewFrame(p.width, p.height)
}
