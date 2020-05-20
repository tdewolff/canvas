package font

import (
	"fmt"
	"testing"

	"github.com/tdewolff/test"
)

func TestWOFF2Error(t *testing.T) {
	var tts = []struct {
		data string
	}{
		{"wOF200000000\x00\x00000000\xff\xff\xff\xff000000000000000000000000"},
	}
	for i, tt := range tts {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			_, err := ParseWOFF2([]byte(tt.data))
			test.T(t, err, ErrInvalidFontData)
		})
	}
}
