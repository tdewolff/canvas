package canvas

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/dsnet/compress/brotli"
	"golang.org/x/image/font/sfnt"
)

func uint32ToString(v uint32) string {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return string(b)
}

type popper struct {
	b []byte
	i int
}

func newPopper(b []byte) *popper {
	return &popper{b, 0}
}

func (p *popper) pop(n int) []byte {
	b := p.b[p.i : p.i+n]
	p.i += n
	return b
}

func (p *popper) pops(n int) string {
	return string(p.pop(n))
}

func (p *popper) pop8() byte {
	return p.pop(1)[0]
}

func (p *popper) pop16() uint16 {
	return binary.BigEndian.Uint16(p.pop(2))
}

func (p *popper) pop32() uint32 {
	return binary.BigEndian.Uint32(p.pop(4))
}

func (p *popper) popBase128() uint32 {
	var accum uint32
	for i := 0; i < 5; i++ {
		dataByte := p.b[p.i+i]
		if i == 0 && dataByte == 0x80 {
			return 0
		}
		if (accum & 0xFE000000) != 0 {
			return 0
		}
		accum = (accum << 7) | uint32(dataByte&0x7F)
		if (dataByte & 0x80) == 0 {
			p.i += i + 1
			return accum
		}
	}
	return 0
}

type pusher struct {
	b []byte
	i int
}

func newPusher(b []byte) *pusher {
	return &pusher{b, 0}
}

func (p *pusher) push(v []byte) {
	p.i += copy(p.b[p.i:], v)
}

func (p *pusher) pushs(v string) {
	p.push([]byte(v))
}

func (p *pusher) push16(v uint16) {
	binary.BigEndian.PutUint16(p.b[p.i:], v)
	p.i += 2
}

func (p *pusher) push32(v uint32) {
	binary.BigEndian.PutUint32(p.b[p.i:], v)
	p.i += 4
}

func parseFont(b []byte) (string, *sfnt.Font, error) {
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
		fmt.Println(uint32ToString(tag), origLength)
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
