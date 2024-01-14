package font

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"reflect"
	"sort"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font/sfnt"
)

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
	} else if tag == "true" || binary.BigEndian.Uint32(b[:4]) == 0x00010000 || tag == "ttcf" {
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

// ToSFNT takes a byte slice and transforms it into an SFNT byte slice. That is, given TTF/OTF/WOFF/WOFF2/EOT input, it will return TTF/OTF output.
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

// NewSFNTReader takes an io.Reader and transforms it into an SFNT reader. That is, given TTF/OTF/WOFF/WOFF2/EOT input, it will return TTF/OTF output.
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

// ParseFont parses a byte slice and of a TTF, OTF, WOFF, WOFF2, or EOT font format. It will return the parsed font and its mimetype.
func ParseFont(b []byte, index int) (*SFNT, error) {
	sfntBytes, err := ToSFNT(b)
	if err != nil {
		return nil, err
	}
	return ParseSFNT(sfntBytes, index)
}

func ParseMetadata(b []byte, index int) (*FontMetadata, error) {
	sfntBytes, err := ToSFNT(b)
	if err != nil {
		return nil, err
	}
	return parseMetadata(sfntBytes, index)
}
func parseMetadata(byt []byte, index int) (*FontMetadata, error) {
	if len(byt) < 12 || uint(math.MaxUint32) < uint(len(byt)) {
		return nil, ErrInvalidFontData
	}

	r := NewBinaryReader(byt)
	sfntVersion := r.ReadString(4)
	isCollection := sfntVersion == "ttcf"
	if isCollection {
		majorVersion := r.ReadUint16()
		minorVersion := r.ReadUint16()
		if majorVersion != 1 && majorVersion != 2 || minorVersion != 0 {
			return nil, fmt.Errorf("bad TTC version")
		}

		numFonts := r.ReadUint32()
		if index < 0 || numFonts <= uint32(index) {
			return nil, fmt.Errorf("bad font index %d", index)
		}
		if r.Len() < 4*numFonts {
			return nil, ErrInvalidFontData
		}

		_ = r.ReadBytes(uint32(4 * index))
		offset := r.ReadUint32()
		var length uint32
		if uint32(index)+1 == numFonts {
			length = uint32(len(byt)) - offset
		} else {
			length = r.ReadUint32() - offset
		}
		if uint32(len(byt))-8 < offset || uint32(len(byt))-8-offset < length {
			return nil, ErrInvalidFontData
		}

		r.Seek(offset)
		sfntVersion = r.ReadString(4)
	} else if index != 0 {
		return nil, fmt.Errorf("bad font index %d", index)
	}
	if sfntVersion != "OTTO" && sfntVersion != "true" && binary.BigEndian.Uint32([]byte(sfntVersion)) != 0x00010000 {
		return nil, fmt.Errorf("bad SFNT version")
	}
	numTables := r.ReadUint16()
	_ = r.ReadUint16()                  // searchRange
	_ = r.ReadUint16()                  // entrySelector
	_ = r.ReadUint16()                  // rangeShift
	if r.Len() < 16*uint32(numTables) { // can never exceed uint32 as numTables is uint16
		return nil, ErrInvalidFontData
	}

	var nameTableData []byte
	for i := 0; i < int(numTables); i++ {
		tag := r.ReadString(4)
		_ = r.ReadUint32() // checksum
		offset := r.ReadUint32()
		length := r.ReadUint32()

		padding := (4 - length&3) & 3
		if uint32(len(byt)) <= offset || uint32(len(byt))-offset < length || uint32(len(byt))-offset-length < padding {
			return nil, ErrInvalidFontData
		}

		if tag == "head" {
			if length < 12 {
				return nil, ErrInvalidFontData
			}
		}

		if string(tag) != "name" {
			continue
		}

		nameTableData = byt[offset : offset+length : offset+length]
		break
	}

	if nameTableData == nil {
		return nil, fmt.Errorf("missing table name")
	}

	r = NewBinaryReader(nameTableData)
	version := r.ReadUint16()
	if version != 0 && version != 1 {
		return nil, fmt.Errorf("name: bad version")
	}
	count := r.ReadUint16()
	storageOffset := r.ReadUint16()
	if uint32(len(nameTableData)) < 6+12*uint32(count) || uint16(len(nameTableData)) < storageOffset {
		return nil, fmt.Errorf("name: bad table")
	}

	var names []string
	var style string

	for i := 0; i < int(count); i++ {
		var record nameRecord
		record.Platform = PlatformID(r.ReadUint16())
		record.Encoding = EncodingID(r.ReadUint16())
		record.Language = r.ReadUint16()
		record.Name = NameID(r.ReadUint16())

		length := r.ReadUint16()
		offset := r.ReadUint16()
		if uint16(len(nameTableData))-storageOffset < offset || uint16(len(nameTableData))-storageOffset-offset < length {
			return nil, fmt.Errorf("name: bad table")
		}
		record.Value = nameTableData[storageOffset+offset : storageOffset+offset+length]

		if record.Name == NameFontFamily ||
			record.Name == NameFull ||
			record.Name == NamePostScript {
			names = append(names, record.String())
		}

		if record.Name == NameFontSubfamily ||
			record.Name == NamePreferredSubfamily {
			style = record.String()
		}
	}

	return &FontMetadata{
		Filename: "",
		Families: names,
		Style:    ParseStyle(style),
	}, nil
}

// FromGoFreetype parses a structure from truetype.Font to a valid SFNT byte slice.
func FromGoFreetype(font *truetype.Font) []byte {
	v := reflect.ValueOf(*font)
	tables := map[string][]byte{}
	tables["cmap"] = v.FieldByName("cmap").Bytes()
	tables["cvt "] = v.FieldByName("cvt").Bytes()
	tables["fpgm"] = v.FieldByName("fpgm").Bytes()
	tables["glyf"] = v.FieldByName("glyf").Bytes()
	tables["hdmx"] = v.FieldByName("hdmx").Bytes()
	tables["head"] = v.FieldByName("head").Bytes()
	tables["hhea"] = v.FieldByName("hhea").Bytes()
	tables["hmtx"] = v.FieldByName("hmtx").Bytes()
	tables["kern"] = v.FieldByName("kern").Bytes()
	tables["loca"] = v.FieldByName("loca").Bytes()
	tables["maxp"] = v.FieldByName("maxp").Bytes()
	tables["name"] = v.FieldByName("name").Bytes()
	tables["OS/2"] = v.FieldByName("os2").Bytes()
	tables["prep"] = v.FieldByName("prep").Bytes()
	tables["vmtx"] = v.FieldByName("vmtx").Bytes()

	// reconstruct missing post table
	post := NewBinaryWriter([]byte{})
	post.WriteUint32(0x00030000) // version
	post.WriteUint32(0)          // italicAngle
	post.WriteInt16(0)           // underlinePosition
	post.WriteInt16(0)           // underlineThickness
	post.WriteUint32(0)          // isFixedPitch
	post.WriteUint32(0)          // minMemType42
	post.WriteUint32(0)          // maxMemType42
	post.WriteUint32(0)          // minMemType1
	post.WriteUint32(0)          // maxMemType1
	tables["post"] = post.Bytes()

	// remove empty tables
	tags := []string{}
	for tag, table := range tables {
		if 0 < len(table) {
			tags = append(tags, tag)
		}
	}
	sort.Strings(tags)

	w := NewBinaryWriter([]byte{})
	w.WriteUint32(0x00010000) // sfntVersion

	numTables := uint16(len(tags))
	entrySelector := uint16(math.Log2(float64(numTables)))
	searchRange := uint16(1 << (entrySelector + 4))
	w.WriteUint16(numTables)                  // numTables
	w.WriteUint16(searchRange)                // searchRange
	w.WriteUint16(entrySelector)              // entrySelector
	w.WriteUint16(numTables<<4 - searchRange) // rangeShift

	// write table directory
	var checksumAdjustmentPos uint32
	offset := w.Len() + 16*uint32(numTables)
	for _, tag := range tags {
		length := uint32(len(tables[tag]))
		padding := (4 - length&3) & 3
		for j := 0; j < int(padding); j++ {
			tables[tag] = append(tables[tag], 0x00)
		}
		if tag == "head" {
			checksumAdjustmentPos = offset + 8
			binary.BigEndian.PutUint32(tables[tag][8:], 0x00000000)
		}
		checksum := calcChecksum(tables[tag])

		w.WriteString(tag)
		w.WriteUint32(checksum)
		w.WriteUint32(offset)
		w.WriteUint32(length)
		offset += length + padding
	}

	// write tables
	for _, tag := range tags {
		w.WriteBytes(tables[tag])
	}

	buf := w.Bytes()
	binary.BigEndian.PutUint32(buf[checksumAdjustmentPos:], 0xB1B0AFBA-calcChecksum(buf))
	return buf
}

// FromGoSFNT parses a structure from sfnt.Font to a valid SFNT byte slice.
func FromGoSFNT(font *sfnt.Font) []byte {
	v := reflect.ValueOf(*font)
	buf := v.FieldByName("src").FieldByName("b").Bytes()
	return buf
}
