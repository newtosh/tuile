package export

import "testing"

func TestSanitizeGlyphsPassesThroughSupportedChars(t *testing.T) {
	face, err := monoFace(16)
	if err != nil {
		t.Fatal(err)
	}
	// Light box-drawing (─│┌) and symbols gomono covers directly.
	for _, ch := range []string{"A", "1", "+", " ", "─", "│", "┌", "•", "●", "○", "►"} {
		if got := sanitizeGlyphs(face, ch); got != ch {
			t.Errorf("sanitizeGlyphs(%q) = %q, want unchanged", ch, got)
		}
	}
}

func TestSanitizeGlyphsSubstitutesBoxDrawing(t *testing.T) {
	face, err := monoFace(16)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"┃": "|", "┏": "+", "▸": ">", "✓": "v",
	}
	for ch, want := range cases {
		if got := sanitizeGlyphs(face, ch); got != want {
			t.Errorf("sanitizeGlyphs(%q) = %q, want %q", ch, got, want)
		}
	}
}

func TestSanitizeGlyphsUnknownUnsupportedFallsBackToBlank(t *testing.T) {
	face, err := monoFace(16)
	if err != nil {
		t.Fatal(err)
	}
	if got := sanitizeGlyphs(face, "😀"); got != " " {
		t.Errorf("sanitizeGlyphs(emoji) = %q, want blank fallback", got)
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

func TestSanitizeGlyphsMultiRuneCell(t *testing.T) {
	face, err := monoFace(16)
	if err != nil {
		t.Fatal(err)
	}
	// CellSnapshot.Ch can hold multi-rune sequences; each rune must be
	// sanitized (regression for the early-return that skipped them).
	got := sanitizeGlyphs(face, "A┃😀")
	if want := "A| "; got != want {
		t.Errorf("sanitizeGlyphs(%q) = %q, want %q", "A┃😀", got, want)
	}
}
