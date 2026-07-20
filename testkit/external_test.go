package testkit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultBaseURL(t *testing.T) {
	if DefaultBaseURL != "http://127.0.0.1:7710" {
		t.Fatalf("DefaultBaseURL = %q", DefaultBaseURL)
	}
	if DefaultListenHostPort() != "127.0.0.1:7710" {
		t.Fatalf("listen = %q", DefaultListenHostPort())
	}
}

func TestFormatViewURL(t *testing.T) {
	got := FormatViewURL("http://127.0.0.1:7710", "abc", "tok")
	want := "http://127.0.0.1:7710/view?session=abc&token=tok"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveBootstrapFromSiblingTuile(t *testing.T) {
	root := findGoModuleRoot(".")
	if root == "" {
		t.Skip("no go.mod")
	}
	sibling := filepath.Clean(filepath.Join(root, "..", "tuile", "tuile.toml"))
	if _, err := os.Stat(sibling); err != nil {
		t.Skip("no sibling tuile.toml")
	}
	got := resolveBootstrapSecret(t)
	if got == "" {
		t.Fatalf("expected bootstrap from %s", sibling)
	}
}
