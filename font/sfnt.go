package font

import (
	"golang.org/x/image/font/sfnt"
)

func ParseSFNT(b []byte) (*Font, error) {
	font, err := sfnt.Parse(b)
	return (*Font)(font), err
}
