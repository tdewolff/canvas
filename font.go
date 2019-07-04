package canvas

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/dsnet/compress/brotli"
	"golang.org/x/image/font/sfnt"
)

var sfntBuffer sfnt.Buffer

// TypographicOptions are the options that can be enabled to make typographic or ligature substitutions automatically.
type TypographicOptions int

const (
	NoTypography TypographicOptions = 2 << iota
	NoRequiredLigatures
	CommonLigatures
	DiscretionaryLigatures
	HistoricalLigatures
)

// Font defines a font of type TTF or OTF which which a FontFace can be generated for use in text drawing operations.
type Font struct {
	// TODO: extend to fully read in sfnt data and read liga tables, generate Raw font data (base on used glyphs), etc
	name     string
	mimetype string
	raw      []byte
	sfnt     *sfnt.Font

	// TODO: use sub/superscript Unicode transformations in ToPath etc. if they exist
	typography  bool
	ligatures   []textSubstitution
	superscript []textSubstitution
	subscript   []textSubstitution
}

func parseFont(name string, b []byte) (*Font, error) {
	mimetype, sfnt, err := parseSFNT(b)
	if err != nil {
		return nil, err
	}
	f := &Font{
		name:     name,
		mimetype: mimetype,
		raw:      b,
		sfnt:     sfnt,
	}
	f.superscript = f.supportedSubstitutions(superscriptSubstitutes)
	f.subscript = f.supportedSubstitutions(subscriptSubstitutes)
	f.Use(0)
	return f, nil
}

// Raw returns the mimetype and raw binary data of the font.
func (f *Font) Raw() (string, []byte) {
	return f.mimetype, f.raw
}

type textSubstitution struct {
	src string
	dst rune
}

// TODO: read from liga tables in OpenType (clig, dlig, hlig) with rlig default enabled
var commonLigatures = []textSubstitution{
	{"ffi", '\uFB03'},
	{"ffl", '\uFB04'},
	{"ff", '\uFB00'},
	{"fi", '\uFB01'},
	{"fl", '\uFB02'},
}

var superscriptSubstitutes = []textSubstitution{
	{"0", '\u2070'},
	{"i", '\u2071'},
	{"2", '\u00B2'},
	{"3", '\u00B3'},
	{"4", '\u2074'},
	{"5", '\u2075'},
	{"6", '\u2076'},
	{"7", '\u2077'},
	{"8", '\u2078'},
	{"9", '\u2079'},
	{"+", '\u207A'},
	{"-", '\u207B'},
	{"=", '\u207C'},
	{"(", '\u207D'},
	{")", '\u207E'},
	{"n", '\u207F'},
}

var subscriptSubstitutes = []textSubstitution{
	{"0", '\u2080'},
	{"1", '\u2081'},
	{"2", '\u2082'},
	{"3", '\u2083'},
	{"4", '\u2084'},
	{"5", '\u2085'},
	{"6", '\u2086'},
	{"7", '\u2087'},
	{"8", '\u2088'},
	{"9", '\u2089'},
	{"+", '\u208A'},
	{"-", '\u208B'},
	{"=", '\u208C'},
	{"(", '\u208D'},
	{")", '\u208E'},
	{"a", '\u2090'},
	{"e", '\u2091'},
	{"o", '\u2092'},
	{"x", '\u2093'},
	{"h", '\u2095'},
	{"k", '\u2096'},
	{"l", '\u2097'},
	{"m", '\u2098'},
	{"n", '\u2099'},
	{"p", '\u209A'},
	{"s", '\u209B'},
	{"t", '\u209C'},
}

func (f *Font) supportedSubstitutions(substitutions []textSubstitution) []textSubstitution {
	supported := []textSubstitution{}
	for _, stn := range substitutions {
		if _, err := f.sfnt.GlyphIndex(&sfntBuffer, stn.dst); err == nil {
			supported = append(supported, stn)
		}
	}
	return supported
}

func (f *Font) Use(options TypographicOptions) {
	if options&NoTypography == 0 {
		f.typography = true
	}

	f.ligatures = []textSubstitution{}
	if options&CommonLigatures != 0 {
		f.ligatures = append(f.ligatures, f.supportedSubstitutions(commonLigatures)...)
	}
}

func (f *Font) substituteLigatures(s string) string {
	for _, stn := range f.ligatures {
		s = strings.ReplaceAll(s, stn.src, string(stn.dst))
	}
	return s
}

func (f *Font) substituteTypography(s string, inSingleQuote, inDoubleQuote bool) (string, bool, bool) {
	if f.typography {
		var rPrev, r rune
		var i, size int
		for {
			rPrev = r
			i += size
			if i >= len(s) {
				break
			}

			r, size = utf8.DecodeRuneInString(s[i:])
			if i+2 < len(s) && s[i] == '.' && s[i+1] == '.' && s[i+2] == '.' {
				s, size = stringReplace(s, i, 3, "\u2026") // ellipsis
				continue
			} else if i+4 < len(s) && s[i] == '.' && s[i+1] == ' ' && s[i+2] == '.' && s[i+3] == ' ' && s[i+4] == '.' {
				s, size = stringReplace(s, i, 5, "\u2026") // ellipsis
				continue
			} else if i+2 < len(s) && s[i] == '-' && s[i+1] == '-' && s[i+2] == '-' {
				s, size = stringReplace(s, i, 3, "\u2014") // em-dash
				continue
			} else if i+1 < len(s) && s[i] == '-' && s[i+1] == '-' {
				s, size = stringReplace(s, i, 2, "\u2013") // en-dash
				continue
			} else if i+2 < len(s) && s[i] == '(' && s[i+1] == 'c' && s[i+2] == ')' {
				s, size = stringReplace(s, i, 3, "\u00A9") // copyright
				continue
			} else if i+2 < len(s) && s[i] == '(' && s[i+1] == 'r' && s[i+2] == ')' {
				s, size = stringReplace(s, i, 3, "\u00AE") // registered
				continue
			} else if i+3 < len(s) && s[i] == '(' && s[i+1] == 't' && s[i+2] == 'm' && s[i+3] == ')' {
				s, size = stringReplace(s, i, 4, "\u2122") // trademark
				continue
			}

			// quotes
			if s[i] == '"' || s[i] == '\'' {
				var rNext rune
				if i+1 < len(s) {
					rNext, _ = utf8.DecodeRuneInString(s[i+1:])
				}
				if s[i] == '"' {
					s, size = quoteReplace(s, i, rPrev, r, rNext, &inDoubleQuote)
					continue
				} else {
					s, size = quoteReplace(s, i, rPrev, r, rNext, &inSingleQuote)
					continue
				}
			}

			// fractions
			if i+2 < len(s) && s[i+1] == '/' && isWordBoundary(rPrev) && rPrev != '/' {
				var rNext rune
				if i+3 < len(s) {
					rNext, _ = utf8.DecodeRuneInString(s[i+3:])
				}
				if isWordBoundary(rNext) && rNext != '/' {
					if s[i] == '1' && s[i+2] == '2' {
						s, size = stringReplace(s, i, 3, "\u00BD") // 1/2
						continue
					} else if s[i] == '1' && s[i+2] == '4' {
						s, size = stringReplace(s, i, 3, "\u00BC") // 1/4
						continue
					} else if s[i] == '3' && s[i+2] == '4' {
						s, size = stringReplace(s, i, 3, "\u00BE") // 3/4
						continue
					} else if s[i] == '+' && s[i+2] == '-' {
						s, size = stringReplace(s, i, 3, "\u00B1") // +/-
						continue
					}
				}
			}
		}
	}
	return s, inSingleQuote, inDoubleQuote
}

////////////////////////////////////////////////////////////////

func parseSFNT(b []byte) (string, *sfnt.Font, error) {
	if len(b) < 4 {
		return "", nil, fmt.Errorf("invalid font file")
	}

	mimetype := ""
	tag := string(b[:4])
	if tag == "wOFF" {
		mimetype = "font/woff"
		var err error
		b, err = parseWOFF(b)
		if err != nil {
			return "", nil, err
		}
	} else if tag == "wOF2" {
		mimetype = "font/woff2"
		var err error
		b, err = parseWOFF2(b)
		if err != nil {
			return "", nil, err
		}
	} else if tag == "true" || binary.BigEndian.Uint32(b[:4]) == 0x00010000 {
		mimetype = "font/truetype"
	} else if tag == "OTTO" {
		mimetype = "font/opentype"
	} else {
		// TODO: support EOT?
		return "", nil, fmt.Errorf("unrecognized font file format")
	}

	sfnt, err := sfnt.Parse(b)
	if err != nil {
		return "", nil, err
	}
	return mimetype, sfnt, nil
}

type woffTable struct {
	tag          uint32
	offset       uint32
	length       uint32
	origLength   uint32
	origChecksum uint32
}

func parseWOFF(b []byte) ([]byte, error) {
	if len(b) < 44 {
		return nil, fmt.Errorf("invalid WOFF data")
	}

	p := newPopper(b)
	signature := p.pops(4)
	if signature != "wOFF" {
		return nil, fmt.Errorf("invalid WOFF data")
	}
	flavor := p.pop32()
	_ = p.pop32() // length
	numTables := p.pop16()
	_ = p.pop16() // reserved
	_ = p.pop32() // totalSfntSize
	_ = p.pop16() // majorVersion
	_ = p.pop16() // minorVersion
	_ = p.pop32() // metaOffset
	_ = p.pop32() // metaLength
	_ = p.pop32() // metaOrigLength
	_ = p.pop32() // privOffset
	_ = p.pop32() // privLength

	tables := []woffTable{}
	sfntLength := uint32(12 + 16*int(numTables))
	for i := 0; i < int(numTables); i++ {
		tag := p.pop32()
		offset := p.pop32()
		compLength := p.pop32()
		origLength := p.pop32()
		origChecksum := p.pop32()
		tables = append(tables, woffTable{
			tag:          tag,
			offset:       offset,
			length:       compLength,
			origLength:   origLength,
			origChecksum: origChecksum,
		})

		sfntLength += origLength
		sfntLength = (sfntLength + 3) & 0xFFFFFFFC // add padding
	}

	var searchRange uint16 = 1
	var entrySelector uint16
	var rangeShift uint16
	for {
		if searchRange*2 > numTables {
			break
		}
		searchRange *= 2
		entrySelector++
	}
	searchRange *= 16
	rangeShift = numTables*16 - searchRange

	out := newPusher(make([]byte, sfntLength))
	out.push32(flavor)
	out.push16(numTables)
	out.push16(searchRange)
	out.push16(entrySelector)
	out.push16(rangeShift)

	sfntOffset := uint32(12 + 16*int(numTables))
	for _, table := range tables {
		out.push32(table.tag)
		out.push32(table.origChecksum)
		out.push32(sfntOffset)
		out.push32(table.origLength)
		sfntOffset += table.origLength
		sfntOffset = (sfntOffset + 3) & 0xFFFFFFFC // add padding
	}

	for _, table := range tables {
		data := b[table.offset : table.offset+table.length]
		if table.length != table.origLength {
			var buf bytes.Buffer
			r, _ := zlib.NewReader(bytes.NewReader(data))
			io.Copy(&buf, r)
			r.Close()
			data = buf.Bytes()
		}

		// TODO: check checksum

		if len(data) != int(table.origLength) {
			panic("font data size mismatch")
		}

		out.push(data)
		nPadding := 4 - len(data)%4
		if nPadding == 4 {
			nPadding = 0
		}
		for i := 0; i < nPadding; i++ {
			out.push([]byte{0x00})
		}
	}
	return out.b, nil
}

type woff2Table struct {
	tag              uint32
	origLength       uint32
	transformVersion int
	transformLength  uint32
}

var woff2TableTags = []string{
	"cmap", "head", "hhea", "hmtx",
	"maxp", "name", "OS/2", "post",
	"cvt ", "fpgm", "glyf", "loca",
	"prep", "CFF ", "VORG", "EBDT",
	"EBLC", "gasp", "hdmx", "kern",
	"LTSH", "PCLT", "VDMX", "vhea",
	"vmtx", "BASE", "GDEF", "GPOS",
	"GSUB", "EBSC", "JSTF", "MATH",
	"CBDT", "CBLC", "COLR", "CPAL",
	"SVG ", "sbix", "acnt", "avar",
	"bdat", "bloc", "bsln", "cvar",
	"fdsc", "feat", "fmtx", "fvar",
	"gvar", "hsty", "just", "lcar",
	"mort", "morx", "opbd", "prop",
	"trak", "Zapf", "Silf", "Glat",
	"Gloc", "Feat", "Sill",
}

func parseWOFF2(b []byte) ([]byte, error) {
	if len(b) < 48 {
		return nil, fmt.Errorf("invalid WOFF2 data")
	}

	p := newPopper(b)
	signature := p.pops(4)
	if signature != "wOF2" {
		return nil, fmt.Errorf("invalid WOFF2 data")
	}
	flavor := p.pop32()
	if uint32ToString(flavor) == "ttcf" {
		panic("collections are unsupported")
	}
	_ = p.pop32() // length
	numTables := p.pop16()
	_ = p.pop16()                    // reserved
	_ = p.pop32()                    // totalSfntSize
	totalCompressedSize := p.pop32() // totalCompressedSize
	_ = p.pop16()                    // majorVersion
	_ = p.pop16()                    // minorVersion
	_ = p.pop32()                    // metaOffset
	_ = p.pop32()                    // metaLength
	_ = p.pop32()                    // metaOrigLength
	_ = p.pop32()                    // privOffset
	_ = p.pop32()                    // privLength

	tables := []woff2Table{}
	sfntLength := uint32(12 + 16*int(numTables))
	for i := 0; i < int(numTables); i++ {
		flags := p.pop8()
		tagIndex := int(flags & 0x3F)
		transformVersion := int((flags & 0xC0) >> 5)

		var tag uint32
		if tagIndex == 63 {
			tag = p.pop32()
		} else {
			tag = binary.BigEndian.Uint32([]byte(woff2TableTags[tagIndex]))
		}
		origLength := p.popBase128()

		var transformLength uint32
		if transformVersion == 0 && (tag == binary.BigEndian.Uint32([]byte("glyf")) || tag == binary.BigEndian.Uint32([]byte("loca")) || transformVersion != 0) {
			transformLength = p.popBase128()
		}
		tables = append(tables, woff2Table{
			tag:              tag,
			origLength:       origLength,
			transformVersion: transformVersion,
			transformLength:  transformLength,
		})
		fmt.Println(uint32ToString(tag), origLength, transformLength, transformVersion)

		sfntLength += origLength
		sfntLength = (sfntLength + 3) & 0xFFFFFFFC // add padding
	}

	var searchRange uint16 = 1
	var entrySelector uint16
	var rangeShift uint16
	for {
		if searchRange*2 > numTables {
			break
		}
		searchRange *= 2
		entrySelector++
	}
	searchRange *= 16
	rangeShift = numTables*16 - searchRange

	out := newPusher(make([]byte, sfntLength))
	out.push32(flavor)
	out.push16(numTables)
	out.push16(searchRange)
	out.push16(entrySelector)
	out.push16(rangeShift)

	data := p.pop(int(totalCompressedSize))

	// decompress Brotlu
	var buf bytes.Buffer
	r, _ := brotli.NewReader(bytes.NewReader(data), nil)
	io.Copy(&buf, r)
	r.Close()
	data = buf.Bytes()

	sfntOffset := uint32(12 + 16*int(numTables))
	for _, table := range tables {
		out.push32(table.tag)
		out.push32(0) // TODO: checksum
		out.push32(sfntOffset)
		out.push32(table.origLength)
	}

	var offset uint32
	for _, table := range tables {
		n := table.origLength
		if table.transformLength != 0 {
			n = table.transformLength
		}
		tableData := data[offset : offset+n]
		offset += n

		switch uint32ToString(table.tag) {
		case "glyf":
			if table.transformVersion == 0 {
				panic("WOFF2 transformed glyf table not supported")
				// TODO: see https://www.w3.org/TR/WOFF2/#glyf_table_format
			} else if table.transformVersion != 3 {
				panic("WOFF2 unknown transformation of glyf table")
			}
		case "loca":
			if table.transformVersion == 0 {
				panic("WOFF2 transformed loca table not supported")
				// TODO
			} else if table.transformVersion != 3 {
				panic("WOFF2 unknown transformation of loca table")
			}
		case "hmtx":
			if table.transformVersion == 1 {
				panic("WOFF2 transformed hmtx table not supported")
				// TODO
			} else if table.transformVersion != 0 {
				panic("WOFF2 unknown transformation of hmtx table")
			}
		default:
			if table.transformVersion != 0 {
				panic(fmt.Sprintf("WOFF2 unknown transformation of %s table", uint32ToString(table.tag)))
			}
		}

		out.push(tableData)
		nPadding := 4 - len(tableData)%4
		if nPadding == 4 {
			nPadding = 0
		}
		for i := 0; i < nPadding; i++ {
			out.push([]byte{0x00})
		}
	}
	return out.b, nil
}
