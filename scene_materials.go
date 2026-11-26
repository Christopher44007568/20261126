package main

// surfaceVariants 是大多数节点使用的单 surface 槽位简写。
func surfaceVariants(summerID, winterID, nightID string) map[Variant]map[string]string {
	return map[Variant]map[string]string{
		VariantSummer: {"surface": summerID},
		VariantWinter: {"surface": winterID},
		VariantNight:  {"surface": nightID},
	}
}

func addCabinMaterials(stage *Stage) {
	materials := []Material{
		{ID: "sky.summer", Shader: "sky", Fill: hexRGB("#79C7EC"), Accent: hexRGB("#BDEBFF")},
		{ID: "sky.winter", Shader: "sky", Fill: hexRGB("#C8DDED"), Accent: hexRGB("#F1F8FF")},
		{ID: "sky.night", Shader: "sky_night", Fill: hexRGB("#0B1638"), Accent: hexRGB("#263E78"), Glow: hexRGB("#FFF3B0")},

		{ID: "sun.summer", Shader: "sun", Fill: hexRGB("#FFD447"), Accent: hexRGB("#FFF0A8")},
		{ID: "sun.winter", Shader: "sun", Fill: hexRGB("#F7E7A3"), Accent: hexRGB("#FFF8D6")},
		{ID: "moon.night", Shader: "moon", Fill: hexRGB("#F4F1D0"), Accent: hexRGB("#C9C49B"), Glow: hexRGB("#DDE8FF")},

		{ID: "mountain.back.summer", Shader: "mountain", Fill: hexRGB("#7895A8"), Accent: hexRGB("#DCEBF4")},
		{ID: "mountain.back.winter", Shader: "mountain_snow", Fill: hexRGB("#9FB8CA"), Accent: hexRGB("#F4FBFF")},
		{ID: "mountain.back.night", Shader: "mountain", Fill: hexRGB("#1B2A4A"), Accent: hexRGB("#31476F")},

		{ID: "mountain.front.summer", Shader: "mountain", Fill: hexRGB("#557383"), Accent: hexRGB("#E8F4FA")},
		{ID: "mountain.front.winter", Shader: "mountain_snow", Fill: hexRGB("#7893A5"), Accent: hexRGB("#FFFFFF")},
		{ID: "mountain.front.night", Shader: "mountain", Fill: hexRGB("#14223D"), Accent: hexRGB("#263A61")},

		{ID: "ground.summer", Shader: "ground", Fill: hexRGB("#5FA85A"), Accent: hexRGB("#477F45")},
		{ID: "ground.winter", Shader: "snow_ground", Fill: hexRGB("#F1FAFF"), Accent: hexRGB("#CFE5F2")},
		{ID: "ground.night", Shader: "ground_night", Fill: hexRGB("#1C3440"), Accent: hexRGB("#294B54")},

		{ID: "trunk.summer", Shader: "bark", Fill: hexRGB("#7A4A2B"), Accent: hexRGB("#5A341E")},
		{ID: "trunk.winter", Shader: "bark", Fill: hexRGB("#604333"), Accent: hexRGB("#3F2B22")},
		{ID: "trunk.night", Shader: "bark", Fill: hexRGB("#2E2438"), Accent: hexRGB("#1D1726")},

		{ID: "crown.summer", Shader: "foliage", Fill: hexRGB("#2F9E44"), Accent: hexRGB("#1F7A36"), Glow: hexRGB("#7BE082")},
		{ID: "crown.winter", Shader: "snow_crown", Fill: hexRGB("#274B3E"), Accent: hexRGB("#F4FBFF"), Glow: hexRGB("#D8ECF8")},
		{ID: "crown.night", Shader: "foliage_night", Fill: hexRGB("#173F35"), Accent: hexRGB("#0D2B25"), Glow: hexRGB("#3D8068")},

		{ID: "wall.summer", Shader: "planks", Fill: hexRGB("#E0B07A"), Accent: hexRGB("#A96F42")},
		{ID: "wall.winter", Shader: "planks_cold", Fill: hexRGB("#B98A61"), Accent: hexRGB("#76523A")},
		{ID: "wall.night", Shader: "planks_night", Fill: hexRGB("#3A3550"), Accent: hexRGB("#201D33")},

		{ID: "roof.summer", Shader: "roof_tiles", Fill: hexRGB("#B9543B"), Accent: hexRGB("#7E2F24")},
		{ID: "roof.winter", Shader: "roof_snow", Fill: hexRGB("#9E4A38"), Accent: hexRGB("#F4FBFF")},
		{ID: "roof.night", Shader: "roof_tiles_night", Fill: hexRGB("#5A2435"), Accent: hexRGB("#27111D")},

		{ID: "door.summer", Shader: "door", Fill: hexRGB("#74482C"), Accent: hexRGB("#F2C14E")},
		{ID: "door.winter", Shader: "door", Fill: hexRGB("#5E3C29"), Accent: hexRGB("#F6D878")},
		{ID: "door.night", Shader: "door_night", Fill: hexRGB("#2A2030"), Accent: hexRGB("#D9A441"), Glow: hexRGB("#FFB84D")},

		{ID: "window.summer", Shader: "glass", Fill: hexRGB("#8ED8FF"), Accent: hexRGB("#FFF4D1"), Glow: hexRGB("#D8F5FF")},
		{ID: "window.winter", Shader: "frosted_glass", Fill: hexRGB("#BFE9FF"), Accent: hexRGB("#F8FDFF"), Glow: hexRGB("#FFFFFF")},
		{ID: "window.night", Shader: "lit_window", Fill: hexRGB("#FFD166"), Accent: hexRGB("#3A2A1A"), Glow: hexRGB("#FFE9A8")},
	}
	for _, material := range materials {
		stage.AddMaterial(material)
	}
}
