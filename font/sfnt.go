package font

import (
	"encoding/binary"
	"fmt"
	"math"
	"runtime/debug"
	"sort"
	"strings"
	"time"
)

const MaxCmapSegments = 20000

type SFNT struct {
	Data              []byte
	IsCFF, IsTrueType bool // only one can be true
	Tables            map[string][]byte

	// required
	Cmap *cmapTable
	Head *headTable
	Hhea *hheaTable
	Hmtx *hmtxTable
	Maxp *maxpTable
	Name *nameTable
	OS2  *os2Table
	Post *postTable

	// TrueType
	Glyf *glyfTable
	Loca *locaTable

	// CFF
	//CFF  *cffTable

	// optional
	Kern *kernTable
	//Gpos *gposTable
	//Gasp *gaspTable

}

func (sfnt *SFNT) GlyphIndex(r rune) uint16 {
	return sfnt.Cmap.Get(r)
}

func (sfnt *SFNT) GlyphName(glyphID uint16) string {
	return sfnt.Post.Get(glyphID)
}

func (sfnt *SFNT) GlyphContour(glyphID uint16) (*glyfContour, error) {
	if !sfnt.IsTrueType {
		return nil, fmt.Errorf("CFF not supported")
	}
	return sfnt.Glyf.Contour(glyphID, 0)
}

func (sfnt *SFNT) GlyphAdvance(glyphID uint16) uint16 {
	return sfnt.Hmtx.Advance(glyphID)
}

func (sfnt *SFNT) Kerning(left, right uint16) int16 {
	return sfnt.Kern.Get(left, right)
}

func ParseSFNT(b []byte) (*SFNT, error) {
	if len(b) < 12 || math.MaxUint32 < len(b) {
		return nil, ErrInvalidFontData
	}

	r := newBinaryReader(b)
	sfntVersion := r.ReadString(4)
	if sfntVersion != "OTTO" && binary.BigEndian.Uint32([]byte(sfntVersion)) != 0x00010000 {
		return nil, fmt.Errorf("bad SFNT version")
	}
	numTables := r.ReadUint16()
	_ = r.ReadUint16() // searchRange
	_ = r.ReadUint16() // entrySelector
	_ = r.ReadUint16() // rangeShift

	frontSize := 12 + 16*uint32(numTables) // can never exceed uint32 as numTables is uint16
	if uint32(len(b)) < frontSize {
		return nil, ErrInvalidFontData
	}

	var checksumAdjustment uint32
	tables := make(map[string][]byte, numTables)
	for i := 0; i < int(numTables); i++ {
		tag := r.ReadString(4)
		checksum := r.ReadUint32()
		offset := r.ReadUint32()
		length := r.ReadUint32()

		padding := (4 - length&3) & 3
		if uint32(len(b)) <= offset || uint32(len(b))-offset < length || uint32(len(b))-offset-length < padding {
			return nil, ErrInvalidFontData
		}

		if tag == "head" {
			if length < 12 {
				return nil, ErrInvalidFontData
			}

			// to check checksum for head table, replace the overal checksum with zero and reset it at the end
			checksumAdjustment = binary.BigEndian.Uint32(b[offset+8:])
			binary.BigEndian.PutUint32(b[offset+8:], 0x00000000)
		}
		if calcChecksum(b[offset:offset+length+padding]) != checksum {
			debug.PrintStack()
			return nil, fmt.Errorf("%s: bad checksum", tag)
		}
		if tag == "head" {
			binary.BigEndian.PutUint32(b[offset+8:], checksumAdjustment)
		}
		tables[tag] = b[offset : offset+length : offset+length]
	}
	// TODO: check file checksum

	sfnt := &SFNT{}
	sfnt.Data = b
	sfnt.IsCFF = sfntVersion == "OTTO"
	sfnt.IsTrueType = binary.BigEndian.Uint32([]byte(sfntVersion)) == 0x00010000
	sfnt.Tables = tables

	requiredTables := []string{"cmap", "head", "hhea", "hmtx", "maxp", "name", "OS/2", "post"}
	if sfnt.IsTrueType {
		requiredTables = append(requiredTables, "glyf", "loca")
	}
	for _, requiredTable := range requiredTables {
		if _, ok := tables[requiredTable]; !ok {
			return nil, fmt.Errorf("%s: missing table", requiredTable)
		}
	}
	if sfnt.IsCFF {
		_, hasCFF := tables["CFF "]
		_, hasCFF2 := tables["CFF2"]
		if !hasCFF && !hasCFF2 {
			return nil, fmt.Errorf("CFF: missing table")
		} else if hasCFF && hasCFF2 {
			return nil, fmt.Errorf("CFF2: CFF table already exists")
		}
	}

	// maxp and hhea tables are required for other tables to be parse first
	if err := sfnt.parseHead(); err != nil {
		return nil, err
	} else if err := sfnt.parseMaxp(); err != nil {
		return nil, err
	} else if err := sfnt.parseHhea(); err != nil {
		return nil, err
	}
	if sfnt.IsTrueType {
		if err := sfnt.parseLoca(); err != nil {
			return nil, err
		}
	}

	tableNames := make([]string, len(tables))
	for tableName, _ := range tables {
		tableNames = append(tableNames, tableName)
	}
	sort.Strings(tableNames)
	for _, tableName := range tableNames {
		var err error
		switch tableName {
		//case "CFF ":
		//	err = sfnt.parseCFF()
		//case "CFF2":
		//	err = sfnt.parseCFF2()
		case "cmap":
			err = sfnt.parseCmap()
		case "glyf":
			err = sfnt.parseGlyf()
		case "hmtx":
			err = sfnt.parseHmtx()
		case "kern":
			err = sfnt.parseKern()
		case "name":
			err = sfnt.parseName()
		case "OS/2":
			err = sfnt.parseOS2()
		case "post":
			err = sfnt.parsePost()
		}
		if err != nil {
			return nil, err
		}
	}
	if sfnt.OS2.Version <= 1 {
		sfnt.estimateOS2()
	}
	return sfnt, nil
}

func (sfnt *SFNT) Subset(glyphIDs []uint16) ([]byte, []uint16) {
	// add dependencies for composite glyphs
	origLen := len(glyphIDs)
	for i := 0; i < origLen; i++ {
		deps, _ := sfnt.Glyf.Dependencies(glyphIDs[i], 0)
		if 0 < len(deps) {
			glyphIDs = append(glyphIDs, deps[1:]...)
		}
	}
	// sort ascending and add default glyph
	sort.Slice(glyphIDs, func(i, j int) bool { return glyphIDs[i] < glyphIDs[j] })
	if len(glyphIDs) == 0 || glyphIDs[0] != 0 {
		glyphIDs = append([]uint16{0}, glyphIDs...)
	}
	// remove duplicate and invalid glyphIDs
	for i := 0; i < len(glyphIDs); i++ {
		if sfnt.Maxp.NumGlyphs <= glyphIDs[i] {
			glyphIDs = glyphIDs[:i]
			break
		} else if 0 < i && glyphIDs[i] == glyphIDs[i-1] {
			glyphIDs = append(glyphIDs[:i], glyphIDs[i+1:]...)
			i--
		}
	}
	glyphMap := make(map[uint16]uint16, len(glyphIDs))
	for i, glyphID := range glyphIDs {
		glyphMap[glyphID] = uint16(i)
	}

	// specify tables to include
	tags := []string{"cmap", "head", "hhea", "hmtx", "maxp", "name", "OS/2", "post"}
	if sfnt.IsTrueType {
		tags = append(tags, "glyf", "loca")
	} else if sfnt.IsCFF {
		// TODO
	}

	// preserve tables
	for _, tag := range []string{"cvt ", "fpgm", "prep"} {
		if _, ok := sfnt.Tables[tag]; ok {
			tags = append(tags, tag)
		}
	}

	// handle kern table that could be removed
	kernSubtables := []kernFormat0{}
	if _, ok := sfnt.Tables["kern"]; ok {
		for _, subtable := range sfnt.Kern.Subtables {
			pairs := []kernPair{}
			iLeft := 0
			iRight := 0
			for _, pair := range subtable.Pairs {
				if pair.Key < uint32(glyphIDs[iLeft])<<16+uint32(glyphIDs[iRight]) {
					continue
				}
				for iLeft < len(glyphIDs) && uint32(glyphIDs[iLeft])<<16 < pair.Key&0xFFFF0000 {
					iLeft++
					iRight = 0
				}
				if iLeft == len(glyphIDs) {
					break
				}
				if uint32(glyphIDs[iLeft])<<16 == pair.Key&0xFFFF0000 {
					for iRight < len(glyphIDs) && uint32(glyphIDs[iRight]) < pair.Key&0x0000FFFF {
						iRight++
					}
					if iRight == len(glyphIDs) {
						iLeft++
						iRight = 0
						if iLeft == len(glyphIDs) {
							break
						}
						continue
					}
					if uint32(glyphIDs[iRight]) == pair.Key&0x0000FFFF {
						pairs = append(pairs, kernPair{
							Key:   uint32(glyphMap[glyphIDs[iLeft]])<<16 + uint32(glyphMap[glyphIDs[iRight]]),
							Value: pair.Value,
						})
					}
				}
			}
			if 0 < len(pairs) {
				kernSubtables = append(kernSubtables, kernFormat0{
					Coverage: subtable.Coverage,
					Pairs:    pairs,
				})
			}
		}
		if 0 < len(kernSubtables) {
			tags = append(tags, "kern")
		}
	}
	sort.Strings(tags)

	// write header
	w := newBinaryWriter([]byte{})
	if sfnt.IsTrueType {
		w.WriteUint32(0x00010000) // sfntVersion
	} else if sfnt.IsCFF {
		w.WriteString("OTTO") // sfntVersion
	}
	numTables := uint16(len(tags))
	entrySelector := uint16(math.Log2(float64(numTables)))
	searchRange := uint16(1 << (entrySelector + 4))
	w.WriteUint16(numTables)                  // numTables
	w.WriteUint16(searchRange)                // searchRange
	w.WriteUint16(entrySelector)              // entrySelector
	w.WriteUint16(numTables<<4 - searchRange) // rangeShift

	// we'll write the table records at the end
	w.WriteBytes(make([]byte, numTables<<4))

	// write tables
	var checksumAdjustmentPos uint32
	offsets, lengths := make([]uint32, numTables), make([]uint32, numTables)
	for i, tag := range tags {
		offsets[i] = w.Len()
		switch tag {
		case "head":
			head := sfnt.Tables["head"]
			w.WriteBytes(head[:8])
			checksumAdjustmentPos = w.Len()
			w.WriteUint32(0) // checksumAdjustment
			w.WriteBytes(head[12:28])
			w.WriteInt64(int64(time.Now().UTC().Sub(time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC)) / 1e9)) // modified
			w.WriteBytes(head[36:])
		case "glyf":
			for _, glyphID := range glyphIDs {
				b := sfnt.Glyf.Get(glyphID)

				// update glyphIDs for composite glyphs, make sure not to write to b
				glyphIDPositions, newGlyphIDs := []uint32{}, []uint16{}
				if 0 < len(b) {
					numberOfContours := int16(binary.BigEndian.Uint16(b))
					if numberOfContours < 0 {
						offset := uint32(10)
						for {
							flags := binary.BigEndian.Uint16(b[offset:])
							subGlyphID := binary.BigEndian.Uint16(b[offset+2:])
							glyphIDPositions = append(glyphIDPositions, offset+2)
							newGlyphIDs = append(newGlyphIDs, glyphMap[subGlyphID])

							length, more := glyfCompositeLength(flags)
							if !more {
								break
							}
							offset += length
						}
					}
				}

				start := w.Len()
				w.WriteBytes(b)
				for i := 0; i < len(glyphIDPositions); i++ {
					binary.BigEndian.PutUint16(w.buf[start+glyphIDPositions[i]:], newGlyphIDs[i])
				}
			}
		case "loca":
			// TODO: change to short format if possible
			if sfnt.Head.IndexToLocFormat == 0 {
				pos := uint16(0)
				for _, glyphID := range glyphIDs {
					w.WriteUint16(pos)
					pos += uint16(sfnt.Loca.Get(glyphID+1) - sfnt.Loca.Get(glyphID))
				}
				w.WriteUint16(pos)
			} else if sfnt.Head.IndexToLocFormat == 1 {
				pos := uint32(0)
				for _, glyphID := range glyphIDs {
					w.WriteUint32(pos)
					pos += sfnt.Loca.Get(glyphID+1) - sfnt.Loca.Get(glyphID)
				}
				w.WriteUint32(pos)
			}
		case "maxp":
			maxp := sfnt.Tables["maxp"]
			w.WriteBytes(maxp[:4])
			w.WriteUint16(uint16(len(glyphIDs))) // numGlyphs
			w.WriteBytes(maxp[6:])
		case "hhea":
			numberOfHMetrics := uint16(0)
			for _, glyphID := range glyphIDs {
				if sfnt.Hhea.NumberOfHMetrics <= glyphID {
					break
				}
				numberOfHMetrics++
			}
			hhea := sfnt.Tables["hhea"]
			w.WriteBytes(hhea[:34])
			w.WriteUint16(numberOfHMetrics) // numberOfHMetrics
		case "hmtx":
			for _, glyphID := range glyphIDs {
				if glyphID < sfnt.Hhea.NumberOfHMetrics {
					w.WriteUint16(sfnt.Hmtx.Advance(glyphID))
				}
				w.WriteInt16(sfnt.Hmtx.LeftSideBearing(glyphID))
			}
		case "post":
			post := sfnt.Tables["post"]
			w.WriteBytes(post[:32])
			if binary.BigEndian.Uint32(post) == 0x00020000 {
				w.WriteUint16(uint16(len(glyphIDs))) // numGlyphs

				i := 0
				b := []byte{}
				for _, glyphID := range glyphIDs {
					if sfnt.Post.GlyphNameIndex[glyphID] < 258 {
						w.WriteUint16(sfnt.Post.GlyphNameIndex[glyphID])
					} else {
						w.WriteUint16(uint16(258 + i))
						name := sfnt.Post.Get(glyphID)
						b = append(b, byte(len(name)))
						b = append(b, []byte(name)...)
						i++
					}
				}
				w.WriteBytes(b)
			}
		case "cmap":
			w.WriteUint16(0)  // version
			w.WriteUint16(1)  // numTables
			w.WriteUint16(0)  // platformID
			w.WriteUint16(4)  // encodingID
			w.WriteUint32(12) // subtableOffset

			// format 12
			start := w.Len()
			w.WriteUint16(12) // format
			w.WriteUint16(0)  // reserved
			w.WriteUint32(0)  // length
			w.WriteUint32(0)  // language

			rs := make([]rune, 0, len(glyphIDs))
			runeMap := make(map[rune]uint16, len(glyphIDs))
			for i, glyphID := range glyphIDs {
				if r := sfnt.Cmap.ToUnicode(glyphID); r != 0 {
					rs = append(rs, r)
					runeMap[r] = uint16(i)
				}
			}
			sort.Slice(rs, func(i, j int) bool { return rs[i] < rs[j] })
			// TODO: optimize ranges
			w.WriteUint32(uint32(len(rs))) // numGroups
			for i, r := range rs {
				if 0 < i && r == rs[i-1] {
					continue
				}
				w.WriteUint32(uint32(r))          // startCharCode
				w.WriteUint32(uint32(r))          // endCharCode
				w.WriteUint32(uint32(runeMap[r])) // startGlyphID
			}
			binary.BigEndian.PutUint32(w.buf[start+4:], w.Len()-start) // set length
		case "kern":
			w.WriteUint16(0)                          // version
			w.WriteUint16(uint16(len(kernSubtables))) // nTables
			for _, subtable := range kernSubtables {
				w.WriteUint16(0)                                     // version
				w.WriteUint16(6 + 8 + 6*uint16(len(subtable.Pairs))) // length
				w.WriteUint8(0)                                      // format
				w.WriteUint8(flagsToUint8(subtable.Coverage))        // coverage

				nPairs := uint16(len(subtable.Pairs))
				entrySelector := uint16(math.Log2(float64(nPairs)))
				searchRange := uint16(1 << (entrySelector + 4))
				w.WriteUint16(nPairs)
				w.WriteUint16(searchRange)
				w.WriteUint16(entrySelector)
				w.WriteUint16(nPairs<<4 - searchRange)
				for _, pair := range subtable.Pairs {
					w.WriteUint32(pair.Key)
					w.WriteInt16(pair.Value)
				}
			}
		default:
			w.WriteBytes(sfnt.Tables[tag])
		}
		lengths[i] = w.Len() - offsets[i]

		padding := (4 - lengths[i]&3) & 3
		for i := 0; i < int(padding); i++ {
			w.WriteByte(0x00)
		}
	}

	// add table record entries
	buf := w.Bytes()
	for i, tag := range tags {
		pos := 12 + i<<4
		copy(buf[pos:], []byte(tag))
		padding := (4 - lengths[i]&3) & 3
		checksum := calcChecksum(buf[offsets[i] : offsets[i]+lengths[i]+padding])
		binary.BigEndian.PutUint32(w.buf[pos+4:], checksum)
		binary.BigEndian.PutUint32(w.buf[pos+8:], offsets[i])
		binary.BigEndian.PutUint32(w.buf[pos+12:], lengths[i])
	}
	binary.BigEndian.PutUint32(w.buf[checksumAdjustmentPos:], 0xB1B0AFBA-calcChecksum(buf))
	return buf, glyphIDs
}

////////////////////////////////////////////////////////////////

type cmapFormat0 struct {
	GlyphIdArray [256]uint8
}

func (subtable *cmapFormat0) Get(r rune) (uint16, bool) {
	if r < 0 || 256 <= r {
		return 0, false
	}
	return uint16(subtable.GlyphIdArray[r]), true
}

func (subtable *cmapFormat0) ToUnicode(glyphID uint16) (rune, bool) {
	if 256 <= glyphID {
		return 0, false
	}
	for r, id := range subtable.GlyphIdArray {
		if id == uint8(glyphID) {
			return rune(r), true
		}
	}
	return 0, false
}

type cmapFormat4 struct {
	StartCode     []uint16
	EndCode       []uint16
	IdDelta       []int16
	IdRangeOffset []uint16
	GlyphIdArray  []uint16
}

func (subtable *cmapFormat4) Get(r rune) (uint16, bool) {
	if r < 0 || 65536 <= r {
		return 0, false
	}
	n := len(subtable.StartCode)
	for i := 0; i < n; i++ {
		if uint16(r) <= subtable.EndCode[i] && subtable.StartCode[i] <= uint16(r) {
			if subtable.IdRangeOffset[i] == 0 {
				// is modulo 65536 with the idDelta cast and addition overflow
				return uint16(subtable.IdDelta[i]) + uint16(r), true
			}
			// idRangeOffset/2  ->  offset value to index of words
			// r-startCode  ->  difference of rune with startCode
			// -(n-1)  ->  subtract offset from the current idRangeOffset item
			index := int(subtable.IdRangeOffset[i]/2) + int(uint16(r)-subtable.StartCode[i]) - (n - i)
			return subtable.GlyphIdArray[index], true // index is always valid
		}
	}
	return 0, false
}

func (subtable *cmapFormat4) ToUnicode(glyphID uint16) (rune, bool) {
	// TODO
	return 0, false
}

type cmapFormat6 struct {
	FirstCode    uint16
	GlyphIdArray []uint16
}

func (subtable *cmapFormat6) Get(r rune) (uint16, bool) {
	if r < int32(subtable.FirstCode) || uint32(len(subtable.GlyphIdArray)) <= uint32(r)-uint32(subtable.FirstCode) {
		return 0, false
	}
	return subtable.GlyphIdArray[uint32(r)-uint32(subtable.FirstCode)], true
}

func (subtable *cmapFormat6) ToUnicode(glyphID uint16) (rune, bool) {
	for i, id := range subtable.GlyphIdArray {
		if id == glyphID {
			return rune(subtable.FirstCode) + rune(i), true
		}
	}
	return 0, false
}

type cmapFormat12 struct {
	StartCharCode []uint32
	EndCharCode   []uint32
	StartGlyphID  []uint32
}

func (subtable *cmapFormat12) Get(r rune) (uint16, bool) {
	if r < 0 {
		return 0, false
	}
	for i := 0; i < len(subtable.StartCharCode); i++ {
		if uint32(r) <= subtable.EndCharCode[i] && subtable.StartCharCode[i] <= uint32(r) {
			return uint16((uint32(r) - subtable.StartCharCode[i]) + subtable.StartGlyphID[i]), true
		}
	}
	return 0, false
}

func (subtable *cmapFormat12) ToUnicode(glyphID uint16) (rune, bool) {
	for i := 0; i < len(subtable.StartCharCode); i++ {
		if subtable.StartGlyphID[i] <= uint32(glyphID) && uint32(glyphID) <= subtable.StartGlyphID[i]+(subtable.EndCharCode[i]-subtable.StartCharCode[i]) {
			return rune((uint32(glyphID) - subtable.StartGlyphID[i]) + subtable.StartCharCode[i]), true
		}
	}
	return 0, false
}

type cmapEncodingRecord struct {
	PlatformID uint16
	EncodingID uint16
	Format     uint16
	Subtable   uint16
}

type cmapSubtable interface {
	Get(rune) (uint16, bool)
	ToUnicode(uint16) (rune, bool)
}

type cmapTable struct {
	EncodingRecords []cmapEncodingRecord
	Subtables       []cmapSubtable
}

func (cmap *cmapTable) Get(r rune) uint16 {
	for _, subtable := range cmap.Subtables {
		if glyphID, ok := subtable.Get(r); ok {
			return glyphID
		}
	}
	return 0
}

func (cmap *cmapTable) ToUnicode(glyphID uint16) rune {
	for _, subtable := range cmap.Subtables {
		if r, ok := subtable.ToUnicode(glyphID); ok {
			return r
		}
	}
	return 0
}

func (sfnt *SFNT) parseCmap() error {
	// requires data from maxp
	b, ok := sfnt.Tables["cmap"]
	if !ok {
		return fmt.Errorf("cmap: missing table")
	} else if len(b) < 4 {
		return fmt.Errorf("cmap: bad table")
	}

	sfnt.Cmap = &cmapTable{}
	r := newBinaryReader(b)
	if r.ReadUint16() != 0 {
		return fmt.Errorf("cmap: bad version")
	}
	numTables := r.ReadUint16()
	if uint32(len(b)) < 4+8*uint32(numTables) {
		return fmt.Errorf("cmap: bad table")
	}

	// find and extract subtables and make sure they don't overlap each other
	offsets, lengths := []uint32{0}, []uint32{4 + 8*uint32(numTables)}
	for j := 0; j < int(numTables); j++ {
		platformID := r.ReadUint16()
		encodingID := r.ReadUint16()
		subtableID := -1

		offset := r.ReadUint32()
		if uint32(len(b))-8 < offset { // subtable must be at least 8 bytes long to extract length
			return fmt.Errorf("cmap: bad subtable %d", j)
		}
		for i := 0; i < len(offsets); i++ {
			if offsets[i] < offset && offset < lengths[i] {
				return fmt.Errorf("cmap: bad subtable %d", j)
			}
		}

		// extract subtable length
		rs := newBinaryReader(b[offset:])
		format := rs.ReadUint16()
		var length uint32
		if format == 0 || format == 2 || format == 4 || format == 6 {
			length = uint32(rs.ReadUint16())
		} else if format == 8 || format == 10 || format == 12 || format == 13 {
			_ = rs.ReadUint16() // reserved
			length = rs.ReadUint32()
		} else if format == 14 {
			length = rs.ReadUint32()
		} else {
			return fmt.Errorf("cmap: bad format %d for subtable %d", format, j)
		}
		if length < 8 || math.MaxUint32-offset < length {
			return fmt.Errorf("cmap: bad subtable %d", j)
		}
		for i := 0; i < len(offsets); i++ {
			if offset == offsets[i] && length == lengths[i] {
				subtableID = int(i)
				break
			} else if offset <= offsets[i] && offsets[i] < offset+length {
				return fmt.Errorf("cmap: bad subtable %d", j)
			}
		}
		rs.buf = rs.buf[:length:length]

		if subtableID == -1 {
			subtableID = len(sfnt.Cmap.Subtables)
			offsets = append(offsets, offset)
			lengths = append(lengths, length)

			switch format {
			case 0:
				if rs.Len() != 258 {
					return fmt.Errorf("cmap: bad subtable %d", j)
				}
				_ = rs.ReadUint16() // languageID

				subtable := &cmapFormat0{}
				copy(subtable.GlyphIdArray[:], rs.ReadBytes(256))
				for _, glyphID := range subtable.GlyphIdArray {
					if sfnt.Maxp.NumGlyphs <= uint16(glyphID) {
						return fmt.Errorf("cmap: bad glyphID in subtable %d", j)
					}
				}
				sfnt.Cmap.Subtables = append(sfnt.Cmap.Subtables, subtable)
			case 4:
				if rs.Len() < 10 {
					return fmt.Errorf("cmap: bad subtable %d", j)
				}
				_ = rs.ReadUint16() // languageID

				segCount := rs.ReadUint16()
				if segCount%2 != 0 {
					return fmt.Errorf("cmap: bad segCount in subtable %d", j)
				}
				segCount /= 2
				if MaxCmapSegments < segCount {
					return fmt.Errorf("cmap: too many segments in subtable %d", j)
				}
				_ = rs.ReadUint16() // searchRange
				_ = rs.ReadUint16() // entrySelector
				_ = rs.ReadUint16() // rangeShift

				subtable := &cmapFormat4{}
				if rs.Len() < 2+8*uint32(segCount) {
					return fmt.Errorf("cmap: bad subtable %d", j)
				}
				subtable.EndCode = make([]uint16, segCount)
				for i := 0; i < int(segCount); i++ {
					endCode := rs.ReadUint16()
					if 0 < i && endCode <= subtable.EndCode[i-1] {
						return fmt.Errorf("cmap: bad endCode in subtable %d", j)
					}
					subtable.EndCode[i] = endCode
				}
				_ = rs.ReadUint16() // reservedPad
				subtable.StartCode = make([]uint16, segCount)
				for i := 0; i < int(segCount); i++ {
					startCode := rs.ReadUint16()
					if subtable.EndCode[i] < startCode || 0 < i && startCode <= subtable.EndCode[i-1] {
						return fmt.Errorf("cmap: bad startCode in subtable %d", j)
					}
					subtable.StartCode[i] = startCode
				}
				subtable.IdDelta = make([]int16, segCount)
				for i := 0; i < int(segCount); i++ {
					subtable.IdDelta[i] = rs.ReadInt16()
				}

				glyphIdArrayLength := rs.Len() - 2*uint32(segCount)
				if glyphIdArrayLength%2 != 0 {
					return fmt.Errorf("cmap: bad subtable %d", j)
				}
				glyphIdArrayLength /= 2

				subtable.IdRangeOffset = make([]uint16, segCount)
				for i := 0; i < int(segCount); i++ {
					idRangeOffset := rs.ReadUint16()
					if idRangeOffset%2 != 0 {
						return fmt.Errorf("cmap: bad idRangeOffset in subtable %d", j)
					} else if idRangeOffset != 0 {
						index := int(idRangeOffset/2) + int(subtable.EndCode[i]-subtable.StartCode[i]) - (int(segCount) - i)
						if index < 0 || glyphIdArrayLength <= uint32(index) {
							return fmt.Errorf("cmap: bad idRangeOffset in subtable %d", j)
						}
					}
					subtable.IdRangeOffset[i] = idRangeOffset
				}
				subtable.GlyphIdArray = make([]uint16, glyphIdArrayLength)
				for i := 0; i < int(glyphIdArrayLength); i++ {
					glyphID := rs.ReadUint16()
					if sfnt.Maxp.NumGlyphs <= glyphID {
						return fmt.Errorf("cmap: bad glyphID in subtable %d", j)
					}
					subtable.GlyphIdArray[i] = glyphID
				}
				sfnt.Cmap.Subtables = append(sfnt.Cmap.Subtables, subtable)
			case 6:
				if rs.Len() < 6 {
					return fmt.Errorf("cmap: bad subtable %d", j)
				}
				_ = rs.ReadUint16() // language

				subtable := &cmapFormat6{}
				subtable.FirstCode = rs.ReadUint16()
				entryCount := rs.ReadUint16()
				if rs.Len() < 2*uint32(entryCount) {
					return fmt.Errorf("cmap: bad subtable %d", j)
				}
				subtable.GlyphIdArray = make([]uint16, entryCount)
				for i := 0; i < int(entryCount); i++ {
					subtable.GlyphIdArray[i] = rs.ReadUint16()
				}
				sfnt.Cmap.Subtables = append(sfnt.Cmap.Subtables, subtable)
			case 12:
				if rs.Len() < 8 {
					return fmt.Errorf("cmap: bad subtable %d", j)
				}
				_ = rs.ReadUint32() // language
				numGroups := rs.ReadUint32()
				if MaxCmapSegments < numGroups {
					return fmt.Errorf("cmap: too many segments in subtable %d", j)
				} else if rs.Len() < 12*numGroups {
					return fmt.Errorf("cmap: bad subtable %d", j)
				}

				subtable := &cmapFormat12{}
				subtable.StartCharCode = make([]uint32, numGroups)
				subtable.EndCharCode = make([]uint32, numGroups)
				subtable.StartGlyphID = make([]uint32, numGroups)
				for i := 0; i < int(numGroups); i++ {
					startCharCode := rs.ReadUint32()
					endCharCode := rs.ReadUint32()
					startGlyphID := rs.ReadUint32()
					if endCharCode < startCharCode || 0 < i && startCharCode <= subtable.EndCharCode[i-1] {
						if 0 < i {
							fmt.Println(startCharCode, endCharCode, "prev", subtable.EndCharCode[i-1])
						} else {
							fmt.Println(startCharCode, endCharCode)
						}
						return fmt.Errorf("cmap: bad character code range in subtable %d", j)
					} else if uint32(sfnt.Maxp.NumGlyphs) <= endCharCode-startCharCode || uint32(sfnt.Maxp.NumGlyphs)-(endCharCode-startCharCode) <= startGlyphID {
						return fmt.Errorf("cmap: bad glyphID in subtable %d", j)
					}
					subtable.StartCharCode[i] = startCharCode
					subtable.EndCharCode[i] = endCharCode
					subtable.StartGlyphID[i] = startGlyphID
				}
				sfnt.Cmap.Subtables = append(sfnt.Cmap.Subtables, subtable)
			}
		}
		sfnt.Cmap.EncodingRecords = append(sfnt.Cmap.EncodingRecords, cmapEncodingRecord{
			PlatformID: platformID,
			EncodingID: encodingID,
			Format:     format,
			Subtable:   uint16(subtableID),
		})
	}
	return nil
}

////////////////////////////////////////////////////////////////

type glyfContour struct {
	GlyphID                uint16
	XMin, YMin, XMax, YMax int16
	EndPoints              []uint16
	Instructions           []byte
	OnCurve                []bool
	XCoordinates           []int16
	YCoordinates           []int16
}

func (contour *glyfContour) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Glyph %v:\n", contour.GlyphID)
	fmt.Fprintf(&b, "  Contours: %v\n", len(contour.EndPoints))
	fmt.Fprintf(&b, "  XMin: %v\n", contour.XMin)
	fmt.Fprintf(&b, "  YMin: %v\n", contour.YMin)
	fmt.Fprintf(&b, "  XMax: %v\n", contour.XMax)
	fmt.Fprintf(&b, "  YMax: %v\n", contour.YMax)
	fmt.Fprintf(&b, "  EndPoints: %v\n", contour.EndPoints)
	fmt.Fprintf(&b, "  Instruction length: %v\n", len(contour.Instructions))
	fmt.Fprintf(&b, "  Coordinates:\n")
	for i := 0; i <= int(contour.EndPoints[len(contour.EndPoints)-1]); i++ {
		fmt.Fprintf(&b, "    ")
		if i < len(contour.XCoordinates) {
			fmt.Fprintf(&b, "%8v", contour.XCoordinates[i])
		} else {
			fmt.Fprintf(&b, "  ----  ")
		}
		if i < len(contour.YCoordinates) {
			fmt.Fprintf(&b, " %8v", contour.YCoordinates[i])
		} else {
			fmt.Fprintf(&b, "   ----  ")
		}
		if i < len(contour.OnCurve) {
			onCurve := "Off"
			if contour.OnCurve[i] {
				onCurve = "On"
			}
			fmt.Fprintf(&b, " %3v\n", onCurve)
		} else {
			fmt.Fprintf(&b, " ---\n")
		}
	}
	return b.String()
}

type glyfTable struct {
	data []byte
	loca *locaTable
}

func (glyf *glyfTable) Get(glyphID uint16) []byte {
	start := glyf.loca.Get(glyphID)
	end := glyf.loca.Get(glyphID + 1)
	if end == 0 {
		return nil
	}
	return glyf.data[start:end]
}

func (glyf *glyfTable) Dependencies(glyphID uint16, level int) ([]uint16, error) {
	deps := []uint16{glyphID}
	b := glyf.Get(glyphID)
	if b == nil {
		return nil, fmt.Errorf("glyf: bad glyphID %v", glyphID)
	} else if len(b) == 0 {
		return deps, nil
	}
	r := newBinaryReader(b)
	if r.Len() < 10 {
		return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
	}
	numberOfContours := r.ReadInt16()
	_ = r.ReadBytes(8)
	if numberOfContours < 0 {
		if 7 < level {
			return nil, fmt.Errorf("glyf: compound glyphs too deeply nested")
		}

		// composite glyph
		for {
			if r.Len() < 4 {
				return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
			}

			flags := r.ReadUint16()
			subGlyphID := r.ReadUint16()
			subDeps, err := glyf.Dependencies(subGlyphID, level+1)
			if err != nil {
				return nil, err
			}
			deps = append(deps, subDeps...)

			length, more := glyfCompositeLength(flags)
			if r.Len() < length-4 {
				return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
			}
			_ = r.ReadBytes(length - 4)
			if !more {
				break
			}
		}
	}
	return deps, nil
}

func glyfCompositeLength(flags uint16) (length uint32, more bool) {
	length = 4 + 2
	if flags&0x0001 != 0 { // ARG_1_AND_2_ARE_WORDS
		length += 2
	}
	if flags&0x0008 != 0 { // WE_HAVE_A_SCALE
		length += 2
	} else if flags&0x0040 != 0 { // WE_HAVE_AN_X_AND_Y_SCALE
		length += 4
	} else if flags&0x0080 != 0 { // WE_HAVE_A_TWO_BY_TWO
		length += 8
	}
	more = flags&0x0020 != 0 // MORE_COMPONENTS
	return
}

func (glyf *glyfTable) Contour(glyphID uint16, level int) (*glyfContour, error) {
	b := glyf.Get(glyphID)
	if b == nil {
		return nil, fmt.Errorf("glyf: bad glyphID %v", glyphID)
	} else if len(b) == 0 {
		return &glyfContour{GlyphID: glyphID}, nil
	}
	r := newBinaryReader(b)
	if r.Len() < 10 {
		return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
	}

	contour := &glyfContour{}
	contour.GlyphID = glyphID
	numberOfContours := r.ReadInt16()
	contour.XMin = r.ReadInt16()
	contour.YMin = r.ReadInt16()
	contour.XMax = r.ReadInt16()
	contour.YMax = r.ReadInt16()
	if 0 <= numberOfContours {
		// simple glyph
		if r.Len() < 2*uint32(numberOfContours)+2 {
			return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
		}
		contour.EndPoints = make([]uint16, numberOfContours)
		for i := 0; i < int(numberOfContours); i++ {
			contour.EndPoints[i] = r.ReadUint16()
		}

		instructionLength := r.ReadUint16()
		if r.Len() < uint32(instructionLength) {
			return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
		}
		contour.Instructions = r.ReadBytes(uint32(instructionLength))

		numPoints := int(contour.EndPoints[numberOfContours-1]) + 1
		flags := make([]byte, numPoints)
		contour.OnCurve = make([]bool, numPoints)
		for i := 0; i < numPoints; i++ {
			if r.Len() < 1 {
				return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
			}

			flags[i] = r.ReadByte()
			contour.OnCurve[i] = flags[i]&0x01 != 0
			if flags[i]&0x08 != 0 { // REPEAT_FLAG
				repeat := r.ReadByte()
				for j := 1; j <= int(repeat); j++ {
					flags[i+j] = flags[i]
					contour.OnCurve[i+j] = contour.OnCurve[i]
				}
				i += int(repeat)
			}
		}

		var x int16
		contour.XCoordinates = make([]int16, numPoints)
		for i := 0; i < numPoints; i++ {
			xShortVector := flags[i]&0x02 != 0
			xIsSameOrPositiveXShortVector := flags[i]&0x10 != 0
			if xShortVector {
				if r.Len() < 1 {
					return nil, fmt.Errorf("glyf: bad table or flags for glyphID %v", glyphID)
				}
				if xIsSameOrPositiveXShortVector {
					x += int16(r.ReadUint8())
				} else {
					x -= int16(r.ReadUint8())
				}
			} else if !xIsSameOrPositiveXShortVector {
				if r.Len() < 2 {
					return nil, fmt.Errorf("glyf: bad table or flags for glyphID %v", glyphID)
				}
				x += r.ReadInt16()
			}
			contour.XCoordinates[i] = x
		}

		var y int16
		contour.YCoordinates = make([]int16, numPoints)
		for i := 0; i < numPoints; i++ {
			yShortVector := flags[i]&0x04 != 0
			yIsSameOrPositiveYShortVector := flags[i]&0x20 != 0
			if yShortVector {
				if r.Len() < 1 {
					return nil, fmt.Errorf("glyf: bad table or flags for glyphID %v", glyphID)
				}
				if yIsSameOrPositiveYShortVector {
					y += int16(r.ReadUint8())
				} else {
					y -= int16(r.ReadUint8())
				}
			} else if !yIsSameOrPositiveYShortVector {
				if r.Len() < 2 {
					return nil, fmt.Errorf("glyf: bad table or flags for glyphID %v", glyphID)
				}
				y += r.ReadInt16()
			}
			contour.YCoordinates[i] = y
		}
	} else {
		if 7 < level {
			return nil, fmt.Errorf("glyf: compound glyphs too deeply nested")
		}

		// composite glyph
		hasInstructions := false
		for {
			if r.Len() < 4 {
				return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
			}

			flags := r.ReadUint16()
			subGlyphID := r.ReadUint16()
			if flags&0x0002 == 0 { // ARGS_ARE_XY_VALUES
				return nil, fmt.Errorf("glyf: composite glyph not supported")
			}
			var dx, dy int16
			if flags&0x0001 != 0 { // ARG_1_AND_2_ARE_WORDS
				if r.Len() < 4 {
					return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
				}
				dx = r.ReadInt16()
				dy = r.ReadInt16()
			} else {
				if r.Len() < 2 {
					return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
				}
				dx = int16(r.ReadInt8())
				dy = int16(r.ReadInt8())
			}
			var txx, txy, tyx, tyy int16
			if flags&0x0008 != 0 { // WE_HAVE_A_SCALE
				if r.Len() < 2 {
					return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
				}
				txx = r.ReadInt16()
				tyy = txx
			} else if flags&0x0040 != 0 { // WE_HAVE_AN_X_AND_Y_SCALE
				if r.Len() < 4 {
					return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
				}
				txx = r.ReadInt16()
				tyy = r.ReadInt16()
			} else if flags&0x0080 != 0 { // WE_HAVE_A_TWO_BY_TWO
				if r.Len() < 8 {
					return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
				}
				txx = r.ReadInt16()
				txy = r.ReadInt16()
				tyx = r.ReadInt16()
				tyy = r.ReadInt16()
			}
			if flags&0x0100 != 0 {
				hasInstructions = true
			}

			subContour, err := glyf.Contour(subGlyphID, level+1)
			if err != nil {
				return nil, err
			}

			var numPoints uint16
			if 0 < len(contour.EndPoints) {
				numPoints = contour.EndPoints[len(contour.EndPoints)-1] + 1
			}
			for i := 0; i < len(subContour.EndPoints); i++ {
				contour.EndPoints = append(contour.EndPoints, numPoints+subContour.EndPoints[i])
			}
			contour.OnCurve = append(contour.OnCurve, subContour.OnCurve...)
			for i := 0; i < len(subContour.XCoordinates); i++ {
				x := subContour.XCoordinates[i]
				y := subContour.YCoordinates[i]
				if flags&0x00C8 != 0 { // has transformation
					const half = 1 << 13
					xt := int16((int64(x)*int64(txx)+half)>>14) + int16((int64(y)*int64(tyx)+half)>>14)
					yt := int16((int64(x)*int64(txy)+half)>>14) + int16((int64(y)*int64(tyy)+half)>>14)
					x, y = xt, yt
				}
				contour.XCoordinates = append(contour.XCoordinates, dx+x)
				contour.YCoordinates = append(contour.YCoordinates, dy+y)
			}
			if flags&0x0020 == 0 { // MORE_COMPONENTS
				break
			}
		}
		if hasInstructions {
			instructionLength := r.ReadUint16()
			if r.Len() < uint32(instructionLength) {
				return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
			}
			contour.Instructions = r.ReadBytes(uint32(instructionLength))
		}
	}
	return contour, nil
}

func (sfnt *SFNT) parseGlyf() error {
	// requires data from loca
	b, ok := sfnt.Tables["glyf"]
	if !ok {
		return fmt.Errorf("glyf: missing table")
	} else if uint32(len(b)) != sfnt.Loca.Get(sfnt.Maxp.NumGlyphs) {
		return fmt.Errorf("glyf: bad table")
	}

	sfnt.Glyf = &glyfTable{
		data: b,
		loca: sfnt.Loca,
	}
	return nil
}

////////////////////////////////////////////////////////////////

type headTable struct {
	FontRevision           uint32
	Flags                  [16]bool
	UnitsPerEm             uint16
	Created, Modified      time.Time
	XMin, YMin, XMax, YMax int16
	MacStyle               [16]bool
	LowestRecPPEM          uint16
	FontDirectionHint      int16
	IndexToLocFormat       int16
	GlyphDataFormat        int16
}

func (sfnt *SFNT) parseHead() error {
	b, ok := sfnt.Tables["head"]
	if !ok {
		return fmt.Errorf("head: missing table")
	} else if len(b) != 54 {
		return fmt.Errorf("head: bad table")
	}

	sfnt.Head = &headTable{}
	r := newBinaryReader(b)
	majorVersion := r.ReadUint16()
	minorVersion := r.ReadUint16()
	if majorVersion != 1 && minorVersion != 0 {
		return fmt.Errorf("head: bad version")
	}
	sfnt.Head.FontRevision = r.ReadUint32()
	_ = r.ReadUint32()                // checksumAdjustment
	if r.ReadUint32() != 0x5F0F3CF5 { // magicNumber
		return fmt.Errorf("head: bad magic version")
	}
	sfnt.Head.Flags = uint16ToFlags(r.ReadUint16())
	sfnt.Head.UnitsPerEm = r.ReadUint16()
	created := r.ReadUint64()
	modified := r.ReadUint64()
	if math.MaxInt64 < created || math.MaxInt64 < modified {
		return fmt.Errorf("head: created and/or modified dates too large")
	}
	sfnt.Head.Created = time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Second * time.Duration(created))
	sfnt.Head.Modified = time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Second * time.Duration(modified))
	sfnt.Head.XMin = r.ReadInt16()
	sfnt.Head.YMin = r.ReadInt16()
	sfnt.Head.XMax = r.ReadInt16()
	sfnt.Head.YMax = r.ReadInt16()
	sfnt.Head.MacStyle = uint16ToFlags(r.ReadUint16())
	sfnt.Head.LowestRecPPEM = r.ReadUint16()
	sfnt.Head.FontDirectionHint = r.ReadInt16()
	sfnt.Head.IndexToLocFormat = r.ReadInt16()
	if sfnt.Head.IndexToLocFormat != 0 && sfnt.Head.IndexToLocFormat != 1 {
		return fmt.Errorf("head: bad indexToLocFormat")
	}
	sfnt.Head.GlyphDataFormat = r.ReadInt16()
	return nil
}

////////////////////////////////////////////////////////////////

type hheaTable struct {
	Ascender            int16
	Descender           int16
	LineGap             int16
	AdvanceWidthMax     uint16
	MinLeftSideBearing  int16
	MinRightSideBearing int16
	XMaxExtent          int16
	CaretSlopeRise      int16
	CaretSlopeRun       int16
	CaretOffset         int16
	MetricDataFormat    int16
	NumberOfHMetrics    uint16
}

func (sfnt *SFNT) parseHhea() error {
	// requires data from maxp
	b, ok := sfnt.Tables["hhea"]
	if !ok {
		return fmt.Errorf("hhea: missing table")
	} else if len(b) != 36 {
		return fmt.Errorf("hhea: bad table")
	}

	sfnt.Hhea = &hheaTable{}
	r := newBinaryReader(b)
	majorVersion := r.ReadUint16()
	minorVersion := r.ReadUint16()
	if majorVersion != 1 && minorVersion != 0 {
		return fmt.Errorf("hhea: bad version")
	}
	sfnt.Hhea.Ascender = r.ReadInt16()
	sfnt.Hhea.Descender = r.ReadInt16()
	sfnt.Hhea.LineGap = r.ReadInt16()
	sfnt.Hhea.AdvanceWidthMax = r.ReadUint16()
	sfnt.Hhea.MinLeftSideBearing = r.ReadInt16()
	sfnt.Hhea.MinRightSideBearing = r.ReadInt16()
	sfnt.Hhea.XMaxExtent = r.ReadInt16()
	sfnt.Hhea.CaretSlopeRise = r.ReadInt16()
	sfnt.Hhea.CaretSlopeRun = r.ReadInt16()
	sfnt.Hhea.CaretOffset = r.ReadInt16()
	_ = r.ReadInt16() // reserved
	_ = r.ReadInt16() // reserved
	_ = r.ReadInt16() // reserved
	_ = r.ReadInt16() // reserved
	sfnt.Hhea.MetricDataFormat = r.ReadInt16()
	sfnt.Hhea.NumberOfHMetrics = r.ReadUint16()
	if sfnt.Maxp.NumGlyphs < sfnt.Hhea.NumberOfHMetrics || sfnt.Hhea.NumberOfHMetrics == 0 {
		return fmt.Errorf("hhea: bad numberOfHMetrics")
	}
	return nil
}

////////////////////////////////////////////////////////////////

type hmtxLongHorMetric struct {
	AdvanceWidth uint16
	Lsb          int16
}

type hmtxTable struct {
	HMetrics         []hmtxLongHorMetric
	LeftSideBearings []int16
}

func (hmtx *hmtxTable) LeftSideBearing(glyphID uint16) int16 {
	if uint16(len(hmtx.HMetrics)) <= glyphID {
		return hmtx.LeftSideBearings[glyphID-uint16(len(hmtx.HMetrics))]
	}
	return hmtx.HMetrics[glyphID].Lsb
}

func (hmtx *hmtxTable) Advance(glyphID uint16) uint16 {
	if uint16(len(hmtx.HMetrics)) <= glyphID {
		glyphID = uint16(len(hmtx.HMetrics)) - 1
	}
	return hmtx.HMetrics[glyphID].AdvanceWidth
}

func (sfnt *SFNT) parseHmtx() error {
	// requires data from hhea and maxp
	b, ok := sfnt.Tables["hmtx"]
	length := 4*uint32(sfnt.Hhea.NumberOfHMetrics) + 2*uint32(sfnt.Maxp.NumGlyphs-sfnt.Hhea.NumberOfHMetrics)
	if !ok {
		return fmt.Errorf("hmtx: missing table")
	} else if uint32(len(b)) != length {
		return fmt.Errorf("hmtx: bad table")
	}

	sfnt.Hmtx = &hmtxTable{}
	// numberOfHMetrics is smaller than numGlyphs
	sfnt.Hmtx.HMetrics = make([]hmtxLongHorMetric, sfnt.Hhea.NumberOfHMetrics)
	sfnt.Hmtx.LeftSideBearings = make([]int16, sfnt.Maxp.NumGlyphs-sfnt.Hhea.NumberOfHMetrics)

	r := newBinaryReader(b)
	for i := 0; i < int(sfnt.Hhea.NumberOfHMetrics); i++ {
		sfnt.Hmtx.HMetrics[i].AdvanceWidth = r.ReadUint16()
		sfnt.Hmtx.HMetrics[i].Lsb = r.ReadInt16()
	}
	for i := 0; i < int(sfnt.Maxp.NumGlyphs-sfnt.Hhea.NumberOfHMetrics); i++ {
		sfnt.Hmtx.LeftSideBearings[i] = r.ReadInt16()
	}
	return nil
}

////////////////////////////////////////////////////////////////

type kernPair struct {
	Key   uint32
	Value int16
}

type kernFormat0 struct {
	Coverage [8]bool
	Pairs    []kernPair
}

func (subtable *kernFormat0) Get(l, r uint16) int16 {
	key := uint32(l)<<16 | uint32(r)
	lo, hi := 0, len(subtable.Pairs)
	for lo < hi {
		mid := (lo + hi) / 2 // can be rounded down if odd
		pair := subtable.Pairs[mid]
		if pair.Key < key {
			lo = mid + 1
		} else if key < pair.Key {
			hi = mid
		} else {
			return pair.Value
		}
	}
	return 0
}

type kernTable struct {
	Subtables []kernFormat0
}

func (kern *kernTable) Get(l, r uint16) (k int16) {
	for _, subtable := range kern.Subtables {
		if !subtable.Coverage[1] {
			k += subtable.Get(l, r)
		} else if min := subtable.Get(l, r); k < min {
			// TODO: test
			k = min
		}
	}
	return
}

func (sfnt *SFNT) parseKern() error {
	b, ok := sfnt.Tables["kern"]
	if !ok {
		return fmt.Errorf("kern: missing table")
	} else if len(b) < 4 {
		return fmt.Errorf("kern: bad table")
	}

	r := newBinaryReader(b)
	version := r.ReadUint16()
	if version != 0 {
		// TODO: supported other kern versions
		return fmt.Errorf("kern: bad version")
	}

	nTables := r.ReadUint16()
	sfnt.Kern = &kernTable{}
	for j := 0; j < int(nTables); j++ {
		if r.Len() < 6 {
			return fmt.Errorf("kern: bad subtable %d", j)
		}

		subtable := kernFormat0{}
		startPos := r.Pos()
		subtableVersion := r.ReadUint16()
		if subtableVersion != 0 {
			// TODO: supported other kern subtable versions
			continue
		}
		length := r.ReadUint16()
		format := r.ReadUint8()
		subtable.Coverage = uint8ToFlags(r.ReadUint8())
		if format != 0 {
			// TODO: supported other kern subtable formats
			continue
		}
		if r.Len() < 8 {
			return fmt.Errorf("kern: bad subtable %d", j)
		}
		nPairs := r.ReadUint16()
		_ = r.ReadUint16() // searchRange
		_ = r.ReadUint16() // entrySelector
		_ = r.ReadUint16() // rangeShift
		if uint32(length) < 14+6*uint32(nPairs) {
			return fmt.Errorf("kern: bad length for subtable %d", j)
		}

		subtable.Pairs = make([]kernPair, nPairs)
		for i := 0; i < int(nPairs); i++ {
			subtable.Pairs[i].Key = r.ReadUint32()
			subtable.Pairs[i].Value = r.ReadInt16()
			if 0 < i && subtable.Pairs[i].Key <= subtable.Pairs[i-1].Key {
				return fmt.Errorf("kern: bad left right pair for subtable %d", j)
			}
		}

		// read unread bytes if length is bigger
		_ = r.ReadBytes(uint32(length) - (r.Pos() - startPos))
		sfnt.Kern.Subtables = append(sfnt.Kern.Subtables, subtable)
	}
	return nil
}

////////////////////////////////////////////////////////////////

type locaTable struct {
	format int16
	data   []byte
}

func (loca *locaTable) Get(glyphID uint16) uint32 {
	if loca.format == 0 && int(glyphID)*2 <= len(loca.data) {
		return binary.BigEndian.Uint32(loca.data[int(glyphID)*2:])
	} else if loca.format == 1 && int(glyphID)*4 <= len(loca.data) {
		return binary.BigEndian.Uint32(loca.data[int(glyphID)*4:])
	}
	return 0
}

func (sfnt *SFNT) parseLoca() error {
	b, ok := sfnt.Tables["loca"]
	if !ok {
		return fmt.Errorf("loca: missing table")
	}

	sfnt.Loca = &locaTable{
		format: sfnt.Head.IndexToLocFormat,
		data:   b,
	}
	//sfnt.Loca.Offsets = make([]uint32, sfnt.Maxp.NumGlyphs+1)
	//r := newBinaryReader(b)
	//if sfnt.Head.IndexToLocFormat == 0 {
	//	if uint32(len(b)) != 2*(uint32(sfnt.Maxp.NumGlyphs)+1) {
	//		return fmt.Errorf("loca: bad table")
	//	}
	//	for i := 0; i < int(sfnt.Maxp.NumGlyphs+1); i++ {
	//		sfnt.Loca.Offsets[i] = uint32(r.ReadUint16())
	//		if 0 < i && sfnt.Loca.Offsets[i] < sfnt.Loca.Offsets[i-1] {
	//			return fmt.Errorf("loca: bad offsets")
	//		}
	//	}
	//} else if sfnt.Head.IndexToLocFormat == 1 {
	//	if uint32(len(b)) != 4*(uint32(sfnt.Maxp.NumGlyphs)+1) {
	//		return fmt.Errorf("loca: bad table")
	//	}
	//	for i := 0; i < int(sfnt.Maxp.NumGlyphs+1); i++ {
	//		sfnt.Loca.Offsets[i] = r.ReadUint32()
	//		if 0 < i && sfnt.Loca.Offsets[i] < sfnt.Loca.Offsets[i-1] {
	//			return fmt.Errorf("loca: bad offsets")
	//		}
	//	}
	//}
	return nil
}

////////////////////////////////////////////////////////////////

type maxpTable struct {
	NumGlyphs             uint16
	MaxPoints             uint16
	MaxContours           uint16
	MaxCompositePoints    uint16
	MaxCompositeContours  uint16
	MaxZones              uint16
	MaxTwilightPoints     uint16
	MaxStorage            uint16
	MaxFunctionDefs       uint16
	MaxInstructionDefs    uint16
	MaxStackElements      uint16
	MaxSizeOfInstructions uint16
	MaxComponentElements  uint16
	MaxComponentDepth     uint16
}

func (sfnt *SFNT) parseMaxp() error {
	b, ok := sfnt.Tables["maxp"]
	if !ok {
		return fmt.Errorf("maxp: missing table")
	}

	sfnt.Maxp = &maxpTable{}
	r := newBinaryReader(b)
	version := r.ReadBytes(4)
	sfnt.Maxp.NumGlyphs = r.ReadUint16()
	if binary.BigEndian.Uint32(version) == 0x00005000 && !sfnt.IsTrueType && len(b) == 6 {
		return nil
	} else if binary.BigEndian.Uint32(version) == 0x00010000 && !sfnt.IsCFF && len(b) == 32 {
		sfnt.Maxp.MaxPoints = r.ReadUint16()
		sfnt.Maxp.MaxContours = r.ReadUint16()
		sfnt.Maxp.MaxCompositePoints = r.ReadUint16()
		sfnt.Maxp.MaxCompositeContours = r.ReadUint16()
		sfnt.Maxp.MaxZones = r.ReadUint16()
		sfnt.Maxp.MaxTwilightPoints = r.ReadUint16()
		sfnt.Maxp.MaxStorage = r.ReadUint16()
		sfnt.Maxp.MaxFunctionDefs = r.ReadUint16()
		sfnt.Maxp.MaxInstructionDefs = r.ReadUint16()
		sfnt.Maxp.MaxStackElements = r.ReadUint16()
		sfnt.Maxp.MaxSizeOfInstructions = r.ReadUint16()
		sfnt.Maxp.MaxComponentElements = r.ReadUint16()
		sfnt.Maxp.MaxComponentDepth = r.ReadUint16()
		return nil
	}
	return fmt.Errorf("maxp: bad table")
}

////////////////////////////////////////////////////////////////

type nameNameRecord struct {
	PlatformID uint16
	EncodingID uint16
	LanguageID uint16
	NameID     uint16
	Offset     uint16
	Length     uint16
}

type nameLangTagRecord struct {
	Offset uint16
	Length uint16
}

type nameTable struct {
	NameRecord    []nameNameRecord
	LangTagRecord []nameLangTagRecord
	Data          []byte
}

func (sfnt *SFNT) parseName() error {
	b, ok := sfnt.Tables["name"]
	if !ok {
		return fmt.Errorf("name: missing table")
	} else if len(b) < 6 {
		return fmt.Errorf("name: bad table")
	}

	sfnt.Name = &nameTable{}
	r := newBinaryReader(b)
	version := r.ReadUint16()
	if version != 0 && version != 1 {
		return fmt.Errorf("name: bad version")
	}
	count := r.ReadUint16()
	_ = r.ReadUint16() // storageOffset
	if uint32(len(b)) < 6+12*uint32(count) {
		return fmt.Errorf("name: bad table")
	}
	sfnt.Name.NameRecord = make([]nameNameRecord, count)
	for i := 0; i < int(count); i++ {
		sfnt.Name.NameRecord[i].PlatformID = r.ReadUint16()
		sfnt.Name.NameRecord[i].EncodingID = r.ReadUint16()
		sfnt.Name.NameRecord[i].LanguageID = r.ReadUint16()
		sfnt.Name.NameRecord[i].NameID = r.ReadUint16()
		sfnt.Name.NameRecord[i].Length = r.ReadUint16()
		sfnt.Name.NameRecord[i].Offset = r.ReadUint16()
	}
	if version == 1 {
		if uint32(len(b)) < 6+12*uint32(count)+2 {
			return fmt.Errorf("name: bad table")
		}
		langTagCount := r.ReadUint16()
		if uint32(len(b)) < 6+12*uint32(count)+2+4*uint32(langTagCount) {
			return fmt.Errorf("name: bad table")
		}
		for i := 0; i < int(langTagCount); i++ {
			sfnt.Name.LangTagRecord[i].Length = r.ReadUint16()
			sfnt.Name.LangTagRecord[i].Offset = r.ReadUint16()
		}
	}
	sfnt.Name.Data = b[r.Pos():]
	return nil
}

////////////////////////////////////////////////////////////////

type os2Table struct {
	Version                 uint16
	XAvgCharWidth           int16
	UsWeightClass           uint16
	UsWidthClass            uint16
	FsType                  uint16
	YSubscriptXSize         int16
	YSubscriptYSize         int16
	YSubscriptXOffset       int16
	YSubscriptYOffset       int16
	YSuperscriptXSize       int16
	YSuperscriptYSize       int16
	YSuperscriptXOffset     int16
	YSuperscriptYOffset     int16
	YStrikeoutSize          int16
	YStrikeoutPosition      int16
	SFamilyClass            int16
	BFamilyType             uint8
	BSerifStyle             uint8
	BWeight                 uint8
	BProportion             uint8
	BContrast               uint8
	BStrokeVariation        uint8
	BArmStyle               uint8
	BLetterform             uint8
	BMidline                uint8
	BXHeight                uint8
	UlUnicodeRange1         uint32
	UlUnicodeRange2         uint32
	UlUnicodeRange3         uint32
	UlUnicodeRange4         uint32
	AchVendID               [4]byte
	FsSelection             uint16
	UsFirstCharIndex        uint16
	UsLastCharIndex         uint16
	STypoAscender           int16
	STypoDescender          int16
	STypoLineGap            int16
	UsWinAscent             uint16
	UsWinDescent            uint16
	UlCodePageRange1        uint32
	UlCodePageRange2        uint32
	SxHeight                int16
	SCapHeight              int16
	UsDefaultChar           uint16
	UsBreakChar             uint16
	UsMaxContent            uint16
	UsLowerOpticalPointSize uint16
	UsUpperOpticalPointSize uint16
}

func (sfnt *SFNT) parseOS2() error {
	b, ok := sfnt.Tables["OS/2"]
	if !ok {
		return fmt.Errorf("OS/2: missing table")
	} else if len(b) < 68 {
		return fmt.Errorf("OS/2: bad table")
	}

	r := newBinaryReader(b)
	sfnt.OS2 = &os2Table{}
	sfnt.OS2.Version = r.ReadUint16()
	if 5 < sfnt.OS2.Version {
		return fmt.Errorf("OS/2: bad version")
	} else if sfnt.OS2.Version == 0 && len(b) != 68 && len(b) != 78 ||
		sfnt.OS2.Version == 1 && len(b) != 86 ||
		2 <= sfnt.OS2.Version && sfnt.OS2.Version <= 4 && len(b) != 96 ||
		sfnt.OS2.Version == 5 && len(b) != 100 {
		return fmt.Errorf("OS/2: bad table")
	}
	sfnt.OS2.XAvgCharWidth = r.ReadInt16()
	sfnt.OS2.UsWeightClass = r.ReadUint16()
	sfnt.OS2.UsWidthClass = r.ReadUint16()
	sfnt.OS2.FsType = r.ReadUint16()
	sfnt.OS2.YSubscriptXSize = r.ReadInt16()
	sfnt.OS2.YSubscriptYSize = r.ReadInt16()
	sfnt.OS2.YSubscriptXOffset = r.ReadInt16()
	sfnt.OS2.YSubscriptYOffset = r.ReadInt16()
	sfnt.OS2.YSuperscriptXSize = r.ReadInt16()
	sfnt.OS2.YSuperscriptYSize = r.ReadInt16()
	sfnt.OS2.YSuperscriptXOffset = r.ReadInt16()
	sfnt.OS2.YSuperscriptYOffset = r.ReadInt16()
	sfnt.OS2.YStrikeoutSize = r.ReadInt16()
	sfnt.OS2.YStrikeoutPosition = r.ReadInt16()
	sfnt.OS2.SFamilyClass = r.ReadInt16()
	sfnt.OS2.BFamilyType = r.ReadUint8()
	sfnt.OS2.BSerifStyle = r.ReadUint8()
	sfnt.OS2.BWeight = r.ReadUint8()
	sfnt.OS2.BProportion = r.ReadUint8()
	sfnt.OS2.BContrast = r.ReadUint8()
	sfnt.OS2.BStrokeVariation = r.ReadUint8()
	sfnt.OS2.BArmStyle = r.ReadUint8()
	sfnt.OS2.BLetterform = r.ReadUint8()
	sfnt.OS2.BMidline = r.ReadUint8()
	sfnt.OS2.BXHeight = r.ReadUint8()
	sfnt.OS2.UlUnicodeRange1 = r.ReadUint32()
	sfnt.OS2.UlUnicodeRange2 = r.ReadUint32()
	sfnt.OS2.UlUnicodeRange3 = r.ReadUint32()
	sfnt.OS2.UlUnicodeRange4 = r.ReadUint32()
	copy(sfnt.OS2.AchVendID[:], r.ReadBytes(4))
	sfnt.OS2.FsSelection = r.ReadUint16()
	sfnt.OS2.UsFirstCharIndex = r.ReadUint16()
	sfnt.OS2.UsLastCharIndex = r.ReadUint16()
	if 78 <= len(b) {
		sfnt.OS2.STypoAscender = r.ReadInt16()
		sfnt.OS2.STypoDescender = r.ReadInt16()
		sfnt.OS2.STypoLineGap = r.ReadInt16()
		sfnt.OS2.UsWinAscent = r.ReadUint16()
		sfnt.OS2.UsWinDescent = r.ReadUint16()
	}
	if sfnt.OS2.Version == 0 {
		return nil
	}
	sfnt.OS2.UlCodePageRange1 = r.ReadUint32()
	sfnt.OS2.UlCodePageRange2 = r.ReadUint32()
	if sfnt.OS2.Version == 1 {
		return nil
	}
	sfnt.OS2.SxHeight = r.ReadInt16()
	sfnt.OS2.SCapHeight = r.ReadInt16()
	sfnt.OS2.UsDefaultChar = r.ReadUint16()
	sfnt.OS2.UsBreakChar = r.ReadUint16()
	sfnt.OS2.UsMaxContent = r.ReadUint16()
	if 2 <= sfnt.OS2.Version && sfnt.OS2.Version <= 4 {
		return nil
	}
	sfnt.OS2.UsLowerOpticalPointSize = r.ReadUint16()
	sfnt.OS2.UsUpperOpticalPointSize = r.ReadUint16()
	return nil
}

func (sfnt *SFNT) estimateOS2() {
	contour, err := sfnt.GlyphContour(sfnt.GlyphIndex('x'))
	if err == nil {
		sfnt.OS2.SxHeight = contour.YMax
	}

	contour, err = sfnt.GlyphContour(sfnt.GlyphIndex('H'))
	if err == nil {
		sfnt.OS2.SCapHeight = contour.YMax
	}
}

////////////////////////////////////////////////////////////////

type postTable struct {
	ItalicAngle        uint32
	UnderlinePosition  int16
	UnderlineThickness int16
	IsFixedPitch       uint32
	MinMemType42       uint32
	MaxMemType42       uint32
	MinMemType1        uint32
	MaxMemType1        uint32
	GlyphNameIndex     []uint16
	stringData         []string
}

func (post *postTable) Get(glyphID uint16) string {
	if len(post.GlyphNameIndex) <= int(glyphID) {
		return ""
	}
	index := post.GlyphNameIndex[glyphID]
	if index < 258 {
		return macintoshGlyphNames[index]
	} else if len(post.stringData) < int(index)-258 {
		return ""
	}
	return post.stringData[index-258]
}

func (sfnt *SFNT) parsePost() error {
	// requires data from maxp
	b, ok := sfnt.Tables["post"]
	if !ok {
		return fmt.Errorf("post: missing table")
	} else if len(b) < 32 {
		return fmt.Errorf("post: bad table")
	}

	sfnt.Post = &postTable{}
	r := newBinaryReader(b)
	version := r.ReadBytes(4)
	sfnt.Post.ItalicAngle = r.ReadUint32()
	sfnt.Post.UnderlinePosition = r.ReadInt16()
	sfnt.Post.UnderlineThickness = r.ReadInt16()
	sfnt.Post.IsFixedPitch = r.ReadUint32()
	sfnt.Post.MinMemType42 = r.ReadUint32()
	sfnt.Post.MaxMemType42 = r.ReadUint32()
	sfnt.Post.MinMemType1 = r.ReadUint32()
	sfnt.Post.MaxMemType1 = r.ReadUint32()
	if binary.BigEndian.Uint32(version) == 0x00010000 && !sfnt.IsCFF && len(b) == 32 {
		return nil
	} else if binary.BigEndian.Uint32(version) == 0x00020000 && !sfnt.IsCFF && 34 <= len(b) {
		if r.ReadUint16() != sfnt.Maxp.NumGlyphs {
			return fmt.Errorf("post: numGlyphs does not match maxp table numGlyphs")
		}
		if uint32(len(b)) < 34+2*uint32(sfnt.Maxp.NumGlyphs) {
			return fmt.Errorf("post: bad table")
		}

		// get string data first
		r.Seek(34 + 2*uint32(sfnt.Maxp.NumGlyphs))
		for 2 <= r.Len() {
			length := r.ReadUint8()
			if r.Len() < uint32(length) {
				return fmt.Errorf("post: bad table")
			}
			sfnt.Post.stringData = append(sfnt.Post.stringData, r.ReadString(uint32(length)))
		}
		if r.Len() != 0 {
			return fmt.Errorf("post: bad table")
		}

		r.Seek(34)
		sfnt.Post.GlyphNameIndex = make([]uint16, sfnt.Maxp.NumGlyphs)
		for i := 0; i < int(sfnt.Maxp.NumGlyphs); i++ {
			index := r.ReadUint16()
			if 258 <= index && len(sfnt.Post.stringData) < int(index)-258 {
				return fmt.Errorf("post: bad table")
			}
			sfnt.Post.GlyphNameIndex[i] = index
		}
		return nil
	} else if binary.BigEndian.Uint32(version) == 0x00025000 && len(b) == 32 {
		return fmt.Errorf("post: version 2.5 not supported")
	} else if binary.BigEndian.Uint32(version) == 0x00030000 && len(b) == 32 {
		return nil
	}
	return fmt.Errorf("post: bad table")
}
