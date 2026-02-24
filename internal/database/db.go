package database

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

type Image struct {
	ID           string    `json:"id"`
	Slug         string    `json:"slug"`
	Folder       string    `json:"folder"`
	OriginalName string    `json:"original_name"`
	MimeType     string    `json:"mime_type"`
	Size         int64     `json:"size"`
	Width        int       `json:"width,omitempty"`
	Height       int       `json:"height,omitempty"`
	Hash         string    `json:"hash"`
	DiskPath     string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type FolderInfo struct {
	Name      string `json:"name"`
	Count     int    `json:"count"`
	CreatedAt string `json:"created_at,omitempty"`
}

func New(dataDir string) (*DB, error) {
	dbPath := filepath.Join(dataDir, "images.db")
	conn, err := sql.Open("sqlite", dbPath+"?_journal=WAL&_timeout=5000&_fk=true")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)
	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("migration: %w", err)
	}
	return db, nil
}

func (db *DB) Close() error { return db.conn.Close() }

func (db *DB) migrate() error {
	// Try add slug column for existing DBs
	db.conn.Exec(`ALTER TABLE images ADD COLUMN slug TEXT NOT NULL DEFAULT ''`)
	db.conn.Exec(`UPDATE images SET slug = id WHERE slug = ''`)

	schema := `
	CREATE TABLE IF NOT EXISTS images (
		id            TEXT PRIMARY KEY,
		slug          TEXT NOT NULL,
		folder        TEXT NOT NULL DEFAULT '',
		original_name TEXT NOT NULL,
		mime_type     TEXT NOT NULL,
		size          INTEGER NOT NULL,
		width         INTEGER DEFAULT 0,
		height        INTEGER DEFAULT 0,
		hash          TEXT NOT NULL,
		disk_path     TEXT NOT NULL,
		created_at    DATETIME DEFAULT (datetime('now')),
		UNIQUE(folder, slug)
	);
	CREATE INDEX IF NOT EXISTS idx_images_folder ON images(folder);
	CREATE INDEX IF NOT EXISTS idx_images_hash ON images(hash);
	CREATE INDEX IF NOT EXISTS idx_images_folder_slug ON images(folder, slug);

	CREATE TABLE IF NOT EXISTS folders (
		path        TEXT PRIMARY KEY,
		created_at  DATETIME DEFAULT (datetime('now'))
	);
	`
	_, err := db.conn.Exec(schema)
	return err
}

// ── Folder CRUD ──

func (db *DB) CreateFolder(path string) error {
	if path == "" {
		return nil
	}
	// Create all ancestors: a → a, a/b → a, a/b
	parts := strings.Split(path, "/")
	for i := range parts {
		ancestor := strings.Join(parts[:i+1], "/")
		db.conn.Exec(`INSERT OR IGNORE INTO folders (path) VALUES (?)`, ancestor)
	}
	return nil
}

func (db *DB) RenameFolder(oldPath, newPath string) (int64, error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Rename exact folder
	tx.Exec(`UPDATE folders SET path = ? WHERE path = ?`, newPath, oldPath)
	// Rename sub-folders
	tx.Exec(`UPDATE folders SET path = ? || substr(path, ?) WHERE path LIKE ?`,
		newPath, len(oldPath)+1, oldPath+"/%")

	// Move images
	r1, _ := tx.Exec(`UPDATE images SET folder = ? WHERE folder = ?`, newPath, oldPath)
	c1, _ := r1.RowsAffected()
	r2, _ := tx.Exec(`UPDATE images SET folder = ? || substr(folder, ?) WHERE folder LIKE ?`,
		newPath, len(oldPath)+1, oldPath+"/%")
	c2, _ := r2.RowsAffected()

	return c1 + c2, tx.Commit()
}

func (db *DB) DeleteFolder(path string) (int64, error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Delete folder and sub-folders
	tx.Exec(`DELETE FROM folders WHERE path = ? OR path LIKE ?`, path, path+"/%")
	// Delete images in folder and sub-folders
	r1, _ := tx.Exec(`DELETE FROM images WHERE folder = ?`, path)
	c1, _ := r1.RowsAffected()
	r2, _ := tx.Exec(`DELETE FROM images WHERE folder LIKE ?`, path+"/%")
	c2, _ := r2.RowsAffected()

	return c1 + c2, tx.Commit()
}

// GetChildFolders returns immediate children of parent path
func (db *DB) GetChildFolders(parent string) ([]FolderInfo, error) {
	prefix := parent
	if prefix != "" {
		prefix += "/"
	}

	rows, err := db.conn.Query(`SELECT path FROM folders ORDER BY path`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// First collect all paths
	var allPaths []string
	for rows.Next() {
		var p string
		rows.Scan(&p)
		allPaths = append(allPaths, p)
	}

	// Then process (no more active query)
	seen := map[string]bool{}
	var result []FolderInfo
	for _, p := range allPaths {
		if parent == "" {
			// Top-level: folders with no /
			if !strings.Contains(p, "/") && !seen[p] {
				seen[p] = true
				cnt := db.countImagesInFolder(p)
				result = append(result, FolderInfo{Name: p, Count: cnt})
			}
		} else if strings.HasPrefix(p, prefix) {
			rest := p[len(prefix):]
			parts := strings.SplitN(rest, "/", 2)
			child := prefix + parts[0]
			if !seen[child] {
				seen[child] = true
				cnt := db.countImagesInFolder(child)
				result = append(result, FolderInfo{Name: child, Count: cnt})
			}
		}
	}
	return result, nil
}

func (db *DB) countImagesInFolder(folder string) int {
	var c int
	db.conn.QueryRow(`SELECT COUNT(*) FROM images WHERE folder = ?`, folder).Scan(&c)
	return c
}

// ListAllFolders returns flat list of all folders
func (db *DB) ListAllFolders() ([]FolderInfo, int, error) {
	var rootCount int
	db.conn.QueryRow(`SELECT COUNT(*) FROM images WHERE folder = ''`).Scan(&rootCount)

	rows, err := db.conn.Query(`SELECT path FROM folders ORDER BY path`)
	if err != nil {
		return nil, rootCount, err
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var p string
		rows.Scan(&p)
		paths = append(paths, p)
	}

	var folders []FolderInfo
	seen := map[string]bool{}
	for _, p := range paths {
		seen[p] = true
		cnt := db.countImagesInFolder(p)
		folders = append(folders, FolderInfo{Name: p, Count: cnt})
	}

	// Also include folders from images not in folders table
	imgRows, _ := db.conn.Query(`SELECT DISTINCT folder FROM images WHERE folder != '' ORDER BY folder`)
	if imgRows != nil {
		defer imgRows.Close()
		var imgFolders []string
		for imgRows.Next() {
			var f string
			imgRows.Scan(&f)
			imgFolders = append(imgFolders, f)
		}
		for _, f := range imgFolders {
			if !seen[f] {
				cnt := db.countImagesInFolder(f)
				folders = append(folders, FolderInfo{Name: f, Count: cnt})
				db.CreateFolder(f)
			}
		}
	}
	return folders, rootCount, nil
}

// ── Image CRUD ──

func (db *DB) CreateImage(img *Image) error {
	_, err := db.conn.Exec(
		`INSERT INTO images (id, slug, folder, original_name, mime_type, size, width, height, hash, disk_path)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		img.ID, img.Slug, img.Folder, img.OriginalName, img.MimeType,
		img.Size, img.Width, img.Height, img.Hash, img.DiskPath,
	)
	return err
}

func (db *DB) GetImageBySlug(folder, slug string) (*Image, error) {
	img := &Image{}
	err := db.conn.QueryRow(
		`SELECT id, slug, folder, original_name, mime_type, size, width, height, hash, disk_path, created_at
		 FROM images WHERE folder = ? AND slug = ?`, folder, slug,
	).Scan(&img.ID, &img.Slug, &img.Folder, &img.OriginalName, &img.MimeType,
		&img.Size, &img.Width, &img.Height, &img.Hash, &img.DiskPath, &img.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return img, err
}

func (db *DB) GetImageByID(id string) (*Image, error) {
	img := &Image{}
	err := db.conn.QueryRow(
		`SELECT id, slug, folder, original_name, mime_type, size, width, height, hash, disk_path, created_at
		 FROM images WHERE id = ?`, id,
	).Scan(&img.ID, &img.Slug, &img.Folder, &img.OriginalName, &img.MimeType,
		&img.Size, &img.Width, &img.Height, &img.Hash, &img.DiskPath, &img.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return img, err
}

func (db *DB) GetImageByHash(hash string) (*Image, error) {
	img := &Image{}
	err := db.conn.QueryRow(
		`SELECT id, slug, folder, original_name, mime_type, size, width, height, hash, disk_path, created_at
		 FROM images WHERE hash = ? LIMIT 1`, hash,
	).Scan(&img.ID, &img.Slug, &img.Folder, &img.OriginalName, &img.MimeType,
		&img.Size, &img.Width, &img.Height, &img.Hash, &img.DiskPath, &img.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return img, err
}

func (db *DB) SlugExists(folder, slug string) (bool, error) {
	var c int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM images WHERE folder = ? AND slug = ?`, folder, slug).Scan(&c)
	return c > 0, err
}

func (db *DB) ListImages(folder string, page, limit int) ([]Image, int, error) {
	offset := (page - 1) * limit
	var total int
	db.conn.QueryRow(`SELECT COUNT(*) FROM images WHERE folder = ?`, folder).Scan(&total)

	rows, err := db.conn.Query(
		`SELECT id, slug, folder, original_name, mime_type, size, width, height, hash, disk_path, created_at
		 FROM images WHERE folder = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		folder, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var imgs []Image
	for rows.Next() {
		var img Image
		rows.Scan(&img.ID, &img.Slug, &img.Folder, &img.OriginalName, &img.MimeType,
			&img.Size, &img.Width, &img.Height, &img.Hash, &img.DiskPath, &img.CreatedAt)
		imgs = append(imgs, img)
	}
	return imgs, total, nil
}

func (db *DB) DeleteImage(id string) (*Image, error) {
	img, err := db.GetImageByID(id)
	if err != nil || img == nil {
		return nil, err
	}
	db.conn.Exec(`DELETE FROM images WHERE id = ?`, id)
	return img, nil
}

func (db *DB) DeleteImagesByIDs(ids []string) ([]Image, error) {
	var del []Image
	for _, id := range ids {
		if img, _ := db.DeleteImage(id); img != nil {
			del = append(del, *img)
		}
	}
	return del, nil
}

func (db *DB) UpdateSlug(id, newSlug string) error {
	r, err := db.conn.Exec(`UPDATE images SET slug = ? WHERE id = ?`, newSlug, id)
	if err != nil {
		return err
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return fmt.Errorf("not found")
	}
	return nil
}

func (db *DB) UpdateFolder(id, newFolder string) error {
	r, err := db.conn.Exec(`UPDATE images SET folder = ? WHERE id = ?`, newFolder, id)
	if err != nil {
		return err
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return fmt.Errorf("not found")
	}
	return nil
}

func (db *DB) MoveImages(ids []string, newFolder string) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, _ := tx.Prepare(`UPDATE images SET folder = ? WHERE id = ?`)
	defer stmt.Close()
	for _, id := range ids {
		stmt.Exec(newFolder, id)
	}
	return tx.Commit()
}

// GetImagesInFolder returns images for deletion when folder is deleted
func (db *DB) GetImagesInFolderTree(folder string) ([]Image, error) {
	rows, err := db.conn.Query(
		`SELECT id, slug, folder, original_name, mime_type, size, width, height, hash, disk_path, created_at
		 FROM images WHERE folder = ? OR folder LIKE ?`, folder, folder+"/%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var imgs []Image
	for rows.Next() {
		var img Image
		rows.Scan(&img.ID, &img.Slug, &img.Folder, &img.OriginalName, &img.MimeType,
			&img.Size, &img.Width, &img.Height, &img.Hash, &img.DiskPath, &img.CreatedAt)
		imgs = append(imgs, img)
	}
	return imgs, nil
}

func (db *DB) EnsureFolder(name string) error {
	return db.CreateFolder(name)
}
