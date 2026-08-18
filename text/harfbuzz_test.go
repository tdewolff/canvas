package text

import (
	"os"
	"testing"

	"github.com/tdewolff/test"
)

// TestShapeGuessedDirection covers the case where the caller leaves the
// direction unset and GuessSegmentProperties picks one.
//
// Shape used to branch on the direction it was passed rather than the one the
// buffer ended up with. For Arabic or Hebrew with no direction given, harfbuzz
// guesses RTL and emits glyphs in visual order, but the walk still read the
// cluster bound off the *next* glyph as if the run were LTR. That index is
// lower than the current cluster in a reversed buffer, so slicing the text
// panicked outright.
//
// Shaping with the direction left unset must give the same glyph text as
// naming RightToLeft explicitly.
func TestShapeGuessedDirection(t *testing.T) {
	b, err := os.ReadFile("../resources/unifont-13.0.05.ttf")
	if err != nil {
		t.Skipf("%v", err)
	}
	shaper, err := NewShaper(b, 0)
	if err != nil {
		t.Skipf("%v", err)
	}

	for _, tt := range []struct {
		text   string
		script Script
	}{
		{"السلام", Arabic},
		{"שלום", Hebrew},
	} {
		explicit := shaper.Shape(tt.text, 100, RightToLeft, tt.script, "", "", "")
		guessed := shaper.Shape(tt.text, 100, DirectionInvalid, tt.script, "", "", "")

		test.T(t, len(guessed), len(explicit))
		for i := range explicit {
			if i < len(guessed) && guessed[i].Text != explicit[i].Text {
				t.Errorf("%q glyph %d: guessed direction gives text %q, explicit RightToLeft gives %q",
					tt.text, i, guessed[i].Text, explicit[i].Text)
			}
		}
	}
}
