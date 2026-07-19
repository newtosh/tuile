package term

import (
	"regexp"
	"strings"
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)

// StripANSI removes common CSI escape sequences from terminal text.
func StripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

// TailFingerprint fingerprints the last maxLines non-empty rows for activity detection.
func TailFingerprint(snap ScreenSnapshot, maxLines int) string {
	lines := TailLines(snap, maxLines)
	if len(lines) == 0 {
		return ""
	}
	parts := make([]string, len(lines))
	for i, ln := range lines {
		text := strings.TrimSpace(StripANSI(ln.Text))
		text = strings.Join(strings.Fields(text), " ")
		parts[i] = text
	}
	return strings.Join(parts, "\n")
}
