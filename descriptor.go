package main

import (
	"errors"
	"fmt"
	"image"
	"sync"
)

// DescriptorRecord is the immutable part of a descriptor plus its mutable
// invalidation state. Reference counts live in CameraPool and TextureStore.
type DescriptorRecord struct {
	Version        DescriptorVersion `json:"version"`
	CameraVersion  CameraVersion     `json:"camera_version"`
	TextureVersion TextureVersion    `json:"texture_version"`
	CameraLabel    string            `json:"camera_label"`
	TextureLabel   string            `json:"texture_label"`
	Matrix         Mat3              `json:"-"`
	Invalidations  InvalidationFlags `json:"-"`
}

// BindingSnapshot captures the exact descriptor revisions a material uses.
// Rebinding creates a new descriptor; it does not rewrite this snapshot.
type BindingSnapshot struct {
	DescriptorVersion DescriptorVersion `json:"descriptor_version"`
	CameraVersion     CameraVersion     `json:"camera_version"`
	TextureVersion    TextureVersion    `json:"texture_version"`
	CameraLabel       string            `json:"camera_label"`
	TextureLabel      string            `json:"texture_label"`
	TextureHash       string            `json:"texture_hash"`
	Matrix            Mat3              `json:"-"`
	Invalidations     InvalidationFlags `json:"-"`
	Stale             bool              `json:"stale"`
}

func (s BindingSnapshot) InvalidationNames() []string {
	return s.Invalidations.Names()
}

// Material owns references to exactly one camera revision and one texture
// revision. The references keep historical GPU-like resources resident.
type Material struct {
	ID             string
	system         *DescriptorSystem
	descriptor     DescriptorVersion
	cameraVersion  CameraVersion
	textureVersion TextureVersion
	textureLabel   string
	textureHash    string
	released       bool
}

// RenderBinding is a resolved material ready for rasterization. Image pixels
// are read-only and remain valid while the material is not released.
type RenderBinding struct {
	MaterialID string
	Snapshot   BindingSnapshot
	Camera     CameraResource
	Texture    TextureResource
	Pixels     *image.RGBA
}

type DescriptorSystem struct {
	cameras  *CameraPool
	textures *TextureStore

	mu        sync.Mutex
	next      DescriptorVersion
	records   map[DescriptorVersion]*DescriptorRecord
	materials map[string]*Material
}

func NewDescriptorSystem(cameras *CameraPool, textures *TextureStore) *DescriptorSystem {
	return &DescriptorSystem{
		cameras:   cameras,
		textures:  textures,
		records:   make(map[DescriptorVersion]*DescriptorRecord),
		materials: make(map[string]*Material),
	}
}

func bindingFlags(cameraVersion CameraVersion, textureVersion TextureVersion, currentCam CameraVersion, currentTex TextureVersion) InvalidationFlags {
	var flags InvalidationFlags
	if cameraVersion != currentCam {
		flags |= InvalidateCameraChanged | InvalidateDescriptorStale
	}
	if textureVersion != currentTex {
		flags |= InvalidateTextureChanged | InvalidateDescriptorStale
	}
	return flags
}

// Bind creates a new descriptor revision and pins both resource revisions.
// A snapshot is never rewritten in place.
func (s *DescriptorSystem) Bind(materialID string, cameraVersion CameraVersion, textureVersion TextureVersion) (*Material, error) {
	if materialID == "" {
		return nil, errors.New("material id must not be empty")
	}
	camera, ok := s.cameras.Acquire(cameraVersion)
	if !ok {
		return nil, fmt.Errorf("camera v%d is not resident", cameraVersion)
	}
	texture, ok := s.textures.Acquire(textureVersion)
	if !ok {
		s.cameras.Release(cameraVersion)
		return nil, fmt.Errorf("texture v%d is not resident", textureVersion)
	}

	s.mu.Lock()
	if _, exists := s.materials[materialID]; exists {
		s.mu.Unlock()
		s.cameras.Release(cameraVersion)
		s.textures.Release(textureVersion)
		return nil, fmt.Errorf("%w: %s", errDuplicateMaterial, materialID)
	}
	s.next++
	version := s.next
	flags := bindingFlags(cameraVersion, textureVersion, s.cameras.Current().Version, s.textures.Current().Version)
	record := &DescriptorRecord{
		Version:        version,
		CameraVersion:  cameraVersion,
		TextureVersion: textureVersion,
		CameraLabel:    camera.Label,
		TextureLabel:   texture.Label,
		Matrix:         camera.Matrix,
		Invalidations:  flags,
	}
	s.records[version] = record
	m := &Material{
		ID:             materialID,
		system:         s,
		descriptor:     version,
		cameraVersion:  cameraVersion,
		textureVersion: textureVersion,
		textureLabel:   texture.Label,
		textureHash:    texture.Hash,
	}
	s.materials[materialID] = m
	s.mu.Unlock()
	return m, nil
}

func (s *DescriptorSystem) recordLocked(version DescriptorVersion) *DescriptorRecord {
	return s.records[version]
}

// Binding returns a copy of the exact snapshot and its latest invalidation
// state. Mutating the returned value cannot affect the material.
func (m *Material) Binding() (BindingSnapshot, error) {
	s := m.system
	s.mu.Lock()
	defer s.mu.Unlock()
	if m.released {
		return BindingSnapshot{}, errMaterialReleased
	}
	record := s.recordLocked(m.descriptor)
	if record == nil {
		return BindingSnapshot{}, fmt.Errorf("descriptor v%d is missing", m.descriptor)
	}
	flags := record.Invalidations
	return BindingSnapshot{
		DescriptorVersion: record.Version,
		CameraVersion:     record.CameraVersion,
		TextureVersion:    record.TextureVersion,
		CameraLabel:       record.CameraLabel,
		TextureLabel:      record.TextureLabel,
		TextureHash:       m.textureHash,
		Matrix:            record.Matrix,
		Invalidations:     flags,
		Stale:             flags.Has(InvalidateDescriptorStale),
	}, nil
}

// Resolve checks that the pinned resources still match the snapshot, then
// returns the immutable pixels and matrices to the renderer.
func (s *DescriptorSystem) Resolve(m *Material) (RenderBinding, error) {
	snapshot, err := m.Binding()
	if err != nil {
		return RenderBinding{}, err
	}
	camera, ok := s.cameras.Get(snapshot.CameraVersion)
	if !ok {
		return RenderBinding{}, fmt.Errorf("camera v%d was evicted while material %s was active", snapshot.CameraVersion, m.ID)
	}
	texture, pixels, ok := s.textures.Get(snapshot.TextureVersion)
	if !ok {
		return RenderBinding{}, fmt.Errorf("texture v%d was evicted while material %s was active", snapshot.TextureVersion, m.ID)
	}

	mismatch := false
	if !camera.Matrix.equal(snapshot.Matrix) {
		mismatch = true
	}
	if texture.Hash != snapshot.TextureHash {
		mismatch = true
	}
	if mismatch {
		s.mu.Lock()
		if record := s.recordLocked(m.descriptor); record != nil {
			record.Invalidations |= InvalidateSnapshotMismatch | InvalidateDescriptorStale
			snapshot.Invalidations = record.Invalidations
			snapshot.Stale = true
		}
		s.mu.Unlock()
		return RenderBinding{}, fmt.Errorf("material %s snapshot no longer matches its resources", m.ID)
	}
	return RenderBinding{
		MaterialID: m.ID,
		Snapshot:   snapshot,
		Camera:     camera,
		Texture:    texture,
		Pixels:     pixels,
	}, nil
}

// UpdateCamera publishes a new current camera revision and marks every older
// descriptor stale. Pinned snapshots continue to reference their old matrix.
func (s *DescriptorSystem) UpdateCamera(matrix Mat3, label string) CameraResource {
	camera := s.cameras.Install(matrix, label)
	s.mu.Lock()
	for _, record := range s.records {
		if record.CameraVersion != camera.Version {
			record.Invalidations |= InvalidateCameraChanged | InvalidateDescriptorStale
		}
	}
	s.mu.Unlock()
	return camera
}

// UpdateTexture publishes a new current texture revision and invalidates older
// texture bindings. Existing snapshots retain the exact old pixels.
func (s *DescriptorSystem) UpdateTexture(pixels *image.RGBA, label string) TextureResource {
	texture := s.textures.Install(pixels, label)
	s.mu.Lock()
	for _, record := range s.records {
		if record.TextureVersion != texture.Version {
			record.Invalidations |= InvalidateTextureChanged | InvalidateDescriptorStale
		}
	}
	s.mu.Unlock()
	return texture
}

// RefreshCamera rebinds a material to a camera revision under a new descriptor
// version. The new reference is acquired before the old one is released.
func (s *DescriptorSystem) RefreshCamera(m *Material, cameraVersion CameraVersion) (BindingSnapshot, error) {
	camera, ok := s.cameras.Acquire(cameraVersion)
	if !ok {
		return BindingSnapshot{}, fmt.Errorf("camera v%d is not resident", cameraVersion)
	}

	s.mu.Lock()
	if m.released {
		s.mu.Unlock()
		s.cameras.Release(cameraVersion)
		return BindingSnapshot{}, errMaterialReleased
	}
	oldView := m.cameraVersion
	oldDescriptor := m.descriptor
	if oldRecord := s.recordLocked(oldDescriptor); oldRecord != nil {
		oldRecord.Invalidations |= InvalidateDescriptorStale
	}
	s.next++
	version := s.next
	flags := bindingFlags(cameraVersion, m.textureVersion, s.cameras.Current().Version, s.textures.Current().Version)
	record := &DescriptorRecord{
		Version:        version,
		CameraVersion:  cameraVersion,
		TextureVersion: m.textureVersion,
		CameraLabel:    camera.Label,
		TextureLabel:   m.textureLabel,
		Matrix:         camera.Matrix,
		Invalidations:  flags,
	}
	s.records[version] = record
	m.descriptor = version
	m.cameraVersion = cameraVersion
	snapshot := BindingSnapshot{
		DescriptorVersion: version,
		CameraVersion:     cameraVersion,
		TextureVersion:    m.textureVersion,
		CameraLabel:       camera.Label,
		TextureLabel:      m.textureLabel,
		TextureHash:       m.textureHash,
		Matrix:            camera.Matrix,
		Invalidations:     flags,
		Stale:             flags.Has(InvalidateDescriptorStale),
	}
	s.mu.Unlock()

	s.cameras.Release(oldView)
	return snapshot, nil
}

// Release drops a material's snapshot references. It is safe to call twice.
func (s *DescriptorSystem) Release(m *Material) {
	s.mu.Lock()
	if m.released {
		s.mu.Unlock()
		return
	}
	m.released = true
	delete(s.materials, m.ID)
	cameraVersion := m.cameraVersion
	textureVersion := m.textureVersion
	s.mu.Unlock()

	s.cameras.Release(cameraVersion)
	s.textures.Release(textureVersion)
}

func (s *DescriptorSystem) Descriptor(version DescriptorVersion) (DescriptorRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record := s.records[version]
	if record == nil {
		return DescriptorRecord{}, false
	}
	return *record, true
}

func (s *DescriptorSystem) DescriptorCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.records)
}

// GC collects unreferenced historical revisions. Current revisions and
// revisions pinned by live material snapshots are retained.
func (s *DescriptorSystem) GC() (removedCameras []CameraVersion, removedTextures []TextureVersion) {
	return s.cameras.GC(), s.textures.GC()
}
