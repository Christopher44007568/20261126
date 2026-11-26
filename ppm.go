package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ReadPPM 解码测试和自检产物所需的 P6/P3 PPM。
// 主流程只写 P6；保留 P3 解析有助于读取常见 PPM 夹具。
func ReadPPM(r io.Reader) (*Image, error) {
	br := bufio.NewReader(r)
	magic, err := readPPMToken(br)
	if err != nil {
		return nil, err
	}
	if magic != "P6" && magic != "P3" {
		return nil, fmt.Errorf("unsupported PPM magic %q", magic)
	}
	widthText, err := readPPMToken(br)
	if err != nil {
		return nil, err
	}
	heightText, err := readPPMToken(br)
	if err != nil {
		return nil, err
	}
	maxText, err := readPPMToken(br)
	if err != nil {
		return nil, err
	}
	width, err := strconv.Atoi(widthText)
	if err != nil {
		return nil, fmt.Errorf("bad width: %w", err)
	}
	height, err := strconv.Atoi(heightText)
	if err != nil {
		return nil, fmt.Errorf("bad height: %w", err)
	}
	maxValue, err := strconv.Atoi(maxText)
	if err != nil || maxValue != 255 {
		return nil, fmt.Errorf("only maxval 255 is supported")
	}

	im := NewImage(width, height)
	pixelCount := width * height
	if magic == "P6" {
		rgb := make([]byte, pixelCount*3)
		if _, err := io.ReadFull(br, rgb); err != nil {
			return nil, err
		}
		for i := 0; i < pixelCount; i++ {
			im.Pix[i*4] = rgb[i*3]
			im.Pix[i*4+1] = rgb[i*3+1]
			im.Pix[i*4+2] = rgb[i*3+2]
			im.Pix[i*4+3] = 255
		}
		return im, nil
	}

	for i := 0; i < pixelCount; i++ {
		components := [3]uint8{}
		for ch := 0; ch < 3; ch++ {
			token, err := readPPMToken(br)
			if err != nil {
				return nil, err
			}
			value, err := strconv.Atoi(token)
			if err != nil || value < 0 || value > 255 {
				return nil, fmt.Errorf("bad P3 component %q", token)
			}
			components[ch] = uint8(value)
		}
		im.Pix[i*4] = components[0]
		im.Pix[i*4+1] = components[1]
		im.Pix[i*4+2] = components[2]
		im.Pix[i*4+3] = 255
	}
	return im, nil
}

func readPPMToken(r *bufio.Reader) (string, error) {
	var b strings.Builder
	for {
		ch, err := r.ReadByte()
		if err != nil {
			return "", err
		}
		if ch == '#' {
			for {
				line, err := r.ReadString('\n')
				if err != nil {
					return "", err
				}
				if strings.HasSuffix(line, "\n") {
					break
				}
			}
			continue
		}
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			if b.Len() > 0 {
				return b.String(), nil
			}
			continue
		}
		b.WriteByte(ch)
	}
}
