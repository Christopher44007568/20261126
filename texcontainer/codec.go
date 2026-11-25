package texcontainer

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"

	"texturelab/colorspace"
)

// Magic marks the custom KTX-style container: "TXCON1" plus two zero bytes.
var Magic = [8]byte{'T', 'X', 'C', 'O', 'N', '1', 0, 0}

const (
	Version           uint32 = 1
	headerSize               = 64
	indexEntrySize           = 40
	maxLevels                = 32
	maxDimension             = 16384
	maxContainerBytes        = 512 << 20
	pixelAlignment           = 4
)

// Encode serializes a texture into the deterministic TXCON1 container format.
func Encode(texture *Texture) ([]byte, error) {
	if err := texture.Validate(); err != nil {
		return nil, err
	}
	if len(texture.Levels) > maxLevels {
		return nil, fmt.Errorf("texcontainer: too many mip levels: %d", len(texture.Levels))
	}

	metadataBytes, err := encodeMetadata(texture.Metadata)
	if err != nil {
		return nil, err
	}

	type location struct {
		offset uint64
		size   uint64
		crc    uint32
	}
	locations := make([]location, len(texture.Levels))
	pixelSection := make([]byte, 0, 1024)

	for i, level := range texture.Levels {
		for len(pixelSection)%pixelAlignment != 0 {
			pixelSection = append(pixelSection, 0)
		}
		offset := uint64(len(pixelSection))
		pixelSection = append(pixelSection, level.Pixels...)
		locations[i] = location{
			offset: offset,
			size:   uint64(len(level.Pixels)),
			crc:    crc32.ChecksumIEEE(level.Pixels),
		}
	}

	levelCount := uint32(len(texture.Levels))
	indexLength := uint32(indexEntrySize * len(texture.Levels))
	header := make([]byte, headerSize)
	copy(header[0:8], Magic[:])
	binary.LittleEndian.PutUint32(header[8:12], Version)
	binary.LittleEndian.PutUint32(header[12:16], headerSize)
	binary.LittleEndian.PutUint32(header[16:20], uint32(texture.Format))
	binary.LittleEndian.PutUint32(header[20:24], uint32(texture.Levels[0].Width))
	binary.LittleEndian.PutUint32(header[24:28], uint32(texture.Levels[0].Height))
	binary.LittleEndian.PutUint32(header[28:32], levelCount)
	binary.LittleEndian.PutUint32(header[32:36], uint32(len(metadataBytes)))
	binary.LittleEndian.PutUint32(header[36:40], indexLength)
	binary.LittleEndian.PutUint64(header[40:48], uint64(len(pixelSection)))
	header[48] = byte(texture.Space)
	header[49] = 0

	index := make([]byte, indexLength)
	for i, level := range texture.Levels {
		entry := index[i*indexEntrySize : (i+1)*indexEntrySize]
		binary.LittleEndian.PutUint32(entry[0:4], uint32(i))
		binary.LittleEndian.PutUint32(entry[4:8], uint32(level.Width))
		binary.LittleEndian.PutUint32(entry[8:12], uint32(level.Height))
		binary.LittleEndian.PutUint32(entry[12:16], 0)
		binary.LittleEndian.PutUint64(entry[16:24], locations[i].offset)
		binary.LittleEndian.PutUint64(entry[24:32], locations[i].size)
		binary.LittleEndian.PutUint32(entry[32:36], locations[i].crc)
		binary.LittleEndian.PutUint32(entry[36:40], 0)
	}

	out := make([]byte, 0, headerSize+len(metadataBytes)+len(index)+len(pixelSection))
	out = append(out, header...)
	out = append(out, metadataBytes...)
	out = append(out, index...)
	out = append(out, pixelSection...)
	return out, nil
}

// Write encodes and writes the container to w.
func Write(w io.Writer, texture *Texture) error {
	data, err := Encode(texture)
	if err != nil {
		return err
	}
	written, err := w.Write(data)
	if err != nil {
		return err
	}
	if written != len(data) {
		return io.ErrShortWrite
	}
	return nil
}

// Read reads at most MaxContainerBytes and decodes a container.
func Read(r io.Reader) (*Texture, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxContainerBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxContainerBytes {
		return nil, fmt.Errorf("texcontainer: container exceeds %d bytes", maxContainerBytes)
	}
	return Decode(data)
}

// Decode parses and validates a complete TXCON1 container.
func Decode(data []byte) (*Texture, error) {
	if len(data) < headerSize {
		return nil, errors.New("texcontainer: truncated header")
	}
	if string(data[0:8]) != string(Magic[:]) {
		return nil, errors.New("texcontainer: bad magic")
	}
	if got := binary.LittleEndian.Uint32(data[8:12]); got != Version {
		return nil, fmt.Errorf("texcontainer: unsupported version %d", got)
	}
	if got := binary.LittleEndian.Uint32(data[12:16]); got != headerSize {
		return nil, fmt.Errorf("texcontainer: unsupported header size %d", got)
	}

	format := Format(binary.LittleEndian.Uint32(data[16:20]))
	baseWidth := int(binary.LittleEndian.Uint32(data[20:24]))
	baseHeight := int(binary.LittleEndian.Uint32(data[24:28]))
	levelCount := int(binary.LittleEndian.Uint32(data[28:32]))
	metadataLength := int(binary.LittleEndian.Uint32(data[32:36]))
	indexLength := int(binary.LittleEndian.Uint32(data[36:40]))
	dataLength := binary.LittleEndian.Uint64(data[40:48])
	space := data[48]

	if format != FormatRGBA8 {
		return nil, fmt.Errorf("%w: %d", ErrInvalidFormat, format)
	}
	if baseWidth <= 0 || baseHeight <= 0 || baseWidth > maxDimension || baseHeight > maxDimension {
		return nil, fmt.Errorf("%w: base %dx%d", ErrInvalidSize, baseWidth, baseHeight)
	}
	if levelCount <= 0 || levelCount > maxLevels {
		return nil, fmt.Errorf("texcontainer: invalid level count %d", levelCount)
	}
	if metadataLength > maxMetadataBytes || metadataLength < 0 {
		return nil, fmt.Errorf("texcontainer: invalid metadata length %d", metadataLength)
	}
	if indexLength != levelCount*indexEntrySize {
		return nil, fmt.Errorf("texcontainer: invalid index length %d for %d levels", indexLength, levelCount)
	}
	if !colorspace.Space(space).Valid() {
		return nil, fmt.Errorf("%w: %d", ErrInvalidSpace, space)
	}

	expectedLength := uint64(headerSize) + uint64(metadataLength) + uint64(indexLength) + dataLength
	if expectedLength > uint64(maxContainerBytes) {
		return nil, errors.New("texcontainer: declared container length is too large")
	}
	if uint64(len(data)) != expectedLength {
		return nil, fmt.Errorf("texcontainer: length mismatch: file=%d declared=%d", len(data), expectedLength)
	}

	pos := headerSize
	metadata, consumed, err := decodeMetadata(data[pos : pos+metadataLength])
	if err != nil {
		return nil, err
	}
	if consumed != metadataLength {
		return nil, fmt.Errorf("texcontainer: metadata padding/consumption mismatch: %d != %d", consumed, metadataLength)
	}
	pos += metadataLength

	indexStart := pos
	pixelStart := indexStart + indexLength
	texture := &Texture{
		Format:   format,
		Space:    colorspace.Space(space),
		Metadata: metadata,
		Levels:   make([]Level, 0, levelCount),
	}

	var cursor uint64
	for i := 0; i < levelCount; i++ {
		entry := data[indexStart+i*indexEntrySize : indexStart+(i+1)*indexEntrySize]
		entryLevel := int(binary.LittleEndian.Uint32(entry[0:4]))
		width := int(binary.LittleEndian.Uint32(entry[4:8]))
		height := int(binary.LittleEndian.Uint32(entry[8:12]))
		offset := binary.LittleEndian.Uint64(entry[16:24])
		size := binary.LittleEndian.Uint64(entry[24:32])
		wantCRC := binary.LittleEndian.Uint32(entry[32:36])

		if entryLevel != i {
			return nil, fmt.Errorf("texcontainer: index entry %d has level %d", i, entryLevel)
		}
		if width <= 0 || height <= 0 || width > maxDimension || height > maxDimension {
			return nil, fmt.Errorf("%w at level %d: %dx%d", ErrInvalidSize, i, width, height)
		}
		if offset%pixelAlignment != 0 || offset < cursor {
			return nil, fmt.Errorf("texcontainer: invalid pixel offset %d at level %d", offset, i)
		}
		if offset > dataLength || size > dataLength-offset {
			return nil, fmt.Errorf("texcontainer: pixel range out of bounds at level %d", i)
		}
		if size != uint64(width)*uint64(height)*uint64(BytesPerPixel) {
			return nil, fmt.Errorf("%w at level %d", ErrInvalidPixels, i)
		}
		cursor = offset + size

		pixels := append([]byte(nil), data[pixelStart+int(offset):pixelStart+int(offset+size)]...)
		if crc32.ChecksumIEEE(pixels) != wantCRC {
			return nil, fmt.Errorf("texcontainer: CRC mismatch at level %d", i)
		}
		texture.Levels = append(texture.Levels, Level{
			Width:  width,
			Height: height,
			Pixels: pixels,
		})
	}

	if texture.Levels[0].Width != baseWidth || texture.Levels[0].Height != baseHeight {
		return nil, errors.New("texcontainer: header base dimensions disagree with level zero")
	}
	return texture, nil
}
