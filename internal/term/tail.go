package term

import (
	"strings"
)

// TailLine is one non-empty row in tail output.
type TailLine struct {
	Y    int    `json:"y"`
	Text string `json:"text"`
}

// TailLines returns the last maxLines non-empty rows (bottom-up), preserving order.
func TailLines(snap ScreenSnapshot, maxLines int) []TailLine {
	if maxLines <= 0 {
		return nil
	}
	var nonempty []TailLine
	for y := len(snap.Lines) - 1; y >= 0; y-- {
		text := strings.TrimRight(snap.Lines[y], " ")
		if text == "" {
			continue
		}
		nonempty = append(nonempty, TailLine{Y: y, Text: text})
		if len(nonempty) >= maxLines {
			break
		}
	}
	// Restore top-to-bottom order.
	for i, j := 0, len(nonempty)-1; i < j; i, j = i+1, j-1 {
		nonempty[i], nonempty[j] = nonempty[j], nonempty[i]
	}
	return nonempty
}

// RegionLines returns rows in [y1, y2] inclusive, clamped to the viewport.
func RegionLines(snap ScreenSnapshot, y1, y2 int) []TailLine {
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	if y1 < 0 {
		y1 = 0
	}
	if y2 >= len(snap.Lines) {
		y2 = len(snap.Lines) - 1
	}
	var out []TailLine
	for y := y1; y <= y2; y++ {
		out = append(out, TailLine{Y: y, Text: snap.Lines[y]})
	}
	return out
}

// JoinTailText joins tail lines into a single newline-delimited string.
func JoinTailText(lines []TailLine) string {
	if len(lines) == 0 {
		return ""
	}
	parts := make([]string, len(lines))
	for i, ln := range lines {
		parts[i] = ln.Text
	}
	return strings.Join(parts, "\n")
}

// VisibleText returns non-empty lines trimmed and joined (full viewport).
func VisibleText(snap ScreenSnapshot) string {
	return JoinTailText(TailLines(snap, len(snap.Lines)))
}

// ContainsText reports whether needle appears in any visible line.
func ContainsText(snap ScreenSnapshot, needle string) bool {
	if needle == "" {
		return true
	}
	for _, line := range snap.Lines {
		if strings.Contains(line, needle) {
			return true
		}
	}
	return false
}
