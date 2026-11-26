package main

// Palette sampled to match the dark proof-board visual language.
var (
	colBgTop    = Hex("#0B0F17")
	colBgBottom = Hex("#0E1420")
	colCard     = Hex("#131A24")
	colCardEdge = Hex("#26313F")
	colViewport = Hex("#0B1018")
	colVPEdge   = Hex("#222B38")
	colInk      = Hex("#F2F5FA")
	colBody     = Hex("#C4CEDA")
	colMuted    = Hex("#93A1B8")
	colFaint    = Hex("#5B6B82")
	colAmber    = Hex("#F7C95B")
	colGreen    = Hex("#70E1A1")
	colRed      = Hex("#FF3355")
	colBanner   = Hex("#E72A4F")
	colRedDark  = Hex("#3A1322")

	colSlate      = Hex("#7B8799")
	colSlateLight = Hex("#93A1B8")
	colNavyRing   = Hex("#2A3648")
	colLens       = Hex("#D7E2EF")
)

const (
	boardW = 1080
	boardH = 720

	pageMargin = 36
	cardW      = 494
	cardH      = 566
	cardY      = 78
	cardGap    = 24
)

func cardRect(index int) (x, y, w, h int) {
	x = pageMargin + index*(cardW+cardGap)
	return x, cardY, cardW, cardH
}
