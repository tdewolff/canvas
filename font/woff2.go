package font

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"sort"

	"github.com/dsnet/compress/brotli"
)

type woff2Table struct {
	tag              string
	origLength       uint32
	transformVersion int
	transformLength  uint32
	data             []byte
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

// ParseWOFF2 parses the WOFF2 font format and returns its contained SFNT font format (TTF or OTF).
// See https://www.w3.org/TR/WOFF2/
func ParseWOFF2(b []byte) ([]byte, uint32, error) {
	// TODO: (WOFF2) could be stricter with parsing, spec is clear when the font should be dismissed
	if len(b) < 48 {
		return nil, 0, fmt.Errorf("invalid data")
	}

	r := newBinaryReader(b)
	signature := r.ReadString(4)
	if signature != "wOF2" {
		return nil, 0, fmt.Errorf("invalid data")
	}
	flavor := r.ReadUint32()
	if uint32ToString(flavor) == "ttcf" {
		return nil, 0, fmt.Errorf("collections are unsupported")
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

	tags := []string{}
	tagTableIndex := map[string]int{}
	tables := []woff2Table{}
	sfntLength := uint32(12 + 16*int(numTables))
	for i := 0; i < int(numTables); i++ {
		flags := r.ReadByte()
		tagIndex := int(flags & 0x3F)
		transformVersion := int((flags & 0xC0) >> 5)

		var tag string
		if tagIndex == 63 {
			tag = uint32ToString(r.ReadUint32())
		} else {
			tag = woff2TableTags[tagIndex]
		}
		origLength := readBase128(r)

		var transformLength uint32
		if transformVersion == 0 && (tag == "glyf" || tag == "loca" || transformVersion != 0) {
			transformLength = readBase128(r)
		}

		if transformVersion == 0 && tag == "loca" && transformLength != 0 {
			return nil, 0, fmt.Errorf("loca: transformLength must be zero")
		}

		tags = append(tags, tag)
		tagTableIndex[tag] = len(tables)
		tables = append(tables, woff2Table{
			tag:              tag,
			origLength:       origLength,
			transformVersion: transformVersion,
			transformLength:  transformLength,
		})

		sfntLength += origLength
		sfntLength = (sfntLength + 3) & 0xFFFFFFFC // add padding
	}

	// decompress font data using Brotli
	var buf bytes.Buffer
	data := r.ReadBytes(totalCompressedSize)
	rBrotli, _ := brotli.NewReader(bytes.NewReader(data), nil)
	io.Copy(&buf, rBrotli)
	rBrotli.Close()
	data = buf.Bytes()

	// read font data and detransform
	var offset uint32
	for i, _ := range tables {
		if tables[i].tag == "loca" && tables[i].transformVersion == 0 {
			continue // already handled
		}

		n := tables[i].origLength
		if tables[i].transformLength != 0 {
			n = tables[i].transformLength
		}
		tables[i].data = data[offset : offset+n]
		offset += n

		switch tables[i].tag {
		case "glyf":
			if tables[i].transformVersion == 0 {
				var locaData []byte
				var err error
				tables[i].data, locaData, err = parseGlyfTransformed(tables[i].data)
				if err != nil {
					return nil, 0, err
				}
				tables[tagTableIndex["loca"]].data = locaData
			} else if tables[i].transformVersion != 3 {
				return nil, 0, fmt.Errorf("glyf: unknown transformation")
			}
		case "loca":
			if tables[i].transformVersion != 0 && tables[i].transformVersion != 3 {
				return nil, 0, fmt.Errorf("loca: unknown transformation")
			}
		case "hmtx":
			if tables[i].transformVersion == 1 {
				panic("WOFF2 transformed hmtx table not supported")
				// TODO: (WOFF2) support hmtx table transformation
			} else if tables[i].transformVersion != 0 {
				return nil, 0, fmt.Errorf("htmx: unknown transformation")
			}
		default:
			if tables[i].transformVersion != 0 {
				return nil, 0, fmt.Errorf("%s: unknown transformation", tables[i].tag)
			}
		}
	}

	// find values for offset table
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

	// write offset table
	w := newBinaryWriter(make([]byte, sfntLength))
	w.WriteUint32(flavor)
	w.WriteUint16(numTables)
	w.WriteUint16(searchRange)
	w.WriteUint16(entrySelector)
	w.WriteUint16(rangeShift)

	// write table record entries, sorted alphabetically
	sort.Strings(tags)
	sfntOffset := uint32(12 + 16*int(numTables))
	for _, tag := range tags {
		table := tables[tagTableIndex[tag]]
		length := uint32(len(table.data))
		w.WriteUint32(binary.BigEndian.Uint32([]byte(table.tag)))
		w.WriteUint32(0) // TODO: (WOFF2) check checksum
		w.WriteUint32(sfntOffset)
		w.WriteUint32(length)
		sfntOffset += length + uint32((4-len(table.data)&3)&3)
	}

	// write tables
	for _, tag := range tags {
		table := tables[tagTableIndex[tag]]
		w.WriteBytes(table.data)

		// add padding
		nPadding := (4 - len(table.data)&3) & 3
		for i := 0; i < nPadding; i++ {
			w.WriteByte(0x00)
		}
	}
	return w.Bytes(), flavor, nil
}

// Remarkable! This code was written on a Sunday evening, and after fixing the compiler errors it worked flawlessly!
func parseGlyfTransformed(b []byte) ([]byte, []byte, error) {
	r := newBinaryReader(b)
	_ = r.ReadUint32() // version
	numGlyphs := r.ReadUint16()
	indexFormat := r.ReadUint16()
	nContourStreamSize := r.ReadUint32()
	nPointsStreamSize := r.ReadUint32()
	flagStreamSize := r.ReadUint32()
	glyphStreamSize := r.ReadUint32()
	compositeStreamSize := r.ReadUint32()
	bboxStreamSize := r.ReadUint32()
	instructionStreamSize := r.ReadUint32()

	bboxBitmapSize := ((uint32(numGlyphs) + 31) >> 5) << 2
	nContourStream := newBinaryReader(r.ReadBytes(nContourStreamSize))
	nPointsStream := newBinaryReader(r.ReadBytes(nPointsStreamSize))
	flagStream := newBinaryReader(r.ReadBytes(flagStreamSize))
	glyphStream := newBinaryReader(r.ReadBytes(glyphStreamSize))
	compositeStream := newBinaryReader(r.ReadBytes(compositeStreamSize))
	bboxBitmap := newBitmapReader(r.ReadBytes(bboxBitmapSize))
	bboxStream := newBinaryReader(r.ReadBytes(bboxStreamSize - bboxBitmapSize))
	instructionStream := newBinaryReader(r.ReadBytes(instructionStreamSize))

	w := newBinaryWriter(make([]byte, 0))
	loca := newBinaryWriter(make([]byte, 0))
	for iGlyph := uint16(0); iGlyph < numGlyphs; iGlyph++ {
		if indexFormat == 0 {
			loca.WriteUint16(uint16(w.Len() / 2))
		} else {
			loca.WriteUint32(w.Len())
		}

		nContours := nContourStream.ReadInt16()

		explicitBbox := bboxBitmap.Read()
		if nContours == 0 && explicitBbox {
			return nil, nil, fmt.Errorf("glyf: empty glyph cannot have explicit bbox definition")
		} else if nContours == -1 && !explicitBbox {
			return nil, nil, fmt.Errorf("glyf: composite glyph must have explicit bbox definition")
		}

		if nContours == 0 { // empty glyph
			continue
		} else if nContours > 0 { // simple glyph
			var xMin, yMin, xMax, yMax int16
			if explicitBbox {
				xMin = bboxStream.ReadInt16()
				yMin = bboxStream.ReadInt16()
				xMax = bboxStream.ReadInt16()
				yMax = bboxStream.ReadInt16()
			}

			var nPoints uint16
			endPtsOfContours := make([]uint16, nContours)
			for iContour := int16(0); iContour < nContours; iContour++ {
				nPoint := read255Uint16(nPointsStream)
				nPoints += nPoint
				endPtsOfContours[iContour] = nPoints - 1
			}

			signInt16 := func(flag byte, pos int) int16 {
				if flag&(1<<pos) != 0 {
					return 1 // positive if bit on position is set
				}
				return -1
			}

			var x, y int16
			outlineFlags := make([]byte, 0, nPoints)
			xCoordinates := make([]int16, 0, nPoints)
			yCoordinates := make([]int16, 0, nPoints)
			for iPoint := uint16(0); iPoint < nPoints; iPoint++ {
				flag := flagStream.ReadByte()
				onCurve := (flag & 0x80) == 0 // unclear in spec, but it is opposite to non-transformed glyf table
				flag &= 0x7f

				// used for reference: https://github.com/fonttools/fonttools/blob/master/Lib/fontTools/ttLib/woff2.py
				var dx, dy int16
				if flag < 10 {
					coord0 := int16(glyphStream.ReadByte())
					dy = signInt16(flag, 0) * (int16(flag&0x0E)<<7 + coord0)
				} else if flag < 20 {
					coord0 := int16(glyphStream.ReadByte())
					dx = signInt16(flag, 0) * (int16((flag-10)&0x0E)<<7 + coord0)
				} else if flag < 84 {
					coord0 := int16(glyphStream.ReadByte())
					dx = signInt16(flag, 0) * (1 + int16((flag-20)&0x30) + coord0>>4)
					dy = signInt16(flag, 1) * (1 + int16((flag-20)&0x0C)<<2 + (coord0 & 0x0F))
				} else if flag < 120 {
					coord0 := int16(glyphStream.ReadByte())
					coord1 := int16(glyphStream.ReadByte())
					dx = signInt16(flag, 0) * (1 + int16((flag-84)/12) + coord0)
					dy = signInt16(flag, 1) * (1 + (int16((flag-84)%12)>>2)<<8 + coord1)
				} else if flag < 124 {
					coord0 := int16(glyphStream.ReadByte())
					coord1 := int16(glyphStream.ReadByte())
					coord2 := int16(glyphStream.ReadByte())
					dx = signInt16(flag, 0) * (coord0<<4 + coord1>>4)
					dy = signInt16(flag, 1) * ((coord1&0x0F)<<8 + coord2)
				} else {
					coord0 := int16(glyphStream.ReadByte())
					coord1 := int16(glyphStream.ReadByte())
					coord2 := int16(glyphStream.ReadByte())
					coord3 := int16(glyphStream.ReadByte())
					dx = signInt16(flag, 0) * (coord0<<8 + coord1)
					dy = signInt16(flag, 1) * (coord2<<8 + coord3)
				}
				xCoordinates = append(xCoordinates, dx)
				yCoordinates = append(yCoordinates, dy)

				// only the OnCurve bit is set, all others zero. That means x and y are two bytes long, this flag is not repeated,
				// and all coordinates are in the coordinate array even if they are the same as the previous.
				if onCurve {
					outlineFlags = append(outlineFlags, 0x01)
				} else {
					outlineFlags = append(outlineFlags, 0x00)
				}

				// calculate bbox
				if !explicitBbox {
					x += dx
					y += dy
					if iPoint == 0 {
						xMin, xMax = x, x
						yMin, yMax = y, y
					} else {
						if x < xMin {
							xMin = x
						} else if xMax < x {
							xMax = x
						}
						if y < yMin {
							yMin = y
						} else if yMax < y {
							yMax = y
						}
					}
				}
			}

			instructionLength := read255Uint16(glyphStream)
			instructions := instructionStream.ReadBytes(uint32(instructionLength))

			// write simple glyph definition
			w.WriteInt16(nContours) // numberOfContours
			w.WriteInt16(xMin)
			w.WriteInt16(yMin)
			w.WriteInt16(xMax)
			w.WriteInt16(yMax)
			for _, endPtsOfContour := range endPtsOfContours {
				w.WriteUint16(endPtsOfContour)
			}
			w.WriteUint16(instructionLength)
			w.WriteBytes(instructions)
			for _, outlineFlag := range outlineFlags {
				w.WriteByte(outlineFlag) // flag
			}
			for _, xCoordinate := range xCoordinates {
				w.WriteInt16(xCoordinate)
			}
			for _, yCoordinate := range yCoordinates {
				w.WriteInt16(yCoordinate)
			}
		} else { // composite glyph
			xMin := bboxStream.ReadInt16()
			yMin := bboxStream.ReadInt16()
			xMax := bboxStream.ReadInt16()
			yMax := bboxStream.ReadInt16()

			// write composite glyph definition
			w.WriteInt16(nContours) // numberOfContours
			w.WriteInt16(xMin)
			w.WriteInt16(yMin)
			w.WriteInt16(xMax)
			w.WriteInt16(yMax)

			hasInstructions := false
			for {
				compositeFlag := compositeStream.ReadUint16()
				argsAreWords := ((compositeFlag >> 0) & 0x01) == 1
				haveScale := ((compositeFlag >> 3) & 0x01) == 1
				moreComponents := ((compositeFlag >> 5) & 0x01) == 1
				haveXYScales := ((compositeFlag >> 6) & 0x01) == 1
				have2by2 := ((compositeFlag >> 7) & 0x01) == 1
				haveInstructions := ((compositeFlag >> 8) & 0x01) == 1

				numBytes := 4 // 2 for glyphIndex and 2 for XY bytes
				if argsAreWords {
					numBytes += 2
				}
				if haveScale {
					numBytes += 2
				} else if haveXYScales {
					numBytes += 4
				} else if have2by2 {
					numBytes += 8
				}
				compositeBytes := compositeStream.ReadBytes(uint32(numBytes))

				w.WriteUint16(compositeFlag)
				w.WriteBytes(compositeBytes)

				if haveInstructions {
					hasInstructions = true
				}
				if !moreComponents {
					break
				}
			}

			if hasInstructions {
				instructionLength := read255Uint16(glyphStream)
				instructions := instructionStream.ReadBytes(uint32(instructionLength))
				w.WriteUint16(instructionLength)
				w.WriteBytes(instructions)
			}
		}

		// padding to allow shorrt version for loca table
		if w.Len()%2 == 1 {
			w.WriteByte(0x00)
		}
	}

	// last entry in loca table
	if indexFormat == 0 {
		loca.WriteUint16(uint16(w.Len() / 2))
	} else {
		loca.WriteUint32(w.Len())
	}

	return w.Bytes(), loca.Bytes(), nil
}

func readBase128(r *binaryReader) uint32 {
	// see https://www.w3.org/TR/WOFF2/#DataTypes
	var accum uint32
	for i := 0; i < 5; i++ {
		dataByte := r.ReadByte()
		if i == 0 && dataByte == 0x80 {
			return 0
		}
		if (accum & 0xFE000000) != 0 {
			return 0
		}
		accum = (accum << 7) | uint32(dataByte&0x7F)
		if (dataByte & 0x80) == 0 {
			return accum
		}
	}
	return 0
}

func read255Uint16(r *binaryReader) uint16 {
	// see https://www.w3.org/TR/WOFF2/#DataTypes
	code := r.ReadByte()
	if code == 253 {
		return r.ReadUint16()
	} else if code == 255 {
		return uint16(r.ReadByte()) + 253
	} else if code == 254 {
		return uint16(r.ReadByte()) + 253*2
	} else {
		return uint16(code)
	}
}
