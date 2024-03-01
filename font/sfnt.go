package font

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"time"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// MaxCmapSegments is the maximum number of cmap segments that will be accepted.
const MaxCmapSegments = 20000

// Pather is an interface to append a glyph's path to canvas.Path.
type Pather interface {
	MoveTo(float64, float64)
	LineTo(float64, float64)
	QuadTo(float64, float64, float64, float64)
	CubeTo(float64, float64, float64, float64, float64, float64)
	Close()
}

// Hinting specifies the type of hinting to use (none supported yes).
type Hinting int

// see Hinting
const (
	NoHinting Hinting = iota
	VerticalHinting
)

// SFNT is a parsed OpenType font.
type SFNT struct {
	Data              []byte
	Version           string
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
	CFF *cffTable

	// optional
	Kern *kernTable
	Vhea *vheaTable
	//Hdmx *hdmxTable // TODO
	Vmtx *vmtxTable
	Gpos *gposgsubTable
	Gsub *gposgsubTable
	Jsft *jsftTable
	//Gasp *gaspTable // TODO
	//Base *baseTable // TODO
	//Prep *baseTable // TODO
	//Fpgm *baseTable // TODO
	//Cvt *baseTable // TODO
}

// NumGlyphs returns the number of glyphs the font contains.
func (sfnt *SFNT) NumGlyphs() uint16 {
	return sfnt.Maxp.NumGlyphs
}

// GlyphIndex returns the glyphID for a given rune. When the rune is not defined it returns 0.
func (sfnt *SFNT) GlyphIndex(r rune) uint16 {
	return sfnt.Cmap.Get(r)
}

// GlyphName returns the name of the glyph.
func (sfnt *SFNT) GlyphName(glyphID uint16) string {
	return sfnt.Post.Get(glyphID)
}

// VerticalMetrics returns the ascender, descender, and line gap values. It returns the "win" values, or the "typo" values if OS/2.FsSelection.USE_TYPO_METRICS is set. If those are zero or not set, default to the "hhea" values.
func (sfnt *SFNT) VerticalMetrics() (uint16, uint16, uint16) {
	// see https://learn.microsoft.com/en-us/typography/opentype/spec/recom#baseline-to-baseline-distances
	var ascender, descender, lineGap uint16
	if 0 < sfnt.Hhea.Ascender {
		ascender = uint16(sfnt.Hhea.Ascender)
	}
	if sfnt.Hhea.Descender < 0 {
		descender = uint16(-sfnt.Hhea.Descender)
	}
	if 0 < sfnt.Hhea.LineGap {
		lineGap = uint16(sfnt.Hhea.LineGap)
	}

	if (sfnt.OS2.FsSelection & 0x0080) != 0 { // USE_TYPO_METRICS
		if 0 < sfnt.OS2.STypoAscender && sfnt.OS2.STypoDescender < 0 {
			ascender = uint16(sfnt.OS2.STypoAscender)
			descender = uint16(-sfnt.OS2.STypoDescender)
			if 0 < sfnt.OS2.STypoLineGap {
				lineGap = uint16(sfnt.OS2.STypoLineGap)
			} else {
				lineGap = 0
			}
		}
	} else {
		if sfnt.OS2.UsWinAscent != 0 && sfnt.OS2.UsWinDescent != 0 {
			ascender, descender = sfnt.OS2.UsWinAscent, sfnt.OS2.UsWinDescent
			externalLeading := int(sfnt.Hhea.Ascender-sfnt.Hhea.Descender+sfnt.Hhea.LineGap) - int(sfnt.OS2.UsWinAscent+sfnt.OS2.UsWinDescent)
			if 0 < externalLeading {
				lineGap = uint16(externalLeading)
			} else {
				lineGap = 0
			}
		}
	}
	return ascender, descender, lineGap
}

// GlyphPath draws the glyph's contour as a path to the pather interface. It will use the specified ppem (pixels-per-EM) for hinting purposes. The path is draws to the (x,y) coordinate and scaled using the given scale factor.
func (sfnt *SFNT) GlyphPath(p Pather, glyphID, ppem uint16, x, y, scale float64, hinting Hinting) error {
	if sfnt.IsTrueType {
		return sfnt.Glyf.ToPath(p, glyphID, ppem, x, y, scale, hinting)
	} else if sfnt.IsCFF {
		return sfnt.CFF.ToPath(p, glyphID, ppem, x, y, scale, hinting)
	}
	return fmt.Errorf("only TrueType and CFF are supported")
}

// GlyphAdvance returns the (horizontal) advance width of the glyph.
func (sfnt *SFNT) GlyphAdvance(glyphID uint16) uint16 {
	return sfnt.Hmtx.Advance(glyphID)
}

// GlyphVerticalAdvance returns the vertical advance width of the glyph.
func (sfnt *SFNT) GlyphVerticalAdvance(glyphID uint16) uint16 {
	if sfnt.Vmtx == nil {
		return sfnt.Head.UnitsPerEm
	}
	return sfnt.Vmtx.Advance(glyphID)
}

type boundsPather struct {
	xmin, ymin, xmax, ymax float64
}

func (p *boundsPather) MoveTo(x, y float64) {
	p.xmin = math.Min(p.xmin, x)
	p.ymin = math.Min(p.ymin, y)
	p.xmax = math.Max(p.xmax, x)
	p.ymax = math.Max(p.ymax, y)
}

func (p *boundsPather) LineTo(x, y float64) {
	p.xmin = math.Min(p.xmin, x)
	p.ymin = math.Min(p.ymin, y)
	p.xmax = math.Max(p.xmax, x)
	p.ymax = math.Max(p.ymax, y)
}

func (p *boundsPather) QuadTo(cpx, cpy, x, y float64) {
	p.xmin = math.Min(p.xmin, cpx)
	p.ymin = math.Min(p.ymin, cpy)
	p.xmax = math.Max(p.xmax, cpx)
	p.ymax = math.Max(p.ymax, cpy)
	p.xmin = math.Min(p.xmin, x)
	p.ymin = math.Min(p.ymin, y)
	p.xmax = math.Max(p.xmax, x)
	p.ymax = math.Max(p.ymax, y)
}

func (p *boundsPather) CubeTo(cp1x, cp1y, cp2x, cp2y, x, y float64) {
	p.xmin = math.Min(p.xmin, cp1x)
	p.ymin = math.Min(p.ymin, cp1y)
	p.xmax = math.Max(p.xmax, cp1x)
	p.ymax = math.Max(p.ymax, cp1y)
	p.xmin = math.Min(p.xmin, cp2x)
	p.ymin = math.Min(p.ymin, cp2y)
	p.xmax = math.Max(p.xmax, cp2x)
	p.ymax = math.Max(p.ymax, cp2y)
	p.xmin = math.Min(p.xmin, x)
	p.ymin = math.Min(p.ymin, y)
	p.xmax = math.Max(p.xmax, x)
	p.ymax = math.Max(p.ymax, y)
}

func (p *boundsPather) Close() {
}

// GlyphBounds returns the bounding rectangle (xmin,ymin,xmax,ymax) of the glyph.
func (sfnt *SFNT) GlyphBounds(glyphID uint16) (int16, int16, int16, int16, error) {
	if sfnt.IsTrueType {
		contour, err := sfnt.Glyf.Contour(glyphID, 0)
		if err != nil {
			return 0, 0, 0, 0, err
		}
		return contour.XMin, contour.YMin, contour.XMax, contour.YMax, nil
	} else if sfnt.IsCFF {
		p := &boundsPather{}
		if err := sfnt.CFF.ToPath(p, glyphID, 0, 0, 0, 1.0, NoHinting); err != nil {
			return 0, 0, 0, 0, err
		}
		return int16(p.xmin), int16(p.ymin), int16(math.Ceil(p.xmax)), int16(math.Ceil(p.ymax)), nil
	}
	return 0, 0, 0, 0, fmt.Errorf("only TrueType is supported")
}

// Kerning returns the kerning between two glyphs, i.e. the advance correction for glyph pairs.
func (sfnt *SFNT) Kerning(left, right uint16) int16 {
	if sfnt.Kern == nil {
		return 0
	}
	return sfnt.Kern.Get(left, right)
}

// ParseSFNT parses an OpenType file format (TTF, OTF, TTC). The index is used for font collections to select a single font.
func ParseSFNT(b []byte, index int) (*SFNT, error) {
	return parseSFNT(b, index, false)
}

// ParseEmbeddedSFNT is like ParseSFNT but for embedded font files in PDFs. It allows font files with fewer required tables.
func ParseEmbeddedSFNT(b []byte, index int) (*SFNT, error) {
	return parseSFNT(b, index, true)
}

func parseSFNT(b []byte, index int, embedded bool) (*SFNT, error) {
	if len(b) < 12 || uint(math.MaxUint32) < uint(len(b)) {
		return nil, ErrInvalidFontData
	}

	r := NewBinaryReader(b)
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
			length = uint32(len(b)) - offset
		} else {
			length = r.ReadUint32() - offset
		}
		if uint32(len(b))-8 < offset || uint32(len(b))-8-offset < length {
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

	tables := make(map[string][]byte, numTables)
	for i := 0; i < int(numTables); i++ {
		tag := r.ReadString(4)
		_ = r.ReadUint32() // checksum
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
			//checksumAdjustment := binary.BigEndian.Uint32(b[offset+8:])
			//binary.BigEndian.PutUint32(b[offset+8:], 0x00000000)
			//if calcChecksum(b[offset:offset+length+padding]) != checksum {
			//	return nil, fmt.Errorf("%s: bad checksum", tag)
			//}
			//binary.BigEndian.PutUint32(b[offset+8:], checksumAdjustment)
			//} else if calcChecksum(b[offset:offset+length+padding]) != checksum {
			//	return nil, fmt.Errorf("%s: bad checksum", tag)
		}
		tables[tag] = b[offset : offset+length : offset+length]
	}
	// TODO: check file checksum

	sfnt := &SFNT{}
	sfnt.Data = b
	sfnt.Version = sfntVersion
	sfnt.IsCFF = sfntVersion == "OTTO"
	sfnt.IsTrueType = sfntVersion == "true" || binary.BigEndian.Uint32([]byte(sfntVersion)) == 0x00010000
	sfnt.Tables = tables
	if isCollection {
		sfnt.Data = sfnt.Write()
	}

	var requiredTables []string
	if embedded {
		// see Table 126 of the PDF32000 specification
		if sfnt.IsTrueType {
			requiredTables = []string{"glyf", "head", "hhea", "hmtx", "loca", "maxp"}
		} else if sfnt.IsCFF {
			requiredTables = []string{"cmap", "CFF "}
		}
	} else {
		requiredTables = []string{"cmap", "head", "hhea", "hmtx", "maxp", "name", "post"} // OS/2 not required by TrueType
		if sfnt.IsTrueType {
			requiredTables = append(requiredTables, "glyf", "loca")
		} else if sfnt.IsCFF {
			_, hasCFF := tables["CFF "]
			_, hasCFF2 := tables["CFF2"]
			if !hasCFF && !hasCFF2 {
				return nil, fmt.Errorf("CFF: missing table")
			} else if hasCFF && hasCFF2 {
				return nil, fmt.Errorf("CFF2: CFF table already exists")
			}
		}
	}
	for _, requiredTable := range requiredTables {
		if _, ok := tables[requiredTable]; !ok {
			return nil, fmt.Errorf("%s: missing table", requiredTable)
		}
	}

	if embedded && sfnt.IsCFF {
		if err := sfnt.parseCFF(); err != nil {
			return nil, err
		} else if err := sfnt.parseCmap(); err != nil {
			return nil, err
		}
		return sfnt, nil
	}

	// maxp and hhea tables are required before parsing other tables
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
	for tableName := range tables {
		tableNames = append(tableNames, tableName)
	}
	sort.Strings(tableNames)
	for _, tableName := range tableNames {
		var err error
		switch tableName {
		case "CFF ":
			err = sfnt.parseCFF()
		case "CFF2":
			err = sfnt.parseCFF2()
		case "cmap":
			err = sfnt.parseCmap()
		case "glyf":
			err = sfnt.parseGlyf()
		case "GPOS":
			err = sfnt.parseGPOS()
		case "GSUB":
			err = sfnt.parseGSUB()
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
		case "vhea":
			err = sfnt.parseVhea()
		case "vmtx":
			err = sfnt.parseVmtx()
		}
		if err != nil {
			return nil, err
		}
	}
	if sfnt.OS2 != nil && sfnt.OS2.Version <= 1 {
		sfnt.estimateOS2()
	}
	return sfnt, nil
}

////////////////////////////////////////////////////////////////

type cmapFormat0 struct {
	GlyphIdArray [256]uint8

	UnicodeMap map[uint16]rune
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
	} else if subtable.UnicodeMap == nil {
		subtable.UnicodeMap = make(map[uint16]rune, 256)
		for r, id := range subtable.GlyphIdArray {
			subtable.UnicodeMap[uint16(id)] = rune(r)
		}
	}
	r, ok := subtable.UnicodeMap[glyphID]
	return r, ok
}

type cmapFormat4 struct {
	StartCode     []uint16
	EndCode       []uint16
	IdDelta       []int16
	IdRangeOffset []uint16
	GlyphIdArray  []uint16

	UnicodeMap map[uint16]rune
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
	if subtable.UnicodeMap == nil {
		subtable.UnicodeMap = map[uint16]rune{}
		n := len(subtable.StartCode)
		for i := 0; i < n; i++ {
			for r := subtable.StartCode[i]; r < subtable.EndCode[i]; r++ {
				var id uint16
				if subtable.IdRangeOffset[i] == 0 {
					// is modulo 65536 with the idDelta cast and addition overflow
					id = uint16(subtable.IdDelta[i]) + r
				} else {
					// idRangeOffset/2  ->  offset value to index of words
					// r-startCode  ->  difference of rune with startCode
					// -(n-1)  ->  subtract offset from the current idRangeOffset item
					index := int(subtable.IdRangeOffset[i]/2) + int(r-subtable.StartCode[i]) - (n - i)
					id = subtable.GlyphIdArray[index]
				}
				subtable.UnicodeMap[id] = rune(r)
			}
		}
	}
	r, ok := subtable.UnicodeMap[glyphID]
	return r, ok
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

	UnicodeMap map[uint16]rune
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
	if subtable.UnicodeMap == nil {
		subtable.UnicodeMap = map[uint16]rune{}
		for i := 0; i < len(subtable.StartCharCode); i++ {
			for r := subtable.StartCharCode[i]; r < subtable.EndCharCode[i]; r++ {
				id := uint16((uint32(r) - subtable.StartCharCode[i]) + subtable.StartGlyphID[i])
				subtable.UnicodeMap[id] = rune(r)
			}
		}
	}
	r, ok := subtable.UnicodeMap[glyphID]
	return r, ok
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
	r := NewBinaryReader(b)
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
		rs := NewBinaryReader(b[offset:])
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
				if segCount%2 != 0 || segCount == 0 {
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
				if subtable.StartCode[segCount-1] != 0xFFFF || subtable.EndCode[segCount-1] != 0xFFFF {
					return fmt.Errorf("cmap: bad last startCode or endCode in subtable %d", j)
				}

				subtable.IdDelta = make([]int16, segCount)
				for i := 0; i < int(segCount-1); i++ {
					subtable.IdDelta[i] = rs.ReadInt16()
				}
				_ = rs.ReadUint16() // last value may be invalid
				subtable.IdDelta[segCount-1] = 1

				glyphIdArrayLength := rs.Len() - 2*uint32(segCount)
				if glyphIdArrayLength%2 != 0 {
					return fmt.Errorf("cmap: bad subtable %d", j)
				}
				glyphIdArrayLength /= 2

				subtable.IdRangeOffset = make([]uint16, segCount)
				for i := 0; i < int(segCount-1); i++ {
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
				_ = rs.ReadUint16() // last value may be invalid
				subtable.IdRangeOffset[segCount-1] = 0

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
	r := NewBinaryReader(b)
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
	sfnt.Head.Flags = Uint16ToFlags(r.ReadUint16())
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
	sfnt.Head.MacStyle = Uint16ToFlags(r.ReadUint16())
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
	r := NewBinaryReader(b)
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

type vheaTable struct {
	Ascender             int16
	Descender            int16
	LineGap              int16
	AdvanceHeightMax     int16
	MinTopSideBearing    int16
	MinBottomSideBearing int16
	YMaxExtent           int16
	CaretSlopeRise       int16
	CaretSlopeRun        int16
	CaretOffset          int16
	MetricDataFormat     int16
	NumberOfVMetrics     uint16
}

func (sfnt *SFNT) parseVhea() error {
	// requires data from maxp
	b, ok := sfnt.Tables["vhea"]
	if !ok {
		return fmt.Errorf("vhea: missing table")
	} else if len(b) != 36 {
		return fmt.Errorf("vhea: bad table")
	}

	sfnt.Vhea = &vheaTable{}
	r := NewBinaryReader(b)
	majorVersion := r.ReadUint16()
	minorVersion := r.ReadUint16()
	if majorVersion != 1 && minorVersion != 0 && minorVersion != 1 {
		return fmt.Errorf("vhea: bad version")
	}
	sfnt.Vhea.Ascender = r.ReadInt16()
	sfnt.Vhea.Descender = r.ReadInt16()
	sfnt.Vhea.LineGap = r.ReadInt16()
	sfnt.Vhea.AdvanceHeightMax = r.ReadInt16()
	sfnt.Vhea.MinTopSideBearing = r.ReadInt16()
	sfnt.Vhea.MinBottomSideBearing = r.ReadInt16()
	sfnt.Vhea.YMaxExtent = r.ReadInt16()
	sfnt.Vhea.CaretSlopeRise = r.ReadInt16()
	sfnt.Vhea.CaretSlopeRun = r.ReadInt16()
	sfnt.Vhea.CaretOffset = r.ReadInt16()
	_ = r.ReadInt16() // reserved
	_ = r.ReadInt16() // reserved
	_ = r.ReadInt16() // reserved
	_ = r.ReadInt16() // reserved
	sfnt.Vhea.MetricDataFormat = r.ReadInt16()
	sfnt.Vhea.NumberOfVMetrics = r.ReadUint16()
	if sfnt.Maxp.NumGlyphs < sfnt.Vhea.NumberOfVMetrics || sfnt.Vhea.NumberOfVMetrics == 0 {
		return fmt.Errorf("vhea: bad numberOfVMetrics")
	}
	return nil
}

////////////////////////////////////////////////////////////////

type hmtxLongHorMetric struct {
	AdvanceWidth    uint16
	LeftSideBearing int16
}

type hmtxTable struct {
	HMetrics         []hmtxLongHorMetric
	LeftSideBearings []int16
}

func (hmtx *hmtxTable) LeftSideBearing(glyphID uint16) int16 {
	if uint16(len(hmtx.HMetrics)) <= glyphID {
		return hmtx.LeftSideBearings[glyphID-uint16(len(hmtx.HMetrics))]
	}
	return hmtx.HMetrics[glyphID].LeftSideBearing
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

	r := NewBinaryReader(b)
	for i := 0; i < int(sfnt.Hhea.NumberOfHMetrics); i++ {
		sfnt.Hmtx.HMetrics[i].AdvanceWidth = r.ReadUint16()
		sfnt.Hmtx.HMetrics[i].LeftSideBearing = r.ReadInt16()
	}
	for i := 0; i < int(sfnt.Maxp.NumGlyphs-sfnt.Hhea.NumberOfHMetrics); i++ {
		sfnt.Hmtx.LeftSideBearings[i] = r.ReadInt16()
	}
	return nil
}

////////////////////////////////////////////////////////////////

type vmtxLongVerMetric struct {
	AdvanceHeight  uint16
	TopSideBearing int16
}

type vmtxTable struct {
	VMetrics        []vmtxLongVerMetric
	TopSideBearings []int16
}

func (vmtx *vmtxTable) TopSideBearing(glyphID uint16) int16 {
	if uint16(len(vmtx.VMetrics)) <= glyphID {
		return vmtx.TopSideBearings[glyphID-uint16(len(vmtx.VMetrics))]
	}
	return vmtx.VMetrics[glyphID].TopSideBearing
}

func (vmtx *vmtxTable) Advance(glyphID uint16) uint16 {
	if uint16(len(vmtx.VMetrics)) <= glyphID {
		glyphID = uint16(len(vmtx.VMetrics)) - 1
	}
	return vmtx.VMetrics[glyphID].AdvanceHeight
}

func (sfnt *SFNT) parseVmtx() error {
	// requires data from vhea and maxp
	if sfnt.Vhea == nil {
		return fmt.Errorf("vhea: missing table")
	}

	b, ok := sfnt.Tables["vmtx"]
	length := 4*uint32(sfnt.Vhea.NumberOfVMetrics) + 2*uint32(sfnt.Maxp.NumGlyphs-sfnt.Vhea.NumberOfVMetrics)
	if !ok {
		return fmt.Errorf("vmtx: missing table")
	} else if uint32(len(b)) != length {
		return fmt.Errorf("vmtx: bad table")
	}

	sfnt.Vmtx = &vmtxTable{}
	// numberOfVMetrics is smaller than numGlyphs
	sfnt.Vmtx.VMetrics = make([]vmtxLongVerMetric, sfnt.Vhea.NumberOfVMetrics)
	sfnt.Vmtx.TopSideBearings = make([]int16, sfnt.Maxp.NumGlyphs-sfnt.Vhea.NumberOfVMetrics)

	r := NewBinaryReader(b)
	for i := 0; i < int(sfnt.Vhea.NumberOfVMetrics); i++ {
		sfnt.Vmtx.VMetrics[i].AdvanceHeight = r.ReadUint16()
		sfnt.Vmtx.VMetrics[i].TopSideBearing = r.ReadInt16()
	}
	for i := 0; i < int(sfnt.Maxp.NumGlyphs-sfnt.Vhea.NumberOfVMetrics); i++ {
		sfnt.Vmtx.TopSideBearings[i] = r.ReadInt16()
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
		if !subtable.Coverage[1] { // kerning values
			k += subtable.Get(l, r)
		} else if min := subtable.Get(l, r); k < min { // minimum values (usually last subtable)
			k = min // TODO: test minimal kerning
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

	r := NewBinaryReader(b)
	majorVersion := r.ReadUint16()
	if majorVersion != 0 && majorVersion != 1 {
		return fmt.Errorf("kern: bad version %d", majorVersion)
	}

	var nTables uint32
	if majorVersion == 0 {
		nTables = uint32(r.ReadUint16())
	} else if majorVersion == 1 {
		minorVersion := r.ReadUint16()
		if minorVersion != 0 {
			return fmt.Errorf("kern: bad minor version %d", minorVersion)
		}
		nTables = r.ReadUint32()
	}

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
		subtable.Coverage = Uint8ToFlags(r.ReadUint8())
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
		if uint32(length) < 14+6*uint32(nPairs) || r.Len() < uint32(length) {
			if j+1 == int(nTables) {
				// for some fonts the subtable's length exceeds what can fit in a uint16
				// we allow only the last subtable to exceed as long as it stays within the table
				pairsLength := 6 * uint32(nPairs)
				pairsLength &= 0xFFFF
				if uint32(length) != 14+pairsLength || r.Len() < pairsLength {
					return fmt.Errorf("kern: bad length for subtable %d", j)
				}
			} else {
				return fmt.Errorf("kern: bad length for subtable %d", j)
			}
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
	r := NewBinaryReader(b)
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

type nameRecord struct {
	Platform PlatformID
	Encoding EncodingID
	Language uint16
	Name     NameID
	Value    []byte
}

func (record nameRecord) String() string {
	var decoder *encoding.Decoder
	if record.Platform == PlatformUnicode || record.Platform == PlatformWindows {
		decoder = unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewDecoder()
	} else if record.Platform == PlatformMacintosh && record.Encoding == EncodingMacintoshRoman {
		decoder = charmap.Macintosh.NewDecoder()
	}
	s, _, err := transform.String(decoder, string(record.Value))
	if err == nil {
		return s
	}
	return string(record.Value)
}

type nameLangTagRecord struct {
	Value []byte
}

func (record nameLangTagRecord) String() string {
	decoder := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewDecoder()
	s, _, err := transform.String(decoder, string(record.Value))
	if err == nil {
		return s
	}
	return string(record.Value)
}

type nameTable struct {
	NameRecord []nameRecord
	LangTag    []nameLangTagRecord
}

func (t *nameTable) Get(name NameID) []nameRecord {
	records := []nameRecord{}
	for _, record := range t.NameRecord {
		if record.Name == name {
			records = append(records, record)
		}
	}
	return records
}

func (sfnt *SFNT) parseName() error {
	b, ok := sfnt.Tables["name"]
	if !ok {
		return fmt.Errorf("name: missing table")
	} else if len(b) < 6 {
		return fmt.Errorf("name: bad table")
	}

	sfnt.Name = &nameTable{}
	r := NewBinaryReader(b)
	version := r.ReadUint16()
	if version != 0 && version != 1 {
		return fmt.Errorf("name: bad version")
	}
	count := r.ReadUint16()
	storageOffset := r.ReadUint16()
	if uint32(len(b)) < 6+12*uint32(count) || uint16(len(b)) < storageOffset {
		return fmt.Errorf("name: bad table")
	}
	sfnt.Name.NameRecord = make([]nameRecord, count)
	for i := 0; i < int(count); i++ {
		sfnt.Name.NameRecord[i].Platform = PlatformID(r.ReadUint16())
		sfnt.Name.NameRecord[i].Encoding = EncodingID(r.ReadUint16())
		sfnt.Name.NameRecord[i].Language = r.ReadUint16()
		sfnt.Name.NameRecord[i].Name = NameID(r.ReadUint16())

		length := r.ReadUint16()
		offset := r.ReadUint16()
		if uint16(len(b))-storageOffset < offset || uint16(len(b))-storageOffset-offset < length {
			return fmt.Errorf("name: bad table")
		}
		sfnt.Name.NameRecord[i].Value = b[storageOffset+offset : storageOffset+offset+length]
	}
	if version == 1 {
		if uint32(len(b)) < 6+12*uint32(count)+2 {
			return fmt.Errorf("name: bad table")
		}
		langTagCount := r.ReadUint16()
		if uint32(len(b)) < 6+12*uint32(count)+2+4*uint32(langTagCount) {
			return fmt.Errorf("name: bad table")
		}
		sfnt.Name.LangTag = make([]nameLangTagRecord, langTagCount)
		for i := 0; i < int(langTagCount); i++ {
			length := r.ReadUint16()
			offset := r.ReadUint16()
			if uint16(len(b))-storageOffset < offset || uint16(len(b))-storageOffset-offset < length {
				return fmt.Errorf("name: bad table")
			}
			sfnt.Name.LangTag[i].Value = b[storageOffset+offset : storageOffset+offset+length]
		}
	}
	if r.Pos() != uint32(storageOffset) {
		return fmt.Errorf("name: bad storageOffset")
	}
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

	r := NewBinaryReader(b)
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

type bboxPather struct {
	xMin, xMax, yMin, yMax float64
}

func (p *bboxPather) MoveTo(x float64, y float64) {
	p.xMin = math.Min(p.xMin, x)
	p.xMax = math.Max(p.xMax, x)
	p.yMin = math.Min(p.yMin, y)
	p.yMax = math.Max(p.yMax, y)
}

func (p *bboxPather) LineTo(x float64, y float64) {
	p.xMin = math.Min(p.xMin, x)
	p.xMax = math.Max(p.xMax, x)
	p.yMin = math.Min(p.yMin, y)
	p.yMax = math.Max(p.yMax, y)
}

func (p *bboxPather) QuadTo(cpx float64, cpy float64, x float64, y float64) {
	p.xMin = math.Min(math.Min(p.xMin, cpx), x)
	p.xMax = math.Max(math.Max(p.xMax, cpx), x)
	p.yMin = math.Min(math.Min(p.yMin, cpy), y)
	p.yMax = math.Max(math.Max(p.yMax, cpy), y)
}

func (p *bboxPather) CubeTo(cpx1 float64, cpy1 float64, cpx2 float64, cpy2 float64, x float64, y float64) {
	p.xMin = math.Min(math.Min(math.Min(p.xMin, cpx1), cpx2), x)
	p.xMax = math.Max(math.Max(math.Max(p.xMax, cpx1), cpx2), x)
	p.yMin = math.Min(math.Min(math.Min(p.yMin, cpy1), cpy2), y)
	p.yMax = math.Max(math.Max(math.Max(p.yMax, cpy1), cpy2), y)
}

func (p *bboxPather) Close() {
}

func (sfnt *SFNT) estimateOS2() {
	if sfnt.IsTrueType {
		contour, err := sfnt.Glyf.Contour(sfnt.GlyphIndex('x'), 0)
		if err == nil {
			sfnt.OS2.SxHeight = contour.YMax
		}

		contour, err = sfnt.Glyf.Contour(sfnt.GlyphIndex('H'), 0)
		if err == nil {
			sfnt.OS2.SCapHeight = contour.YMax
		}
	} else if sfnt.IsCFF {
		p := &bboxPather{}
		if err := sfnt.CFF.ToPath(p, sfnt.GlyphIndex('x'), 0, 0, 0, 1.0, NoHinting); err == nil {
			sfnt.OS2.SxHeight = int16(p.yMax)
		}

		p = &bboxPather{}
		if err := sfnt.CFF.ToPath(p, sfnt.GlyphIndex('H'), 0, 0, 0, 1.0, NoHinting); err == nil {
			sfnt.OS2.SCapHeight = int16(p.yMax)
		}
	}
}

////////////////////////////////////////////////////////////////

type postTable struct {
	ItalicAngle        float64
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
	// requires data from maxp and CFF2
	b, ok := sfnt.Tables["post"]
	if !ok {
		return fmt.Errorf("post: missing table")
	} else if len(b) < 32 {
		return fmt.Errorf("post: bad table")
	}

	_, isCFF2 := sfnt.Tables["CFF2"]

	sfnt.Post = &postTable{}
	r := NewBinaryReader(b)
	version := r.ReadUint32()
	sfnt.Post.ItalicAngle = float64(r.ReadInt32()) / (1 << 16)
	sfnt.Post.UnderlinePosition = r.ReadInt16()
	sfnt.Post.UnderlineThickness = r.ReadInt16()
	sfnt.Post.IsFixedPitch = r.ReadUint32()
	sfnt.Post.MinMemType42 = r.ReadUint32()
	sfnt.Post.MaxMemType42 = r.ReadUint32()
	sfnt.Post.MinMemType1 = r.ReadUint32()
	sfnt.Post.MaxMemType1 = r.ReadUint32()
	if version == 0x00010000 && sfnt.IsTrueType && len(b) == 32 {
		sfnt.Post.GlyphNameIndex = make([]uint16, 258)
		for i := 0; i < 258; i++ {
			sfnt.Post.GlyphNameIndex[i] = uint16(i)
		}
		return nil
	} else if version == 0x00020000 && (sfnt.IsTrueType || isCFF2) && 34 <= len(b) {
		// can be used for TrueType and CFF2 fonts, we check for this in the CFF table
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
			if r.Len() < uint32(length) || 63 < length {
				return fmt.Errorf("post: bad stringData")
			}
			sfnt.Post.stringData = append(sfnt.Post.stringData, r.ReadString(uint32(length)))
		}
		if 1 < r.Len() {
			return fmt.Errorf("post: bad stringData")
		}

		r.Seek(34)
		sfnt.Post.GlyphNameIndex = make([]uint16, sfnt.Maxp.NumGlyphs)
		for i := 0; i < int(sfnt.Maxp.NumGlyphs); i++ {
			index := r.ReadUint16()
			if 258 <= index && len(sfnt.Post.stringData) < int(index)-258 {
				return fmt.Errorf("post: bad stringData")
			}
			sfnt.Post.GlyphNameIndex[i] = index
		}
		return nil
	} else if version == 0x00025000 && sfnt.IsTrueType && len(b) == 32 {
		return fmt.Errorf("post: version 2.5 not supported")
	} else if version == 0x00030000 && len(b) == 32 {
		// no PostScript glyph names provided
		return nil
	}
	return fmt.Errorf("post: bad version")
}
