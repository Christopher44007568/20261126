package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"renderpipeline/internal/pipeline"
)

const (
	frameWidth  = 64
	frameHeight = 48
)

// validPipeline 是接线正确的管线：
// geometry -> shadow -> lighting -> postprocess -> output，
// color 并行汇入 lighting。
func validPipeline() *pipeline.Pipeline {
	p := pipeline.New("correct-wiring", frameWidth, frameHeight)
	mustAdd(p, pipeline.GeometryPass())
	mustAdd(p, pipeline.ShadowPass())
	mustAdd(p, pipeline.ColorPass())
	mustAdd(p, pipeline.LightingPass())
	mustAdd(p, pipeline.PostProcessPass())
	mustAdd(p, pipeline.OutputPass())
	return p
}

// swappedPipeline 模拟用户把光照与后处理的两条边接反：
// 本来 lighting 写 lit、postprocess 读 lit 写 final_color；
// 接反后 lighting 改读 final_color 并写 lit，postprocess 读 lit 写 final_color，
// 两个 Pass 互相等待对方先产出，形成 lighting <-> postprocess 资源环。
func swappedPipeline() *pipeline.Pipeline {
	p := pipeline.New("swapped-edges", frameWidth, frameHeight)
	mustAdd(p, pipeline.GeometryPass())
	mustAdd(p, pipeline.ShadowPass())
	mustAdd(p, pipeline.ColorPass())

	mustAdd(p, swappedLightingPass())
	mustAdd(p, swappedPostProcessPass())

	mustAdd(p, pipeline.OutputPass())
	return p
}

// swappedLightingPass 错误地把 final_color 当作输入：它本该由后处理产出，
// 却被光照 Pass 提前消费，造成“后处理等光照、光照等后处理”的环。
func swappedLightingPass() *pipeline.Pass {
	base := pipeline.LightingPass()
	base.Reads = []string{pipeline.ResColor, pipeline.ResShadow, pipeline.ResFinalColor}
	return base
}

// swappedPostProcessPass 保持读 lit、写 final_color；与上面的错误光照组合后即成环。
func swappedPostProcessPass() *pipeline.Pass {
	return pipeline.PostProcessPass()
}

// missingInputPipeline 删除 shadow pass，却保留 lighting 对 shadow 的读取，
// 属于不合法输入：资源没有生产者。
func missingInputPipeline() *pipeline.Pipeline {
	p := pipeline.New("missing-input", frameWidth, frameHeight)
	mustAdd(p, pipeline.GeometryPass())
	mustAdd(p, pipeline.ColorPass())
	mustAdd(p, pipeline.LightingPass())
	mustAdd(p, pipeline.PostProcessPass())
	mustAdd(p, pipeline.OutputPass())
	return p
}

func mustAdd(p *pipeline.Pipeline, pass *pipeline.Pass) {
	if err := p.Add(pass); err != nil {
		panic(err)
	}
}

func printHeader(title string) {
	fmt.Println("============================================================")
	fmt.Println(title)
	fmt.Println("============================================================")
}

func runScenario(name, outPath string) error {
	printHeader("scenario: " + name)

	var p *pipeline.Pipeline
	switch name {
	case "valid":
		p = validPipeline()
	case "swapped":
		p = swappedPipeline()
	case "missing":
		p = missingInputPipeline()
	default:
		return fmt.Errorf("unknown scenario %q (valid|swapped|missing)", name)
	}

	// 先建图：静态资源校验失败时直接拒绝，不执行任何 Pass。
	graph, err := p.BuildGraph()
	if err != nil {
		fmt.Printf("[reject] static validation failed: %v\n", err)
		fmt.Println("[reject] no frame executed, no image generated.")
		return err
	}

	fmt.Println("node graph (producer -> consumer):")
	fmt.Print(graph.String())

	// 再做拓扑排序：有环时给出完整循环路径并拒绝运行。
	plan, err := p.Plan()
	if err != nil {
		var cycleErr *pipeline.CycleError
		if errors.As(err, &cycleErr) {
			fmt.Printf("[reject] cycle detected: %s\n", pipeline.FormatPath(cycleErr.Path))
			fmt.Println("[reject] cyclic chain blocks execution; pipeline refuses to run.")
			fmt.Println("[reject] no frame executed, no image generated.")
			return err
		}
		return err
	}

	fmt.Println("execution order (topological):")
	for i, name := range plan.Order() {
		fmt.Printf("  %d. %s\n", i+1, name)
	}

	frame := p.NewFrame()
	if err := plan.Run(frame); err != nil {
		fmt.Printf("[error] execution aborted: %v\n", err)
		return err
	}

	buf, err := frame.Get(pipeline.ResPPMColor)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	if err := pipeline.WritePPM(outPath, buf); err != nil {
		return err
	}
	fmt.Printf("[ok] frame buffers: %d, final PPM written to %s (%dx%d)\n",
		len(frame.Buffers), outPath, p.Width(), p.Height())
	return nil
}

func main() {
	scenario := "valid"
	outPath := filepath.Join("artifacts", "frame.ppm")
	if len(os.Args) > 1 {
		scenario = os.Args[1]
	}
	if len(os.Args) > 2 {
		outPath = os.Args[2]
	}

	if err := runScenario(scenario, outPath); err != nil {
		os.Exit(1)
	}
}
