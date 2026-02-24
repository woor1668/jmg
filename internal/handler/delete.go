package handler

import (
	"encoding/json"
	"net/http"

	"jmg/internal/database"
	"jmg/internal/storage"
)

type DeleteHandler struct {
	db   *database.DB
	disk *storage.Disk
}

func NewDeleteHandler(db *database.DB, disk *storage.Disk) *DeleteHandler {
	return &DeleteHandler{db: db, disk: disk}
}

// Single delete: DELETE /api/images/:id
func (h *DeleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get ID from query param
	id := r.URL.Query().Get("id")
	if id == "" {
		jsonError(w, "missing id", http.StatusBadRequest)
		return
	}

	img, err := h.db.DeleteImage(id)
	if err != nil {
		jsonError(w, "failed to delete", http.StatusInternalServerError)
		return
	}
	if img == nil {
		jsonError(w, "image not found", http.StatusNotFound)
		return
	}

	h.disk.Delete(img.DiskPath)
	w.WriteHeader(http.StatusNoContent)
}

// BulkDeleteHandler handles POST /api/bulk-delete
type BulkDeleteHandler struct {
	db   *database.DB
	disk *storage.Disk
}

func NewBulkDeleteHandler(db *database.DB, disk *storage.Disk) *BulkDeleteHandler {
	return &BulkDeleteHandler{db: db, disk: disk}
}

type BulkDeleteRequest struct {
	IDs []string `json:"ids"`
}

func (h *BulkDeleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req BulkDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.IDs) == 0 {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	deleted, err := h.db.DeleteImagesByIDs(req.IDs)
	if err != nil {
		jsonError(w, "failed to delete", http.StatusInternalServerError)
		return
	}

	for _, img := range deleted {
		h.disk.Delete(img.DiskPath)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"deleted": len(deleted)})
}
