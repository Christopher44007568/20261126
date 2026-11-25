package render

import (
	"animationtiming/raster"
	"animationtiming/timeline"
)

func drawHero(c *raster.Canvas, cx, groundY int, pose timeline.Pose) {
	ellipse(c, cx, groundY+5, 36, 7, raster.Hex(0xd6deea))

	hip := point{X: float64(cx), Y: float64(groundY-68) - pose.OffY}

	kneeL := polar(hip, pose.LegL, 31)
	ankleL := polar(kneeL, pose.LegL+pose.KneeL, 31)
	kneeR := polar(hip, pose.LegR, 31)
	ankleR := polar(kneeR, pose.LegR+pose.KneeR, 31)

	lineF(c, hip, kneeL, 11, ink)
	lineF(c, kneeL, ankleL, 11, ink)
	lineF(c, hip, kneeR, 11, ink)
	lineF(c, kneeR, ankleR, 11, ink)
	ellipse(c, int(ankleL.X)-2, int(ankleL.Y)+2, 11, 6, blue)
	ellipse(c, int(ankleR.X)+2, int(ankleR.Y)+2, 11, 6, blue)

	shoulder := polar(hip, 180-pose.Lean, 35)
	neck := polar(hip, 180-pose.Lean, 44)
	lineF(c, hip, neck, 22, ink)
	lineF(c, hip, neck, 14, blue)

	elbowL := polar(shoulder, pose.ArmL, 25)
	handL := polar(elbowL, pose.ArmL+pose.ElbowL, 23)
	elbowR := polar(shoulder, pose.ArmR, 25)
	handR := polar(elbowR, pose.ArmR+pose.ElbowR, 23)
	lineF(c, shoulder, elbowL, 9, ink)
	lineF(c, elbowL, handL, 9, ink)
	lineF(c, shoulder, elbowR, 9, ink)
	lineF(c, elbowR, handR, 9, ink)
	discF(c, handL, 6, blue)
	discF(c, handR, 6, blue)

	head := polar(hip, 180-pose.Lean, 58)
	discF(c, head, 16, ink)
	c.FillRect(int(head.X)-14, int(head.Y)-7, 28, 5, blue)
	discF(c, point{X: head.X - 5, Y: head.Y - 1}, 3, white)
	discF(c, point{X: head.X + 6, Y: head.Y - 1}, 3, white)
	discF(c, point{X: head.X - 5, Y: head.Y - 1}, 1, ink)
	discF(c, point{X: head.X + 6, Y: head.Y - 1}, 1, ink)
}

func drawBeacon(c *raster.Canvas, cx, groundY int) {
	ellipse(c, cx, groundY+4, 22, 6, raster.Hex(0xd6deea))
	ellipse(c, cx, groundY, 18, 6, raster.Hex(0x9aa7b8))
	c.FillRect(cx-4, groundY-72, 8, 72, raster.Hex(0x7c8aa0))
	c.FillRect(cx-2, groundY-70, 2, 68, raster.Hex(0xc4cdd9))

	orbY := groundY - 89
	discF(c, point{X: float64(cx), Y: float64(orbY)}, 18, greenPale)
	c.Circle(cx, orbY, 14, 4, teal)
	discF(c, point{X: float64(cx), Y: float64(orbY)}, 9, green)
	discF(c, point{X: float64(cx - 4), Y: float64(orbY - 4)}, 3, white)
}

func drawBuddy(c *raster.Canvas, cx, groundY int, pose timeline.Pose) {
	ellipse(c, cx, groundY+5, 37, 7, raster.Hex(0xd6deea))
	cy := groundY - 28

	lineF(c, point{X: float64(cx + 4), Y: float64(cy - 27)},
		point{X: float64(cx + 10), Y: float64(cy - 43)}, 3, purpleDk)
	discF(c, point{X: float64(cx + 11), Y: float64(cy - 46)}, 5, gold)

	leftShoulder := point{X: float64(cx - 27), Y: float64(cy - 2)}
	rightShoulder := point{X: float64(cx + 27), Y: float64(cy - 2)}
	leftHand := polar(leftShoulder, pose.ArmL, 18)
	rightHand := polar(rightShoulder, pose.ArmR, 18)
	lineF(c, leftShoulder, leftHand, 7, purpleDk)
	lineF(c, rightShoulder, rightHand, 7, purpleDk)
	discF(c, leftHand, 5, purple)
	discF(c, rightHand, 5, purple)

	ellipse(c, cx-15, groundY-2, 11, 6, purpleDk)
	ellipse(c, cx+15, groundY-2, 11, 6, purpleDk)
	ellipse(c, cx, cy, 35, 30, purple)
	ellipse(c, cx, cy+7, 22, 18, raster.Hex(0xa78bfa))

	discF(c, point{X: float64(cx - 11), Y: float64(cy - 7)}, 8, white)
	discF(c, point{X: float64(cx + 11), Y: float64(cy - 7)}, 8, white)
	discF(c, point{X: float64(cx - 9), Y: float64(cy - 6)}, 3, ink)
	discF(c, point{X: float64(cx + 13), Y: float64(cy - 6)}, 3, ink)
	lineF(c, point{X: float64(cx - 6), Y: float64(cy + 9)},
		point{X: float64(cx + 6), Y: float64(cy + 9)}, 2, purpleDk)
}
