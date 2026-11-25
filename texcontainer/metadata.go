package texcontainer

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
)

const (
	maxMetadataBytes = 1 << 20
	maxMetadataKeys  = 256
)

var errMetadataTruncated = errors.New("texcontainer: truncated metadata record")

func encodeMetadata(metadata map[string]string) ([]byte, error) {
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	size := 4
	for _, key := range keys {
		if len(key) == 0 || len(key) > 65535 {
			return nil, fmt.Errorf("texcontainer: metadata key length invalid: %q", key)
		}
		if len(metadata[key]) > 0xffffffff {
			return nil, fmt.Errorf("texcontainer: metadata value too long for %q", key)
		}
		size += 2 + len(key) + 4 + len(metadata[key])
	}
	if size > maxMetadataBytes {
		return nil, fmt.Errorf("texcontainer: metadata is %d bytes, limit %d", size, maxMetadataBytes)
	}

	buf := make([]byte, size)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(len(keys)))
	pos := 4
	for _, key := range keys {
		value := metadata[key]
		binary.LittleEndian.PutUint16(buf[pos:pos+2], uint16(len(key)))
		pos += 2
		copy(buf[pos:pos+len(key)], key)
		pos += len(key)
		binary.LittleEndian.PutUint32(buf[pos:pos+4], uint32(len(value)))
		pos += 4
		copy(buf[pos:pos+len(value)], value)
		pos += len(value)
	}
	return buf, nil
}

func decodeMetadata(data []byte) (map[string]string, int, error) {
	if len(data) < 4 {
		return nil, 0, errMetadataTruncated
	}
	count := int(binary.LittleEndian.Uint32(data[0:4]))
	if count > maxMetadataKeys {
		return nil, 0, fmt.Errorf("texcontainer: too many metadata keys: %d", count)
	}
	metadata := make(map[string]string, count)
	pos := 4
	for i := 0; i < count; i++ {
		if pos+2 > len(data) {
			return nil, 0, errMetadataTruncated
		}
		keyLen := int(binary.LittleEndian.Uint16(data[pos : pos+2]))
		pos += 2
		if keyLen == 0 || pos+keyLen+4 > len(data) {
			return nil, 0, errMetadataTruncated
		}
		key := string(data[pos : pos+keyLen])
		pos += keyLen
		valueLen := int(binary.LittleEndian.Uint32(data[pos : pos+4]))
		pos += 4
		if pos+valueLen > len(data) {
			return nil, 0, errMetadataTruncated
		}
		if _, exists := metadata[key]; exists {
			return nil, 0, fmt.Errorf("texcontainer: duplicate metadata key %q", key)
		}
		metadata[key] = string(data[pos : pos+valueLen])
		pos += valueLen
	}
	return metadata, pos, nil
}
