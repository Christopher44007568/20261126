package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type ObjectMappingRecord struct {
	ObjectID      ObjectID   `json:"object_id"`
	ObjectName    string     `json:"object_name"`
	HitGroupID    HitGroupID `json:"hit_group_id"`
	HitGroupName  string     `json:"hit_group_name"`
	MaterialID    MaterialID `json:"material_id"`
	MaterialIndex int        `json:"material_index"`
	MaterialName  string     `json:"material_name"`
}

type HitGroupRecord struct {
	ID            HitGroupID `json:"id"`
	Name          string     `json:"name"`
	MaterialID    MaterialID `json:"material_id"`
	MaterialIndex int        `json:"material_index"`
	PaletteColor  string     `json:"palette_color"`
}

type MaterialRecord struct {
	ID           MaterialID `json:"id"`
	Index        int        `json:"index"`
	Name         string     `json:"name"`
	PaletteColor string     `json:"palette_color"`
}

type RenderReport struct {
	Width                   int                   `json:"width"`
	Height                  int                   `json:"height"`
	Outputs                 map[string]string     `json:"outputs"`
	ReferenceObjectMappings []ObjectMappingRecord `json:"reference_object_mappings"`
	BrokenObjectMappings    []ObjectMappingRecord `json:"broken_object_mappings"`
	HitGroups               []HitGroupRecord      `json:"hit_groups"`
	Materials               []MaterialRecord      `json:"materials"`
	DeliberateChanges       []MappingChange       `json:"deliberate_changes"`
	FirstError              FirstMismatch         `json:"first_error"`
	FirstErrorSummary       string                `json:"first_error_summary"`
}

func buildReport(
	scene *Scene,
	reference *RenderResult,
	referenceMapping, brokenMapping ObjectMapping,
	changes []MappingChange,
	first FirstMismatch,
	outputs map[string]string,
) RenderReport {
	report := RenderReport{
		Width: reference.Width, Height: reference.Height,
		Outputs: outputs, DeliberateChanges: changes,
		FirstError: first, FirstErrorSummary: first.Summary(),
	}
	for _, group := range scene.Catalog.Groups {
		report.HitGroups = append(report.HitGroups, HitGroupRecord{
			ID: group.ID, Name: group.Name, MaterialID: group.MaterialID,
			MaterialIndex: int(group.MaterialID),
			PaletteColor:  colorHex8(hitGroupColor(group.ID)),
		})
	}
	for _, material := range scene.Catalog.Materials {
		report.Materials = append(report.Materials, MaterialRecord{
			ID: material.ID, Index: int(material.ID), Name: material.Name,
			PaletteColor: colorHex8(materialColor(material.ID)),
		})
	}
	report.ReferenceObjectMappings = mappingRecords(scene, referenceMapping)
	report.BrokenObjectMappings = mappingRecords(scene, brokenMapping)
	return report
}

func mappingRecords(scene *Scene, mapping ObjectMapping) []ObjectMappingRecord {
	records := make([]ObjectMappingRecord, 0, len(scene.Objects))
	for _, object := range scene.Objects {
		groupID := mapping[object.ID]
		group, _ := scene.Catalog.Group(groupID)
		material, _ := scene.Catalog.MaterialForGroup(groupID)
		records = append(records, ObjectMappingRecord{
			ObjectID: object.ID, ObjectName: object.Name,
			HitGroupID: groupID, HitGroupName: group.Name,
			MaterialID: material.ID, MaterialIndex: int(material.ID),
			MaterialName: material.Name,
		})
	}
	return records
}

func writeReport(path string, report RenderReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".report-*.tmp")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempName, path); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	return nil
}
