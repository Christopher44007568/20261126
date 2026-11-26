package main

import "testing"

func TestSelectionPropagationAndOverrideRestore(t *testing.T) {
	stage := BuildCabin()

	if err := stage.SelectVariant(AssetPath, VariantWinter); err != nil {
		t.Fatal(err)
	}
	left, err := stage.Resolve(LeftWindowPath)
	if err != nil {
		t.Fatal(err)
	}
	if left.Variant != VariantWinter || left.SourcePath != AssetPath {
		t.Fatalf("parent propagation = %+v, want winter from %s", left, AssetPath)
	}

	token, err := stage.Override(WindowBankPath, VariantNight)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{LeftWindowPath, RightWindowPath} {
		resolution, resolveErr := stage.Resolve(path)
		if resolveErr != nil {
			t.Fatal(resolveErr)
		}
		if resolution.Variant != VariantNight || resolution.SourcePath != WindowBankPath {
			t.Fatalf("%s = %+v, want night from WindowBank", path, resolution)
		}
	}

	roof, err := stage.Resolve(RoofPath)
	if err != nil {
		t.Fatal(err)
	}
	if roof.Variant != VariantWinter || roof.SourcePath != AssetPath {
		t.Fatalf("sibling roof = %+v, want winter from Asset", roof)
	}

	if err := stage.Restore(token); err != nil {
		t.Fatal(err)
	}
	left, _ = stage.Resolve(LeftWindowPath)
	if left.Variant != VariantWinter || left.SourcePath != AssetPath {
		t.Fatalf("token restore = %+v, want inherited winter", left)
	}
	if _, authored := stage.selections[WindowBankPath]; authored {
		t.Fatal("restore did not delete absent override opinion")
	}

	if err := stage.SelectVariant(AssetPath, VariantSummer); err != nil {
		t.Fatal(err)
	}
	final, _ := stage.Resolve(LeftWindowPath)
	if final.Variant != VariantSummer || final.SourcePath != AssetPath {
		t.Fatalf("original selection = %+v, want summer", final)
	}
}

func TestSwitchBackRestoresPixelsExactly(t *testing.T) {
	stage := BuildCabin()
	renderer := NewRenderer()

	original, err := renderer.Render(stage, RenderOptions{})
	if err != nil {
		t.Fatal(err)
	}

	for _, variant := range []Variant{VariantWinter, VariantNight, VariantWinter, VariantSummer} {
		if err := stage.SelectVariant(AssetPath, variant); err != nil {
			t.Fatal(err)
		}
		frame, renderErr := renderer.Render(stage, RenderOptions{})
		if renderErr != nil {
			t.Fatal(renderErr)
		}
		if frame.PixelSHA256() == original.PixelSHA256() && variant != VariantSummer {
			t.Fatalf("variant %s unexpectedly matches summer", variant)
		}
	}

	token, err := stage.Override(WindowBankPath, VariantNight)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := renderer.Render(stage, RenderOptions{HighlightPath: LeftWindowPath}); err != nil {
		t.Fatal(err)
	}
	if err := stage.Restore(token); err != nil {
		t.Fatal(err)
	}
	if err := stage.SelectVariant(AssetPath, VariantSummer); err != nil {
		t.Fatal(err)
	}

	restored, err := renderer.Render(stage, RenderOptions{})
	if err != nil {
		t.Fatal(err)
	}
	comparison := compareImages(original, restored)
	if !comparison.Identical || comparison.DifferentPixels != 0 {
		t.Fatalf("restore comparison = %+v", comparison)
	}
	if string(original.Pix) != string(restored.Pix) {
		t.Fatal("RGBA buffers differ despite zero RGB differences")
	}
}

func TestNameOnlyChangeIsNotBinding(t *testing.T) {
	stage := BuildCabin()
	renderer := NewRenderer()
	before, err := renderer.Render(stage, RenderOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if err := stage.ChangeMaterialNameOnly(RoofPath, "surface", "roof.winter"); err != nil {
		t.Fatal(err)
	}
	if err := stage.ValidateBindings(); err == nil {
		t.Fatal("ValidateBindings accepted name-only binding")
	}
	if _, err := renderer.Render(stage, RenderOptions{}); err == nil {
		t.Fatal("renderer accepted name-only binding")
	}

	if err := stage.Rebind(); err != nil {
		t.Fatal(err)
	}
	after, err := renderer.Render(stage, RenderOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if comparison := compareImages(before, after); !comparison.Identical {
		t.Fatalf("rebind repair changed %d pixels", comparison.DifferentPixels)
	}
}
