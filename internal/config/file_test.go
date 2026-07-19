package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/newtosh/tuile/internal/config"
)

func TestLoadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tuile.toml")
	if err := os.WriteFile(path, []byte(`
bootstrap_secret = "dev-bootstrap"
listen = "127.0.0.1:9999"
allowed_origins = ["http://127.0.0.1:9999"]
`), 0o600); err != nil {
		t.Fatal(err)
	}

	f, err := config.LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if f.BootstrapSecret != "dev-bootstrap" {
		t.Fatalf("bootstrap = %q", f.BootstrapSecret)
	}
	if f.Listen != "127.0.0.1:9999" {
		t.Fatalf("listen = %q", f.Listen)
	}
	if len(f.AllowedOrigins) != 1 {
		t.Fatalf("origins = %+v", f.AllowedOrigins)
	}
}

func TestFindFileWalksUp(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "tuile.toml")
	if err := os.WriteFile(path, []byte(`bootstrap_secret = "nested"`), 0o600); err != nil {
		t.Fatal(err)
	}

	found, err := config.FindFile(nested)
	if err != nil {
		t.Fatal(err)
	}
	if found != path {
		t.Fatalf("found = %q, want %q", found, path)
	}
}

func TestFindFileMissing(t *testing.T) {
	_, err := config.FindFile(t.TempDir())
	if err != config.ErrConfigNotFound {
		t.Fatalf("err = %v, want ErrConfigNotFound", err)
	}
}
