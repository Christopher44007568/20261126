package main

import (
	"fmt"
	"path/filepath"

	"shadowflags/internal/shadow"
)

func main() {
	scene := shadow.BuildScene()
	outDir := "artifacts"

	for _, cfg := range shadow.RenderConfigs {
		img := scene.Render(cfg.Flag)
		path := filepath.Join(outDir, "shadow_"+cfg.Name+".png")
		if err := shadow.WritePNG(path, img); err != nil {
			fmt.Println("write error:", err)
			return
		}
		counts := scene.CountTags(cfg.Flag)
		fmt.Printf("%-18s -> %s  extra=%d ignored=%d closureExit=%d\n",
			cfg.Name, path, counts.ExtraOccluder, counts.IgnoredBackface, counts.ClosureExit)
	}

	cubeTop := shadow.Vec3{X: 4.5, Y: shadow.CubeTopY, Z: 0.5}
	ray := shadow.ShadowRay(shadow.PointUnderObject(cubeTop))
	fmt.Printf("\ncube center ray energy: backfaceCull=%.2f doubleSided=%.2f closure=%.2f naiveDoubleCount=%.2f\n",
		scene.Evaluate(ray, shadow.BackfaceCull).Energy,
		scene.Evaluate(ray, shadow.DoubleSidedHit).Energy,
		scene.Evaluate(ray, shadow.ReflectionClosure).Energy,
		scene.NaiveDoubleCount(ray),
	)
}
