package export

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	imagedraw "image/draw"
	"image/png"
	"io"

	"github.com/newtosh/tuile/internal/term"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// RenderPNG rasterizes the composed export to PNG bytes.
func RenderPNG(snap term.ScreenSnapshot, opts Options) ([]byte, error) {
	return RenderPNGWithBackground(snap, opts, nil)
}

// RenderPNGWithBackground rasterizes export, optionally decoding a custom backdrop.
func RenderPNGWithBackground(snap term.ScreenSnapshot, opts Options, custom io.Reader) ([]byte, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}
	layout := ComputeLayout(snap, opts)
	img := image.NewRGBA(image.Rect(0, 0, layout.RenderOuterW, layout.RenderOuterH))
	drawBackground(img, layout, opts, custom)
	drawChrome(img, layout, opts)
	if err := drawTerminal(img, snap, layout, opts); err != nil {
		return nil, err
	}
	drawGridLabelOverlay(img, layout, opts)
	if layout.RenderOuterW != layout.OuterW || layout.RenderOuterH != layout.OuterH {
		img = downscaleRGBA(img, layout.OuterW, layout.OuterH)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func drawBackground(img *image.RGBA, layout Layout, opts Options, custom io.Reader) {
	w, h := layout.RenderOuterW, layout.RenderOuterH
	switch opts.BackgroundMode {
	case BackgroundTransparent:
		return
	case BackgroundPreset:
		spec, ok := BackgroundPresets[opts.BackgroundPreset]
		if !ok {
			return
		}
		if spec.Kind == "solid" {
			c := parseColor(spec.Color, false).(color.RGBA)
			fillRect(img, 0, 0, w, h, c)
			return
		}
		start := parseColor(spec.Start, false).(color.RGBA)
		end := parseColor(spec.End, false).(color.RGBA)
		for y := 0; y < h; y++ {
			t := float64(y) / float64(maxInt(h-1, 1))
			c := lerpColor(start, end, t)
			for x := 0; x < w; x++ {
				img.Set(x, y, c)
			}
		}
	case BackgroundCustom:
		if custom == nil {
			return
		}
		bg, _, err := image.Decode(custom)
		if err != nil {
			return
		}
		dst := image.NewRGBA(image.Rect(0, 0, w, h))
		draw.Draw(dst, dst.Bounds(), bg, bg.Bounds().Min, imagedraw.Over)
		draw.Draw(img, img.Bounds(), dst, image.Point{}, imagedraw.Over)
	}
}

func drawChrome(img *image.RGBA, layout Layout, opts Options) {
	if opts.ChromePreset == ChromeOSWireframe {
		drawWireframeChrome(img, layout, opts)
		return
	}
	drawViewerFrame(img, layout)
}

func drawGridLabelOverlay(img *image.RGBA, layout Layout, opts Options) {
	if opts.ChromePreset == ChromeOSWireframe || !opts.ShowGridSize {
		return
	}
	drawGridLabel(img, layout)
}

func drawWireframeChrome(img *image.RGBA, layout Layout, opts Options) {
	frame := color.RGBA{22, 22, 26, 255}
	border := color.RGBA{139, 139, 158, 255}
	w, h := layout.RenderOuterW, layout.RenderOuterH
	fillRect(img, 0, 0, w, h, frame)
	inset := layout.ChromePad
	strokeRect(img, inset/2, inset/2, w-inset, h-inset, border, 2*layout.RenderScale)
	strokeRect(img, inset, inset, w-inset*2, layout.TitleBar, border, 2*layout.RenderScale)
	drawDots(img, layout)
	fontPx := EffectiveFontPx(opts)
	face, err := monoFace(float64(fontPx * layout.RenderScale * 7 / 10))
	if err != nil {
		return
	}
	drawText(img, face, opts.Title, w/2, inset+layout.TitleBar*2/3, color.RGBA{228, 228, 231, 255}, true)
}

func drawViewerFrame(img *image.RGBA, layout Layout) {
	frameBg := color.RGBA{11, 12, 15, 255}
	border := color.RGBA{51, 78, 96, 255}
	fillRoundRect(img, 0, 0, layout.FrameW, layout.FrameH, layout.FrameRadius, frameBg)
	strokeRoundRect(img, 0, 0, layout.FrameW, layout.FrameH, layout.FrameRadius, border, 1)
}

func drawGridLabel(img *image.RGBA, layout Layout) {
	label := formatGridLabel(layout.Cols, layout.Rows)
	fontSize := int(GridLabelFontPx() * float64(layout.RenderScale))
	face, err := monoFace(float64(fontSize))
	if err != nil {
		return
	}
	adv := font.MeasureString(face, label).Round()
	padX := 6 * layout.RenderScale
	padY := 2 * layout.RenderScale
	boxW := adv + padX*2
	boxH := fontSize + padY*2
	anchorX := layout.FrameW
	anchorY := layout.FrameH
	x := anchorX - boxW - 6*layout.RenderScale
	y := anchorY - boxH - 5*layout.RenderScale
	fillRoundRect(img, x, y, boxW, boxH, 4*layout.RenderScale, color.RGBA{22, 22, 28, 224})
	strokeRoundRect(img, x, y, boxW, boxH, 4*layout.RenderScale, color.RGBA{51, 78, 96, 140}, 1)
	drawText(img, face, label, x+padX, y+padY+fontSize*88/100, color.RGBA{142, 200, 224, 255}, false)
}

func formatGridLabel(cols, rows int) string {
	return fmt.Sprintf("%d×%d", cols, rows)
}

func drawDots(img *image.RGBA, layout Layout) {
	dot := 10 * layout.RenderScale
	gap := 8 * layout.RenderScale
	left := layout.ChromePad + 10*layout.RenderScale
	cy := layout.ChromePad + layout.TitleBar/2
	colors := []color.RGBA{
		{255, 95, 87, 255},
		{254, 188, 46, 255},
		{40, 200, 64, 255},
	}
	for i, c := range colors {
		cx := left + i*(dot+gap) + dot/2
		fillCircle(img, cx, cy, dot/2, c)
	}
}

func drawTerminal(img *image.RGBA, snap term.ScreenSnapshot, layout Layout, opts Options) error {
	termBG := color.RGBA{10, 10, 10, 255}
	fillRect(img, layout.TermOffsetX, layout.TermOffsetY, layout.TermW, layout.TermH, termBG)
	face, err := monoFace(float64(EffectiveFontPx(opts) * layout.RenderScale))
	if err != nil {
		return err
	}
	if len(snap.Grid) > 0 {
		for _, row := range snap.Grid {
			y := row.Y
			xOff := layout.TermOffsetX
			yOff := layout.TermOffsetY + y*layout.CellH
			for _, cell := range row.Cells {
				bg := parseColor(cell.Bg, false).(color.RGBA)
				fillRect(img, xOff, yOff, layout.CellW, layout.CellH, bg)
				fg := parseColor(cell.Fg, true).(color.RGBA)
				if cell.Ch != "" && cell.Ch != " " {
					drawText(img, face, cell.Ch, xOff+2, yOff+layout.CellH*4/5, fg, false)
				}
				xOff += layout.CellW
			}
		}
		return nil
	}
	fg := color.RGBA{201, 209, 217, 255}
	for y, line := range snap.Lines {
		drawText(img, face, line, layout.TermOffsetX+4, layout.TermOffsetY+y*layout.CellH+layout.CellH*4/5, fg, false)
	}
	return nil
}

func monoFace(size float64) (font.Face, error) {
	if size < 8 {
		size = 8
	}
	f, err := opentype.Parse(gomono.TTF)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72})
}

func drawText(img *image.RGBA, face font.Face, text string, x, y int, col color.RGBA, center bool) {
	if text == "" {
		return
	}
	d := &font.Drawer{Dst: img, Src: image.NewUniform(col), Face: face}
	if center {
		adv := font.MeasureString(face, text)
		x -= adv.Round() / 2
	}
	d.Dot = fixed.P(x, y)
	d.DrawString(text)
}

func fillRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			if image.Pt(xx, yy).In(img.Bounds()) {
				img.Set(xx, yy, c)
			}
		}
	}
}

func strokeRect(img *image.RGBA, x, y, w, h int, c color.RGBA, thickness int) {
	for t := 0; t < thickness; t++ {
		for xx := x; xx < x+w; xx++ {
			img.Set(xx, y+t, c)
			img.Set(xx, y+h-1-t, c)
		}
		for yy := y; yy < y+h; yy++ {
			img.Set(x+t, yy, c)
			img.Set(x+w-1-t, yy, c)
		}
	}
}

func fillCircle(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if dx*dx+dy*dy <= r*r {
				img.Set(cx+dx, cy+dy, c)
			}
		}
	}
}

func fillRoundRect(img *image.RGBA, x, y, w, h, r int, c color.RGBA) {
	if r > w/2 {
		r = w / 2
	}
	if r > h/2 {
		r = h / 2
	}
	fillRect(img, x+r, y, w-2*r, h, c)
	fillRect(img, x, y+r, w, h-2*r, c)
	fillCircle(img, x+r, y+r, r, c)
	fillCircle(img, x+w-r, y+r, r, c)
	fillCircle(img, x+r, y+h-r, r, c)
	fillCircle(img, x+w-r, y+h-r, r, c)
}

func strokeRoundRect(img *image.RGBA, x, y, w, h, r int, c color.RGBA, thickness int) {
	for t := 0; t < thickness; t++ {
		strokeRect(img, x+t, y+t, w-2*t, h-2*t, c, 1)
	}
}

func downscaleRGBA(src *image.RGBA, width, height int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), imagedraw.Over, nil)
	return dst
}

func lerpColor(a, b color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(a.R)*(1-t) + float64(b.R)*t),
		G: uint8(float64(a.G)*(1-t) + float64(b.G)*t),
		B: uint8(float64(a.B)*(1-t) + float64(b.B)*t),
		A: 255,
	}
}
