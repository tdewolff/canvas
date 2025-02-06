package text

import (
	"testing"

	"github.com/tdewolff/test"
)

func TestScriptItemizer(t *testing.T) {
	var tests = []struct {
		str   string
		items []ScriptItem
	}{
		{"abc", []ScriptItem{{Latin, 0, "abc"}}},
		{"\u064bياعادلا", []ScriptItem{{Arabic, 1, "\u064bياعادلا"}}},
	}

	for _, tt := range tests {
		t.Run(tt.str, func(t *testing.T) {
			runes := []rune(tt.str)
			embeddingLevels := EmbeddingLevels(runes)
			items := ScriptItemizer(runes, embeddingLevels)
			test.T(t, items, tt.items)
		})
	}
}
