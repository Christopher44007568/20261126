package dds

import (
	"fmt"
	"math/bits"
)

func (h *header) setDXGIFormat() *FieldError {
	type dxgi struct {
		name string
		bits int
		r, g uint32
		b, a uint32
	}
	formats := map[uint32]dxgi{
		28: {"RGBA8_UNORM", 32, 0x000000FF, 0x0000FF00, 0x00FF0000, 0xFF000000},
		29: {"RGBA8_SRGB", 32, 0x000000FF, 0x0000FF00, 0x00FF0000, 0xFF000000},
		87: {"BGRA8_UNORM", 32, 0x00FF0000, 0x0000FF00, 0x000000FF, 0xFF000000},
		88: {"BGRX8_UNORM", 32, 0x00FF0000, 0x0000FF00, 0x000000FF, 0},
		91: {"BGRA8_SRGB", 32, 0x00FF0000, 0x0000FF00, 0x000000FF, 0xFF000000},
		24: {"R10G10B10A2_UNORM", 32, 0x000003FF, 0x000FFC00, 0x3FF00000, 0xC0000000},
		61: {"R8_UNORM", 8, 0x000000FF, 0, 0, 0},
		65: {"A8_UNORM", 8, 0, 0, 0, 0x000000FF},
		49: {"R8G8_UNORM", 16, 0x000000FF, 0x0000FF00, 0, 0},
	}
	f, ok := formats[h.dxgiFormat]
	if !ok {
		return &FieldError{Field: "dx10.dxgiFormat", Offset: 128,
			Message: "DXGI format is outside this uncompressed subset", Expected: "28, 29, 87, 88, 91, 24, 61, 65 or 49",
			Actual: fmt.Sprint(h.dxgiFormat)}
	}
	h.bitCount = f.bits
	h.rMask, h.gMask, h.bMask, h.aMask = f.r, f.g, f.b, f.a
	h.format = PixelFormat{
		Name: f.name, FourCC: "DX10", BitCount: f.bits,
		RMask: f.r, GMask: f.g, BMask: f.b, AMask: f.a, DXGI: h.dxgiFormat,
	}
	return nil
}

func mipSize(width, height, level int) (int, int) {
	w, h := width, height
	for i := 0; i < level; i++ {
		if w > 1 {
			w /= 2
		}
		if h > 1 {
			h /= 2
		}
	}
	return w, h
}

func sampleBytes(bitCount int) int {
	return (bitCount + 7) / 8
}

func (h *header) rowPitch(level, width int) (int, *FieldError) {
	minPitch := width * sampleBytes(h.bitCount)
	if level != 0 {
		return minPitch, nil
	}
	if h.flags&flagPitch != 0 {
		if h.pitch <= 0 {
			return 0, &FieldError{
				Field: "header.dwPitchOrLinearSize", Offset: 20,
				Message:  "DDSD_PITCH is set but the row pitch is not positive",
				Expected: ">= 1", Actual: fmt.Sprint(h.pitch),
			}
		}
		return checkedPitch(h.pitch, minPitch)
	}
	if h.flags&flagLinearSize != 0 {
		if h.pitch <= 0 {
			return 0, &FieldError{
				Field: "header.dwPitchOrLinearSize", Offset: 20,
				Message:  "DDSD_LINEARSIZE is set but the linear size is not positive",
				Expected: ">= 1", Actual: fmt.Sprint(h.pitch),
			}
		}
		if h.height <= 0 || h.pitch%h.height != 0 {
			return 0, &FieldError{
				Field: "header.dwPitchOrLinearSize", Offset: 20,
				Message:  "top-level linear size cannot be split into equally pitched rows",
				Expected: fmt.Sprintf("a multiple of height %d", h.height), Actual: fmt.Sprint(h.pitch),
			}
		}
		return checkedPitch(h.pitch/h.height, minPitch)
	}
	if h.pitch > 0 {
		return checkedPitch(h.pitch, minPitch)
	}
	return minPitch, nil
}

func checkedPitch(pitch, minPitch int) (int, *FieldError) {
	if pitch < minPitch {
		return 0, &FieldError{
			Field: "header.dwPitchOrLinearSize", Offset: 20,
			Message: "row pitch is too small for the base mip", Expected: fmt.Sprintf(">= %d", minPitch),
			Actual: fmt.Sprint(pitch),
		}
	}
	const maxReasonablePitch = 1 << 28
	if pitch > maxReasonablePitch {
		return 0, &FieldError{
			Field: "header.dwPitchOrLinearSize", Offset: 20,
			Message: "row pitch exceeds the safety limit", Expected: fmt.Sprintf("<= %d", maxReasonablePitch),
			Actual: fmt.Sprint(pitch),
		}
	}
	return pitch, nil
}

func (h *header) levelLayout(level int) (w, ht, pitch, bytes int, ferr *FieldError) {
	w, ht = mipSize(h.width, h.height, level)
	pitch, ferr = h.rowPitch(level, w)
	if ferr != nil {
		return
	}
	bytes = pitch * ht
	return
}

func describeMasks(f PixelFormat) string {
	if f.Name != "" {
		return f.Name
	}
	if bits.OnesCount32(f.RMask) == 8 && bits.OnesCount32(f.GMask) == 8 &&
		bits.OnesCount32(f.BMask) == 8 && f.AMask == 0 && f.BitCount == 32 &&
		f.RMask == 0x00FF0000 && f.GMask == 0x0000FF00 && f.BMask == 0x000000FF {
		return "BGRX8"
	}
	if f.RMask == 0xFF && f.GMask == 0xFF00 && f.BMask == 0xFF0000 {
		if f.AMask == 0xFF000000 {
			return "RGBA8"
		}
		if f.AMask == 0 {
			return "RGBX8"
		}
	}
	if f.RMask == 0xFF0000 && f.GMask == 0xFF00 && f.BMask == 0xFF {
		if f.AMask == 0xFF000000 {
			return "BGRA8"
		}
		if f.AMask == 0 {
			return "BGRX8"
		}
	}
	if f.RMask != 0 && f.GMask == 0 && f.BMask == 0 {
		return fmt.Sprintf("L%d", f.BitCount)
	}
	if f.RMask != 0 && f.RMask == f.GMask && f.GMask == f.BMask {
		return fmt.Sprintf("L%d", f.BitCount)
	}
	return fmt.Sprintf("masked%d", f.BitCount)
}

func bitsToByte(value, mask uint32) uint8 {
	if mask == 0 {
		return 0
	}
	shift := bits.TrailingZeros32(mask)
	width := bits.OnesCount32(mask)
	v := (value & mask) >> uint(shift)
	if width >= 8 {
		return uint8(v >> uint(width-8))
	}
	maxValue := uint32((1 << uint(width)) - 1)
	return uint8((v*255 + maxValue/2) / maxValue)
}
