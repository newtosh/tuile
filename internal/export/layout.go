package export

import (
	"github.com/newtosh/tuile/internal/term"
)

const (
	MinExportFontPx      = 14
	CompactSuperSample   = 2
)

// Layout describes export canvas dimensions in pixels.
type Layout struct {
	ExportScale   int
	RenderScale   int
	Downscale     int
	CellW         int
	CellH         int
	TermW         int
	TermH         int
	ChromePad     int
	TitleBar      int
	InnerGap      int
	FramePad      int
	FrameRadius   int
	FrameW        int
	FrameH        int
	RenderOuterW  int
	RenderOuterH  int
	OuterW        int
	OuterH        int
	TermOffsetX   int
	TermOffsetY   int
	Cols          int
	Rows          int
	Wireframe     bool
	OSStyle       string
	WindowRadius  int
	Border        int
	ScenePad      int
	ChromeW       int
	ChromeH       int
	ChromeOffsetX int
	ChromeOffsetY int
}

// EffectiveFontPx returns the export font size with a readability floor.
func EffectiveFontPx(opts Options) int {
	px := opts.FontSizePx
	if px < MinExportFontPx {
		px = MinExportFontPx
	}
	return px
}

// InternalRenderScale returns the rasterization multiplier before optional downscale.
func InternalRenderScale(exportScale int) int {
	if exportScale == 1 {
		return CompactSuperSample
	}
	return exportScale
}

// ComputeLayout derives pixel dimensions from screen and options.
func ComputeLayout(snap term.ScreenSnapshot, opts Options) Layout {
	exportScale := opts.LayoutScale()
	renderScale := InternalRenderScale(exportScale)
	fontPx := EffectiveFontPx(opts)
	cols := snap.Cols
	if cols == 0 {
		cols = maxInt(1, longestLineCols(snap.Lines))
	}
	rows := snap.Rows
	if rows == 0 {
		rows = len(snap.Lines)
	}
	if rows == 0 {
		rows = 1
	}
	cellW, cellH, termW, termH := layoutCellGeometry(cols, rows, fontPx, renderScale, opts)

	if opts.IsOSChrome() {
		osStyle := opts.ResolvedOSStyle()
		if osStyle == OSStyleMacOS || osStyle == OSStyleWindows {
			titleBar := MacOSTitleBarHeight() * renderScale
			termInset := MacOSTerminalInset() * renderScale
			radius := MacOSWindowRadius() * renderScale
			if osStyle == OSStyleWindows {
				titleBar = WindowsTitleBarHeight() * renderScale
				termInset = WindowsTerminalInset() * renderScale
				radius = WindowsWindowRadius() * renderScale
			}
			renderOuterW := termW + termInset*2
			renderOuterH := titleBar + termH + termInset*2
			return applyCustomBackgroundScene(Layout{
				ExportScale:  exportScale,
				RenderScale:  renderScale,
				Downscale:    renderScale / exportScale,
				CellW:        cellW,
				CellH:        cellH,
				TermW:        termW,
				TermH:        termH,
				TitleBar:     titleBar,
				FramePad:     termInset,
				WindowRadius: radius,
				OSStyle:      osStyle,
				Cols:         cols,
				Rows:         rows,
				Wireframe:    false,
				RenderOuterW: renderOuterW,
				RenderOuterH: renderOuterH,
				OuterW:       renderOuterW / (renderScale / exportScale),
				OuterH:       renderOuterH / (renderScale / exportScale),
				TermOffsetX:  termInset,
				TermOffsetY:  titleBar + termInset,
			}, opts)
		}

		pad := ChromePadding() * renderScale
		title := TitleBarHeight(ChromeOS, OSStyleWireframe) * renderScale
		inner := ChromeInnerGap() * renderScale
		renderOuterW := termW + pad*2
		renderOuterH := pad + title + inner + termH + pad
		return applyCustomBackgroundScene(Layout{
			ExportScale:  exportScale,
			RenderScale:  renderScale,
			Downscale:    renderScale / exportScale,
			CellW:        cellW,
			CellH:        cellH,
			TermW:        termW,
			TermH:        termH,
			ChromePad:    pad,
			TitleBar:     title,
			InnerGap:     inner,
			OSStyle:      OSStyleWireframe,
			Cols:         cols,
			Rows:         rows,
			Wireframe:    true,
			RenderOuterW: renderOuterW,
			RenderOuterH: renderOuterH,
			OuterW:       renderOuterW / (renderScale / exportScale),
			OuterH:       renderOuterH / (renderScale / exportScale),
			TermOffsetX:  pad,
			TermOffsetY:  pad + title + inner,
		}, opts)
	}

	framePad := ViewerFramePad() * renderScale
	frameW := termW + framePad*2
	frameH := termH + framePad*2
	renderOuterW := frameW
	renderOuterH := frameH
	return applyCustomBackgroundScene(Layout{
		ExportScale:  exportScale,
		RenderScale:  renderScale,
		Downscale:    renderScale / exportScale,
		CellW:        cellW,
		CellH:        cellH,
		TermW:        termW,
		TermH:        termH,
		FramePad:     framePad,
		FrameRadius:  ViewerFrameRadius() * renderScale,
		FrameW:       frameW,
		FrameH:       frameH,
		Cols:         cols,
		Rows:         rows,
		Wireframe:    false,
		RenderOuterW: renderOuterW,
		RenderOuterH: renderOuterH,
		OuterW:       renderOuterW / (renderScale / exportScale),
		OuterH:       renderOuterH / (renderScale / exportScale),
		TermOffsetX:  framePad,
		TermOffsetY:  framePad,
	}, opts)
}

func applyCustomBackgroundScene(layout Layout, opts Options) Layout {
	layout.ChromeW = layout.RenderOuterW
	layout.ChromeH = layout.RenderOuterH
	if opts.BackgroundMode != BackgroundCustom {
		return layout
	}
	scenePad := CustomBackgroundScenePad() * layout.RenderScale
	layout.ScenePad = scenePad
	layout.ChromeOffsetX = scenePad
	layout.ChromeOffsetY = scenePad
	layout.RenderOuterW = layout.ChromeW + scenePad*2
	layout.RenderOuterH = layout.ChromeH + scenePad*2
	if layout.Downscale > 0 {
		layout.OuterW = layout.RenderOuterW / layout.Downscale
		layout.OuterH = layout.RenderOuterH / layout.Downscale
	} else {
		layout.OuterW = layout.RenderOuterW
		layout.OuterH = layout.RenderOuterH
	}
	layout.TermOffsetX += scenePad
	layout.TermOffsetY += scenePad
	return layout
}

func layoutCellGeometry(cols, rows, fontPx, renderScale int, opts Options) (cellW, cellH, termW, termH int) {
	if opts.TermWPx > 0 && opts.TermHPx > 0 {
		termW = opts.TermWPx * renderScale
		termH = opts.TermHPx * renderScale
		if cols > 0 {
			cellW = termW / cols
		} else {
			cellW = 8 * renderScale
		}
		if rows > 0 {
			cellH = termH / rows
		} else {
			cellH = (fontPx + 6) * renderScale
		}
		return cellW, cellH, termW, termH
	}
	baseW := fontPx * 6 / 10
	if baseW < 8 {
		baseW = 8
	}
	baseH := fontPx + 6
	cellW = baseW * renderScale
	cellH = baseH * renderScale
	termW = cols * cellW
	termH = rows * cellH
	return cellW, cellH, termW, termH
}

func longestLineCols(lines []string) int {
	max := 0
	for _, line := range lines {
		if n := len([]rune(line)); n > max {
			max = n
		}
	}
	return max
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ScaleLayoutToOuter converts render-space layout into final SVG/output pixels.
// Gwenview and other Qt SVG renderers mishandle nested transform scale groups.
func ScaleLayoutToOuter(layout Layout) Layout {
	if layout.Downscale <= 1 {
		return layout
	}
	d := layout.Downscale
	round := func(v int) int {
		if v == 0 {
			return 0
		}
		return (v + d/2) / d
	}
	layout.RenderScale = round(layout.RenderScale)
	layout.CellW = round(layout.CellW)
	layout.CellH = round(layout.CellH)
	layout.TermW = round(layout.TermW)
	layout.TermH = round(layout.TermH)
	layout.ChromePad = round(layout.ChromePad)
	layout.TitleBar = round(layout.TitleBar)
	layout.InnerGap = round(layout.InnerGap)
	layout.FramePad = round(layout.FramePad)
	layout.FrameRadius = round(layout.FrameRadius)
	layout.FrameW = round(layout.FrameW)
	layout.FrameH = round(layout.FrameH)
	layout.WindowRadius = round(layout.WindowRadius)
	layout.Border = round(layout.Border)
	layout.ScenePad = round(layout.ScenePad)
	layout.ChromeW = round(layout.ChromeW)
	layout.ChromeH = round(layout.ChromeH)
	layout.ChromeOffsetX = round(layout.ChromeOffsetX)
	layout.ChromeOffsetY = round(layout.ChromeOffsetY)
	layout.TermOffsetX = round(layout.TermOffsetX)
	layout.TermOffsetY = round(layout.TermOffsetY)
	layout.RenderOuterW = layout.OuterW
	layout.RenderOuterH = layout.OuterH
	layout.Downscale = 1
	return layout
}
