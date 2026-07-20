package export_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/newtosh/tuile/internal/export"
	"github.com/newtosh/tuile/internal/term"
)

func TestRenderPNGCustomBackgroundChangesOutput(t *testing.T) {
	snap := term.ScreenSnapshot{
		Cols:  8,
		Rows:  2,
		Lines: []string{"custom", "bg"},
	}
	base := export.DefaultOptions()
	base.ChromePreset = export.ChromeOS
	base.ChromeOSStyle = export.OSStyleWireframe
	base.BackgroundMode = export.BackgroundTransparent

	custom := export.DefaultOptions()
	custom.ChromePreset = export.ChromeOS
	custom.ChromeOSStyle = export.OSStyleWireframe
	custom.BackgroundMode = export.BackgroundCustom

	transparentPNG, err := export.RenderPNG(snap, base)
	if err != nil {
		t.Fatal(err)
	}
	customPNG, err := export.RenderPNGWithBackground(snap, custom, bytes.NewReader(solidPNG(t, 220, 40, 120)))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(transparentPNG, customPNG) {
		t.Fatal("custom background export identical to transparent export")
	}
}

func TestRenderPNGCustomBackgroundRejectsOversizedUpload(t *testing.T) {
	opts := export.DefaultOptions()
	opts.BackgroundMode = export.BackgroundCustom
	snap := term.ScreenSnapshot{Cols: 3, Rows: 1, Lines: []string{"x"}}
	oversized := bytes.Repeat([]byte{0x89}, export.MaxBackgroundBytes+1)
	_, _, err := export.Render(snap, opts, bytes.NewReader(oversized))
	if err == nil {
		t.Fatal("expected oversize custom background error")
	}
}

func TestRenderPNGCustomBackgroundShowsThroughWireframe(t *testing.T) {
	opts := export.DefaultOptions()
	opts.BackgroundMode = export.BackgroundCustom
	opts.ChromePreset = export.ChromeOS
	opts.ChromeOSStyle = export.OSStyleWireframe
	snap := term.ScreenSnapshot{Cols: 6, Rows: 1, Lines: []string{"ok"}}
	pngBytes, err := export.RenderPNGWithBackground(snap, opts, bytes.NewReader(solidPNG(t, 255, 0, 0)))
	if err != nil {
		t.Fatal(err)
	}
	img, err := pngDecodeRGBA(pngBytes)
	if err != nil {
		t.Fatal(err)
	}
	r, _, _, _ := img.RGBAAt(0, 0).RGBA()
	if r>>8 < 200 {
		t.Fatalf("top-left red channel = %d want custom background visible", r>>8)
	}
}

func TestRenderPNGCustomBackgroundShowsThroughMinimalChrome(t *testing.T) {
	opts := export.DefaultOptions()
	opts.BackgroundMode = export.BackgroundCustom
	opts.Scale = 2
	snap := term.ScreenSnapshot{Cols: 6, Rows: 1, Lines: []string{"ok"}}
	pngBytes, err := export.RenderPNGWithBackground(snap, opts, bytes.NewReader(solidPNG(t, 0, 120, 255)))
	if err != nil {
		t.Fatal(err)
	}
	img, err := pngDecodeRGBA(pngBytes)
	if err != nil {
		t.Fatal(err)
	}
	layout := export.ComputeLayout(snap, opts)
	sampleX := layout.FramePad / 2
	sampleY := layout.FramePad / 2
	_, _, b, _ := img.RGBAAt(sampleX, sampleY).RGBA()
	if b>>8 < 200 {
		t.Fatalf("frame padding blue channel = %d want custom background visible at (%d,%d)", b>>8, sampleX, sampleY)
	}
}

func solidPNG(t *testing.T, r, g, b uint8) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	fill := color.RGBA{R: r, G: g, B: b, A: 255}
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.Set(x, y, fill)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
