package main

import (
	"fmt"
	"sort"
)

type ObjectID uint32
type HitGroupID uint32
type MaterialID uint32

const (
	ObjectFloor ObjectID = iota + 1
	ObjectStoneWall
	ObjectSideWall
	ObjectGlassPanel
	ObjectAreaLight
	ObjectCeramicSphere
	ObjectMetalSphere
	ObjectPlinth
)

const (
	GroupStone HitGroupID = iota + 1
	GroupGlass
	GroupFloor
	GroupCeramic
	GroupMetal
	GroupLight
)

const (
	MaterialStone MaterialID = iota + 1
	MaterialGlass
	MaterialFloor
	MaterialCeramic
	MaterialMetal
	MaterialLight
)

type Pattern int

const (
	PatternNone Pattern = iota
	PatternBrick
	PatternChecker
)

type Material struct {
	ID           MaterialID
	Name         string
	Albedo       Vec3
	Emission     Vec3
	Specular     float64
	Shininess    float64
	Reflectivity float64
	Transparency float64
	IOR          float64
	Pattern      Pattern
}

type HitGroup struct {
	ID         HitGroupID
	Name       string
	MaterialID MaterialID
}

type GroupCatalog struct {
	Groups    []HitGroup
	Materials []Material
	groupByID map[HitGroupID]HitGroup
	matByID   map[MaterialID]Material
}

func NewGroupCatalog() *GroupCatalog {
	catalog := &GroupCatalog{
		Groups: []HitGroup{
			{ID: GroupStone, Name: "stone", MaterialID: MaterialStone},
			{ID: GroupGlass, Name: "glass", MaterialID: MaterialGlass},
			{ID: GroupFloor, Name: "floor", MaterialID: MaterialFloor},
			{ID: GroupCeramic, Name: "ceramic", MaterialID: MaterialCeramic},
			{ID: GroupMetal, Name: "metal", MaterialID: MaterialMetal},
			{ID: GroupLight, Name: "area_light", MaterialID: MaterialLight},
		},
		Materials: []Material{
			{
				ID: MaterialStone, Name: "rough_stone", Albedo: Vec3{0.42, 0.39, 0.34},
				Specular: 0.08, Shininess: 18, Pattern: PatternBrick,
			},
			{
				ID: MaterialGlass, Name: "clear_glass", Albedo: Vec3{0.70, 0.88, 0.96},
				Specular: 0.9, Shininess: 180, Reflectivity: 0.16,
				Transparency: 0.76, IOR: 1.5,
			},
			{
				ID: MaterialFloor, Name: "sandstone_tiles", Albedo: Vec3{0.58, 0.54, 0.46},
				Specular: 0.12, Shininess: 32, Pattern: PatternChecker,
			},
			{
				ID: MaterialCeramic, Name: "orange_ceramic", Albedo: Vec3{0.82, 0.18, 0.06},
				Specular: 0.7, Shininess: 100,
			},
			{
				ID: MaterialMetal, Name: "brushed_metal", Albedo: Vec3{0.70, 0.76, 0.84},
				Specular: 1, Shininess: 220, Reflectivity: 0.82,
			},
			{
				ID: MaterialLight, Name: "warm_light", Albedo: Vec3{1, 0.72, 0.42},
				Emission: Vec3{10, 7.2, 4.4},
			},
		},
	}
	catalog.groupByID = make(map[HitGroupID]HitGroup, len(catalog.Groups))
	for _, group := range catalog.Groups {
		catalog.groupByID[group.ID] = group
	}
	catalog.matByID = make(map[MaterialID]Material, len(catalog.Materials))
	for _, material := range catalog.Materials {
		catalog.matByID[material.ID] = material
	}
	return catalog
}

func (c *GroupCatalog) Group(id HitGroupID) (HitGroup, bool) {
	group, ok := c.groupByID[id]
	return group, ok
}

func (c *GroupCatalog) MaterialForGroup(id HitGroupID) (Material, bool) {
	group, ok := c.groupByID[id]
	if !ok {
		return Material{}, false
	}
	material, ok := c.matByID[group.MaterialID]
	return material, ok
}

// ObjectMapping is the only place where an object gets material semantics.
type ObjectMapping map[ObjectID]HitGroupID

func CorrectMapping() ObjectMapping {
	return ObjectMapping{
		ObjectFloor:         GroupFloor,
		ObjectStoneWall:     GroupStone,
		ObjectSideWall:      GroupStone,
		ObjectGlassPanel:    GroupGlass,
		ObjectAreaLight:     GroupLight,
		ObjectCeramicSphere: GroupCeramic,
		ObjectMetalSphere:   GroupMetal,
		ObjectPlinth:        GroupStone,
	}
}

func (m ObjectMapping) clone() ObjectMapping {
	copy := make(ObjectMapping, len(m))
	for object, group := range m {
		copy[object] = group
	}
	return copy
}

func (m ObjectMapping) Validate(scene *Scene) error {
	for _, object := range scene.Objects {
		if _, ok := m[object.ID]; !ok {
			return fmt.Errorf("object %d (%s) has no hit-group mapping", object.ID, object.Name)
		}
	}
	for objectID, groupID := range m {
		if _, ok := scene.Object(objectID); !ok {
			return fmt.Errorf("mapping references unknown object %d", objectID)
		}
		if _, ok := scene.Catalog.Group(groupID); !ok {
			return fmt.Errorf("object %d maps to unknown hit group %d", objectID, groupID)
		}
	}
	return nil
}

func RemapOne(mapping ObjectMapping, objectID ObjectID, wrongGroup HitGroupID) (ObjectMapping, error) {
	if _, ok := mapping[objectID]; !ok {
		return nil, fmt.Errorf("cannot remap missing object %d", objectID)
	}
	changed := mapping.clone()
	changed[objectID] = wrongGroup
	return changed, nil
}

type MappingChange struct {
	ObjectID     ObjectID   `json:"object_id"`
	ObjectName   string     `json:"object_name"`
	CorrectGroup HitGroupID `json:"correct_hit_group"`
	BrokenGroup  HitGroupID `json:"broken_hit_group"`
	CorrectName  string     `json:"correct_hit_group_name"`
	BrokenName   string     `json:"broken_hit_group_name"`
}

func MappingChanges(scene *Scene, correct, broken ObjectMapping) []MappingChange {
	changes := make([]MappingChange, 0)
	for objectID, correctGroup := range correct {
		brokenGroup := broken[objectID]
		if correctGroup == brokenGroup {
			continue
		}
		object, _ := scene.Object(objectID)
		correctInfo, _ := scene.Catalog.Group(correctGroup)
		brokenInfo, _ := scene.Catalog.Group(brokenGroup)
		changes = append(changes, MappingChange{
			ObjectID: objectID, ObjectName: object.Name,
			CorrectGroup: correctGroup, BrokenGroup: brokenGroup,
			CorrectName: correctInfo.Name, BrokenName: brokenInfo.Name,
		})
	}
	sort.Slice(changes, func(i, j int) bool {
		return changes[i].ObjectID < changes[j].ObjectID
	})
	return changes
}
