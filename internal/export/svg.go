package export

import (
	"bytes"
	"fmt"
	"html"
	"strings"

	"github.com/newtosh/tuile/internal/term"
)

// RenderSVG returns an SVG document for the composed export.
func RenderSVG(snap term.ScreenSnapshot, opts Options) ([]byte, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}
	layout := ComputeLayout(snap, opts)
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	fmt.Fprintf(&buf, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`, layout.OuterW, layout.OuterH, layout.RenderOuterW, layout.RenderOuterH)
	writeSVGBackground(&buf, layout, opts)
	writeSVGChrome(&buf, layout, opts)
	writeSVGTerminal(&buf, snap, layout, opts)
	writeSVGGridLabel(&buf, layout, opts)
	buf.WriteString(`</svg>`)
	return buf.Bytes(), nil
}

func writeSVGBackground(buf *bytes.Buffer, layout Layout, opts Options) {
	switch opts.BackgroundMode {
	case BackgroundTransparent:
		return
	case BackgroundPreset:
		spec, ok := BackgroundPresets[opts.BackgroundPreset]
		if !ok {
			return
		}
		if spec.Kind == "solid" {
			fmt.Fprintf(buf, `<rect width="%d" height="%d" fill="%s"/>`, layout.RenderOuterW, layout.RenderOuterH, html.EscapeString(spec.Color))
			return
		}
		id := "bg-grad"
		fmt.Fprintf(buf, `<defs><linearGradient id="%s" x1="0%%" y1="0%%" x2="100%%" y2="100%%"><stop offset="0%%" stop-color="%s"/><stop offset="100%%" stop-color="%s"/></linearGradient></defs>`, id, html.EscapeString(spec.Start), html.EscapeString(spec.End))
		fmt.Fprintf(buf, `<rect width="%d" height="%d" fill="url(#%s)"/>`, layout.RenderOuterW, layout.RenderOuterH, id)
	}
}

func writeSVGChrome(buf *bytes.Buffer, layout Layout, opts Options) {
	if opts.ChromePreset == ChromeOSWireframe {
		writeSVGWireframeChrome(buf, layout, opts)
		return
	}
	accent := ThemeChromeAccentFor(opts)
	fmt.Fprintf(buf, `<rect x="0" y="0" width="%d" height="%d" fill="#0a0a0a"/>`, layout.FrameW, layout.FrameH)
	fmt.Fprintf(buf, `<rect x="0" y="0" width="%d" height="%d" rx="%d" fill="%s" stroke="%s" stroke-width="1"/>`, layout.FrameW, layout.FrameH, layout.FrameRadius, html.EscapeString(accent.FrameBg), html.EscapeString(accent.Border))
}

func writeSVGGridLabel(buf *bytes.Buffer, layout Layout, opts Options) {
	if opts.ChromePreset == ChromeOSWireframe || !opts.ShowGridSize {
		return
	}
	accent := ThemeChromeAccentFor(opts)
	label := formatGridLabel(layout.Cols, layout.Rows)
	fontSize := int(GridLabelFontPx() * float64(layout.RenderScale))
	padX := 6 * layout.RenderScale
	padY := 2 * layout.RenderScale
	boxW := len(label)*fontSize*6/10 + padX*2
	boxH := fontSize + padY*2
	anchorX := layout.FrameW
	anchorY := layout.FrameH
	lx := anchorX - boxW - 6*layout.RenderScale
	ly := anchorY - boxH - 5*layout.RenderScale
	fmt.Fprintf(buf, `<rect x="%d" y="%d" width="%d" height="%d" rx="%d" fill="%s" stroke="%s" stroke-width="1"/>`, lx, ly, boxW, boxH, 4*layout.RenderScale, html.EscapeString(accent.LabelBg), html.EscapeString(accent.LabelBorder))
	fmt.Fprintf(buf, `<text x="%d" y="%d" fill="%s" font-family="JetBrains Mono, ui-monospace, monospace" font-size="%d" font-weight="500">%s</text>`, lx+padX, ly+padY+fontSize*88/100, html.EscapeString(accent.LabelText), fontSize, html.EscapeString(label))
}

func writeSVGWireframeChrome(buf *bytes.Buffer, layout Layout, opts Options) {
	stroke := "#8b8b9e"
	fill := "#16161a"
	fmt.Fprintf(buf, `<rect width="%d" height="%d" fill="%s" stroke="%s" stroke-width="%d" stroke-dasharray="5 4"/>`, layout.RenderOuterW, layout.RenderOuterH, fill, stroke, 2*layout.RenderScale)
	fmt.Fprintf(buf, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="%d" stroke-dasharray="5 4"/>`, layout.ChromePad, layout.ChromePad+layout.TitleBar, layout.RenderOuterW-layout.ChromePad, layout.ChromePad+layout.TitleBar, stroke, 2*layout.RenderScale)
	dot := 10 * layout.RenderScale
	gap := 8 * layout.RenderScale
	left := layout.ChromePad + 10*layout.RenderScale
	cy := layout.ChromePad + layout.TitleBar/2
	colors := []string{"#ff5f57", "#febc2e", "#28c840"}
	for i, col := range colors {
		dx := left + i*(dot+gap)
		fmt.Fprintf(buf, `<circle cx="%d" cy="%d" r="%d" fill="%s"/>`, dx+dot/2, cy, dot/2, col)
	}
	fmt.Fprintf(buf, `<text x="%d" y="%d" text-anchor="middle" fill="#e4e4e7" font-family="system-ui,sans-serif" font-size="%d" font-weight="600">%s</text>`, layout.RenderOuterW/2, layout.ChromePad+layout.TitleBar*2/3, 12*layout.RenderScale, html.EscapeString(opts.Title))
}

func writeSVGTerminal(buf *bytes.Buffer, snap term.ScreenSnapshot, layout Layout, opts Options) {
	fontSize := EffectiveFontPx(opts) * layout.RenderScale
	fmt.Fprintf(buf, `<g transform="translate(%d,%d)">`, layout.TermOffsetX, layout.TermOffsetY)
	fmt.Fprintf(buf, `<rect width="%d" height="%d" fill="#0a0a0a"/>`, layout.TermW, layout.TermH)
	if len(snap.Grid) > 0 {
		for _, row := range snap.Grid {
			y := row.Y
			if y < 0 || y >= len(snap.Lines) {
				continue
			}
			xOff := 0
			for _, cell := range row.Cells {
				bg := parseColor(cell.Bg, false)
				fg := parseColor(cell.Fg, true)
				r, g, b, _ := bg.RGBA()
				fmt.Fprintf(buf, `<rect x="%d" y="%d" width="%d" height="%d" fill="rgb(%d,%d,%d)"/>`, xOff, y*layout.CellH, layout.CellW, layout.CellH, r>>8, g>>8, b>>8)
				fr, fgC, fb, _ := fg.RGBA()
				ch := html.EscapeString(cell.Ch)
				if ch == "" || ch == " " {
					xOff += layout.CellW
					continue
				}
				fmt.Fprintf(buf, `<text x="%d" y="%d" text-anchor="middle" fill="rgb(%d,%d,%d)" font-family="monospace" font-size="%d">%s</text>`, xOff+layout.CellW/2, y*layout.CellH+fontSize, fr>>8, fgC>>8, fb>>8, fontSize, ch)
				xOff += layout.CellW
			}
		}
	} else {
		for y, line := range snap.Lines {
			if line == "" {
				continue
			}
			fmt.Fprintf(buf, `<text x="%d" y="%d" fill="#e4e4e4" font-family="monospace" font-size="%d">%s</text>`, 4, y*layout.CellH+fontSize, fontSize, html.EscapeString(line))
		}
	}
	buf.WriteString(`</g>`)
}

func escapeSVGText(s string) string {
	return strings.TrimSpace(html.EscapeString(s))
}
