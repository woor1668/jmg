package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Storage   StorageConfig   `yaml:"storage"`
	Thumbnail ThumbnailConfig `yaml:"thumbnail"`
	Auth      AuthConfig      `yaml:"auth"`
	ID        IDConfig        `yaml:"id"`
}

type ServerConfig struct {
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	BaseURL string `yaml:"base_url"`
}

type StorageConfig struct {
	DataDir      string   `yaml:"data_dir"`
	MaxFileSize  int64    `yaml:"max_file_size"`
	AllowedTypes []string `yaml:"allowed_types"`
}

type ThumbnailConfig struct {
	Enabled bool   `yaml:"enabled"`
	Sizes   []int  `yaml:"sizes"`
	Format  string `yaml:"format"`
	Quality int    `yaml:"quality"`
}

type AuthConfig struct {
	Token        string `yaml:"token"`
	PublicBrowse bool   `yaml:"public_browse"`
}

type IDConfig struct {
	Length   int    `yaml:"length"`
	Alphabet string `yaml:"alphabet"`
}

func defaults() *Config {
	return &Config{
		Server: ServerConfig{
			Host:    "0.0.0.0",
			Port:    8080,
			BaseURL: "",
		},
		Storage: StorageConfig{
			DataDir:     "./data",
			MaxFileSize: 50 * 1024 * 1024,
			AllowedTypes: []string{
				"image/png",
				"image/jpeg",
				"image/gif",
				"image/webp",
				"image/svg+xml",
				"image/avif",
				"image/bmp",
				"image/tiff",
			},
		},
		Thumbnail: ThumbnailConfig{
			Enabled: true,
			Sizes:   []int{200, 400, 800},
			Format:  "jpeg",
			Quality: 80,
		},
		Auth: AuthConfig{
			Token:        "",
			PublicBrowse: false,
		},
		ID: IDConfig{
			Length:   6,
			Alphabet: "0123456789abcdefghijklmnopqrstuvwxyz",
		},
	}
}

func Load(path string) (*Config, error) {
	cfg := defaults()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			applyEnvOverrides(cfg)
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	applyEnvOverrides(cfg)

	if cfg.Auth.Token == "" {
		// 토큰 없으면 개발 모드 (인증 스킵)
		fmt.Println("⚠️  경고: auth.token이 비어있습니다. 인증 없이 실행됩니다.")
	}

	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("AUTH_TOKEN"); v != "" {
		cfg.Auth.Token = v
	}
	if v := os.Getenv("BASE_URL"); v != "" {
		cfg.Server.BaseURL = v
	}
	if v := os.Getenv("PORT"); v != "" {
		fmt.Sscanf(v, "%d", &cfg.Server.Port)
	}
	if v := os.Getenv("DATA_DIR"); v != "" {
		cfg.Storage.DataDir = v
	}
	if v := os.Getenv("HOST"); v != "" {
		cfg.Server.Host = v
	}
	if v := os.Getenv("PUBLIC_BROWSE"); strings.ToLower(v) == "true" {
		cfg.Auth.PublicBrowse = true
	}
}

func (c *Config) GetBaseURL() string {
	if c.Server.BaseURL != "" {
		return strings.TrimRight(c.Server.BaseURL, "/")
	}
	return fmt.Sprintf("http://%s:%d", c.Server.Host, c.Server.Port)
}

func (c *Config) IsAllowedType(mimeType string) bool {
	for _, t := range c.Storage.AllowedTypes {
		if t == mimeType {
			return true
		}
	}
	return false
}
