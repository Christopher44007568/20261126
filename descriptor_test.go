package main

import (
	"image/color"
	"testing"
)

func TestStaleSnapshotKeepsHistoricalResourcesAlive(t *testing.T) {
	m1 := CameraMatrix(100, 100, 40, 30, 0, 0)
	m2 := CameraMatrix(230, 170, 70, 60, -0.05, 0.01)
	m3 := CameraMatrix(400, 220, 95, 80, 0.07, 0.02)

	cameras := NewCameraPool(m1, "CAM.V1")
	textures := NewTextureStore()
	texture := textures.Install(MakeMaterialTexture(128), "TEX.V1")
	system := NewDescriptorSystem(cameras, textures)

	previous := system.UpdateCamera(m2, "CAM.V2.PREVIOUS")
	stale, err := system.Bind("MAT.STALE", previous.Version, texture.Version)
	if err != nil {
		t.Fatal(err)
	}
	oldSnapshot, err := stale.Binding()
	if err != nil {
		t.Fatal(err)
	}
	if oldSnapshot.Stale {
		t.Fatal("snapshot was stale before the next camera was published")
	}

	current := system.UpdateCamera(m3, "CAM.V3.CURRENT")
	staleSnapshot, err := stale.Binding()
	if err != nil {
		t.Fatal(err)
	}
	if !staleSnapshot.Stale || !staleSnapshot.Invalidations.Has(InvalidateCameraChanged) {
		t.Fatalf("expected camera invalidation, got %s", staleSnapshot.Invalidations)
	}
	if staleSnapshot.CameraVersion != previous.Version {
		t.Fatal("camera update rewrote the material's binding snapshot")
	}
	if staleSnapshot.DescriptorVersion != oldSnapshot.DescriptorVersion {
		t.Fatal("descriptor version changed without a rebind")
	}
	if !staleSnapshot.Matrix.equal(m2) {
		t.Fatal("snapshot matrix was overwritten")
	}
	if cameras.RefCount(previous.Version) != 1 {
		t.Fatalf("pinned camera refs = %d, want 1", cameras.RefCount(previous.Version))
	}

	normal, err := system.Bind("MAT.NORMAL", current.Version, texture.Version)
	if err != nil {
		t.Fatal(err)
	}
	normalSnapshot, err := normal.Binding()
	if err != nil {
		t.Fatal(err)
	}
	if normalSnapshot.Stale || normalSnapshot.Invalidations != InvalidateNone {
		t.Fatalf("current material should be fresh, got %s", normalSnapshot.Invalidations)
	}
	if normalSnapshot.DescriptorVersion <= staleSnapshot.DescriptorVersion {
		t.Fatal("descriptor versions are not monotonic")
	}
	if textures.RefCount(texture.Version) != 2 {
		t.Fatalf("texture refs = %d, want 2", textures.RefCount(texture.Version))
	}

	pinnedRender, err := system.Resolve(stale)
	if err != nil {
		t.Fatal(err)
	}
	normalRender, err := system.Resolve(normal)
	if err != nil {
		t.Fatal(err)
	}
	if !pinnedRender.Camera.Matrix.equal(m2) {
		t.Fatal("resolved stale material no longer uses the historical matrix")
	}
	if !normalRender.Camera.Matrix.equal(m3) {
		t.Fatal("resolved fresh material does not use the current matrix")
	}
	if pinnedRender.Texture.Hash != normalRender.Texture.Hash {
		t.Fatal("normal and stale materials resolved different textures")
	}

	removed, _ := system.GC()
	if containsCameraVersion(removed, previous.Version) {
		t.Fatal("GC collected the camera pinned by the stale material")
	}
	if !cameras.Alive(previous.Version) || !textures.Alive(texture.Version) {
		t.Fatal("pinned resources are not resident")
	}

	system.Release(stale)
	removed, _ = system.GC()
	if !containsCameraVersion(removed, previous.Version) {
		t.Fatal("GC did not collect the camera after the stale material released it")
	}
	if cameras.Alive(previous.Version) {
		t.Fatal("released camera is still resident")
	}
	if _, err := system.Resolve(stale); err == nil {
		t.Fatal("resolving a released material should fail")
	}
}

func TestTextureUpdateDoesNotModifyPinnedMaterial(t *testing.T) {
	cameras := NewCameraPool(CameraMatrix(100, 100, 40, 30, 0, 0), "CAM.V1")
	textures := NewTextureStore()
	first := textures.Install(solidImage(8, color.RGBA{R: 10, G: 20, B: 30, A: 255}), "TEX.V1")
	system := NewDescriptorSystem(cameras, textures)
	oldMaterial, err := system.Bind("MAT.OLD", cameras.Current().Version, first.Version)
	if err != nil {
		t.Fatal(err)
	}
	oldSnapshot, _ := oldMaterial.Binding()

	second := system.UpdateTexture(solidImage(8, color.RGBA{R: 200, G: 120, B: 40, A: 255}), "TEX.V2")
	invalidated, err := oldMaterial.Binding()
	if err != nil {
		t.Fatal(err)
	}
	if !invalidated.Invalidations.Has(InvalidateTextureChanged) {
		t.Fatalf("expected texture invalidation, got %s", invalidated.Invalidations)
	}
	if invalidated.TextureHash != oldSnapshot.TextureHash {
		t.Fatal("texture update rewrote the pinned snapshot hash")
	}
	oldRender, err := system.Resolve(oldMaterial)
	if err != nil {
		t.Fatal(err)
	}
	if got := oldRender.Pixels.RGBAAt(0, 0); got.R != 10 || got.G != 20 || got.B != 30 {
		t.Fatalf("pinned texture pixels were overwritten: %#v", got)
	}

	newMaterial, err := system.Bind("MAT.NEW", cameras.Current().Version, second.Version)
	if err != nil {
		t.Fatal(err)
	}
	newRender, err := system.Resolve(newMaterial)
	if err != nil {
		t.Fatal(err)
	}
	if got := newRender.Pixels.RGBAAt(0, 0); got.R != 200 || got.G != 120 || got.B != 40 {
		t.Fatalf("new material did not use new texture: %#v", got)
	}
	if oldRender.Texture.Hash == newRender.Texture.Hash {
		t.Fatal("texture versions have identical hashes")
	}
}

func TestRefreshCameraCreatesNewDescriptor(t *testing.T) {
	cameras := NewCameraPool(CameraMatrix(100, 100, 40, 30, 0, 0), "CAM.V1")
	textures := NewTextureStore()
	texture := textures.Install(MakeMaterialTexture(128), "TEX.V1")
	system := NewDescriptorSystem(cameras, textures)
	material, err := system.Bind("MAT.A", cameras.Current().Version, texture.Version)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := material.Binding()
	next := system.UpdateCamera(CameraMatrix(210, 160, 60, 50, 0.04, 0.01), "CAM.V2")
	after, err := system.RefreshCamera(material, next.Version)
	if err != nil {
		t.Fatal(err)
	}
	if after.DescriptorVersion <= before.DescriptorVersion {
		t.Fatal("refresh did not allocate a new descriptor version")
	}
	if after.Stale || after.CameraVersion != next.Version {
		t.Fatalf("refresh did not produce a fresh snapshot: %+v", after)
	}
	oldRecord, ok := system.Descriptor(before.DescriptorVersion)
	if !ok || !oldRecord.Invalidations.Has(InvalidateDescriptorStale) {
		t.Fatal("old descriptor was not marked stale")
	}
	if cameras.RefCount(before.CameraVersion) != 0 {
		t.Fatal("refresh leaked the old camera reference")
	}
}

func TestResolveFlagsResourceOverwrite(t *testing.T) {
	m1 := CameraMatrix(100, 100, 40, 30, 0, 0)
	cameras := NewCameraPool(m1, "CAM.V1")
	textures := NewTextureStore()
	texture := textures.Install(MakeMaterialTexture(128), "TEX.V1")
	system := NewDescriptorSystem(cameras, textures)
	material, err := system.Bind("MAT.A", cameras.Current().Version, texture.Version)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, _ := material.Binding()

	overwritten := CameraMatrix(500, 400, 20, 10, 0, 0)
	cameras.mu.Lock()
	resource := cameras.byID[snapshot.CameraVersion]
	resource.Matrix = overwritten
	cameras.byID[snapshot.CameraVersion] = resource
	cameras.mu.Unlock()

	if _, err := system.Resolve(material); err == nil {
		t.Fatal("resolve accepted an overwritten camera resource")
	}
	record, ok := system.Descriptor(snapshot.DescriptorVersion)
	if !ok || !record.Invalidations.Has(InvalidateSnapshotMismatch) {
		t.Fatalf("snapshot mismatch flag not set: %s", record.Invalidations)
	}
}
