package audit

import "math"

// SRGBToLinear decodes an IEC 61966-2-1 encoded channel to linear light.
func SRGBToLinear(value float64) float64 {
	value = clamp01(value)
	if value <= 0.04045 {
		return value / 12.92
	}
	return math.Pow((value+0.055)/1.055, 2.4)
}

// LinearToSRGB encodes linear light for an ordinary 8-bit display image.
func LinearToSRGB(value float64) float64 {
	value = clamp01(value)
	if value <= 0.0031308 {
		return 12.92 * value
	}
	return 1.055*math.Pow(value, 1.0/2.4) - 0.055
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func clamp(value, low, high float64) float64 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func mix(low, high, amount float64) float64 {
	return low + (high-low)*amount
}

func srgb8(value float64) uint8 {
	return uint8(math.Round(LinearToSRGB(value) * 255))
}
