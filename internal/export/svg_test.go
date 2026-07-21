package export_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/newtosh/tuile/internal/export"
	"github.com/newtosh/tuile/internal/term"
)

func TestRenderSVGRootDimensionsMatchViewBox(t *testing.T) {
	opts := export.DefaultOptions()
	opts.Format = export.FormatSVG
	opts.Scale = 1
	snap := term.ScreenSnapshot{Cols: 8, Rows: 2, Lines: []string{"line one", "line two"}}
	svg, err := export.RenderSVG(snap, opts)
	if err != nil {
		t.Fatal(err)
	}
	layout := export.ComputeLayout(snap, opts)
	want := fmt.Sprintf(`width="%d" height="%d" viewBox="0 0 %d %d"`, layout.OuterW, layout.OuterH, layout.OuterW, layout.OuterH)
	if !bytes.Contains(svg, []byte(want)) {
		t.Fatalf("svg root mismatch, want %q in:\n%s", want, svg)
	}
	if bytes.Contains(svg, []byte(`transform="scale(`)) {
		t.Fatal("expected svg coordinates in outer space without scale wrapper")
	}
}

func TestRenderSVGTerminalFontFitsCells(t *testing.T) {
	opts := export.DefaultOptions()
	opts.Format = export.FormatSVG
	opts.FontSizePx = 14
	snap := term.ScreenSnapshot{
		Cols:  80,
		Rows:  1,
		Lines: []string{"hello"},
		Grid: []term.RowSnapshot{{
			Y: 0,
			Cells: []term.CellSnapshot{
				{Ch: "h", Fg: "#ffffff", Bg: "#000000"},
				{Ch: "i", Fg: "#ffffff", Bg: "#000000"},
			},
		}},
	}
	svg, err := export.RenderSVG(snap, opts)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(svg, []byte(`text-anchor="start"`)) {
		t.Fatal("expected left-anchored terminal glyphs")
	}
	layout := export.ScaleLayoutToOuter(export.ComputeLayout(snap, opts))
	wantFont := layout.CellH - 6*layout.RenderScale
	if wantFont < 8 {
		wantFont = 8
	}
	if !bytes.Contains(svg, []byte(fmt.Sprintf(`font-size="%d"`, wantFont))) {
		t.Fatalf("expected font-size %d in svg:\n%s", wantFont, svg)
	}
}

func TestRenderSVGMinimalChrome(t *testing.T) {
	opts := export.DefaultOptions()
	opts.Format = export.FormatSVG
	opts.BackgroundMode = export.BackgroundTransparent
	snap := term.ScreenSnapshot{Cols: 6, Rows: 1, Lines: []string{"ok"}}
	svg, err := export.RenderSVG(snap, opts)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(svg, []byte("<svg")) || !bytes.Contains(svg, []byte("ok")) {
		t.Fatalf("unexpected svg: %s", svg)
	}
	if !bytes.Contains(svg, []byte(`fill="#0b0c0f"`)) {
		t.Fatal("expected opaque viewer frame in transparent svg export")
	}
}

func TestRenderSVGCustomBackgroundEmbedsImage(t *testing.T) {
	opts := export.DefaultOptions()
	opts.Format = export.FormatSVG
	opts.BackgroundMode = export.BackgroundCustom
	opts.ChromePreset = export.ChromeOS
	opts.ChromeOSStyle = export.OSStyleMacOS
	snap := term.ScreenSnapshot{Cols: 4, Rows: 1, Lines: []string{"x"}}
	pngBytes := solidPNG(t, 255, 0, 0)
	svg, err := export.RenderSVGWithBackground(snap, opts, pngBytes)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(svg, []byte("data:image/png;base64,")) {
		t.Fatal("expected embedded custom background image")
	}
	layout := export.ComputeLayout(snap, opts)
	if !bytes.Contains(svg, []byte(`viewBox="0 0 `)) {
		t.Fatal("missing viewBox")
	}
	if layout.ScenePad == 0 {
		t.Fatal("expected scene pad in custom svg layout")
	}
}

func TestRenderSVGMacOSChromeUsesSceneOffset(t *testing.T) {
	opts := export.DefaultOptions()
	opts.Format = export.FormatSVG
	opts.BackgroundMode = export.BackgroundCustom
	opts.ChromePreset = export.ChromeOS
	opts.ChromeOSStyle = export.OSStyleMacOS
	snap := term.ScreenSnapshot{Cols: 4, Rows: 1, Lines: []string{"x"}}
	svg, err := export.RenderSVGWithBackground(snap, opts, solidPNG(t, 0, 0, 255))
	if err != nil {
		t.Fatal(err)
	}
	layout := export.ComputeLayout(snap, opts)
	if !bytes.Contains(svg, []byte(`x="`)) {
		t.Fatal("expected positioned chrome elements")
	}
	if layout.ChromeOffsetX == 0 {
		t.Fatal("expected chrome offset")
	}
}
