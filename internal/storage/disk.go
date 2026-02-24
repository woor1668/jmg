package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

type Disk struct {
	basePath   string
	imagesPath string
}

func NewDisk(dataDir string) (*Disk, error) {
	imagesPath := filepath.Join(dataDir, "images")
	if err := os.MkdirAll(imagesPath, 0755); err != nil {
		return nil, fmt.Errorf("creating images dir: %w", err)
	}
	return &Disk{
		basePath:   dataDir,
		imagesPath: imagesPath,
	}, nil
}

func (d *Disk) Save(id string, data []byte, ext string) (string, error) {
	prefix := id[:2]
	dir := filepath.Join(d.imagesPath, prefix)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating prefix dir: %w", err)
	}

	filename := id + ext
	fullPath := filepath.Join(dir, filename)

	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", fmt.Errorf("writing file: %w", err)
	}

	relPath := filepath.Join("images", prefix, filename)
	return relPath, nil
}

func (d *Disk) FullPath(relPath string) string {
	return filepath.Join(d.basePath, relPath)
}

func (d *Disk) Delete(relPath string) error {
	fullPath := filepath.Join(d.basePath, relPath)
	return os.Remove(fullPath)
}

func (d *Disk) Exists(relPath string) bool {
	fullPath := filepath.Join(d.basePath, relPath)
	_, err := os.Stat(fullPath)
	return err == nil
}

func MimeToExt(mime string) string {
	switch mime {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"
	case "image/avif":
		return ".avif"
	case "image/bmp":
		return ".bmp"
	case "image/tiff":
		return ".tiff"
	default:
		return ".bin"
	}
}
