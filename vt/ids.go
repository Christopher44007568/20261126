// Package vt implements a dependency-free software virtual texturing demo.
package vt

import (
	"errors"
	"fmt"
	"math"
)

// PageID identifies one virtual page in a particular mip level.
type PageID struct {
	Mip int
	X   int
	Y   int
}

func (p PageID) String() string {
	return fmt.Sprintf("m%d_%02d_%02d", p.Mip, p.X, p.Y)
}

// Info describes a power-of-two tiled virtual texture.
type Info struct {
	BaseSize     int
	PageSize     int
	PagesPerSide int
	MaxMip       int
}

// NewInfo validates a square power-of-two page layout.
func NewInfo(baseSize, pageSize int) (Info, error) {
	if baseSize <= 0 || pageSize <= 0 {
		return Info{}, errors.New("sizes must be positive")
	}
	if baseSize%pageSize != 0 {
		return Info{}, errors.New("base size must be divisible by page size")
	}
	pages := baseSize / pageSize
	if pages&(pages-1) != 0 {
		return Info{}, errors.New("pages per side must be a power of two")
	}
	if pageSize&(pageSize-1) != 0 {
		return Info{}, errors.New("page size must be a power of two")
	}
	maxMip := int(math.Log2(float64(pages)))
	return Info{
		BaseSize:     baseSize,
		PageSize:     pageSize,
		PagesPerSide: pages,
		MaxMip:       maxMip,
	}, nil
}

func MustInfo(baseSize, pageSize int) Info {
	info, err := NewInfo(baseSize, pageSize)
	if err != nil {
		panic(err)
	}
	return info
}

func DefaultInfo() Info { return MustInfo(512, 64) }

// Count is the number of pages along one side at a mip level.
func (i Info) Count(mip int) int {
	mip = i.ClampMip(mip)
	return i.PagesPerSide >> mip
}

func (i Info) ClampMip(mip int) int {
	if mip < 0 {
		return 0
	}
	if mip > i.MaxMip {
		return i.MaxMip
	}
	return mip
}

func (i Info) Valid(id PageID) bool {
	if id.Mip < 0 || id.Mip > i.MaxMip {
		return false
	}
	n := i.Count(id.Mip)
	return id.X >= 0 && id.X < n && id.Y >= 0 && id.Y < n
}

// ClampUV keeps coordinates on the wall texture rather than in repeat wrap.
func ClampUV(u, v float64) (float64, float64) {
	return math.Min(1, math.Max(0, u)), math.Min(1, math.Max(0, v))
}

// PageAtUV returns the page covering UV at the requested mip.
func (i Info) PageAtUV(u, v float64, mip int) PageID {
	u, v = ClampUV(u, v)
	mip = i.ClampMip(mip)
	n := i.Count(mip)
	x := int(math.Floor(u * float64(n)))
	y := int(math.Floor(v * float64(n)))
	if x >= n {
		x = n - 1
	}
	if y >= n {
		y = n - 1
	}
	return PageID{Mip: mip, X: x, Y: y}
}

// Bounds returns the normalized UV rectangle of a page.
func (i Info) Bounds(id PageID) (u0, v0, u1, v1 float64) {
	n := i.Count(id.Mip)
	fn := float64(n)
	return float64(id.X) / fn, float64(id.Y) / fn,
		float64(id.X+1) / fn, float64(id.Y+1) / fn
}

// Parent returns the immediate coarser page.
func (i Info) Parent(id PageID) (PageID, bool) {
	if id.Mip >= i.MaxMip {
		return PageID{}, false
	}
	return PageID{Mip: id.Mip + 1, X: id.X / 2, Y: id.Y / 2}, true
}

// Ancestor maps id to mip, where mip >= id.Mip.
func (i Info) Ancestor(id PageID, mip int) PageID {
	mip = i.ClampMip(mip)
	if mip < id.Mip {
		mip = id.Mip
	}
	shift := uint(mip - id.Mip)
	return PageID{Mip: mip, X: id.X >> shift, Y: id.Y >> shift}
}

// Chain returns id followed by its resident-candidate ancestors.
func (i Info) Chain(id PageID) []PageID {
	chain := make([]PageID, 0, i.MaxMip-id.Mip+1)
	chain = append(chain, id)
	current := id
	for {
		parent, ok := i.Parent(current)
		if !ok {
			return chain
		}
		chain = append(chain, parent)
		current = parent
	}
}
