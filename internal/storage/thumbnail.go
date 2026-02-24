package storage

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"

	_ "image/gif"

	"jmg/internal/config"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

type ThumbnailGenerator struct {
	basePath string
	thumbDir string
	config   config.ThumbnailConfig
}

func NewThumbnailGenerator(dataDir string, cfg config.ThumbnailConfig) *ThumbnailGenerator {
	thumbDir := filepath.Join(dataDir, "thumbnails")
	os.MkdirAll(thumbDir, 0755)
	return &ThumbnailGenerator{
		basePath: dataDir,
		thumbDir: thumbDir,
		config:   cfg,
	}
}

func (tg *ThumbnailGenerator) GetOrCreate(diskPath string, id string, width int) (string, error) {
	if !tg.config.Enabled {
		return filepath.Join(tg.basePath, diskPath), nil
	}

	prefix := id[:2]
	thumbName := fmt.Sprintf("%s_%d.jpg", id, width)
	thumbRelPath := filepath.Join("thumbnails", prefix, thumbName)
	thumbFullPath := filepath.Join(tg.basePath, thumbRelPath)

	if _, err := os.Stat(thumbFullPath); err == nil {
		return thumbFullPath, nil
	}

	srcPath := filepath.Join(tg.basePath, diskPath)
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("opening source: %w", err)
	}
	defer srcFile.Close()

	srcImg, _, err := image.Decode(srcFile)
	if err != nil {
		return srcPath, nil
	}

	bounds := srcImg.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	if srcW <= width {
		return srcPath, nil
	}

	newH := int(float64(srcH) * float64(width) / float64(srcW))

	dst := image.NewRGBA(image.Rect(0, 0, width, newH))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), srcImg, bounds, draw.Over, nil)

	thumbDirPath := filepath.Join(tg.thumbDir, prefix)
	os.MkdirAll(thumbDirPath, 0755)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: tg.config.Quality}); err != nil {
		return "", fmt.Errorf("encoding thumbnail: %w", err)
	}

	if err := os.WriteFile(thumbFullPath, buf.Bytes(), 0644); err != nil {
		return "", fmt.Errorf("writing thumbnail: %w", err)
	}

	return thumbFullPath, nil
}

var _ = png.Encode
