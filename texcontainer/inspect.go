package texcontainer

import (
	"encoding/binary"
	"fmt"
)

// LevelInfo is the persisted index record for one mip level.
type LevelInfo struct {
	Level          int    `json:"level"`
	Width          int    `json:"width"`
	Height         int    `json:"height"`
	RelativeOffset uint64 `json:"relative_offset"`
	AbsoluteOffset uint64 `json:"absolute_offset"`
	Size           uint64 `json:"size"`
	CRC32          string `json:"crc32"`
}

// Info describes a container header, metadata and level index without requiring
// callers to calculate binary offsets themselves.
type Info struct {
	Version            uint32            `json:"version"`
	Format             Format            `json:"format"`
	ColorSpace         string            `json:"color_space"`
	BaseWidth          int               `json:"base_width"`
	BaseHeight         int               `json:"base_height"`
	LevelCount         int               `json:"level_count"`
	HeaderSize         int               `json:"header_size"`
	MetadataLength     int               `json:"metadata_length"`
	IndexLength        int               `json:"index_length"`
	PixelSectionOffset uint64            `json:"pixel_section_offset"`
	PixelSectionLength uint64            `json:"pixel_section_length"`
	FileSize           int               `json:"file_size"`
	Metadata           map[string]string `json:"metadata"`
	Levels             []LevelInfo       `json:"levels"`
}

// Inspect validates data and returns its KTX-style layout information.
func Inspect(data []byte) (*Info, error) {
	texture, err := Decode(data)
	if err != nil {
		return nil, err
	}
	if len(data) < headerSize {
		return nil, ErrInvalidSize
	}

	metadataLength := int(binary.LittleEndian.Uint32(data[32:36]))
	indexLength := int(binary.LittleEndian.Uint32(data[36:40]))
	pixelLength := binary.LittleEndian.Uint64(data[40:48])
	indexStart := headerSize + metadataLength
	pixelStart := uint64(indexStart + indexLength)

	info := &Info{
		Version:            binary.LittleEndian.Uint32(data[8:12]),
		Format:             Format(binary.LittleEndian.Uint32(data[16:20])),
		ColorSpace:         texture.Space.String(),
		BaseWidth:          int(binary.LittleEndian.Uint32(data[20:24])),
		BaseHeight:         int(binary.LittleEndian.Uint32(data[24:28])),
		LevelCount:         len(texture.Levels),
		HeaderSize:         headerSize,
		MetadataLength:     metadataLength,
		IndexLength:        indexLength,
		PixelSectionOffset: pixelStart,
		PixelSectionLength: pixelLength,
		FileSize:           len(data),
		Metadata:           texture.Metadata,
		Levels:             make([]LevelInfo, 0, len(texture.Levels)),
	}

	for i := 0; i < len(texture.Levels); i++ {
		entry := data[indexStart+i*indexEntrySize : indexStart+(i+1)*indexEntrySize]
		relativeOffset := binary.LittleEndian.Uint64(entry[16:24])
		size := binary.LittleEndian.Uint64(entry[24:32])
		crc := binary.LittleEndian.Uint32(entry[32:36])
		info.Levels = append(info.Levels, LevelInfo{
			Level:          int(binary.LittleEndian.Uint32(entry[0:4])),
			Width:          int(binary.LittleEndian.Uint32(entry[4:8])),
			Height:         int(binary.LittleEndian.Uint32(entry[8:12])),
			RelativeOffset: relativeOffset,
			AbsoluteOffset: pixelStart + relativeOffset,
			Size:           size,
			CRC32:          fmt.Sprintf("%08x", crc),
		})
	}
	return info, nil
}
