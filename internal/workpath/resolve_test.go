package workpath_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/newtosh/tuile/internal/workpath"
)

func TestResolveAbsoluteDirectory(t *testing.T) {
	dir := t.TempDir()
	got, err := workpath.Resolve(dir)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("Resolve() = %q, want %q", got, want)
	}
}

func TestResolveRelativeDirectory(t *testing.T) {
	base := t.TempDir()
	t.Chdir(base)

	got, err := workpath.Resolve(".")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want, err := filepath.EvalSymlinks(base)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("Resolve() = %q, want %q", got, want)
	}
}

func TestResolveRejectsMissingDirectory(t *testing.T) {
	_, err := workpath.Resolve(filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("expected error for missing workspace")
	}
}

func TestResolveRejectsFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := workpath.Resolve(file)
	if err == nil {
		t.Fatal("expected error for file workspace")
	}
}

func TestResolveRejectsNULByte(t *testing.T) {
	_, err := workpath.Resolve("foo\x00bar")
	if err == nil {
		t.Fatal("expected error for NUL in path")
	}
}

func TestResolveRejectsEmptyPath(t *testing.T) {
	_, err := workpath.Resolve("")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestResolveFollowsSymlinkWithinRoot(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "target")
	link := filepath.Join(base, "link")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	got, err := workpath.Resolve(link)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("Resolve() = %q, want %q", got, want)
	}
}
