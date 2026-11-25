package main

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"texpath/internal/texresolver"
)

//go:embed web/*
var webFiles embed.FS

type config struct {
	SearchOrder []texresolver.Root `json:"searchOrder"`
}

type materialSpec struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Eyebrow     string    `json:"eyebrow"`
	Description string    `json:"description"`
	Accent      string    `json:"accent"`
	Maps        []mapSpec `json:"-"`
}

type mapSpec struct {
	Kind     string `json:"kind"`
	Label    string `json:"label"`
	Filename string `json:"filename"`
}

type mapView struct {
	texresolver.Resolution
	URL         string `json:"url"`
	Status      string `json:"status"`
	StatusLabel string `json:"statusLabel"`
	MapLabel    string `json:"mapLabel"`
}

type materialView struct {
	materialSpec
	Maps        []mapView `json:"maps"`
	Resolved    int       `json:"resolved"`
	Placeholder int       `json:"placeholder"`
}

type orderStep struct {
	texresolver.Root
	Hits       int    `json:"hits"`
	Duplicates int    `json:"duplicates"`
	State      string `json:"state"`
}

type statusGroup struct {
	ID               string   `json:"id"`
	Label            string   `json:"label"`
	Kind             string   `json:"kind"`
	Path             string   `json:"path"`
	Accent           string   `json:"accent"`
	Description      string   `json:"description"`
	Count            int      `json:"count"`
	FileCount        int      `json:"fileCount"`
	PlaceholderCount int      `json:"placeholderCount"`
	ConflictCount    int      `json:"conflictCount"`
	Files            []string `json:"files"`
}

type sceneSummary struct {
	Total         int `json:"total"`
	Resolved      int `json:"resolved"`
	Placeholder   int `json:"placeholder"`
	ConflictMaps  int `json:"conflictMaps"`
	ConflictFiles int `json:"conflictFiles"`
}

type sceneResponse struct {
	GeneratedAt string         `json:"generatedAt"`
	SearchOrder []orderStep    `json:"searchOrder"`
	Summary     sceneSummary   `json:"summary"`
	Groups      []statusGroup  `json:"groups"`
	Materials   []materialView `json:"materials"`
}

var demoMaterials = []materialSpec{
	{
		ID: "oak", Name: "烟熏橡木", Eyebrow: "ASSET GROUP · 01",
		Description: "本地资产目录优先命中。缓存里的同名贴图仍然保留为可追踪候选，但不会覆盖最终结果。",
		Accent:      "#d9a25f",
		Maps: []mapSpec{
			{Kind: "albedo", Label: "ALBEDO", Filename: "oak_albedo.png"},
			{Kind: "normal", Label: "NORMAL", Filename: "oak_normal.png"},
			{Kind: "roughness", Label: "ROUGHNESS", Filename: "oak_roughness.png"},
		},
	},
	{
		ID: "brass", Name: "拉丝黄铜", Eyebrow: "CACHE GROUP · 02",
		Description: "本地资产缺位，缓存路径接管。黄铜反照率同时存在于回退目录，最终采用缓存版本。",
		Accent:      "#8ec5ad",
		Maps: []mapSpec{
			{Kind: "albedo", Label: "ALBEDO", Filename: "brass_albedo.png"},
			{Kind: "normal", Label: "NORMAL", Filename: "brass_normal.png"},
			{Kind: "roughness", Label: "ROUGHNESS", Filename: "brass_roughness.png"},
		},
	},
	{
		ID: "clay", Name: "手作陶土", Eyebrow: "FALLBACK GROUP · 03",
		Description: "反照率与粗糙度在所有路径中缺失，以棋盘格占位；场景光照、背景和其余材质继续正常渲染。",
		Accent:      "#e58b4a",
		Maps: []mapSpec{
			{Kind: "albedo", Label: "ALBEDO", Filename: "clay_albedo.png"},
			{Kind: "normal", Label: "NORMAL", Filename: "clay_normal.png"},
			{Kind: "roughness", Label: "ROUGHNESS", Filename: "clay_roughness.png"},
		},
	},
}

func defaultConfig() config {
	return config{SearchOrder: []texresolver.Root{
		{ID: "assets", Label: "本地资产", Kind: "asset", Path: "assets/textures", Priority: 1},
		{ID: "cache", Label: "运行时缓存", Kind: "cache", Path: ".cache/textures", Priority: 2},
		{ID: "fallback", Label: "最终回退", Kind: "fallback", Path: "assets/fallback", Priority: 3},
	}}
}

func loadConfig() (config, string, error) {
	path := os.Getenv("TEXPATH_CONFIG")
	if path == "" {
		path = "texturepaths.json"
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return defaultConfig(), "built-in defaults", nil
	}
	if err != nil {
		return config{}, path, err
	}
	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return config{}, path, err
	}
	if len(cfg.SearchOrder) == 0 {
		return config{}, path, errors.New("searchOrder must contain at least one root")
	}
	seen := make(map[string]bool, len(cfg.SearchOrder))
	for i := range cfg.SearchOrder {
		root := &cfg.SearchOrder[i]
		root.Priority = i + 1
		if root.ID == "" || root.Path == "" {
			return config{}, path, fmt.Errorf("searchOrder[%d] needs id and path", i)
		}
		if seen[root.ID] {
			return config{}, path, fmt.Errorf("duplicate search root id %q", root.ID)
		}
		seen[root.ID] = true
		if root.Kind == "" {
			root.Kind = "asset"
		}
		if root.Label == "" {
			root.Label = root.ID
		}
		root.Path = filepath.Clean(root.Path)
	}
	return cfg, path, nil
}
