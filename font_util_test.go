package canvas

import (
	"fmt"
	"testing"
)

func TestParseWOFF(t *testing.T) {
	_, err = LoadFontFile("DejaVuSerif", Regular, "example/DejaVuSerif.woff")
	fmt.Println(err)
}
