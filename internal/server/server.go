package server

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"jmg/internal/config"
	"jmg/internal/database"
	"jmg/internal/handler"
	"jmg/internal/middleware"
	"jmg/internal/storage"
	"jmg/web"
)

type Server struct {
	cfg      *config.Config
	httpSrv  *http.Server
	db       *database.DB
	disk     *storage.Disk
	thumbGen *storage.ThumbnailGenerator
}

func New(cfg *config.Config, db *database.DB, disk *storage.Disk, thumbGen *storage.ThumbnailGenerator) *Server {
	s := &Server{cfg: cfg, db: db, disk: disk, thumbGen: thumbGen}
	mux := http.NewServeMux()
	s.setupRoutes(mux)
	var h http.Handler = mux
	h = middleware.CORS(h)
	h = middleware.Logger(h)
	s.httpSrv = &http.Server{
		Addr: fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port), Handler: h,
		ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 120 * time.Second,
	}
	return s
}

func (s *Server) setupRoutes(mux *http.ServeMux) {
	auth := middleware.Auth(s.cfg.Auth.Token)

	up := handler.NewUploadHandler(s.cfg, s.db, s.disk)
	br := handler.NewBrowseHandler(s.cfg, s.db)
	fl := handler.NewFoldersHandler(s.db)
	dl := handler.NewDeleteHandler(s.db, s.disk)
	bdl := handler.NewBulkDeleteHandler(s.db, s.disk)
	iu := handler.NewImageUpdateHandler(s.db)
	bm := handler.NewBulkMoveHandler(s.db)
	fr := handler.NewFolderRenameHandler(s.db)
	fc := handler.NewFolderCreateHandler(s.db)
	fd := handler.NewFolderDeleteHandler(s.db, s.disk)
	sv := handler.NewServeHandler(s.db, s.disk, s.thumbGen)

	mux.Handle("/api/upload", auth(up))
	mux.Handle("/api/images", auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodDelete:
			dl.ServeHTTP(w, r)
		case http.MethodPatch:
			iu.ServeHTTP(w, r)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})))
	mux.Handle("/api/bulk-delete", auth(bdl))
	mux.Handle("/api/bulk-move", auth(bm))
	mux.Handle("/api/folder-rename", auth(fr))
	mux.Handle("/api/folder-create", auth(fc))
	mux.Handle("/api/folder-delete", auth(fd))
	mux.Handle("/api/folders", s.optionalAuth(fl))
	mux.Handle("/api/browse/", s.optionalAuth(br))
	mux.Handle("/api/browse", s.optionalAuth(br))
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
	mux.Handle("/admin/", s.adminHandler(web.StaticFiles))
	mux.Handle("/admin", http.RedirectHandler("/admin/", http.StatusMovedPermanently))
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	mux.Handle("/", sv)
}

func (s *Server) optionalAuth(h http.Handler) http.Handler {
	if s.cfg.Auth.PublicBrowse {
		return h
	}
	return middleware.Auth(s.cfg.Auth.Token)(h)
}

func (s *Server) adminHandler(staticFS embed.FS) http.Handler {
	sub, _ := fs.Sub(staticFS, "static")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/admin")
		path = strings.TrimPrefix(path, "/")
		if path == "" {
			path = "index.html"
		}
		data, err := fs.ReadFile(sub, path)
		if err != nil {
			data, _ = fs.ReadFile(sub, "index.html")
			path = "index.html"
		}
		switch {
		case strings.HasSuffix(path, ".html"):
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		case strings.HasSuffix(path, ".css"):
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		case strings.HasSuffix(path, ".js"):
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		}
		w.Write(data)
	})
}

func (s *Server) Start() error  { return s.httpSrv.ListenAndServe() }
func (s *Server) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	s.httpSrv.Shutdown(ctx)
}
