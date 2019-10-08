package font

import (
	"encoding/binary"
	"fmt"

	"golang.org/x/image/font/sfnt"
)

func ParseFontOld(b []byte) (string, *sfnt.Font, error) {
	font, err := ParseFont(b)
	if err != nil {
		return "", nil, err
	}

	mimetype := ""
	tag := string(b[:4])
	if tag == "wOFF" {
		mimetype = "font/woff"
	} else if tag == "wOF2" {
		mimetype = "font/woff2"
	} else if tag == "true" || binary.BigEndian.Uint32(b[:4]) == 0x00010000 {
		mimetype = "font/truetype"
	} else if tag == "OTTO" {
		mimetype = "font/opentype"
	}
	return mimetype, (*sfnt.Font)(font), nil
}

type Font sfnt.Font

func ParseFont(b []byte) (*Font, error) {
	// TODO: support Type1 and EOT font format?
	if len(b) < 4 {
		return nil, fmt.Errorf("invalid font file")
	}

	tag := string(b[:4])
	if tag == "wOFF" {
		return ParseWOFF(b)
	} else if tag == "wOF2" {
		return nil, fmt.Errorf("WOFF2 not yet supported")
		//return ParseWOFF2(b)
	} else if tag == "true" || binary.BigEndian.Uint32(b[:4]) == 0x00010000 {
		return ParseSFNT(b)
	} else if tag == "OTTO" {
		return ParseSFNT(b)
	}
	return nil, fmt.Errorf("unrecognized font file format")
}
