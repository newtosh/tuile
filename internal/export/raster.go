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
	if err := drawBackground(img, layout, opts, custom); err != nil {
		return nil, err
	}
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

func drawBackground(img *image.RGBA, layout Layout, opts Options, custom io.Reader) error {
	w, h := layout.RenderOuterW, layout.RenderOuterH
	switch opts.BackgroundMode {
	case BackgroundTransparent:
		return nil
	case BackgroundPreset:
		spec, ok := BackgroundPresets[opts.BackgroundPreset]
		if !ok {
			return nil
		}
		if spec.Kind == "solid" {
			c := parseColor(spec.Color, false).(color.RGBA)
			fillRect(img, 0, 0, w, h, c)
			return nil
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
		return nil
	case BackgroundCustom:
		if custom == nil {
			return fmt.Errorf("background_image required")
		}
		bg, _, err := image.Decode(custom)
		if err != nil {
			return fmt.Errorf("decode background image: %w", err)
		}
		scaled := image.NewRGBA(image.Rect(0, 0, w, h))
		draw.ApproxBiLinear.Scale(scaled, scaled.Bounds(), bg, bg.Bounds(), draw.Over, nil)
		imagedraw.Draw(img, img.Bounds(), scaled, image.Point{}, imagedraw.Src)
		return nil
	default:
		return nil
	}
}

func drawChrome(img *image.RGBA, layout Layout, opts Options) {
	if opts.IsOSChrome() {
		switch opts.ResolvedOSStyle() {
		case OSStyleMacOS:
			drawMacOSChrome(img, layout, opts)
		case OSStyleWindows:
			drawWindowsChrome(img, layout, opts)
		default:
			drawWireframeChrome(img, layout, opts)
		}
		return
	}
	drawViewerFrame(img, layout, opts)
}

func chromeRect(layout Layout) (x, y, w, h int) {
	if layout.ScenePad > 0 {
		return layout.ChromeOffsetX, layout.ChromeOffsetY, layout.ChromeW, layout.ChromeH
	}
	return 0, 0, layout.RenderOuterW, layout.RenderOuterH
}

func drawGridLabelOverlay(img *image.RGBA, layout Layout, opts Options) {
	if opts.IsOSChrome() || !opts.ShowGridSize {
		return
	}
	drawGridLabel(img, layout, opts)
}

func drawWireframeChrome(img *image.RGBA, layout Layout, opts Options) {
	border := color.RGBA{139, 139, 158, 255}
	ox, oy, w, h := chromeRect(layout)
	if opts.BackgroundMode != BackgroundCustom {
		frame := color.RGBA{22, 22, 26, 255}
		fillRect(img, ox, oy, w, h, frame)
	}
	inset := layout.ChromePad
	strokeRect(img, ox+inset/2, oy+inset/2, w-inset, h-inset, border, 2*layout.RenderScale)
	strokeRect(img, ox+inset, oy+inset, w-inset*2, layout.TitleBar, border, 2*layout.RenderScale)
	drawDots(img, layout, ox, oy)
	fontPx := EffectiveFontPx(opts)
	face, err := monoFace(float64(fontPx * layout.RenderScale * 7 / 10))
	if err != nil {
		return
	}
	drawText(img, face, opts.Title, ox+w/2, oy+inset+layout.TitleBar*2/3, color.RGBA{228, 228, 231, 255}, true)
}

func drawMacOSChrome(img *image.RGBA, layout Layout, opts Options) {
	ox, oy, w, h := chromeRect(layout)
	radius := layout.WindowRadius
	titleBar := layout.TitleBar
	windowBg := MacOSWindowBg(opts)
	light := opts.Theme == "light"
	border := color.RGBA{0, 0, 0, 89}
	titleColor := color.RGBA{245, 245, 247, 183}
	if light {
		border = color.RGBA{0, 0, 0, 31}
		titleColor = color.RGBA{60, 60, 67, 183}
	}

	fillRoundRect(img, ox, oy, w, h, radius, windowBg)
	drawMacOSTrafficLights(img, layout, ox, oy)
	fontPx := EffectiveFontPx(opts)
	face, err := monoFace(float64(fontPx * layout.RenderScale * 13 / 20))
	if err == nil {
		drawText(img, face, opts.Title, ox+w/2, oy+int(float64(titleBar)*0.62), titleColor, true)
	}
	strokeRoundRect(img, ox, oy, w, h, radius, border, 1)
}

func drawMacOSTrafficLights(img *image.RGBA, layout Layout, ox, oy int) {
	dot := MacOSTrafficLightSize() * layout.RenderScale
	gap := MacOSTrafficLightGap() * layout.RenderScale
	left := ox + MacOSTrafficLightInset()*layout.RenderScale
	top := oy + MacOSTrafficLightInset()*layout.RenderScale
	cy := top + dot/2
	ring := color.RGBA{0, 0, 0, 26}
	colors := []color.RGBA{
		{249, 96, 87, 255},
		{248, 206, 82, 255},
		{95, 207, 101, 255},
	}
	for i, c := range colors {
		cx := left + i*(dot+gap) + dot/2
		fillCircle(img, cx, cy, dot/2, c)
		strokeCircle(img, cx, cy, dot/2, ring)
	}
}

func drawWindowsChrome(img *image.RGBA, layout Layout, opts Options) {
	ox, oy, w, h := chromeRect(layout)
	radius := layout.WindowRadius
	windowBg := WindowsWindowBg(opts)
	light := opts.Theme == "light"
	border := color.RGBA{255, 255, 255, 15}
	captionColor := color.RGBA{255, 255, 255, 230}
	if light {
		border = color.RGBA{0, 0, 0, 31}
		captionColor = color.RGBA{0, 0, 0, 230}
	}

	fillRoundRect(img, ox, oy, w, h, radius, windowBg)
	drawWindowsTabRow(img, layout, opts, ox, oy)
	drawWindowsCaptionButtons(img, layout, captionColor, ox, oy)
	strokeRoundRect(img, ox, oy, w, h, radius, border, 1)
}

func drawWindowsTabRow(img *image.RGBA, layout Layout, opts Options, ox, oy int) {
	titleBar := layout.TitleBar
	light := opts.Theme == "light"
	tabText := color.RGBA{204, 204, 204, 255}
	captionColor := color.RGBA{255, 255, 255, 230}
	if light {
		tabText = color.RGBA{26, 26, 26, 255}
		captionColor = color.RGBA{0, 0, 0, 230}
	}
	scale := layout.RenderScale
	tabX := ox + WindowsTabRowMarginX()*scale
	tabY := oy + WindowsTabRowMarginTop()*scale
	tabPad := WindowsTabPaddingX() * scale
	tabW := WindowsTabWidth() * scale
	tabH := titleBar - WindowsTabRowMarginTop()*scale
	tabR := WindowsTabTopRadius() * scale
	iconSize := WindowsTabIconSize() * scale
	iconGap := WindowsTabIconGap() * scale
	fontSize := 12 * scale
	appName := WindowsAppName()
	tabRowBg := WindowsTabRowBg(opts)
	tabActiveBg := WindowsWindowBg(opts)
	tabAccent := WindowsTabActiveTopAccent(opts)
	fillRect(img, ox, oy, layout.ChromeW, titleBar, tabRowBg)
	fillRoundRectTopOnly(img, tabX, tabY, tabW, tabH, tabR, tabActiveBg)
	fillRect(img, tabX+tabR, tabY, tabW-2*tabR, maxInt(1, scale), tabAccent)
	iconX := tabX + tabPad
	iconY := tabY + (tabH-iconSize)/2
	drawTuileFavicon(img, iconX, iconY, iconSize)
	face, err := monoFace(float64(fontSize))
	if err == nil {
		drawText(img, face, appName, iconX+iconSize+iconGap, tabY+tabH/2+fontSize*2/5, tabText, false)
	}
	drawWindowsTabCloseButton(img, tabX, tabY, tabW, tabH, captionColor, scale)
	controlsCy := tabY + tabH/2
	controlsX := tabX + tabW
	drawWindowsNewTabButton(img, controlsX, controlsCy, captionColor, scale)
	drawWindowsTabMenuChevron(img, controlsX+WindowsNewTabButtonWidth()*scale, controlsCy, captionColor, scale)
}

func drawWindowsTabCloseButton(img *image.RGBA, tabX, tabY, tabW, tabH int, icon color.RGBA, scale int) {
	closeW := WindowsTabCloseButtonWidth() * scale
	pad := WindowsTabPaddingX() * scale
	cx := tabX + tabW - pad - closeW/2
	cy := tabY + tabH/2
	iconSize := int(3.5*float64(scale) + 0.5)
	thickness := scale
	if thickness < 1 {
		thickness = 1
	}
	strokeLine(img, cx-iconSize, cy-iconSize, cx+iconSize, cy+iconSize, icon, thickness)
	strokeLine(img, cx+iconSize, cy-iconSize, cx-iconSize, cy+iconSize, icon, thickness)
}

func drawWindowsTabMenuChevron(img *image.RGBA, x, cy int, icon color.RGBA, scale int) {
	btnW := WindowsTabMenuButtonWidth() * scale
	cx := x + btnW/2
	half := int(3.5*float64(scale) + 0.5)
	thickness := scale
	if thickness < 1 {
		thickness = 1
	}
	strokeLine(img, cx-half, cy-int(float64(half)*0.35), cx, cy+int(float64(half)*0.65), icon, thickness)
	strokeLine(img, cx, cy+int(float64(half)*0.65), cx+half, cy-int(float64(half)*0.35), icon, thickness)
}

func drawTuileFavicon(img *image.RGBA, x, y, size int) {
	u := float64(size) / 32.0
	bg := color.RGBA{12, 12, 14, 255}
	tile := color.RGBA{232, 165, 75, 255}
	fillRoundRect(img, x, y, size, size, int(6*u+0.5), bg)
	squares := []struct {
		x, y int
		o    uint8
	}{
		{6, 6, 255},
		{17, 6, 209},
		{6, 17, 209},
		{17, 17, 255},
	}
	for _, sq := range squares {
		c := tile
		c.A = sq.o
		sx := x + int(float64(sq.x)*u+0.5)
		sy := y + int(float64(sq.y)*u+0.5)
		sw := int(9*u + 0.5)
		fillRoundRect(img, sx, sy, sw, sw, int(1.5*u+0.5), c)
	}
}

func drawWindowsNewTabButton(img *image.RGBA, x, cy int, icon color.RGBA, scale int) {
	btnW := WindowsNewTabButtonWidth() * scale
	cx := x + btnW/2
	iconSize := 5 * scale
	thickness := scale
	if thickness < 1 {
		thickness = 1
	}
	strokeHLine(img, cx-iconSize, cx+iconSize, cy, icon, thickness)
	strokeVLine(img, cx, cy-iconSize, cy+iconSize, icon, thickness)
}

func strokeVLine(img *image.RGBA, x, y1, y2 int, c color.RGBA, thickness int) {
	for t := 0; t < thickness; t++ {
		for y := y1; y <= y2; y++ {
			img.Set(x+t, y, c)
		}
	}
}

func drawWindowsCaptionButtons(img *image.RGBA, layout Layout, icon color.RGBA, ox, oy int) {
	btnW := WindowsCaptionButtonWidth() * layout.RenderScale
	titleBar := layout.TitleBar
	w := layout.ChromeW
	if layout.ScenePad == 0 {
		w = layout.RenderOuterW
	}
	iconSize := 4 * layout.RenderScale
	thickness := layout.RenderScale
	if thickness < 1 {
		thickness = 1
	}
	kinds := []string{"minimize", "maximize", "close"}
	for i, kind := range kinds {
		x := ox + w - (len(kinds)-i)*btnW
		cx := x + btnW/2
		cy := oy + titleBar/2
		switch kind {
		case "minimize":
			strokeHLine(img, cx-iconSize, cx+iconSize, cy, icon, thickness)
		case "maximize":
			strokeRect(img, cx-iconSize, cy-iconSize, iconSize*2, iconSize*2, icon, thickness)
		case "close":
			strokeLine(img, cx-iconSize, cy-iconSize, cx+iconSize, cy+iconSize, icon, thickness)
			strokeLine(img, cx+iconSize, cy-iconSize, cx-iconSize, cy+iconSize, icon, thickness)
		}
	}
}

func strokeHLine(img *image.RGBA, x1, x2, y int, c color.RGBA, thickness int) {
	for t := 0; t < thickness; t++ {
		for x := x1; x <= x2; x++ {
			img.Set(x, y+t, c)
		}
	}
}

func strokeLine(img *image.RGBA, x1, y1, x2, y2 int, c color.RGBA, thickness int) {
	dx := absInt(x2 - x1)
	dy := absInt(y2 - y1)
	sx := -1
	if x1 < x2 {
		sx = 1
	}
	sy := -1
	if y1 < y2 {
		sy = 1
	}
	err := dx - dy
	for {
		for t := 0; t < thickness; t++ {
			img.Set(x1, y1+t, c)
		}
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := err * 2
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func drawViewerFrame(img *image.RGBA, layout Layout, opts Options) {
	accent := ThemeChromeAccentFor(opts)
	frameBg := parseColor(accent.FrameBg, false).(color.RGBA)
	border := parseColor(accent.Border, false).(color.RGBA)
	ox, oy, w, h := chromeRect(layout)
	if layout.FrameW > 0 && layout.ScenePad == 0 {
		w = layout.FrameW
		h = layout.FrameH
	}
	if opts.BackgroundMode != BackgroundCustom {
		termBG := color.RGBA{10, 10, 10, 255}
		fillRoundRectEars(img, ox, oy, w, h, layout.FrameRadius, termBG)
		fillRoundRect(img, ox, oy, w, h, layout.FrameRadius, frameBg)
	}
	strokeRoundRect(img, ox, oy, w, h, layout.FrameRadius, border, 1)
}

func drawGridLabel(img *image.RGBA, layout Layout, opts Options) {
	accent := ThemeChromeAccentFor(opts)
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
	ox, oy, cw, ch := chromeRect(layout)
	frameW := layout.FrameW
	frameH := layout.FrameH
	if frameW > 0 && layout.ScenePad == 0 {
		cw = frameW
		ch = frameH
	}
	anchorX := ox + cw
	anchorY := oy + ch
	x := anchorX - boxW - 6*layout.RenderScale
	y := anchorY - boxH - 5*layout.RenderScale
	labelBg := parseColor(accent.LabelBg, false).(color.RGBA)
	labelBorder := parseColor(accent.LabelBorder, false).(color.RGBA)
	labelText := parseColor(accent.LabelText, true).(color.RGBA)
	fillRoundRect(img, x, y, boxW, boxH, 4*layout.RenderScale, labelBg)
	strokeRoundRect(img, x, y, boxW, boxH, 4*layout.RenderScale, labelBorder, 1)
	drawText(img, face, label, x+padX, y+padY+fontSize*88/100, labelText, false)
}

func formatGridLabel(cols, rows int) string {
	return fmt.Sprintf("%d×%d", cols, rows)
}

func drawDots(img *image.RGBA, layout Layout, ox, oy int) {
	dot := 10 * layout.RenderScale
	gap := 8 * layout.RenderScale
	left := ox + layout.ChromePad + 10*layout.RenderScale
	cy := oy + layout.ChromePad + layout.TitleBar/2
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
					drawText(img, face, cell.Ch, xOff+cellTextX(layout.CellW, face, cell.Ch), yOff+cellTextY(layout.CellH), fg, false)
				}
				xOff += layout.CellW
			}
		}
		return nil
	}
	fg := color.RGBA{201, 209, 217, 255}
	for y, line := range snap.Lines {
		drawText(img, face, line, layout.TermOffsetX+2, layout.TermOffsetY+y*layout.CellH+cellTextY(layout.CellH), fg, false)
	}
	return nil
}

func cellTextX(cellW int, face font.Face, ch string) int {
	if ch == "" || ch == " " {
		return 0
	}
	adv := font.MeasureString(face, ch).Round()
	pad := (cellW - adv) / 2
	if pad < 0 {
		return 0
	}
	if pad > 2 {
		return 2
	}
	return pad
}

func cellTextY(cellH int) int {
	return cellH * 4 / 5
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

func strokeCircle(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	outer := r
	inner := r - 1
	if inner < 0 {
		inner = 0
	}
	for dy := -outer; dy <= outer; dy++ {
		for dx := -outer; dx <= outer; dx++ {
			d2 := dx*dx + dy*dy
			if d2 <= outer*outer && d2 >= inner*inner {
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

func fillRoundRectTopOnly(img *image.RGBA, x, y, w, h, r int, c color.RGBA) {
	if r > w/2 {
		r = w / 2
	}
	if r > h {
		r = h
	}
	fillRect(img, x, y+r, w, h-r, c)
	fillRect(img, x+r, y, w-2*r, r, c)
	fillCircle(img, x+r, y+r, r, c)
	fillCircle(img, x+w-r, y+r, r, c)
}

func fillRoundRectEars(img *image.RGBA, x, y, w, h, r int, c color.RGBA) {
	for py := y; py < y+h; py++ {
		for px := x; px < x+w; px++ {
			if !insideRoundRect(px-x, py-y, w, h, r) {
				if image.Pt(px, py).In(img.Bounds()) {
					img.Set(px, py, c)
				}
			}
		}
	}
}

func insideRoundRect(lx, ly, w, h, r int) bool {
	if lx < 0 || ly < 0 || lx >= w || ly >= h {
		return false
	}
	if r > w/2 {
		r = w / 2
	}
	if r > h/2 {
		r = h / 2
	}
	if lx >= r && lx < w-r {
		return true
	}
	if ly >= r && ly < h-r {
		return true
	}
	var cx, cy float64
	switch {
	case lx < r && ly < r:
		cx, cy = float64(r), float64(r)
	case lx >= w-r && ly < r:
		cx, cy = float64(w-r), float64(r)
	case lx < r && ly >= h-r:
		cx, cy = float64(r), float64(h-r)
	default:
		cx, cy = float64(w-r), float64(h-r)
	}
	dx := float64(lx) - cx
	dy := float64(ly) - cy
	return dx*dx+dy*dy <= float64(r*r)
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
