package font

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/dsnet/compress/brotli"
)

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

func parseWOFF2(b []byte) (*Font, error) {
	if len(b) < 48 {
		return nil, fmt.Errorf("invalid WOFF2 data")
	}

	r := newBinaryReader(b)
	signature := r.ReadString(4)
	if signature != "wOF2" {
		return nil, fmt.Errorf("invalid WOFF2 data")
	}
	flavor := r.ReadUint32()
	if uint32ToString(flavor) == "ttcf" {
		panic("collections are unsupported")
	}
	_ = r.ReadUint32() // length
	numTables := r.ReadUint16()
	_ = r.ReadUint16()                    // reserved
	_ = r.ReadUint32()                    // totalSfntSize
	totalCompressedSize := r.ReadUint32() // totalCompressedSize
	_ = r.ReadUint16()                    // majorVersion
	_ = r.ReadUint16()                    // minorVersion
	_ = r.ReadUint32()                    // metaOffset
	_ = r.ReadUint32()                    // metaLength
	_ = r.ReadUint32()                    // metaOrigLength
	_ = r.ReadUint32()                    // privOffset
	_ = r.ReadUint32()                    // privLength

	tables := []woff2Table{}
	sfntLength := uint32(12 + 16*int(numTables))
	for i := 0; i < int(numTables); i++ {
		flags := r.ReadByte()
		tagIndex := int(flags & 0x3F)
		transformVersion := int((flags & 0xC0) >> 5)

		var tag uint32
		if tagIndex == 63 {
			tag = r.ReadUint32()
		} else {
			tag = binary.BigEndian.Uint32([]byte(woff2TableTags[tagIndex]))
		}
		origLength := r.ReadBase128()

		var transformLength uint32
		if transformVersion == 0 && (tag == binary.BigEndian.Uint32([]byte("glyf")) || tag == binary.BigEndian.Uint32([]byte("loca")) || transformVersion != 0) {
			transformLength = r.ReadBase128()
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

	w := newBinaryWriter(make([]byte, sfntLength))
	w.WriteUint32(flavor)
	w.WriteUint16(numTables)
	w.WriteUint16(searchRange)
	w.WriteUint16(entrySelector)
	w.WriteUint16(rangeShift)

	data := r.ReadBytes(int(totalCompressedSize))

	// decompress Brotli
	var buf bytes.Buffer
	rBrotli, _ := brotli.NewReader(bytes.NewReader(data), nil)
	io.Copy(&buf, rBrotli)
	rBrotli.Close()
	data = buf.Bytes()

	sfntOffset := uint32(12 + 16*int(numTables))
	for _, table := range tables {
		w.WriteUint32(table.tag)
		w.WriteUint32(0) // TODO: (WOFF2) checksum
		w.WriteUint32(sfntOffset)
		w.WriteUint32(table.origLength)
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
				// TODO: (WOFF2) see https://www.w3.org/TR/WOFF2/#glyf_table_format
			} else if table.transformVersion != 3 {
				panic("WOFF2 unknown transformation of glyf table")
			}
		case "loca":
			if table.transformVersion == 0 {
				panic("WOFF2 transformed loca table not supported")
				// TODO: (WOFF2)
			} else if table.transformVersion != 3 {
				panic("WOFF2 unknown transformation of loca table")
			}
		case "hmtx":
			if table.transformVersion == 1 {
				panic("WOFF2 transformed hmtx table not supported")
				// TODO: (WOFF2)
			} else if table.transformVersion != 0 {
				panic("WOFF2 unknown transformation of hmtx table")
			}
		default:
			if table.transformVersion != 0 {
				panic(fmt.Sprintf("WOFF2 unknown transformation of %s table", uint32ToString(table.tag)))
			}
		}

		w.WriteBytes(tableData)
		nPadding := 4 - len(tableData)%4
		if nPadding == 4 {
			nPadding = 0
		}
		for i := 0; i < nPadding; i++ {
			w.WriteByte(0x00)
		}
	}
	return ParseSFNT(w.Bytes())
}
