package handler

import (
	"net/http"
	"strconv"
	"strings"

	"jmg/internal/database"
	"jmg/internal/storage"
)

type ServeHandler struct {
	db       *database.DB
	disk     *storage.Disk
	thumbGen *storage.ThumbnailGenerator
}

func NewServeHandler(db *database.DB, disk *storage.Disk, thumbGen *storage.ThumbnailGenerator) *ServeHandler {
	return &ServeHandler{db: db, disk: disk, thumbGen: thumbGen}
}

func (h *ServeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		http.NotFound(w, r)
		return
	}

	// Parse: everything up to last segment is folder, last segment is slug
	parts := strings.Split(path, "/")
	slug := parts[len(parts)-1]
	folder := ""
	if len(parts) > 1 {
		folder = strings.Join(parts[:len(parts)-1], "/")
	}

	// Strip extension if present (abc123.png → abc123)
	if dotIdx := strings.LastIndex(slug, "."); dotIdx > 0 {
		slug = slug[:dotIdx]
	}

	// Lookup by folder + slug
	img, err := h.db.GetImageBySlug(folder, slug)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if img == nil {
		http.NotFound(w, r)
		return
	}

	// Thumbnail request
	if wStr := r.URL.Query().Get("w"); wStr != "" {
		width, err := strconv.Atoi(wStr)
		if err == nil && width > 0 && width <= 2000 {
			thumbPath, err := h.thumbGen.GetOrCreate(img.DiskPath, img.ID, width)
			if err == nil {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				w.Header().Set("Content-Type", "image/jpeg")
				http.ServeFile(w, r, thumbPath)
				return
			}
		}
	}

	fullPath := h.disk.FullPath(img.DiskPath)

	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Content-Type", img.MimeType)
	w.Header().Set("ETag", `"`+img.Hash[:16]+`"`)
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if match := r.Header.Get("If-None-Match"); match != "" {
		if strings.Contains(match, img.Hash[:16]) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	http.ServeFile(w, r, fullPath)
}
