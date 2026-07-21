package export_test

import (
	"testing"

	"github.com/newtosh/tuile/internal/export"
	"github.com/newtosh/tuile/internal/term"
)

func TestScaleLayoutToOuterRemovesDownscale(t *testing.T) {
	opts := export.DefaultOptions()
	opts.Scale = 1
	snap := term.ScreenSnapshot{Cols: 80, Rows: 24, Lines: make([]string, 24)}
	layout := export.ScaleLayoutToOuter(export.ComputeLayout(snap, opts))
	if layout.Downscale != 1 {
		t.Fatalf("downscale = %d want 1", layout.Downscale)
	}
	if layout.RenderOuterW != layout.OuterW {
		t.Fatalf("render outer %dx%d != outer %dx%d", layout.RenderOuterW, layout.RenderOuterH, layout.OuterW, layout.OuterH)
	}
	if layout.CellW <= 0 || layout.CellH <= 0 {
		t.Fatalf("unexpected cell size %dx%d", layout.CellW, layout.CellH)
	}
}

func TestComputeLayoutUsesViewerTermDimensions(t *testing.T) {
	opts := export.DefaultOptions()
	opts.TermWPx = 1080
	opts.TermHPx = 720
	if err := opts.Validate(); err != nil {
		t.Fatal(err)
	}
	snap := term.ScreenSnapshot{Cols: 120, Rows: 36, Lines: make([]string, 36)}
	layout := export.ComputeLayout(snap, opts)
	if layout.OuterW != 1108 {
		t.Fatalf("outer width = %d want 1108", layout.OuterW)
	}
	if layout.OuterH != 748 {
		t.Fatalf("outer height = %d want 748", layout.OuterH)
	}
	if layout.CellW != 18 {
		t.Fatalf("cell width = %d want 18", layout.CellW)
	}
	if layout.CellH != 40 {
		t.Fatalf("cell height = %d want 40", layout.CellH)
	}
}

func TestOptionsValidateRequiresBothTermDimensions(t *testing.T) {
	opts := export.DefaultOptions()
	opts.TermWPx = 100
	if err := opts.Validate(); err == nil {
		t.Fatal("expected validation error for partial term dimensions")
	}
}
