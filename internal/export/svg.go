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
	if opts.IsOSChrome() {
		switch opts.ResolvedOSStyle() {
		case OSStyleMacOS:
			writeSVGMacOSChrome(buf, layout, opts)
		case OSStyleWindows:
			writeSVGWindowsChrome(buf, layout, opts)
		default:
			writeSVGWireframeChrome(buf, layout, opts)
		}
		return
	}
	accent := ThemeChromeAccentFor(opts)
	if opts.BackgroundMode == BackgroundPreset {
		fmt.Fprintf(buf, `<rect x="0" y="0" width="%d" height="%d" fill="#0a0a0a"/>`, layout.FrameW, layout.FrameH)
		fmt.Fprintf(buf, `<rect x="0" y="0" width="%d" height="%d" rx="%d" fill="%s" stroke="%s" stroke-width="1"/>`, layout.FrameW, layout.FrameH, layout.FrameRadius, html.EscapeString(accent.FrameBg), html.EscapeString(accent.Border))
		return
	}
	fmt.Fprintf(buf, `<rect x="0" y="0" width="%d" height="%d" rx="%d" fill="none" stroke="%s" stroke-width="1"/>`, layout.FrameW, layout.FrameH, layout.FrameRadius, html.EscapeString(accent.Border))
}

func writeSVGGridLabel(buf *bytes.Buffer, layout Layout, opts Options) {
	if opts.IsOSChrome() || !opts.ShowGridSize {
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
	fill := "none"
	if opts.BackgroundMode == BackgroundPreset {
		fill = "#16161a"
	}
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

func writeSVGMacOSChrome(buf *bytes.Buffer, layout Layout, opts Options) {
	light := opts.Theme == "light"
	windowBg := "#0a0a0a"
	border := "rgba(0,0,0,0.35)"
	titleColor := "rgba(245,245,247,0.72)"
	if light {
		windowBg = "#ffffff"
		border = "rgba(0,0,0,0.12)"
		titleColor = "rgba(60,60,67,0.72)"
	}
	radius := layout.WindowRadius
	titleBar := layout.TitleBar
	dot := MacOSTrafficLightSize() * layout.RenderScale
	gap := MacOSTrafficLightGap() * layout.RenderScale
	left := MacOSTrafficLightInset() * layout.RenderScale
	top := MacOSTrafficLightInset() * layout.RenderScale
	cy := top + dot/2
	fmt.Fprintf(buf, `<rect x="0" y="0" width="%d" height="%d" rx="%d" fill="%s" stroke="%s" stroke-width="0.5"/>`, layout.RenderOuterW, layout.RenderOuterH, radius, windowBg, border)
	colors := []string{"#F96057", "#F8CE52", "#5FCF65"}
	for i, col := range colors {
		dx := left + i*(dot+gap)
		fmt.Fprintf(buf, `<circle cx="%d" cy="%d" r="%d" fill="%s" stroke="rgba(0,0,0,0.1)" stroke-width="0.5"/>`, dx+dot/2, cy, dot/2, col)
	}
	fontSize := 13 * layout.RenderScale
	fmt.Fprintf(buf, `<text x="%d" y="%d" text-anchor="middle" fill="%s" font-family="-apple-system,BlinkMacSystemFont,&quot;SF Pro Text&quot;,system-ui,sans-serif" font-size="%d" font-weight="500">%s</text>`, layout.RenderOuterW/2, int(float64(titleBar)*0.62), titleColor, fontSize, html.EscapeString(opts.Title))
}

func writeSVGWindowsChrome(buf *bytes.Buffer, layout Layout, opts Options) {
	light := opts.Theme == "light"
	windowBg := "#0C0C0C"
	tabRowBg := "#333333"
	tabAccent := "rgba(255,255,255,0.14)"
	border := "rgba(255,255,255,0.06)"
	tabText := "#CCCCCC"
	captionColor := "rgba(255,255,255,0.9)"
	if light {
		windowBg = "#F3F3F3"
		tabRowBg = "#ECECEC"
		tabAccent = "rgba(0,0,0,0.12)"
		border = "rgba(0,0,0,0.12)"
		tabText = "#1A1A1A"
		captionColor = "rgba(0,0,0,0.9)"
	}
	radius := layout.WindowRadius
	titleBar := layout.TitleBar
	btnW := WindowsCaptionButtonWidth() * layout.RenderScale
	icon := 4 * layout.RenderScale
	thickness := layout.RenderScale
	if thickness < 1 {
		thickness = 1
	}
	fmt.Fprintf(buf, `<rect x="0" y="0" width="%d" height="%d" rx="%d" fill="%s" stroke="%s" stroke-width="0.5"/>`, layout.RenderOuterW, layout.RenderOuterH, radius, windowBg, border)
	scale := layout.RenderScale
	tabX := WindowsTabRowMarginX() * scale
	tabY := WindowsTabRowMarginTop() * scale
	tabPad := WindowsTabPaddingX() * scale
	tabW := WindowsTabWidth() * scale
	tabH := titleBar - WindowsTabRowMarginTop()*scale
	tabR := WindowsTabTopRadius() * scale
	iconSize := WindowsTabIconSize() * scale
	iconGap := WindowsTabIconGap() * scale
	fontSize := 12 * scale
	appName := WindowsAppName()
	iconX := tabX + tabPad
	iconY := tabY + (tabH-iconSize)/2
	textX := iconX + iconSize + iconGap
	closeCx := tabX + tabW - tabPad - WindowsTabCloseButtonWidth()*scale/2
	closeIcon := int(3.5*float64(scale) + 0.5)
	controlsX := tabX + tabW
	controlsCy := tabY + tabH/2
	newTabCx := controlsX + WindowsNewTabButtonWidth()*scale/2
	menuCx := controlsX + WindowsNewTabButtonWidth()*scale + WindowsTabMenuButtonWidth()*scale/2
	plus := 5 * scale
	chev := int(3.5*float64(scale) + 0.5)
	fmt.Fprintf(buf, `<rect x="0" y="0" width="%d" height="%d" fill="%s"/>`, layout.RenderOuterW, titleBar, tabRowBg)
	fmt.Fprintf(buf, `<path d="M%d %d L%d %d Q%d %d %d %d L%d %d Q%d %d %d %d L%d %d Z" fill="%s"/>`, tabX, tabY+tabH, tabX, tabY+tabR, tabX, tabY, tabX+tabR, tabY, tabX+tabW-tabR, tabY, tabX+tabW, tabY, tabX+tabW, tabY+tabR, tabX+tabW, tabY+tabH, windowBg)
	fmt.Fprintf(buf, `<rect x="%d" y="%d" width="%d" height="%d" fill="%s"/>`, tabX, tabY, tabW, maxInt(1, scale), tabAccent)
	fmt.Fprintf(buf, `<g transform="translate(%d,%d) scale(%g)"><rect width="32" height="32" rx="6" fill="#0c0c0e"/><rect x="6" y="6" width="9" height="9" rx="1.5" fill="#e8a54b"/><rect x="17" y="6" width="9" height="9" rx="1.5" fill="#e8a54b" opacity="0.82"/><rect x="6" y="17" width="9" height="9" rx="1.5" fill="#e8a54b" opacity="0.82"/><rect x="17" y="17" width="9" height="9" rx="1.5" fill="#e8a54b"/></g>`, iconX, iconY, float64(iconSize)/32)
	fmt.Fprintf(buf, `<text x="%d" y="%d" dominant-baseline="middle" fill="%s" font-family="Segoe UI Variable,Segoe UI,system-ui,sans-serif" font-size="%d" font-weight="400">%s</text>`, textX, controlsCy, tabText, fontSize, html.EscapeString(appName))
	fmt.Fprintf(buf, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="%d" stroke-linecap="round"/>`, closeCx-closeIcon, controlsCy-closeIcon, closeCx+closeIcon, controlsCy+closeIcon, captionColor, thickness)
	fmt.Fprintf(buf, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="%d" stroke-linecap="round"/>`, closeCx+closeIcon, controlsCy-closeIcon, closeCx-closeIcon, controlsCy+closeIcon, captionColor, thickness)
	fmt.Fprintf(buf, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="%d" stroke-linecap="round"/>`, newTabCx-plus, controlsCy, newTabCx+plus, controlsCy, captionColor, thickness)
	fmt.Fprintf(buf, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="%d" stroke-linecap="round"/>`, newTabCx, controlsCy-plus, newTabCx, controlsCy+plus, captionColor, thickness)
	fmt.Fprintf(buf, `<polyline points="%d,%d %d,%d %d,%d" fill="none" stroke="%s" stroke-width="%d" stroke-linecap="round" stroke-linejoin="round"/>`, menuCx-chev, controlsCy-chev*35/100, menuCx, controlsCy+chev*65/100, menuCx+chev, controlsCy-chev*35/100, captionColor, thickness)
	kinds := []string{"minimize", "maximize", "close"}
	for i, kind := range kinds {
		x := layout.RenderOuterW - (len(kinds)-i)*btnW
		cx := x + btnW/2
		cy := titleBar / 2
		switch kind {
		case "minimize":
			fmt.Fprintf(buf, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="%d"/>`, cx-icon, cy, cx+icon, cy, captionColor, thickness)
		case "maximize":
			fmt.Fprintf(buf, `<rect x="%d" y="%d" width="%d" height="%d" fill="none" stroke="%s" stroke-width="%d"/>`, cx-icon, cy-icon, icon*2, icon*2, captionColor, thickness)
		case "close":
			fmt.Fprintf(buf, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="%d"/>`, cx-icon, cy-icon, cx+icon, cy+icon, captionColor, thickness)
			fmt.Fprintf(buf, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="%d"/>`, cx+icon, cy-icon, cx-icon, cy+icon, captionColor, thickness)
		}
	}
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
