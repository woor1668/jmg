package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"jmg/internal/config"
	"jmg/internal/database"
	"jmg/internal/storage"
	"jmg/internal/util"
)

type UploadHandler struct {
	cfg  *config.Config
	db   *database.DB
	disk *storage.Disk
}

func NewUploadHandler(cfg *config.Config, db *database.DB, disk *storage.Disk) *UploadHandler {
	return &UploadHandler{cfg: cfg, db: db, disk: disk}
}

type UploadResponse struct {
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	Folder    string `json:"folder,omitempty"`
	URL       string `json:"url"`
	Markdown  string `json:"markdown"`
	HTML      string `json:"html"`
	BBCode    string `json:"bbcode"`
	Size      int64  `json:"size"`
	MimeType  string `json:"mime_type"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	CreatedAt string `json:"created_at"`
}

func (h *UploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	maxSize := h.cfg.Storage.MaxFileSize + (1 << 20)
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)

	if err := r.ParseMultipartForm(maxSize); err != nil {
		jsonError(w, "file too large or invalid form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		jsonError(w, "missing file field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		jsonError(w, "failed to read file", http.StatusInternalServerError)
		return
	}

	if int64(len(data)) > h.cfg.Storage.MaxFileSize {
		jsonError(w, fmt.Sprintf("file too large (max %dMB)", h.cfg.Storage.MaxFileSize/(1024*1024)), http.StatusBadRequest)
		return
	}

	mime := util.DetectMimeType(data)
	if !h.cfg.IsAllowedType(mime) {
		if header.Header.Get("Content-Type") == "image/svg+xml" {
			mime = "image/svg+xml"
		}
		if !h.cfg.IsAllowedType(mime) {
			jsonError(w, fmt.Sprintf("file type not allowed: %s", mime), http.StatusBadRequest)
			return
		}
	}

	hash := util.HashBytes(data)
	folder := util.SanitizeFolder(r.FormValue("folder"))

	// Strip directory from filename (some browsers send full path)
	filename := header.Filename
	if idx := strings.LastIndex(filename, "/"); idx >= 0 {
		filename = filename[idx+1:]
	}
	if idx := strings.LastIndex(filename, "\\"); idx >= 0 {
		filename = filename[idx+1:]
	}

	// Generate slug from filename or custom slug
	customSlug := r.FormValue("slug")
	var slug string
	if customSlug != "" {
		slug = util.SanitizeSlug(customSlug)
	} else {
		slug = util.ToSlug(filename)
	}

	// Ensure slug is unique in folder
	slug, err = h.ensureUniqueSlug(folder, slug)
	if err != nil {
		jsonError(w, "failed to generate unique name", http.StatusInternalServerError)
		return
	}

	// Generate internal ID
	id, err := util.NanoID(h.cfg.ID.Alphabet, h.cfg.ID.Length)
	if err != nil {
		jsonError(w, "failed to generate ID", http.StatusInternalServerError)
		return
	}

	width, height, _ := util.ImageDimensions(bytes.NewReader(data))

	ext := storage.MimeToExt(mime)
	diskPath, err := h.disk.Save(id, data, ext)
	if err != nil {
		jsonError(w, "failed to save file", http.StatusInternalServerError)
		return
	}

	img := &database.Image{
		ID:           id,
		Slug:         slug,
		Folder:       folder,
		OriginalName: header.Filename,
		MimeType:     mime,
		Size:         int64(len(data)),
		Width:        width,
		Height:       height,
		Hash:         hash,
		DiskPath:     diskPath,
	}

	if err := h.db.CreateImage(img); err != nil {
		jsonError(w, "failed to save metadata", http.StatusInternalServerError)
		return
	}

	h.db.EnsureFolder(folder)

	url := buildImageURL(h.cfg, folder, slug)
	resp := UploadResponse{
		ID:        id,
		Slug:      slug,
		Folder:    folder,
		URL:       url,
		Markdown:  fmt.Sprintf("![image](%s)", url),
		HTML:      fmt.Sprintf(`<img src="%s">`, url),
		BBCode:    fmt.Sprintf("[img]%s[/img]", url),
		Size:      int64(len(data)),
		MimeType:  mime,
		Width:     width,
		Height:    height,
		CreatedAt: "just now",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *UploadHandler) ensureUniqueSlug(folder, slug string) (string, error) {
	exists, err := h.db.SlugExists(folder, slug)
	if err != nil {
		return "", err
	}
	if !exists {
		return slug, nil
	}

	// Append number: image → image-2, image-3, ...
	for i := 2; i < 10000; i++ {
		candidate := fmt.Sprintf("%s-%d", slug, i)
		exists, err := h.db.SlugExists(folder, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("too many duplicates")
}

func buildImageURL(cfg *config.Config, folder, slug string) string {
	base := cfg.GetBaseURL()
	if folder != "" {
		return fmt.Sprintf("%s/%s/%s", base, folder, slug)
	}
	return fmt.Sprintf("%s/%s", base, slug)
}

func buildImagePath(folder, slug string) string {
	if folder != "" {
		return "/" + folder + "/" + slug
	}
	return "/" + slug
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
