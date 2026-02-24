package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"jmg/internal/config"
	"jmg/internal/database"
)

type BrowseHandler struct {
	cfg *config.Config
	db  *database.DB
}

func NewBrowseHandler(cfg *config.Config, db *database.DB) *BrowseHandler {
	return &BrowseHandler{cfg: cfg, db: db}
}

type BrowseResponse struct {
	Folder       string      `json:"folder"`
	Images       []ImageItem `json:"images"`
	Total        int         `json:"total"`
	Page         int         `json:"page"`
	Pages        int         `json:"pages"`
	ChildFolders []string    `json:"child_folders,omitempty"`
}

type ImageItem struct {
	ID           string `json:"id"`
	Slug         string `json:"slug"`
	URL          string `json:"url"`
	Thumbnail    string `json:"thumbnail"`
	OriginalName string `json:"original_name"`
	Folder       string `json:"folder"`
	MimeType     string `json:"mime_type"`
	Size         int64  `json:"size"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
	CreatedAt    string `json:"created_at"`
}

func (h *BrowseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	folder := strings.TrimPrefix(r.URL.Path, "/api/browse")
	folder = strings.Trim(folder, "/")

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 50
	}

	images, total, err := h.db.ListImages(folder, page, limit)
	if err != nil {
		jsonError(w, "failed to list images", http.StatusInternalServerError)
		return
	}

	pages := total / limit
	if total%limit != 0 {
		pages++
	}

	items := make([]ImageItem, 0, len(images))
	for _, img := range images {
		var urlPath string
		if img.Folder != "" {
			urlPath = "/" + img.Folder + "/" + img.Slug
		} else {
			urlPath = "/" + img.Slug
		}
		items = append(items, ImageItem{
			ID:           img.ID,
			Slug:         img.Slug,
			URL:          urlPath,
			Thumbnail:    urlPath + "?w=200",
			OriginalName: img.OriginalName,
			Folder:       img.Folder,
			MimeType:     img.MimeType,
			Size:         img.Size,
			Width:        img.Width,
			Height:       img.Height,
			CreatedAt:    img.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	childFolderInfos, _ := h.db.GetChildFolders(folder)
	var childFolderNames []string
	for _, f := range childFolderInfos {
		childFolderNames = append(childFolderNames, f.Name)
	}

	resp := BrowseResponse{
		Folder:       folder,
		Images:       items,
		Total:        total,
		Page:         page,
		Pages:        pages,
		ChildFolders: childFolderNames,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// FoldersHandler returns all folders
type FoldersHandler struct {
	db *database.DB
}

func NewFoldersHandler(db *database.DB) *FoldersHandler {
	return &FoldersHandler{db: db}
}

type FoldersResponse struct {
	Folders   []database.FolderInfo `json:"folders"`
	RootCount int                   `json:"root_count"`
}

func (h *FoldersHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	folders, rootCount, err := h.db.ListAllFolders()
	if err != nil {
		jsonError(w, "failed to list folders", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(FoldersResponse{Folders: folders, RootCount: rootCount})
}
