package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"sort"
	"sync"
)

// CameraPool publishes immutable camera revisions. Installing a revision
// allocates a new entry; it never writes over an older matrix. Entries remain
// resident while a material snapshot holds a reference.
type CameraPool struct {
	mu      sync.RWMutex
	next    uint64
	current CameraVersion
	byID    map[CameraVersion]CameraResource
	refs    map[CameraVersion]int
}

func NewCameraPool(initial Mat3, label string) *CameraPool {
	p := &CameraPool{
		byID: make(map[CameraVersion]CameraResource),
		refs: make(map[CameraVersion]int),
	}
	p.installLocked(initial, label)
	return p
}

func (p *CameraPool) installLocked(matrix Mat3, label string) CameraResource {
	p.next++
	version := CameraVersion(p.next)
	if label == "" {
		label = fmt.Sprintf("camera.v%d", version)
	}
	res := CameraResource{
		Version:  version,
		Label:    label,
		Matrix:   matrix,
		Sequence: p.next,
	}
	p.byID[version] = res
	p.current = version
	return res
}

// Install publishes a new camera revision and makes it current.
func (p *CameraPool) Install(matrix Mat3, label string) CameraResource {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.installLocked(matrix, label)
}

func (p *CameraPool) Current() CameraResource {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.byID[p.current]
}

func (p *CameraPool) Acquire(version CameraVersion) (CameraResource, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	res, ok := p.byID[version]
	if !ok {
		return CameraResource{}, false
	}
	p.refs[version]++
	return res, true
}

func (p *CameraPool) Release(version CameraVersion) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.refs[version] > 0 {
		p.refs[version]--
	}
}

func (p *CameraPool) Get(version CameraVersion) (CameraResource, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	res, ok := p.byID[version]
	return res, ok
}

func (p *CameraPool) Alive(version CameraVersion) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, ok := p.byID[version]
	return ok
}

func (p *CameraPool) RefCount(version CameraVersion) int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.refs[version]
}

// GC removes non-current camera revisions with no snapshot references.
func (p *CameraPool) GC() []CameraVersion {
	p.mu.Lock()
	defer p.mu.Unlock()
	removed := make([]CameraVersion, 0)
	for version := range p.byID {
		if version == p.current || p.refs[version] > 0 {
			continue
		}
		delete(p.byID, version)
		delete(p.refs, version)
		removed = append(removed, version)
	}
	sort.Slice(removed, func(i, j int) bool { return removed[i] < removed[j] })
	return removed
}

func (p *CameraPool) RetainedVersions() []uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]uint64, 0, len(p.byID))
	for version := range p.byID {
		out = append(out, uint64(version))
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func (p *CameraPool) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.byID)
}

type textureEntry struct {
	resource TextureResource
	image    *image.RGBA
}

// TextureStore also publishes immutable revisions. Publishing copies the
// source pixels, preventing later edits from changing a material in flight.
type TextureStore struct {
	mu      sync.RWMutex
	next    uint64
	current TextureVersion
	byID    map[TextureVersion]textureEntry
	refs    map[TextureVersion]int
}

func NewTextureStore() *TextureStore {
	return &TextureStore{
		byID: make(map[TextureVersion]textureEntry),
		refs: make(map[TextureVersion]int),
	}
}

func (s *TextureStore) Install(img *image.RGBA, label string) TextureResource {
	if img == nil {
		panic("TextureStore.Install: nil image")
	}
	copied := image.NewRGBA(image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy()))
	draw.Draw(copied, copied.Bounds(), img, img.Bounds().Min, draw.Src)
	sum := sha256.Sum256(copied.Pix)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	version := TextureVersion(s.next)
	if label == "" {
		label = fmt.Sprintf("texture.v%d", version)
	}
	res := TextureResource{
		Version: version,
		Label:   label,
		Width:   copied.Bounds().Dx(),
		Height:  copied.Bounds().Dy(),
		Hash:    hex.EncodeToString(sum[:]),
	}
	s.byID[version] = textureEntry{resource: res, image: copied}
	s.current = version
	return res
}

func (s *TextureStore) Current() TextureResource {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.byID[s.current].resource
}

func (s *TextureStore) Acquire(version TextureVersion) (TextureResource, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.byID[version]
	if !ok {
		return TextureResource{}, false
	}
	s.refs[version]++
	return entry.resource, true
}

func (s *TextureStore) Release(version TextureVersion) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.refs[version] > 0 {
		s.refs[version]--
	}
}

func (s *TextureStore) Get(version TextureVersion) (TextureResource, *image.RGBA, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.byID[version]
	if !ok {
		return TextureResource{}, nil, false
	}
	return entry.resource, entry.image, true
}

func (s *TextureStore) RefCount(version TextureVersion) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.refs[version]
}

func (s *TextureStore) Alive(version TextureVersion) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.byID[version]
	return ok
}

func (s *TextureStore) RetainedVersions() []uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]uint64, 0, len(s.byID))
	for version := range s.byID {
		out = append(out, uint64(version))
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// GC removes non-current textures that no material still references.
func (s *TextureStore) GC() []TextureVersion {
	s.mu.Lock()
	defer s.mu.Unlock()
	removed := make([]TextureVersion, 0)
	for version := range s.byID {
		if version == s.current || s.refs[version] > 0 {
			continue
		}
		delete(s.byID, version)
		delete(s.refs, version)
		removed = append(removed, version)
	}
	sort.Slice(removed, func(i, j int) bool { return removed[i] < removed[j] })
	return removed
}

var (
	errMaterialReleased  = errors.New("material has been released")
	errDuplicateMaterial = errors.New("material id is already active")
)
