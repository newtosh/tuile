package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// ErrConfigNotFound indicates no tuile.toml was found while searching.
var ErrConfigNotFound = errors.New("tuile.toml not found")

// File is optional project configuration loaded from tuile.toml.
type File struct {
	BootstrapSecret string   `toml:"bootstrap_secret"`
	Listen          string   `toml:"listen"`
	AllowedOrigins  []string `toml:"allowed_origins"`
}

// LoadFile reads configuration from path.
func LoadFile(path string) (File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return File{}, err
	}
	var f File
	if err := toml.Unmarshal(data, &f); err != nil {
		return File{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return f, nil
}

// FindFile searches start and parent directories for tuile.toml.
func FindFile(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		path := filepath.Join(dir, "tuile.toml")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrConfigNotFound
		}
		dir = parent
	}
}

// LoadNearest reads tuile.toml from start or a parent directory.
func LoadNearest(start string) (File, string, error) {
	path, err := FindFile(start)
	if err != nil {
		return File{}, "", err
	}
	f, err := LoadFile(path)
	if err != nil {
		return File{}, path, err
	}
	return f, path, nil
}

// ApplyFile merges file settings into server config.
func ApplyFile(cfg *Server, f File) {
	if f.Listen != "" {
		cfg.Listen = f.Listen
	}
	if len(f.AllowedOrigins) > 0 {
		cfg.AllowedOrigins = append([]string(nil), f.AllowedOrigins...)
	}
	if f.BootstrapSecret != "" {
		cfg.BootstrapSecret = f.BootstrapSecret
	}
}
