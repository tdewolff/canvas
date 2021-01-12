package font

import (
	"encoding/binary"
	"math"
	"sort"
	"time"
)

func (sfnt *SFNT) Subset(glyphIDs []uint16) ([]byte, []uint16) {
	if sfnt.IsCFF {
		// TODO: support CFF
		glyphIDs = glyphIDs[:0]
		for glyphID := uint16(0); glyphID < sfnt.Maxp.NumGlyphs; glyphID++ {
			glyphIDs = append(glyphIDs, glyphID)
		}
		return sfnt.Data, glyphIDs
	}

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
		if _, ok := sfnt.Tables["CFF2"]; ok {
			tags = append(tags, "CFF2")
		} else {
			tags = append(tags, "CFF ")
		}
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
	iGlyf := 0
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
			w.WriteBytes(head[36:50])

			// glyf comes before head
			if lengths[iGlyf] <= math.MaxUint16 {
				w.WriteInt16(0) // indexToLocFormat
			} else {
				w.WriteInt16(1) // indexToLocFormat
			}
			w.WriteBytes(head[52:])
		case "glyf":
			iGlyf = i
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
			// glyf comes before loca
			if lengths[iGlyf] <= math.MaxUint16 {
				pos := uint16(0)
				for _, glyphID := range glyphIDs {
					w.WriteUint16(uint16(pos))
					pos1, _ := sfnt.Loca.Get(glyphID)
					pos2, _ := sfnt.Loca.Get(glyphID + 1)
					pos += uint16(pos2 - pos1)
				}
				w.WriteUint16(pos)
			} else {
				pos := uint32(0)
				for _, glyphID := range glyphIDs {
					w.WriteUint32(pos)
					pos1, _ := sfnt.Loca.Get(glyphID)
					pos2, _ := sfnt.Loca.Get(glyphID + 1)
					pos += pos2 - pos1
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
