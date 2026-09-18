package export

import "testing"

// TestSanitizeGlyphPassesThroughSupportedChars proves ordinary ASCII and
// the box-drawing/symbol glyphs gomono (the raster export's embedded font)
// genuinely covers are returned unchanged, not run through the fallback map
// unnecessarily.
func TestSanitizeGlyphPassesThroughSupportedChars(t *testing.T) {
	face, err := monoFace(16)
	if err != nil {
		t.Fatal(err)
	}
	// │┌┐└┘├┤┬┴┼ are the LIGHT box-drawing set; confirmed covered by
	// gomono directly (unlike the HEAVY set ┃┏┓┗┛, which isn't — see
	// TestSanitizeGlyphSubstitutesBoxDrawing).
	for _, ch := range []string{"A", "1", "+", " ", "─", "│", "┌", "•", "●", "○", "►"} {
		if got := sanitizeGlyph(face, ch); got != ch {
			t.Errorf("sanitizeGlyph(%q) = %q, want unchanged", ch, got)
		}
	}
}

// TestSanitizeGlyphSubstitutesBoxDrawing is the regression test for the
// tofu-box bug: the HEAVY box-drawing glyphs and a few TUI markers gomono
// doesn't cover should come back as their mapped ASCII look-alike, not the
// original (tofu-box) glyph.
func TestSanitizeGlyphSubstitutesBoxDrawing(t *testing.T) {
	face, err := monoFace(16)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"┃": "|", "┏": "+", "▸": ">", "✓": "v",
	}
	for ch, want := range cases {
		if got := sanitizeGlyph(face, ch); got != want {
			t.Errorf("sanitizeGlyph(%q) = %q, want %q", ch, got, want)
		}
	}
}

// TestSanitizeGlyphUnknownUnsupportedFallsBackToBlank proves a glyph with
// no known fallback becomes a blank space rather than the original
// (tofu-box) glyph reaching drawText.
func TestSanitizeGlyphUnknownUnsupportedFallsBackToBlank(t *testing.T) {
	face, err := monoFace(16)
	if err != nil {
		t.Fatal(err)
	}
	// U+1F600 GRINNING FACE: an emoji gomono definitely doesn't cover, and
	// not in boxDrawingFallback either.
	if got := sanitizeGlyph(face, "😀"); got != " " {
		t.Errorf("sanitizeGlyph(emoji) = %q, want blank fallback", got)
	}
}

func TestSanitizeGlyphsAppliesAcrossWholeLine(t *testing.T) {
	face, err := monoFace(16)
	if err != nil {
		t.Fatal(err)
	}
	got := sanitizeGlyphs(face, "A┃B")
	if want := "A|B"; got != want {
		t.Errorf("sanitizeGlyphs(%q) = %q, want %q", "A┃B", got, want)
	}
}
