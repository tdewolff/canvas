package font

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"

	"golang.org/x/image/font/sfnt"
)

// Font currently uses golang.org/x/image/font/sfnt
type Font sfnt.Font

// MediaType returns the media type (MIME) for a given font.
func MediaType(b []byte) (string, error) {
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

// Extension returns the file extension for a given font. An empty string is returned when the font is not recognized.
func Extension(b []byte) string {
	mediatype, err := MediaType(b)
	if err != nil {
		return ""
	}
	switch mediatype {
	case "font/truetype":
		return ".ttf"
	case "font/opentype":
		return ".otf"
	case "font/woff":
		return ".woff"
	case "font/woff2":
		return ".woff2"
	case "font/eot":
		return ".eot"
	}
	return ""
}

// ToSFNT takes a byte-slice and transforms it into an SFNT byte-slice. That is, given TTF/OTF/WOFF/WOFF2/EOT input, it will return TTF/OTF output.
func ToSFNT(b []byte) ([]byte, error) {
	mediatype, err := MediaType(b)
	if err != nil {
		return nil, err
	}
	switch mediatype {
	case "font/truetype":
		return b, nil
	case "font/opentype":
		return b, nil
	case "font/woff":
		if b, err = ParseWOFF(b); err != nil {
			return nil, fmt.Errorf("WOFF: %w", err)
		}
		return b, nil
	case "font/woff2":
		if b, err = ParseWOFF2(b); err != nil {
			return nil, fmt.Errorf("WOFF2: %w", err)
		}
		return b, nil
	case "font/eot":
		if b, err = ParseEOT(b); err != nil {
			return nil, fmt.Errorf("EOT: %w", err)
		}
		return b, nil
	}
	return nil, fmt.Errorf("unrecognized font file format")
}

// NewReader takes an io.Reader and transforms it into an SFNT reader. That is, given TTF/OTF/WOFF/WOFF2/EOT input, it will return TTF/OTF output.
func NewSFNTReader(r io.Reader) (*bytes.Reader, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if b, err = ToSFNT(b); err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

// ParseFont parses a byte slice and recognized whether it is a TTF, OTF, WOFF, WOFF2, or EOT font format. It will return the parsed font and its mimetype. Currently returns instance of golang.org/x/image/font/sfnt.
func ParseFont(b []byte) (*Font, error) {
	sfntBytes, err := ToSFNT(b)
	if err != nil {
		return nil, err
	}
	return ParseSFNT(sfntBytes)
}
