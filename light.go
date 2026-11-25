package main

import "math"

type LightState struct {
	Name string
	CCT  float64
	RGB  RGB
}

func lightSequence() []LightState {
	states := make([]LightState, frameCount)
	sunset := blackbodyLight("sunset", 2700)
	fluorescent := fluorescentLight()
	shadow := blackbodyLight("shadow", 7500)

	for i := range states {
		switch {
		case i < 10:
			states[i] = sunset
		case i < 14:
			states[i] = mixLight(sunset, fluorescent, float64(i-10)/4)
		case i < 24:
			states[i] = fluorescent
		case i < 28:
			states[i] = mixLight(fluorescent, shadow, float64(i-24)/4)
		default:
			states[i] = shadow
		}
	}
	return states
}

func blackbodyLight(name string, cct float64) LightState {
	xyz := integrateSpectrum(func(wavelength float64) RGB {
		power := blackbodySpectrum(wavelength, cct)
		return RGB{R: cmfX(wavelength) * power, G: cmfY(wavelength) * power, B: cmfZ(wavelength) * power}
	})
	return makeLight(name, cct, xyz)
}

func fluorescentLight() LightState {
	base := integrateSpectrum(func(wavelength float64) RGB {
		broad := 0.45 * math.Exp(-math.Pow((wavelength-545)/55, 2))
		line := 0.85*math.Exp(-math.Pow((wavelength-435)/3, 2)) +
			0.95*math.Exp(-math.Pow((wavelength-545)/3, 2)) +
			0.75*math.Exp(-math.Pow((wavelength-615)/3, 2))
		power := broad + line
		return RGB{R: cmfX(wavelength) * power, G: cmfY(wavelength) * power, B: cmfZ(wavelength) * power}
	})
	return makeLight("fluorescent", cctFromXYZ(base), base)
}

func mixLight(a, b LightState, t float64) LightState {
	rgb := RGB{
		R: a.RGB.R*(1-t) + b.RGB.R*t,
		G: a.RGB.G*(1-t) + b.RGB.G*t,
		B: a.RGB.B*(1-t) + b.RGB.B*t,
	}
	name := "transition"
	if a.Name == "sunset" {
		name = "sunset_to_fluorescent"
	}
	if a.Name == "fluorescent" {
		name = "fluorescent_to_shadow"
	}
	return LightState{Name: name, CCT: a.CCT*(1-t) + b.CCT*t, RGB: normalizeRGB(rgb)}
}

func makeLight(name string, cct float64, xyz RGB) LightState {
	rgb := linearSRGBFromXYZ(xyz)
	return LightState{Name: name, CCT: cct, RGB: normalizeRGB(rgb)}
}

func normalizeRGB(c RGB) RGB {
	mean := (c.R + c.G + c.B) / 3
	return RGB{R: c.R / mean, G: c.G / mean, B: c.B / mean}
}

func integrateSpectrum(sample func(float64) RGB) RGB {
	total := RGB{}
	for wavelength := 380.0; wavelength <= 730.0; wavelength += 5 {
		part := sample(wavelength)
		total.R += part.R * 5
		total.G += part.G * 5
		total.B += part.B * 5
	}
	return total
}

func blackbodySpectrum(wavelengthNM, cct float64) float64 {
	waveM := wavelengthNM * 1e-9
	return 1 / (math.Pow(waveM, 5) * (math.Exp(1.4387769e-2/(waveM*cct)) - 1))
}

func cmfGaussian(wavelength, peak, widthLeft, widthRight float64) float64 {
	width := widthLeft
	if wavelength > peak {
		width = widthRight
	}
	return math.Exp(-0.5 * math.Pow((wavelength-peak)/width, 2))
}

func cmfX(wavelength float64) float64 {
	return 1.056*cmfGaussian(wavelength, 599.8, 37.9, 31.0) +
		0.362*cmfGaussian(wavelength, 442.0, 16.0, 26.7) -
		0.065*cmfGaussian(wavelength, 501.1, 20.4, 26.2)
}

func cmfY(wavelength float64) float64 {
	return 0.821*cmfGaussian(wavelength, 568.8, 46.9, 40.5) +
		0.286*cmfGaussian(wavelength, 530.9, 16.3, 31.1)
}

func cmfZ(wavelength float64) float64 {
	return 1.217*cmfGaussian(wavelength, 437.0, 11.8, 36.0) +
		0.681*cmfGaussian(wavelength, 459.0, 26.0, 13.8)
}

func linearSRGBFromXYZ(xyz RGB) RGB {
	return RGB{
		R: 3.2406*xyz.R - 1.5372*xyz.G - 0.4986*xyz.B,
		G: -0.9689*xyz.R + 1.8758*xyz.G + 0.0415*xyz.B,
		B: 0.0557*xyz.R - 0.2040*xyz.G + 1.0570*xyz.B,
	}
}

func xyzFromLinearSRGB(rgb RGB) RGB {
	return RGB{
		R: 0.4124*rgb.R + 0.3576*rgb.G + 0.1805*rgb.B,
		G: 0.2126*rgb.R + 0.7152*rgb.G + 0.0722*rgb.B,
		B: 0.0193*rgb.R + 0.1192*rgb.G + 0.9505*rgb.B,
	}
}

func cctFromXYZ(xyz RGB) float64 {
	sum := xyz.R + xyz.G + xyz.B
	if sum == 0 {
		return 6500
	}
	x := xyz.R / sum
	y := xyz.G / sum
	if y == 0 {
		return 6500
	}
	n := (x - 0.3320) / (0.1858 - y)
	cct := 437*n*n*n + 3601*n*n + 6861*n + 5517
	return math.Max(1500, math.Min(50000, cct))
}

func cctFromRGB(rgb RGB) float64 {
	return cctFromXYZ(xyzFromLinearSRGB(rgb))
}
