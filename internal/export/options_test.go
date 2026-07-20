package export_test

import (
	"bytes"
	"testing"

	"github.com/newtosh/tuile/internal/export"
	"github.com/newtosh/tuile/internal/term"
)

func TestOptionsValidateRejectsInvalidChrome(t *testing.T) {
	opts := export.DefaultOptions()
	opts.ChromePreset = "native-macos"
	if err := opts.Validate(); err == nil {
		t.Fatal("expected invalid chrome error")
	}
}

func TestOptionsScaleDoublesLayoutWidth(t *testing.T) {
	opts := export.DefaultOptions()
	opts.Scale = 2
	if err := opts.Validate(); err != nil {
		t.Fatal(err)
	}
	snap := term.ScreenSnapshot{Cols: 10, Rows: 2, Lines: []string{"hello", "world"}}
	layout1 := export.ComputeLayout(snap, export.DefaultOptions())
	layout2 := export.ComputeLayout(snap, opts)
	if layout2.OuterW != layout1.OuterW*2 {
		t.Fatalf("scale 2 width = %d want %d", layout2.OuterW, layout1.OuterW*2)
	}
}

func TestRenderPNGMinimalChrome(t *testing.T) {
	opts := export.DefaultOptions()
	opts.BackgroundPreset = "slate"
	snap := term.ScreenSnapshot{
		Cols:  5,
		Rows:  1,
		Lines: []string{"ok"},
	}
	png, err := export.RenderPNG(snap, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(png) < 8 || !bytes.HasPrefix(png, []byte{0x89, 'P', 'N', 'G'}) {
		t.Fatalf("invalid png prefix %v", png[:min(8, len(png))])
	}
}

func TestRenderSVGContainsText(t *testing.T) {
	opts := export.DefaultOptions()
	opts.Format = export.FormatSVG
	snap := term.ScreenSnapshot{Cols: 3, Rows: 1, Lines: []string{"hi"}}
	svg, err := export.RenderSVG(snap, opts)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(svg, []byte("<svg")) || !bytes.Contains(svg, []byte("hi")) {
		t.Fatalf("unexpected svg: %s", svg)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
