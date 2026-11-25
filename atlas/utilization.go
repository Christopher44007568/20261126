package atlas

type SpriteUtilization struct {
	ID            string  `json:"id"`
	Rotated       bool    `json:"rotated"`
	FrameArea     int     `json:"frame_area"`
	CellArea      int     `json:"cell_area"`
	PaddingArea   int     `json:"padding_area"`
	CellFillRatio float64 `json:"cell_fill_ratio"`
	AtlasShare    float64 `json:"atlas_share"`
}

type Utilization struct {
	AtlasWidth  int `json:"atlas_width"`
	AtlasHeight int `json:"atlas_height"`
	AtlasArea   int `json:"atlas_area"`
	SpriteArea  int `json:"sprite_area"`
	FrameArea   int `json:"frame_area"`
	CellArea    int `json:"cell_area"`
	PaddingArea int `json:"padding_area"`
	FreeArea    int `json:"free_area"`

	// TextureUtilization is actual sprite texels divided by atlas area.
	TextureUtilization float64 `json:"texture_utilization"`

	// CellUtilization includes reserved mip-safe padding cells.
	CellUtilization float64 `json:"cell_utilization"`

	// PackingEfficiency is sprite area divided by reserved cell area.
	PackingEfficiency float64 `json:"packing_efficiency"`

	MipLevelCount     int                 `json:"mip_level_count"`
	MipChainArea      int                 `json:"mip_chain_area"`
	MipChainAreaRatio float64             `json:"mip_chain_area_ratio"`
	Sprites           []SpriteUtilization `json:"sprites"`
}

func CalculateUtilization(layout *Layout, mipCount int) Utilization {
	u := Utilization{
		AtlasWidth: layout.Width, AtlasHeight: layout.Height,
		AtlasArea:     layout.Width * layout.Height,
		MipLevelCount: maxInt(1, mipCount),
		Sprites:       make([]SpriteUtilization, 0, len(layout.Placements)),
	}

	for _, p := range layout.Placements {
		frameArea := p.Frame.Area()
		cellArea := p.Cell.Area()
		u.SpriteArea += p.SourceW * p.SourceH
		u.FrameArea += frameArea
		u.CellArea += cellArea
		u.Sprites = append(u.Sprites, SpriteUtilization{
			ID: p.ID, Rotated: p.Rotated,
			FrameArea: frameArea, CellArea: cellArea,
			PaddingArea:   cellArea - frameArea,
			CellFillRatio: ratio(frameArea, cellArea),
			AtlasShare:    ratio(frameArea, u.AtlasArea),
		})
	}

	u.PaddingArea = u.CellArea - u.FrameArea
	u.FreeArea = u.AtlasArea - u.CellArea
	u.TextureUtilization = ratio(u.SpriteArea, u.AtlasArea)
	u.CellUtilization = ratio(u.CellArea, u.AtlasArea)
	u.PackingEfficiency = ratio(u.SpriteArea, u.CellArea)

	w, h := layout.Width, layout.Height
	for level := 0; level < u.MipLevelCount; level++ {
		u.MipChainArea += maxInt(1, w>>level) * maxInt(1, h>>level)
	}
	u.MipChainAreaRatio = ratio(u.MipChainArea, u.AtlasArea)
	return u
}

func ratio(a, b int) float64 {
	if b <= 0 {
		return 0
	}
	return float64(a) / float64(b)
}
