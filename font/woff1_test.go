package font

import (
	"fmt"
	"testing"

	"github.com/tdewolff/test"
)

func TestWOFFError(t *testing.T) {
	var tts = []struct {
		data string
		err  string
	}{
		{"wOFF00000000\x00\x01\x00\x0000000000000000000000i00000000000\xff\xff\xff\xfc\x00\x00\x0000000000000000000", ErrInvalidFontData.Error()},
		{"wOFF\x01bwOFF u\x00\x01\x00\x00de\x80\x00orma\x10\x00wOFF\x01b dunicF u\x00r\xbd\xbf\xef^\x00\x00\x00\x00 \x00\x00\x00 :pur  oes ?ite:\t", ErrInvalidFontData.Error()},
	}
	for i, tt := range tts {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			_, err := ParseWOFF1([]byte(tt.data))
			test.T(t, err.Error(), tt.err)
		})
	}
}
