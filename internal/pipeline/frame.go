package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
)

// Buffer 是一块具名的逐像素资源（深度、阴影、颜色等）。
// 每个像素占 Channels 个 float64，取值约定在 [0,1]。
type Buffer struct {
	Name     string
	Width    int
	Height   int
	Channels int
	Data     []float64
}

// At 返回像素 (x,y) 第 ch 个通道的指针，便于 Pass 直接写入。
func (b *Buffer) At(x, y, ch int) *float64 {
	return &b.Data[(y*b.Width+x)*b.Channels+ch]
}

// Get 读取像素 (x,y) 第 ch 个通道。
func (b *Buffer) Get(x, y, ch int) float64 {
	return b.Data[(y*b.Width+x)*b.Channels+ch]
}

// Frame 持有一次管线执行过程中的全部缓冲。
type Frame struct {
	Width   int
	Height  int
	Buffers map[string]*Buffer
}

// NewFrame 创建空帧。
func NewFrame(width, height int) *Frame {
	return &Frame{
		Width:   width,
		Height:  height,
		Buffers: make(map[string]*Buffer),
	}
}

// Ensure 返回名为 name 的缓冲；不存在时按 channels 分配。
func (f *Frame) Ensure(name string, channels int) *Buffer {
	if buf, ok := f.Buffers[name]; ok {
		return buf
	}
	buf := &Buffer{
		Name:     name,
		Width:    f.Width,
		Height:   f.Height,
		Channels: channels,
		Data:     make([]float64, f.Width*f.Height*channels),
	}
	f.Buffers[name] = buf
	return buf
}

// Get 取缓冲，不存在时返回错误而不是静默生成半张画面。
func (f *Frame) Get(name string) (*Buffer, error) {
	buf, ok := f.Buffers[name]
	if !ok {
		return nil, fmt.Errorf("frame: buffer %q not found", name)
	}
	return buf, nil
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// EncodePPM 把一个三通道（或以上）颜色缓冲编码为二进制 PPM(P6)。
func EncodePPM(buf *Buffer) ([]byte, error) {
	if buf.Channels < 3 {
		return nil, fmt.Errorf("ppm: buffer %q has %d channels, need at least 3", buf.Name, buf.Channels)
	}
	header := fmt.Sprintf("P6\n%d %d\n255\n", buf.Width, buf.Height)
	out := make([]byte, 0, len(header)+buf.Width*buf.Height*3)
	out = append(out, header...)
	for px := 0; px < buf.Width*buf.Height; px++ {
		base := px * buf.Channels
		for ch := 0; ch < 3; ch++ {
			out = append(out, byte(clamp01(buf.Data[base+ch])*255+0.5))
		}
	}
	return out, nil
}

// WritePPM 原子写入 PPM：先写同目录临时文件再 rename，避免失败时留下半张画面。
func WritePPM(path string, buf *Buffer) error {
	data, err := EncodePPM(buf)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".ppm-*")
	if err != nil {
		return fmt.Errorf("ppm: create temp file: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("ppm: write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("ppm: close temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("ppm: rename to %s: %w", path, err)
	}
	return nil
}
