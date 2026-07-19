package term

import (
	"fmt"

	xterm "github.com/gitpod-io/xterm-go"
)

// ScrollRegion describes the active terminal scroll margins.
type ScrollRegion struct {
	Top    int `json:"top"`
	Bottom int `json:"bottom"`
}

// CellSnapshot is one grid cell for the headless API.
type CellSnapshot struct {
	Ch   string   `json:"ch"`
	Fg   string   `json:"fg,omitempty"`
	Bg   string   `json:"bg,omitempty"`
	Attr []string `json:"attr,omitempty"`
}

// RowSnapshot is one viewport row with optional per-cell detail.
type RowSnapshot struct {
	Y     int            `json:"y"`
	Text  string         `json:"text"`
	Cells []CellSnapshot `json:"cells,omitempty"`
}

// ScreenSnapshot is a JSON-serializable grid for the headless API (U3).
type ScreenSnapshot struct {
	Cols   int           `json:"cols"`
	Rows   int           `json:"rows"`
	Cursor struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"cursor"`
	Lines  []string      `json:"lines"`
	Alt    bool          `json:"alt_buffer"`
	Scroll *ScrollRegion `json:"scroll,omitempty"`
	Grid   []RowSnapshot `json:"grid,omitempty"`
}

// SnapshotOptions controls structured screen capture.
type SnapshotOptions struct {
	IncludeCells bool
}

// LineChange is one row update in an incremental screen diff.
type LineChange struct {
	Y    int    `json:"y"`
	Text string `json:"text"`
}

// ScreenDiff captures incremental updates between two screen versions.
type ScreenDiff struct {
	Cursor       *struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"cursor,omitempty"`
	Scroll       *ScrollRegion `json:"scroll,omitempty"`
	ChangedLines []LineChange  `json:"changed_lines,omitempty"`
}

// DiffSnapshots returns line-level changes between two line snapshots.
func DiffSnapshots(before, after ScreenSnapshot) ScreenDiff {
	var diff ScreenDiff
	if before.Cursor.X != after.Cursor.X || before.Cursor.Y != after.Cursor.Y {
		diff.Cursor = &after.Cursor
	}
	if !scrollEqual(before.Scroll, after.Scroll) {
		diff.Scroll = after.Scroll
	}
	rows := after.Rows
	if rows == 0 {
		rows = len(after.Lines)
	}
	for y := 0; y < rows; y++ {
		var prev, next string
		if y < len(before.Lines) {
			prev = before.Lines[y]
		}
		if y < len(after.Lines) {
			next = after.Lines[y]
		}
		if prev != next {
			diff.ChangedLines = append(diff.ChangedLines, LineChange{Y: y, Text: next})
		}
	}
	return diff
}

func scrollEqual(a, b *ScrollRegion) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Top == b.Top && a.Bottom == b.Bottom
}

func snapshotFromTerminal(term *xterm.Terminal, opts SnapshotOptions) ScreenSnapshot {
	snap := ScreenSnapshot{
		Cols:  term.Cols(),
		Rows:  term.Rows(),
		Alt:   term.IsAltBufferActive(),
		Lines: make([]string, term.Rows()),
	}
	snap.Cursor.X = term.CursorX()
	snap.Cursor.Y = term.CursorY()

	buf := term.Buffer()
	snap.Scroll = &ScrollRegion{
		Top:    buf.ScrollTop,
		Bottom: buf.ScrollBottom,
	}

	cellScratch := xterm.NewCellData()
	for y := 0; y < term.Rows(); y++ {
		line := buf.Lines.Get(buf.YBase + y)
		if line == nil {
			continue
		}
		text := line.TranslateToString(true, 0, -1)
		snap.Lines[y] = text
		if !opts.IncludeCells {
			continue
		}
		row := RowSnapshot{Y: y, Text: text}
		for x := 0; x < line.Len; x++ {
			line.LoadCell(x, cellScratch)
			if cellScratch.GetWidth() == 0 {
				continue
			}
			ch := cellScratch.GetChars()
			if ch == "" {
				ch = " "
			}
			cell := CellSnapshot{Ch: ch}
			if fg := encodeColor(cellScratch, true); fg != "" {
				cell.Fg = fg
			}
			if bg := encodeColor(cellScratch, false); bg != "" {
				cell.Bg = bg
			}
			if attrs := encodeAttrs(cellScratch); len(attrs) > 0 {
				cell.Attr = attrs
			}
			row.Cells = append(row.Cells, cell)
		}
		snap.Grid = append(snap.Grid, row)
	}
	return snap
}

func encodeColor(cell *xterm.CellData, fg bool) string {
	if fg {
		if cell.IsFgDefault() {
			return ""
		}
		if cell.IsFgRGB() {
			rgb := xterm.ToColorRGB(uint32(cell.GetFgColor()))
			return fmt.Sprintf("#%02x%02x%02x", rgb[0], rgb[1], rgb[2])
		}
		return fmt.Sprintf("p%d", cell.GetFgColor())
	}
	if cell.IsBgDefault() {
		return ""
	}
	if cell.IsBgRGB() {
		rgb := xterm.ToColorRGB(uint32(cell.GetBgColor()))
		return fmt.Sprintf("#%02x%02x%02x", rgb[0], rgb[1], rgb[2])
	}
	return fmt.Sprintf("p%d", cell.GetBgColor())
}

func encodeAttrs(cell *xterm.CellData) []string {
	var attrs []string
	if cell.IsBold() != 0 {
		attrs = append(attrs, "bold")
	}
	if cell.IsDim() != 0 {
		attrs = append(attrs, "dim")
	}
	if cell.IsItalic() != 0 {
		attrs = append(attrs, "italic")
	}
	if cell.IsUnderline() != 0 {
		attrs = append(attrs, "underline")
	}
	if cell.IsBlink() != 0 {
		attrs = append(attrs, "blink")
	}
	if cell.IsInverse() != 0 {
		attrs = append(attrs, "inverse")
	}
	if cell.IsInvisible() != 0 {
		attrs = append(attrs, "invisible")
	}
	if cell.IsStrikethrough() != 0 {
		attrs = append(attrs, "strikethrough")
	}
	if cell.IsOverline() != 0 {
		attrs = append(attrs, "overline")
	}
	return attrs
}
