package handler

import (
	"encoding/json"
	"net/http"

	"jmg/internal/database"
	"jmg/internal/storage"
	"jmg/internal/util"
)

// ImageUpdateHandler handles PATCH /api/images
type ImageUpdateHandler struct {
	db *database.DB
}

func NewImageUpdateHandler(db *database.DB) *ImageUpdateHandler {
	return &ImageUpdateHandler{db: db}
}

type UpdateRequest struct {
	ID        string `json:"id"`
	NewSlug   string `json:"slug,omitempty"`
	NewFolder string `json:"folder,omitempty"`
	Action    string `json:"action"`
}

func (h *ImageUpdateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	img, err := h.db.GetImageByID(req.ID)
	if err != nil || img == nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	switch req.Action {
	case "rename":
		slug := util.SanitizeSlug(req.NewSlug)
		if slug == "" {
			jsonError(w, "invalid name", http.StatusBadRequest)
			return
		}
		if exists, _ := h.db.SlugExists(img.Folder, slug); exists && slug != img.Slug {
			jsonError(w, "name already exists", http.StatusConflict)
			return
		}
		h.db.UpdateSlug(req.ID, slug)
		img.Slug = slug
	case "move":
		folder := util.SanitizeFolder(req.NewFolder)
		if exists, _ := h.db.SlugExists(folder, img.Slug); exists {
			jsonError(w, "name exists in target", http.StatusConflict)
			return
		}
		h.db.UpdateFolder(req.ID, folder)
		h.db.EnsureFolder(folder)
		img.Folder = folder
	default:
		jsonError(w, "action: rename|move", http.StatusBadRequest)
		return
	}
	url := buildImagePath(img.Folder, img.Slug)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "url": url, "slug": img.Slug, "folder": img.Folder})
}

// BulkMoveHandler POST /api/bulk-move
type BulkMoveHandler struct{ db *database.DB }

func NewBulkMoveHandler(db *database.DB) *BulkMoveHandler { return &BulkMoveHandler{db: db} }

func (h *BulkMoveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		IDs    []string `json:"ids"`
		Folder string   `json:"folder"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil || len(req.IDs) == 0 {
		jsonError(w, "invalid", http.StatusBadRequest)
		return
	}
	folder := util.SanitizeFolder(req.Folder)
	h.db.MoveImages(req.IDs, folder)
	if folder != "" {
		h.db.EnsureFolder(folder)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "moved": len(req.IDs)})
}

// FolderRenameHandler POST /api/folder-rename
type FolderRenameHandler struct{ db *database.DB }

func NewFolderRenameHandler(db *database.DB) *FolderRenameHandler {
	return &FolderRenameHandler{db: db}
}

func (h *FolderRenameHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		OldName string `json:"old_name"`
		NewName string `json:"new_name"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		jsonError(w, "invalid", http.StatusBadRequest)
		return
	}
	old := util.SanitizeFolder(req.OldName)
	nw := util.SanitizeFolder(req.NewName)
	if old == "" || nw == "" {
		jsonError(w, "names required", http.StatusBadRequest)
		return
	}
	cnt, err := h.db.RenameFolder(old, nw)
	if err != nil {
		jsonError(w, "failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "affected": cnt})
}

// FolderCreateHandler POST /api/folder-create
type FolderCreateHandler struct{ db *database.DB }

func NewFolderCreateHandler(db *database.DB) *FolderCreateHandler {
	return &FolderCreateHandler{db: db}
}

func (h *FolderCreateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		jsonError(w, "invalid", http.StatusBadRequest)
		return
	}
	path := util.SanitizeFolder(req.Path)
	if path == "" {
		jsonError(w, "path required", http.StatusBadRequest)
		return
	}
	h.db.CreateFolder(path)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "path": path})
}

// FolderDeleteHandler POST /api/folder-delete
type FolderDeleteHandler struct {
	db   *database.DB
	disk *storage.Disk
}

func NewFolderDeleteHandler(db *database.DB, disk *storage.Disk) *FolderDeleteHandler {
	return &FolderDeleteHandler{db: db, disk: disk}
}

func (h *FolderDeleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		jsonError(w, "invalid", http.StatusBadRequest)
		return
	}
	path := util.SanitizeFolder(req.Path)
	if path == "" {
		jsonError(w, "path required", http.StatusBadRequest)
		return
	}

	// Get images to delete from disk
	imgs, _ := h.db.GetImagesInFolderTree(path)
	cnt, err := h.db.DeleteFolder(path)
	if err != nil {
		jsonError(w, "failed", http.StatusInternalServerError)
		return
	}
	for _, img := range imgs {
		h.disk.Delete(img.DiskPath)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "deleted_images": cnt})
}
