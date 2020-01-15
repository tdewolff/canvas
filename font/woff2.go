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
func ParseWOFF2(b []byte) ([]byte, error) {
	if len(b) < 48 {
		return nil, ErrInvalidFontData
	}

	r := newBinaryReader(b)
	signature := r.ReadString(4)
	if signature != "wOF2" {
		return nil, ErrInvalidFontData
	}
	flavor := r.ReadUint32()
	if uint32ToString(flavor) == "ttcf" {
		return nil, fmt.Errorf("collections are unsupported")
	}
	_ = r.ReadUint32() // length
	numTables := r.ReadUint16()
	_ = r.ReadUint16()                    // reserved
	totalSfntSize := r.ReadUint32()       // totalSfntSize
	totalCompressedSize := r.ReadUint32() // totalCompressedSize
	_ = r.ReadUint16()                    // majorVersion
	_ = r.ReadUint16()                    // minorVersion
	_ = r.ReadUint32()                    // metaOffset
	_ = r.ReadUint32()                    // metaLength
	_ = r.ReadUint32()                    // metaOrigLength
	_ = r.ReadUint32()                    // privOffset
	_ = r.ReadUint32()                    // privLength
	if r.EOF() {
		return nil, ErrInvalidFontData
	}

	tags := []string{}
	tagTableIndex := map[string]int{}
	tables := []woff2Table{}
	var uncompressedSize uint32
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

		origLength, err := readUintBase128(r) // if EOF is encountered above, this will return ErrInvalidFontData
		if err != nil {
			return nil, err
		}

		var transformLength uint32
		if transformVersion == 0 && (tag == "glyf" || tag == "loca") || transformVersion != 0 {
			transformLength, err = readUintBase128(r)
			if err != nil {
				return nil, err
			}
			uncompressedSize += transformLength
		} else {
			uncompressedSize += origLength
		}

		if tag == "loca" {
			if transformLength != 0 {
				return nil, fmt.Errorf("loca: transformLength must be zero")
			}
			if _, ok := tagTableIndex["glyf"]; !ok {
				return nil, fmt.Errorf("loca: must follow 'glyf' table")
			}
		}

		tags = append(tags, tag)
		tagTableIndex[tag] = len(tables)
		tables = append(tables, woff2Table{
			tag:              tag,
			origLength:       origLength,
			transformVersion: transformVersion,
			transformLength:  transformLength,
		})
	}

	// TODO: (WOFF2) parse collection directory format

	// decompress font data using Brotli
	data := r.ReadBytes(totalCompressedSize)
	if r.EOF() {
		return nil, ErrInvalidFontData
	}

	var dataBuf bytes.Buffer
	rBrotli, _ := brotli.NewReader(bytes.NewReader(data), nil) // err is always nil
	io.Copy(&dataBuf, rBrotli)
	if err := rBrotli.Close(); err != nil {
		return nil, fmt.Errorf("brotli: %v", err)
	}

	data = dataBuf.Bytes()
	if uint32(len(data)) != uncompressedSize {
		return nil, ErrInvalidFontData
	}

	// read font data
	var offset uint32
	for i, _ := range tables {
		if tables[i].tag == "loca" && tables[i].transformVersion == 0 {
			continue // will be reconstructed
		}

		n := tables[i].origLength
		if tables[i].transformLength != 0 {
			n = tables[i].transformLength
		}
		tables[i].data = data[offset : offset+n : offset+n]
		offset += n

		switch tables[i].tag {
		case "glyf":
			if tables[i].transformVersion != 0 && tables[i].transformVersion != 3 {
				return nil, fmt.Errorf("glyf: unknown transformation")
			}
		case "loca":
			if tables[i].transformVersion != 0 && tables[i].transformVersion != 3 {
				return nil, fmt.Errorf("loca: unknown transformation")
			}
		case "hmtx":
			if tables[i].transformVersion != 0 && tables[i].transformVersion != 1 {
				return nil, fmt.Errorf("htmx: unknown transformation")
			}
		default:
			if tables[i].transformVersion != 0 {
				return nil, fmt.Errorf("%s: unknown transformation", tables[i].tag)
			}
		}
	}

	// detransform font data tables
	iGlyf, hasGlyf := tagTableIndex["glyf"]
	iLoca, hasLoca := tagTableIndex["loca"]
	if hasGlyf != hasLoca || tables[iGlyf].transformVersion != tables[iLoca].transformVersion {
		return nil, fmt.Errorf("glyf and loca tables must be both present and either be both transformed or not")
	}
	if hasGlyf {
		if tables[iGlyf].transformVersion == 0 {
			var err error
			tables[iGlyf].data, tables[iLoca].data, err = parseGlyfTransformed(tables[iGlyf].data)
			if err != nil {
				return nil, err
			}
			if tables[iLoca].origLength != uint32(len(tables[iLoca].data)) {
				return nil, fmt.Errorf("loca: invalid value for origLength")
			}
		} else {
			rGlyf := newBinaryReader(tables[iGlyf].data)
			_ = rGlyf.ReadUint32() // version
			numGlyphs := uint32(rGlyf.ReadUint16())
			indexFormat := rGlyf.ReadUint16()
			if rGlyf.EOF() {
				return nil, ErrInvalidFontData
			}
			if indexFormat == 0 && tables[iLoca].origLength != (numGlyphs+1)*2 || indexFormat == 1 && tables[iLoca].origLength != (numGlyphs+1)*4 {
				return nil, fmt.Errorf("loca: invalid value for origLength")
			}
		}
	}

	if iHmtx, hasHmtx := tagTableIndex["hmtx"]; hasHmtx && tables[iHmtx].transformVersion == 1 {
		iMaxp, ok := tagTableIndex["maxp"]
		if !ok {
			return nil, fmt.Errorf("hmtx: maxp table must be defined in order to rebuild hmtx table")
		}
		iHhea, ok := tagTableIndex["hhea"]
		if !ok {
			return nil, fmt.Errorf("hmtx: hhea table must be defined in order to rebuild hmtx table")
		}
		var err error
		tables[iHmtx].data, err = parseHmtxTransformed(tables[iHmtx].data, tables[iGlyf].data, tables[iMaxp].data, tables[iHhea].data)
		if err != nil {
			return nil, err
		}
	}

	// set checkSumAdjustment to zero to enable calculation of table checksum and overal checksum
	// also clear 11th bit in flags field
	iHead, hasHead := tagTableIndex["head"]
	if !hasHead || len(tables[iHead].data) < 18 {
		return nil, fmt.Errorf("head: must be present")
	} else {
		binary.BigEndian.PutUint32(tables[iHead].data[8:], 0x00000000) // clear checkSumAdjustment

		flags := binary.BigEndian.Uint16(tables[iHead].data[16:])
		flags &= ^(uint16(1 << 11)) // clear bit 11
		binary.BigEndian.PutUint16(tables[iHead].data[16:], flags)
	}

	// remove DSIG table
	if iDSIG, hasDSIG := tagTableIndex["DSIG"]; hasDSIG {
		tags = append(tags[:iDSIG], tags[iDSIG+1:]...)
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
	w := newBinaryWriter(make([]byte, totalSfntSize)) // initial guess, will be bigger
	w.WriteUint32(flavor)
	w.WriteUint16(numTables)
	w.WriteUint16(searchRange)
	w.WriteUint16(entrySelector)
	w.WriteUint16(rangeShift)

	// write table record entries, sorted alphabetically
	sort.Strings(tags)
	sfntOffset := uint32(12 + 16*int(numTables))
	for _, tag := range tags {
		i := tagTableIndex[tag]
		actualLength := len(tables[i].data)

		// add padding
		nPadding := (4 - actualLength&3) & 3
		for j := 0; j < nPadding; j++ {
			tables[i].data = append(tables[i].data, 0x00)
		}

		w.WriteUint32(binary.BigEndian.Uint32([]byte(tables[i].tag)))
		w.WriteUint32(calcChecksum(tables[i].data))
		w.WriteUint32(sfntOffset)
		w.WriteUint32(uint32(actualLength))
		sfntOffset += uint32(len(tables[i].data))
	}

	// write tables
	var iCheckSumAdjustment uint32
	for _, tag := range tags {
		if tag == "head" {
			iCheckSumAdjustment = w.Len() + 8
		}
		table := tables[tagTableIndex[tag]]
		w.WriteBytes(table.data)
	}

	buf := w.Bytes()
	checkSumAdjustment := 0xB1B0AFBA - calcChecksum(buf)
	binary.BigEndian.PutUint32(buf[iCheckSumAdjustment:], checkSumAdjustment)
	return buf, nil
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
	if r.EOF() || nContourStreamSize != 2*uint32(numGlyphs) {
		return nil, nil, ErrInvalidFontData
	}

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

		nContours := nContourStream.ReadInt16() // EOF cannot occur

		explicitBbox := bboxBitmap.Read() // EOF cannot occur
		if nContours == 0 && explicitBbox {
			return nil, nil, fmt.Errorf("glyf: empty glyph cannot have bbox definition")
		} else if nContours == -1 && !explicitBbox {
			return nil, nil, fmt.Errorf("glyf: composite glyph must have bbox definition")
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
				if bboxStream.EOF() {
					return nil, nil, ErrInvalidFontData
				}
			}

			var nPoints uint16
			endPtsOfContours := make([]uint16, nContours)
			for iContour := int16(0); iContour < nContours; iContour++ {
				nPoint := read255Uint16(nPointsStream)
				nPoints += nPoint
				endPtsOfContours[iContour] = nPoints - 1
			}
			if nPointsStream.EOF() {
				return nil, nil, ErrInvalidFontData
			}

			signInt16 := func(flag byte, pos int) int16 {
				if flag&(1<<uint(pos)) != 0 {
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
			if flagStream.EOF() || glyphStream.EOF() {
				return nil, nil, ErrInvalidFontData
			}

			instructionLength := read255Uint16(glyphStream)
			instructions := instructionStream.ReadBytes(uint32(instructionLength))
			if instructionStream.EOF() {
				return nil, nil, ErrInvalidFontData
			}

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
			if bboxStream.EOF() {
				return nil, nil, ErrInvalidFontData
			}

			// write composite glyph definition
			w.WriteInt16(nContours) // numberOfContours
			w.WriteInt16(xMin)
			w.WriteInt16(yMin)
			w.WriteInt16(xMax)
			w.WriteInt16(yMax)

			hasInstructions := false
			for {
				compositeFlag := compositeStream.ReadUint16()
				argsAreWords := (compositeFlag & 0x0001) != 0
				haveScale := (compositeFlag & 0x0008) != 0
				moreComponents := (compositeFlag & 0x0020) != 0
				haveXYScales := (compositeFlag & 0x0040) != 0
				have2by2 := (compositeFlag & 0x0080) != 0
				haveInstructions := (compositeFlag & 0x0100) != 0

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
				if compositeStream.EOF() {
					return nil, nil, ErrInvalidFontData
				}

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
				if instructionStream.EOF() {
					return nil, nil, ErrInvalidFontData
				}
				w.WriteUint16(instructionLength)
				w.WriteBytes(instructions)
			}
		}

		// padding to allow short version for loca table
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

func parseHmtxTransformed(b []byte, glyf []byte, maxp []byte, hhea []byte) ([]byte, error) {
	r := newBinaryReader(b)
	flags := r.ReadByte() // flags
	if flags&0xFC != 0 {
		return nil, fmt.Errorf("hmtx: the flag's 6 most significant bits must be zero")
	}
	reconstructLsb := flags&0x01 != 0
	reconstructLeftSideBearing := flags&0x02 != 0
	if !reconstructLsb && !reconstructLeftSideBearing {
		return nil, fmt.Errorf("hmtx: must either reconstruct lsb or LeftSideBearing arrays")
	}

	// get numGlyphs
	rMaxp := newBinaryReader(maxp)
	_ = rMaxp.ReadUint32() // version
	numGlyphs := uint32(rMaxp.ReadUint16())
	if rMaxp.EOF() {
		return nil, ErrInvalidFontData
	}

	// get numHMetrics
	rHhea := newBinaryReader(hhea)
	_ = rHhea.ReadBytes(34) // skip all but the last header field
	numHMetrics := uint32(rHhea.ReadUint16())
	if rHhea.EOF() {
		return nil, ErrInvalidFontData
	}

	n := numHMetrics * 2
	if !reconstructLsb {
		n += numHMetrics * 2
	} else if !reconstructLeftSideBearing {
		n += (numGlyphs - numHMetrics) * 2
	}
	fmt.Println("hmtx", n, "==?", r.Len())
	if n != r.Len() {
		return nil, ErrInvalidFontData
	}

	w := newBinaryWriter(make([]byte, 0))

	advanceWidths := make([]uint16, numHMetrics)
	for iHMetric := uint32(0); iHMetric < numHMetrics; iHMetric++ {
		advanceWidths[iHMetric] = r.ReadUint16()
	}
	if !reconstructLsb {
		for iHMetric := uint32(0); iHMetric < numHMetrics; iHMetric++ {
			w.WriteUint16(advanceWidths[iHMetric])
			w.WriteInt16(r.ReadInt16()) // lsb
		}
	}

	rGlyf := newBinaryReader(glyf)
	for iGlyph := uint32(0); iGlyph < numGlyphs; iGlyph++ {
		numContours := rGlyf.ReadInt16()
		xMin := rGlyf.ReadInt16()
		if reconstructLsb && iGlyph < numHMetrics {
			w.WriteUint16(advanceWidths[iGlyph])
			w.WriteInt16(xMin) // lsb
		} else if reconstructLeftSideBearing && numHMetrics <= iGlyph {
			w.WriteInt16(xMin) // leftSideBearing
		}

		// skip through rest of glyf table
		_ = rGlyf.ReadBytes(6) // yMin, xMax, yMax
		if 0 < numContours {
			_ = rGlyf.ReadBytes(2 * uint32(numContours)) // endPtsOfContours except last
			numPoints := rGlyf.ReadUint16() + 1
			instructionLength := rGlyf.ReadUint16()
			_ = rGlyf.ReadBytes(uint32(instructionLength)) // instructions

			var xLength, yLength uint32
			for iPoint := uint16(0); iPoint < numPoints; iPoint++ {
				flag := rGlyf.ReadByte()
				xShort := (flag & 0x02) != 0
				yShort := (flag & 0x04) != 0
				repeat := (flag & 0x08) != 0
				xSame := (flag & 0x10) != 0
				ySame := (flag & 0x20) != 0

				var dx, dy uint32
				if xShort {
					dx = 1
				} else if !xSame {
					dx = 2
				}
				if yShort {
					dy = 1
				} else if !ySame {
					dy = 2
				}
				if repeat {
					n := rGlyf.ReadByte()
					dx *= uint32(n)
					dy *= uint32(n)
					iPoint += uint16(n)
				}
				xLength += dx
				yLength += dy
			}
			_ = rGlyf.ReadBytes(xLength) // xCoordinates
			_ = rGlyf.ReadBytes(yLength) // yCoordinates
		} else if numContours < 0 {
			for {
				compositeFlag := rGlyf.ReadUint16()
				argsAreWords := (compositeFlag & 0x0001) != 0
				haveScale := (compositeFlag & 0x0008) != 0
				moreComponents := (compositeFlag & 0x0020) != 0
				haveXYScales := (compositeFlag & 0x0040) != 0
				have2by2 := (compositeFlag & 0x0080) != 0

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
				_ = rGlyf.ReadBytes(uint32(numBytes))
				if !moreComponents {
					break
				}
			}
		}
		if rGlyf.EOF() {
			return nil, ErrInvalidFontData
		}
	}
	fmt.Println("hmtx left:", r.Len())
	return w.Bytes(), nil
}

func readUintBase128(r *binaryReader) (uint32, error) {
	// see https://www.w3.org/TR/WOFF2/#DataTypes
	var accum uint32
	for i := 0; i < 5; i++ {
		dataByte := r.ReadByte()
		if r.EOF() {
			return 0, ErrInvalidFontData
		}
		if i == 0 && dataByte == 0x80 {
			return 0, fmt.Errorf("readUintBase128: must not start with leading zeros")
		}
		if (accum & 0xFE000000) != 0 {
			return 0, fmt.Errorf("readUintBase128: overflow")
		}
		accum = (accum << 7) | uint32(dataByte&0x7F)
		if (dataByte & 0x80) == 0 {
			return accum, nil
		}
	}
	return 0, fmt.Errorf("readUintBase128: exceeds 5 bytes")
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
