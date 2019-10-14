package font

import (
	"encoding/binary"
	"fmt"

	"golang.org/x/image/font/sfnt"
)

func ParseFontOld(b []byte) (string, *sfnt.Font, error) {
	font, tag, err := ParseFont(b)
	if err != nil {
		return "", nil, err
	}

	mimetype := ""
	if tag == "wOFF" {
		mimetype = "font/woff"
	} else if tag == "wOF2" {
		mimetype = "font/woff2"
	} else if tag == "true" || binary.BigEndian.Uint32([]byte(tag)) == 0x00010000 {
		mimetype = "font/truetype"
	} else if tag == "OTTO" {
		mimetype = "font/opentype"
	}
	return mimetype, (*sfnt.Font)(font), nil
}

type Font sfnt.Font

func ParseFont(b []byte) (*Font, string, error) {
	// TODO: support Type1 and EOT font format?
	if len(b) < 4 {
		return nil, "", fmt.Errorf("empty font file")
	}

	tag := string(b[:4])
	if tag == "wOFF" {
		var flavor uint32
		var err error
		b, flavor, err = ParseWOFF(b)
		if err != nil {
			return nil, "", fmt.Errorf("WOFF: %w", err)
		}
		tag = uint32ToString(flavor)
	} else if tag == "wOF2" {
		var flavor uint32
		var err error
		b, flavor, err = ParseWOFF2(b)
		if err != nil {
			return nil, "", fmt.Errorf("WOFF2: %w", err)
		}
		tag = uint32ToString(flavor)
	} else if tag == "true" || binary.BigEndian.Uint32(b[:4]) == 0x00010000 {
		// TTF
	} else if tag == "OTTO" {
		// OTF
	} else {
		return nil, "", fmt.Errorf("unrecognized font file format")
	}

	sfnt, err := ParseSFNT(b)
	if err != nil {
		return nil, "", fmt.Errorf("SFNT: %w", err)
	}
	return sfnt, tag, nil
}
