package cli_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/newtosh/tuile/internal/cli"
	"github.com/newtosh/tuile/internal/pty"
)

func TestResolveUnknownCLI(t *testing.T) {
	_, err := cli.Resolve("not-a-cli")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveFromPATH(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "claude")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	d, err := cli.Resolve(cli.Claude)
	if err != nil {
		t.Fatal(err)
	}
	if d.Bin != fake {
		t.Fatalf("bin = %q, want %q", d.Bin, fake)
	}
	opts := d.SessionOptions()
	if opts.Command != fake {
		t.Fatalf("command = %q", opts.Command)
	}
	if opts.Cols != 120 || opts.Rows != 36 {
		t.Fatalf("dims = %dx%d, want 120x36", opts.Cols, opts.Rows)
	}
}

func TestResolveMissingBinary(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, err := cli.Resolve(cli.Codex)
	if !errors.Is(err, cli.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestResolveCursorCLI(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "cursor-agent")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	d, err := cli.Resolve(cli.CursorCLI)
	if err != nil {
		t.Fatal(err)
	}
	if d.Bin != fake {
		t.Fatalf("bin = %q, want %q", d.Bin, fake)
	}
	if d.Name != cli.CursorCLI {
		t.Fatalf("name = %q, want %q", d.Name, cli.CursorCLI)
	}
	if len(d.Args) != 1 || d.Args[0] != "--yolo" {
		t.Fatalf("args = %v, want [--yolo]", d.Args)
	}
	if d.InputStrategy != pty.StrategyBracketedPaste {
		t.Fatalf("strategy = %v, want bracketed paste", d.InputStrategy)
	}
}

func TestInputStrategyCursorCLI(t *testing.T) {
	if got := cli.InputStrategy(cli.CursorCLI); got != pty.StrategyBracketedPaste {
		t.Fatalf("strategy = %v, want bracketed paste", got)
	}
}

func TestResolveCopilotCLI(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "copilot")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	d, err := cli.Resolve(cli.CopilotCLI)
	if err != nil {
		t.Fatal(err)
	}
	if d.Bin != fake {
		t.Fatalf("bin = %q, want %q", d.Bin, fake)
	}
	if len(d.Args) != 1 || d.Args[0] != "--yolo" {
		t.Fatalf("args = %v, want [--yolo]", d.Args)
	}
	if d.InputStrategy != pty.StrategyBracketedPaste {
		t.Fatalf("strategy = %v, want bracketed paste", d.InputStrategy)
	}
}

func TestResolveOpencodeCLI(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "opencode")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	d, err := cli.Resolve(cli.OpencodeCLI)
	if err != nil {
		t.Fatal(err)
	}
	if d.Bin != fake {
		t.Fatalf("bin = %q, want %q", d.Bin, fake)
	}
	if len(d.Args) != 1 || d.Args[0] != "--auto" {
		t.Fatalf("args = %v, want [--auto]", d.Args)
	}
	if d.InputStrategy != pty.StrategyStandard {
		t.Fatalf("strategy = %v, want standard", d.InputStrategy)
	}
}

func TestInputStrategyCopilotCLI(t *testing.T) {
	if got := cli.InputStrategy(cli.CopilotCLI); got != pty.StrategyBracketedPaste {
		t.Fatalf("strategy = %v, want bracketed paste", got)
	}
}
