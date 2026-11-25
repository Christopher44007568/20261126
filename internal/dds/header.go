package dds

import (
	"encoding/binary"
	"fmt"
	"math/bits"
)

const (
	Magic             = "DDS "
	HeaderSize        = 124
	DX10HeaderSize    = 20
	MaxDimension      = 16384
	MaxDecodedBytes   = 512 << 20
	flagCaps          = 0x1
	flagHeight        = 0x2
	flagWidth         = 0x4
	flagPitch         = 0x8
	flagPixelFormat   = 0x1000
	flagMipMapCount   = 0x20000
	flagLinearSize    = 0x80000
	headerRequired    = flagHeight | flagWidth | flagPixelFormat
	ddspfAlphaPixels  = 0x1
	ddspfFourCC       = 0x4
	ddspfRGB          = 0x40
	ddspfLuminance    = 0x20000
	caps2Cubemap      = 0x200
	cubeFaceAll       = 0xFC00
	dx10ResourceCube  = 0x4
	dimensionTexture2 = 3
)

type header struct {
	flags       uint32
	width       int
	height      int
	pitch       int
	depth       int
	mipCount    int
	pfFlags     uint32
	fourCC      string
	bitCount    int
	rMask       uint32
	gMask       uint32
	bMask       uint32
	aMask       uint32
	caps2       uint32
	dxgiFormat  uint32
	resourceDim uint32
	arraySize   uint32
	miscFlag    uint32
	isCubemap   bool
	faceCount   int
	dataOffset  int64
	format      PixelFormat
}

func parseHeader(data []byte) (*header, *DecodeError) {
	errs := &DecodeError{}
	if len(data) < 4 {
		errs.addAt("magic", 0, "file is too short to contain the DDS signature", "4 bytes", fmt.Sprintf("%d bytes", len(data)))
		return nil, errs
	}
	if string(data[:4]) != Magic {
		errs.addAt("magic", 0, "not a DDS file", Magic, fmt.Sprintf("%q", string(data[:4])))
	}
	if len(data) < 4+HeaderSize {
		errs.addAt("header", 4, "file is too short to contain the complete 124-byte header",
			fmt.Sprintf(">= %d bytes", 4+HeaderSize), fmt.Sprintf("%d bytes", len(data)))
		return nil, errs
	}

	h := &header{}
	h.flags = u32(data, 8)
	h.height = int(i32(data, 12))
	h.width = int(i32(data, 16))
	h.pitch = int(i32(data, 20))
	h.depth = int(i32(data, 24))
	h.mipCount = int(u32(data, 28))
	h.pfFlags = u32(data, 80)
	h.fourCC = string(data[84:88])
	h.bitCount = int(u32(data, 88))
	h.rMask = u32(data, 92)
	h.gMask = u32(data, 96)
	h.bMask = u32(data, 100)
	h.aMask = u32(data, 104)
	h.caps2 = u32(data, 112)
	h.dataOffset = 128

	if size := u32(data, 4); size != HeaderSize {
		errs.addAt("header.dwSize", 4, "invalid header size", "124", fmt.Sprint(size))
	}
	if h.flags&headerRequired != headerRequired {
		errs.addAt("header.dwFlags", 8, "required WIDTH, HEIGHT and PIXELFORMAT flags are missing",
			fmt.Sprintf("0x%x", uint32(headerRequired)), fmt.Sprintf("0x%x", h.flags))
	}
	if h.width <= 0 || h.height <= 0 {
		errs.addAt("header.dwWidth/dwHeight", 12, "width and height must both be positive",
			"1..16384", fmt.Sprintf("%dx%d", h.width, h.height))
	} else if h.width > MaxDimension || h.height > MaxDimension {
		errs.addAt("header.dwWidth/dwHeight", 12, "dimensions exceed the supported limit",
			fmt.Sprintf("<= %d", MaxDimension), fmt.Sprintf("%dx%d", h.width, h.height))
	}
	if h.depth != 0 && h.depth != 1 {
		errs.addAt("header.dwDepth", 24, "volume textures are outside this DDS subset", "0 or 1", fmt.Sprint(h.depth))
	}

	h.parseMipCount(errs)
	h.parsePixelFormat(errs, data)
	if len(errs.Errors) > 0 {
		return nil, errs
	}
	if err := h.validateMasks(); err != nil {
		errs.Errors = append(errs.Errors, *err)
	}
	h.parseFaces(errs)
	h.validateByteBudget(errs)
	if len(errs.Errors) > 0 {
		return nil, errs
	}
	return h, nil
}

func (h *header) parseMipCount(errs *DecodeError) {
	if h.flags&flagMipMapCount == 0 {
		h.mipCount = 1
		return
	}
	if h.mipCount < 1 {
		errs.addAt("header.dwMipMapCount", 28, "declared mip count must be positive", ">= 1", fmt.Sprint(h.mipCount))
		return
	}
	maxLevels := fullMipCount(h.width, h.height)
	if h.width > 0 && h.height > 0 && h.mipCount > maxLevels {
		errs.addAt("header.dwMipMapCount", 28, "mip chain is longer than the base dimensions allow",
			fmt.Sprintf("<= %d", maxLevels), fmt.Sprint(h.mipCount))
	}
}

func fullMipCount(width, height int) int {
	largest := max(width, height)
	if largest <= 0 {
		return 0
	}
	return bits.Len(uint(largest))
}

func (h *header) parsePixelFormat(errs *DecodeError, data []byte) {
	if size := u32(data, 76); size != 32 {
		errs.addAt("header.ddspf.dwSize", 76, "invalid pixel format size", "32", fmt.Sprint(size))
	}
	switch {
	case h.pfFlags&ddspfFourCC != 0:
		h.parseFourCC(errs, data)
	case h.pfFlags&(ddspfRGB|ddspfLuminance) != 0:
		h.parseMasks(errs)
	default:
		errs.addAt("header.ddspf.dwFlags", 80, "pixel format is neither FOURCC nor RGB/luminance",
			"FOURCC or RGB", fmt.Sprintf("0x%x", h.pfFlags))
	}
}

func (h *header) parseFourCC(errs *DecodeError, data []byte) {
	switch h.fourCC {
	case "DX10":
		if len(data) < 128+DX10HeaderSize {
			errs.addAt("dx10", 128, "DX10 header is truncated", "20 bytes", fmt.Sprintf("%d bytes", len(data)-128))
			return
		}
		h.dxgiFormat = u32(data, 128)
		h.resourceDim = u32(data, 132)
		h.miscFlag = u32(data, 136)
		h.arraySize = u32(data, 140)
		h.dataOffset = 148
		if h.resourceDim != dimensionTexture2 {
			errs.addAt("dx10.resourceDimension", 132, "only 2D and cubemap resources are supported", "3", fmt.Sprint(h.resourceDim))
		}
		if h.miscFlag&dx10ResourceCube != 0 {
			h.isCubemap = true
		}
		if h.arraySize == 0 {
			errs.addAt("dx10.arraySize", 140, "array size must be positive", ">= 1", "0")
		} else if h.arraySize != 1 && h.arraySize != 6 {
			errs.addAt("dx10.arraySize", 140, "texture arrays are outside this DDS subset", "1 or 6", fmt.Sprint(h.arraySize))
		}
		if h.arraySize == 6 && !h.isCubemap {
			errs.addAt("dx10.arraySize", 140, "six-slice texture arrays are outside this DDS subset",
				"1 without a cubemap, or 1/6 with a cubemap", fmt.Sprint(h.arraySize))
		}
		if h.isCubemap && h.arraySize != 1 && h.arraySize != 6 {
			errs.addAt("dx10.arraySize", 140, "cubemap array must describe one cube", "1 or 6", fmt.Sprint(h.arraySize))
		}
		if err := h.setDXGIFormat(); err != nil {
			errs.Errors = append(errs.Errors, *err)
		}
	case "DXT1", "DXT3", "DXT5", "BC1 ", "BC2 ", "BC3 ":
		h.format = PixelFormat{Name: h.fourCC, FourCC: h.fourCC, Compressed: true}
		errs.addAt("header.ddspf.dwFourCC", 84, "compressed DDS formats are outside this minimal subset", "uncompressed RGBA", h.fourCC)
	default:
		name := fmt.Sprintf("%q", h.fourCC)
		errs.addAt("header.ddspf.dwFourCC", 84, "unsupported FOURCC", "DX10 or uncompressed masks", name)
	}
}

func (h *header) parseMasks(errs *DecodeError) {
	if h.bitCount != 8 && h.bitCount != 16 && h.bitCount != 24 && h.bitCount != 32 {
		errs.addAt("header.ddspf.dwRGBBitCount", 88, "unsupported sample width", "8, 16, 24 or 32", fmt.Sprint(h.bitCount))
	}
	if h.rMask == 0 && h.gMask == 0 && h.bMask == 0 {
		errs.addAt("header.ddspf.dwRBitMask", 92, "at least one color/luminance mask is required", "non-zero", "0")
	}
	h.format = PixelFormat{
		BitCount: h.bitCount,
		RMask:    h.rMask, GMask: h.gMask, BMask: h.bMask, AMask: h.aMask,
	}
}

func (h *header) validateMasks() *FieldError {
	if h.format.Compressed {
		return nil
	}
	masks := []uint32{h.rMask, h.gMask, h.bMask, h.aMask}
	used := uint32(0)
	for _, mask := range masks {
		if mask == 0 {
			continue
		}
		if mask&used != 0 {
			return &FieldError{Field: "header.ddspf.*BitMask", Offset: 92,
				Message: "channel masks overlap", Expected: "disjoint masks"}
		}
		used |= mask
	}
	if h.bitCount > 0 && used >= 1<<uint(h.bitCount) && h.bitCount < 32 {
		return &FieldError{Field: "header.ddspf.*BitMask", Offset: 92,
			Message: "channel mask is wider than the source sample", Expected: fmt.Sprintf("%d-bit sample", h.bitCount)}
	}
	hasColor := h.rMask != 0 || h.gMask != 0 || h.bMask != 0
	if !hasColor {
		return &FieldError{Field: "header.ddspf.dwRGBBitMask", Offset: 92,
			Message: "an RGB or luminance mask is required", Expected: "non-zero channel mask"}
	}
	return nil
}

func (h *header) parseFaces(errs *DecodeError) {
	cubeByCaps := h.caps2&caps2Cubemap != 0
	if cubeByCaps {
		if h.caps2&cubeFaceAll != cubeFaceAll {
			errs.addAt("header.dwCaps2", 112, "cubemap is missing one or more face flags",
				fmt.Sprintf("0x%x", uint32(cubeFaceAll)), fmt.Sprintf("0x%x", h.caps2))
		}
		h.isCubemap = true
	} else if h.caps2&cubeFaceAll != 0 {
		errs.addAt("header.dwCaps2", 112, "cube face bits are set without the CUBEMAP flag",
			fmt.Sprintf("CUBEMAP|0x%x", uint32(cubeFaceAll)), fmt.Sprintf("0x%x", h.caps2))
	}
	if h.isCubemap {
		h.faceCount = 6
		if h.width > 0 && h.height > 0 && h.width != h.height {
			errs.addAt("header.dwWidth/dwHeight", 12, "cubemap faces must be square",
				"width == height", fmt.Sprintf("%dx%d", h.width, h.height))
		}
	} else {
		h.faceCount = 1
	}
}

func (h *header) validateByteBudget(errs *DecodeError) {
	total := int64(0)
	for level := 0; level < h.mipCount; level++ {
		w, ht := mipSize(h.width, h.height, level)
		total += int64(w) * int64(ht) * 4 * int64(h.faceCount)
	}
	if total > MaxDecodedBytes {
		errs.addAt("header.dwWidth/dwHeight", 12, "decoded texture exceeds the memory budget",
			fmt.Sprintf("<= %d bytes", MaxDecodedBytes), fmt.Sprintf("%d bytes", total))
	}
}

func u32(data []byte, off int) uint32 {
	return binary.LittleEndian.Uint32(data[off : off+4])
}

func i32(data []byte, off int) int32 {
	return int32(binary.LittleEndian.Uint32(data[off : off+4]))
}
