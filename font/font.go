package font

import (
	"encoding/binary"
	"fmt"

	"golang.org/x/image/font/sfnt"
)

// Font currently uses golang.org/x/image/font/sfnt
type Font sfnt.Font

func Mimetype(b []byte) (string, error) {
	if len(b) < 4 {
		return "", fmt.Errorf("empty font file")
	}

	tag := string(b[:4])
	if tag == "wOFF" {
		return "font/woff", nil
	} else if tag == "wOF2" {
		return "font/woff2", nil
	} else if tag == "true" || binary.BigEndian.Uint32(b[:4]) == 0x00010000 {
		return "font/truetype", nil
	} else if tag == "OTTO" {
		return "font/opentype", nil
	} else if 36 < len(b) && binary.LittleEndian.Uint16(b[34:36]) == 0x504C {
		return "font/eot", nil
	}
	return "", fmt.Errorf("unrecognized font file format")

}

func ToSFNT(b []byte) ([]byte, string, error) {
	mimetype, err := Mimetype(b)
	if err != nil {
		return nil, "", err
	}

	if mimetype == "font/woff" {
		b, err = ParseWOFF1(b)
		if err != nil {
			return nil, "", fmt.Errorf("WOFF: %w", err)
		}
	} else if mimetype == "font/woff2" {
		b, err = ParseWOFF2(b)
		if err != nil {
			return nil, "", fmt.Errorf("WOFF2: %w", err)
		}
	} else if mimetype == "font/eot" {
		b, err = ParseEOT(b)
		if err != nil {
			return nil, "", fmt.Errorf("EOT: %w", err)
		}
	}

	mimetype, err = Mimetype(b)
	if err != nil {
		return nil, "", err
	}
	return b, mimetype, nil
}

// ParseFont parses a byte slice and recognized whether it is a TTF, OTF, WOFF, WOFF2, or EOT font format. It will return the parsed font and its mimetype.
func ParseFont(b []byte) (*Font, error) {
	sfntBytes, _, err := ToSFNT(b)
	if err != nil {
		return nil, err
	}

	sfnt, err := ParseSFNT(sfntBytes)
	if err != nil {
		return nil, err
	}
	return sfnt, nil
}
