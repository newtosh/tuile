package export_test

import (
	"bytes"
	"image"
	"image/draw"
	"image/png"
	"testing"

	"github.com/newtosh/tuile/internal/export"
	"github.com/newtosh/tuile/internal/term"
)

func TestOptionsValidateNormalizesLegacyWireframe(t *testing.T) {
	opts := export.DefaultOptions()
	opts.ChromePreset = export.ChromeOSWireframe
	if err := opts.Validate(); err != nil {
		t.Fatal(err)
	}
	if opts.ChromePreset != export.ChromeOS {
		t.Fatalf("chrome preset = %q want %q", opts.ChromePreset, export.ChromeOS)
	}
	if opts.ChromeOSStyle != export.OSStyleWireframe {
		t.Fatalf("chrome os style = %q want %q", opts.ChromeOSStyle, export.OSStyleWireframe)
	}
}

func TestRenderPNGMacOSChrome(t *testing.T) {
	opts := export.DefaultOptions()
	opts.ChromePreset = export.ChromeOS
	opts.ChromeOSStyle = export.OSStyleMacOS
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

func TestRenderPNGMacOSTransparentKeepsOpaqueChrome(t *testing.T) {
	opts := export.DefaultOptions()
	opts.ChromePreset = export.ChromeOS
	opts.ChromeOSStyle = export.OSStyleMacOS
	opts.BackgroundMode = export.BackgroundTransparent
	snap := term.ScreenSnapshot{
		Cols:  5,
		Rows:  1,
		Lines: []string{"ok"},
	}
	png, err := export.RenderPNG(snap, opts)
	if err != nil {
		t.Fatal(err)
	}
	img, err := pngDecodeRGBA(png)
	if err != nil {
		t.Fatal(err)
	}
	// Title bar should be opaque even when the export backdrop is transparent.
	if a := img.RGBAAt(20, 10).A; a == 0 {
		t.Fatalf("title bar alpha = 0 want opaque chrome")
	}
	// Padding band below the title bar should also be opaque.
	layout := export.ComputeLayout(snap, opts)
	down := layout.Downscale
	if down < 1 {
		down = 1
	}
	titleX := 16 / down
	titleY := 10 / down
	if a := img.RGBAAt(titleX, titleY).A; a == 0 {
		t.Fatalf("title bar alpha = 0 want opaque chrome at (%d,%d)", titleX, titleY)
	}
}

func pngDecodeRGBA(data []byte) (*image.RGBA, error) {
	decoded, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	bounds := decoded.Bounds()
	out := image.NewRGBA(bounds)
	draw.Draw(out, bounds, decoded, bounds.Min, draw.Src)
	return out, nil
}

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
