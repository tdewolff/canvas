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

	var x0, x1 uint16
	x0 = 3314 //font.Cmap.Map('A')
	x1 = 1933 //font.Cmap.Map('V')

	fmt.Println(x0, x1, font.Kern.Get(x0, x1))
	return

	if font.IsTrueType {
		contour, err := font.Glyf.Contour(20)
		test.Error(t, err)
		fmt.Println(contour)
	}
}
