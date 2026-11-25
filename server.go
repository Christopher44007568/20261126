package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"texpath/internal/texresolver"
)

type app struct {
	cfg         config
	resolver    texresolver.Resolver
	placeholder []byte
	web         fs.FS
}

func main() {
	cfg, configSource, err := loadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if err := ensureRoots(cfg); err != nil {
		log.Fatalf("prepare search roots: %v", err)
	}
	if err := texturegenEnsure(); err != nil {
		log.Fatalf("generate demo assets: %v", err)
	}
	placeholder, err := checkerPNG()
	if err != nil {
		log.Fatalf("build placeholder: %v", err)
	}
	web, err := fs.Sub(webFiles, "web")
	if err != nil {
		log.Fatalf("open embedded web assets: %v", err)
	}
	handler := &app{
		cfg: cfg, resolver: texresolver.Resolver{Roots: cfg.SearchOrder},
		placeholder: placeholder, web: web,
	}

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("texture search order: %s", describeOrder(cfg.SearchOrder))
	log.Printf("config: %s", configSource)
	log.Printf("listening on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, handler.routes()); err != nil {
		log.Fatal(err)
	}
}

func (a *app) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/scene", a.handleScene)
	mux.HandleFunc("/api/resolve", a.handleResolve)
	mux.HandleFunc("/texture/", a.handleTexture)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, "ok\n")
	})
	mux.Handle("/", http.FileServer(http.FS(a.web)))
	return mux
}

func (a *app) handleScene(w http.ResponseWriter, _ *http.Request) {
	scene, err := a.buildScene()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(scene)
}

func (a *app) handleResolve(w http.ResponseWriter, request *http.Request) {
	name := request.URL.Query().Get("name")
	result, err := a.resolver.Resolve(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(result)
}

func (a *app) handleTexture(w http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	trimmed := strings.TrimPrefix(request.URL.Path, "/texture/")
	if trimmed == "_placeholder.png" {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		_, _ = w.Write(a.placeholder)
		return
	}
	rootID, filename, ok := strings.Cut(trimmed, "/")
	if !ok || filename == "" || filepath.Base(filename) != filename ||
		strings.ContainsAny(filename, `/\`) {
		http.NotFound(w, request)
		return
	}
	root, ok := a.rootByID(rootID)
	if !ok {
		http.NotFound(w, request)
		return
	}
	full := filepath.Join(root.Path, filename)
	info, err := os.Stat(full)
	if err != nil || !info.Mode().IsRegular() {
		http.NotFound(w, request)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	http.ServeFile(w, request, full)
}

func (a *app) rootByID(id string) (texresolver.Root, bool) {
	for _, root := range a.cfg.SearchOrder {
		if root.ID == id {
			return root, true
		}
	}
	return texresolver.Root{}, false
}

func (a *app) buildScene() (sceneResponse, error) {
	response := sceneResponse{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		SearchOrder: make([]orderStep, len(a.cfg.SearchOrder)),
		Groups:      make([]statusGroup, len(a.cfg.SearchOrder)),
		Materials:   make([]materialView, 0, len(demoMaterials)),
	}
	groupIndex := make(map[string]int, len(a.cfg.SearchOrder))
	for i, root := range a.cfg.SearchOrder {
		response.SearchOrder[i] = orderStep{
			Root: root, State: orderState(root.Kind),
		}
		response.Groups[i] = newStatusGroup(root)
		groupIndex[root.ID] = i
	}

	for _, material := range demoMaterials {
		view := materialView{materialSpec: material}
		for _, spec := range material.Maps {
			resolution, err := a.resolver.Resolve(spec.Filename)
			if err != nil {
				return sceneResponse{}, err
			}
			item := newMapView(spec, resolution)
			view.Maps = append(view.Maps, item)
			response.Summary.Total++
			if item.Placeholder {
				response.Summary.Placeholder++
			} else {
				response.Summary.Resolved++
				view.Resolved++
			}
			if item.Placeholder {
				view.Placeholder++
			}
			if len(item.Conflicts) > 0 {
				response.Summary.ConflictMaps++
				response.Summary.ConflictFiles += len(item.Conflicts)
			}
			a.aggregate(&response, groupIndex, resolution, item)
		}
		response.Materials = append(response.Materials, view)
	}
	return response, nil
}

func (a *app) aggregate(response *sceneResponse, index map[string]int, resolution texresolver.Resolution, item mapView) {
	groupPosition := index["fallback"]
	if resolution.Selected != nil {
		groupPosition = index[resolution.Selected.RootID]
		if idx, ok := index[resolution.Selected.RootID]; ok {
			response.SearchOrder[idx].Hits++
		}
	}
	group := &response.Groups[groupPosition]
	group.Count++
	group.Files = append(group.Files, item.Filename)
	if item.Placeholder {
		group.PlaceholderCount++
	} else if resolution.Selected != nil && resolution.Selected.Kind == "fallback" {
		group.FileCount++
	}
	if len(item.Conflicts) > 0 {
		group.ConflictCount++
		for _, conflict := range item.Conflicts {
			if idx, ok := index[conflict.RootID]; ok {
				response.SearchOrder[idx].Duplicates++
			}
		}
	}
}

func newMapView(spec mapSpec, resolution texresolver.Resolution) mapView {
	item := mapView{
		Resolution: resolution,
		MapLabel:   spec.Label,
	}
	if resolution.Placeholder {
		item.URL = "/texture/_placeholder.png"
		item.Status = "placeholder"
		item.StatusLabel = "棋盘格占位"
		return item
	}
	item.URL = fmt.Sprintf("/texture/%s/%s",
		url.PathEscape(resolution.Selected.RootID), url.PathEscape(spec.Filename))
	switch resolution.Selected.Kind {
	case "cache":
		item.Status = "cache"
		item.StatusLabel = "缓存接管"
	case "fallback":
		item.Status = "fallback"
		item.StatusLabel = "最终回退"
	default:
		item.Status = "asset"
		item.StatusLabel = "本地命中"
	}
	return item
}

func newStatusGroup(root texresolver.Root) statusGroup {
	group := statusGroup{
		ID: root.ID, Label: root.Label, Kind: root.Kind, Path: filepath.ToSlash(root.Path),
		Files: make([]string, 0, 3),
	}
	switch root.Kind {
	case "cache":
		group.Accent = "#8ec5ad"
		group.Description = "资产缺失时介入；保留被覆盖记录。"
	case "fallback", "config":
		group.Accent = "#e58b4a"
		group.Description = "最后一道保险；仍缺失则生成棋盘格。"
	default:
		group.Accent = "#d9a25f"
		group.Description = "程序目录优先；同名文件只采用最高优先级。"
	}
	return group
}

func orderState(kind string) string {
	switch kind {
	case "cache":
		return "standby"
	case "fallback", "config":
		return "last-resort"
	default:
		return "primary"
	}
}

func ensureRoots(cfg config) error {
	for _, root := range cfg.SearchOrder {
		if err := os.MkdirAll(root.Path, 0o755); err != nil {
			return errors.New(root.Path + ": " + err.Error())
		}
	}
	return nil
}
