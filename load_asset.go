package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"os"
)

func LoadAsset(path string) (*RelocatedAS, *AssetFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	headerSize := binary.Size(diskHeader{})
	if len(data) < headerSize+4 {
		return nil, nil, fmt.Errorf("asset %q is truncated", path)
	}

	var header diskHeader
	if err := binary.Read(bytes.NewReader(data[:headerSize]), binary.LittleEndian, &header); err != nil {
		return nil, nil, fmt.Errorf("read header: %w", err)
	}
	if string(header.Magic[:]) != assetMagic {
		return nil, nil, fmt.Errorf("bad asset magic %q", string(header.Magic[:]))
	}
	if header.Version != assetVersion {
		return nil, nil, fmt.Errorf("unsupported asset version %d", header.Version)
	}
	if int(header.HeaderSize) != headerSize {
		return nil, nil, fmt.Errorf("header size mismatch: disk=%d runtime=%d", header.HeaderSize, headerSize)
	}

	payloadEnd := len(data) - 4
	expectedTrailer := binary.LittleEndian.Uint32(data[payloadEnd:])
	actualTrailer := crc32.ChecksumIEEE(data[:payloadEnd])
	if expectedTrailer != actualTrailer {
		return nil, nil, fmt.Errorf("trailer CRC mismatch: stored=%08x actual=%08x", expectedTrailer, actualTrailer)
	}
	payload := data[headerSize:payloadEnd]
	if header.PayloadCRC != crc32.ChecksumIEEE(payload) {
		return nil, nil, fmt.Errorf("payload CRC mismatch: stored=%08x actual=%08x",
			header.PayloadCRC, crc32.ChecksumIEEE(payload))
	}

	r := bytes.NewReader(payload)
	file := &AssetFile{Header: header, Checksum: expectedTrailer}

	file.Nodes = make([]DiskNode, header.NodeCount)
	if err := binary.Read(r, binary.LittleEndian, file.Nodes); err != nil {
		return nil, nil, fmt.Errorf("read nodes: %w", err)
	}

	diskTriangles := make([]DiskTriangle, header.TriangleCount)
	if err := binary.Read(r, binary.LittleEndian, diskTriangles); err != nil {
		return nil, nil, fmt.Errorf("read triangles: %w", err)
	}
	file.Triangles = make([]Triangle, len(diskTriangles))
	for i, rec := range diskTriangles {
		file.Triangles[i] = rec.Runtime()
	}

	file.Order = make([]uint32, header.OrderCount)
	if err := binary.Read(r, binary.LittleEndian, file.Order); err != nil {
		return nil, nil, fmt.Errorf("read order: %w", err)
	}

	file.InstanceNames = make([]string, header.InstanceCount)
	file.Instances = make([]DiskInstance, header.InstanceCount)
	for i := range file.Instances {
		if err := binary.Read(r, binary.LittleEndian, &file.Instances[i]); err != nil {
			return nil, nil, fmt.Errorf("read instance %d: %w", i, err)
		}
		var nameLen uint16
		if err := binary.Read(r, binary.LittleEndian, &nameLen); err != nil {
			return nil, nil, fmt.Errorf("read instance name length: %w", err)
		}
		nameBytes := make([]byte, nameLen)
		if _, err := r.Read(nameBytes); err != nil {
			return nil, nil, fmt.Errorf("read instance name: %w", err)
		}
		file.InstanceNames[i] = string(nameBytes)
	}

	file.DiskToOriginal = make([]uint32, header.PermutationCount)
	if err := binary.Read(r, binary.LittleEndian, file.DiskToOriginal); err != nil {
		return nil, nil, fmt.Errorf("read permutation: %w", err)
	}
	if r.Len() != 0 {
		return nil, nil, fmt.Errorf("asset has %d unexpected trailing bytes", r.Len())
	}

	rt := RelocateAsset(file, path)
	return rt, file, nil
}
