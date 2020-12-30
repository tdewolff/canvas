package font

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/tdewolff/test"
)

func TestSFNTDejaVuSerifTTF(t *testing.T) {
	b, err := ioutil.ReadFile("DejaVuSerif.ttf")
	test.Error(t, err)

	font, err := ParseSFNT(b)
	test.Error(t, err)

	id := font.GlyphIndex(' ')
	fmt.Println(font.GlyphName(id))

	contour, err := font.GlyphContour(id)
	test.Error(t, err)
	fmt.Println(contour)
}
