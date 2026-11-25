package preview

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"dds-preview/internal/dds"
)

// PPM encodes a binary P6 PPM. The decoded RGBA alpha is ignored; callers
// that need compositing must do it before calling this function.
func PPM(img dds.Image) ([]byte, error) {
	if img.Width <= 0 || img.Height <= 0 {
		return nil, fmt.Errorf("invalid image dimensions %dx%d", img.Width, img.Height)
	}
	if len(img.RGBA) != img.Width*img.Height*4 {
		return nil, fmt.Errorf("RGBA buffer has %d bytes, want %d", len(img.RGBA), img.Width*img.Height*4)
	}
	var out bytes.Buffer
	fmt.Fprintf(&out, "P6\n%d %d\n255\n", img.Width, img.Height)
	expected := out.Len() + img.Width*img.Height*3
	out.Grow(img.Width * img.Height * 3)
	for i := 0; i < len(img.RGBA); i += 4 {
		out.WriteByte(img.RGBA[i])
		out.WriteByte(img.RGBA[i+1])
		out.WriteByte(img.RGBA[i+2])
	}
	if out.Len() != expected {
		return nil, fmt.Errorf("PPM size mismatch: wrote %d bytes, want %d", out.Len(), expected)
	}
	return out.Bytes(), nil
}

func writeFile(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	cleanup := func() {
		temp.Close()
		os.Remove(tempName)
	}
	if _, err := temp.Write(data); err != nil {
		cleanup()
		return err
	}
	if err := temp.Chmod(mode); err != nil {
		cleanup()
		return err
	}
	if err := temp.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := temp.Close(); err != nil {
		os.Remove(tempName)
		return err
	}
	if err := os.Rename(tempName, path); err != nil {
		os.Remove(tempName)
		return err
	}
	return nil
}
