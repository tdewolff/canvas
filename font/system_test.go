package font

import (
	"fmt"
	"testing"
	"time"
)

func TestFindSystemFonts(t *testing.T) {
	start := time.Now()
	dirs := DefaultFontDirs()
	fonts, err := FindSystemFonts(dirs)
	if err != nil {
		t.Error(err)
		return
	}

	fmt.Println(fonts, time.Since(start))
}
