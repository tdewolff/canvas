package font

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

type woffTable struct {
	tag          string
	offset       uint32
	length       uint32
	origLength   uint32
	origChecksum uint32
}

type tablePositions struct {
	offsets, lengths []uint32
}

func (table *tablePositions) Add(offset, length uint32) {
	i := 0
	for i < len(table.offsets) && table.offsets[i] < offset {
		i++
	}
	if i == len(table.offsets) {
		table.offsets = append(table.offsets, offset)
		table.lengths = append(table.lengths, length)
	} else {
		table.offsets = append(table.offsets[:i], append([]uint32{offset}, table.offsets[i:]...)...)
		table.lengths = append(table.lengths[:i], append([]uint32{length}, table.lengths[i:]...)...)
	}
}

func (table *tablePositions) HasOverlap() bool {
	for i := 1; i < len(table.offsets); i++ {
		if table.offsets[i] < table.offsets[i-1]+table.lengths[i-1] {
			return true
		}
	}
	return false
}

// ParseWOFF parses the WOFF font format and returns its contained SFNT font format (TTF or OTF).
// See https://www.w3.org/TR/WOFF/
func ParseWOFF(b []byte) ([]byte, error) {
	if len(b) < 44 {
		return nil, ErrInvalidFontData
	}

	r := newBinaryReader(b)
	signature := r.ReadString(4)
	if signature != "wOFF" {
		return nil, fmt.Errorf("bad signature")
	}
	flavor := r.ReadUint32()
	if uint32ToString(flavor) == "ttcf" {
		return nil, fmt.Errorf("collections are unsupported")
	}
	length := r.ReadUint32()         // length
	numTables := r.ReadUint16()      // numTables
	reserved := r.ReadUint16()       // reserved
	totalSfntSize := r.ReadUint32()  // totalSfntSize
	_ = r.ReadUint16()               // majorVersion
	_ = r.ReadUint16()               // minorVersion
	metaOffset := r.ReadUint32()     // metaOffset
	metaLength := r.ReadUint32()     // metaLength
	metaOrigLength := r.ReadUint32() // metaOrigLength
	privOffset := r.ReadUint32()     // privOffset
	privLength := r.ReadUint32()     // privLength

	frontSize := 44 + 20*uint32(numTables) // can never exceed uint32 as numTables is uint16
	if length != uint32(len(b)) || numTables == 0 || length <= frontSize || reserved != 0 {
		// file size does not match length in header
		// or numTables is zero
		// or table directory is bigger or equal to the file size
		// or reserved is not zero (stated explicitly by the spec)
		return nil, ErrInvalidFontData
	}

	tables := []woffTable{}
	tablePos := &tablePositions{[]uint32{}, []uint32{}}
	tablePos.Add(0, frontSize)
	sfntOffset := 12 + 16*uint32(numTables) // can never exceed uint32 as numTables is uint16
	for i := 0; i < int(numTables); i++ {
		// EOF already checked above
		tag := uint32ToString(r.ReadUint32())
		offset := r.ReadUint32()
		compLength := r.ReadUint32()
		origLength := r.ReadUint32()
		origChecksum := r.ReadUint32()
		if length-compLength < offset || origLength < compLength || 0 < i && tag < tables[i-1].tag {
			// table extends beyond file
			// or compressed size larger then uncompressed
			// or tables not sorted alphabetically
			return nil, ErrInvalidFontData
		}
		sfntOrigLength := (origLength + 3) & 0xFFFFFFFC // add padding
		if math.MaxUint32-sfntOrigLength < sfntOffset {
			return nil, ErrInvalidFontData
		}
		sfntOffset += sfntOrigLength

		tables = append(tables, woffTable{
			tag:          tag,
			offset:       offset,
			length:       compLength,
			origLength:   origLength,
			origChecksum: origChecksum,
		})
		tablePos.Add(offset, compLength)
	}

	if totalSfntSize != sfntOffset {
		// totalSfntSize does not coincide with sfnt header + table directory + data blocks
		return nil, ErrInvalidFontData
	}
	if (metaOffset == 0) != (metaLength == 0) || (metaOffset == 0) != (metaOrigLength == 0) {
		return nil, ErrInvalidFontData
	} else if metaOffset != 0 {
		tablePos.Add(metaOffset, metaLength)
	}
	if (privOffset == 0) != (privLength == 0) {
		return nil, ErrInvalidFontData
	} else if privOffset != 0 {
		tablePos.Add(privOffset, privLength)
	}
	if tablePos.HasOverlap() {
		return nil, ErrInvalidFontData
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

	w := newBinaryWriter(make([]byte, totalSfntSize))
	w.WriteUint32(flavor)
	w.WriteUint16(numTables)
	w.WriteUint16(searchRange)
	w.WriteUint16(entrySelector)
	w.WriteUint16(rangeShift)

	sfntOffset = 12 + 16*uint32(numTables) // can never exceed uint32 as numTables is uint16
	for _, table := range tables {
		w.WriteUint32(binary.BigEndian.Uint32([]byte(table.tag)))
		w.WriteUint32(table.origChecksum)
		w.WriteUint32(sfntOffset) // offset already verified
		w.WriteUint32(table.origLength)
		sfntOffset += (table.origLength + 3) & 0xFFFFFFFC // add padding
	}

	var iCheckSumAdjustment uint32
	var checkSumAdjustment uint32
	for _, table := range tables {
		data := b[table.offset : table.offset+table.length : table.offset+table.length]
		if table.length != table.origLength {
			var buf bytes.Buffer
			r, err := zlib.NewReader(bytes.NewReader(data))
			if err != nil {
				return nil, fmt.Errorf("%s: %v", table.tag, err)
			}
			if _, err = io.Copy(&buf, r); err != nil {
				return nil, fmt.Errorf("%s: %v", table.tag, err)
			}
			if err = r.Close(); err != nil {
				return nil, fmt.Errorf("%s: %v", table.tag, err)
			}
			data = buf.Bytes()
		}

		if len(data) != int(table.origLength) {
			return nil, ErrInvalidFontData
		}

		// add padding
		nPadding := (4 - len(data)&3) & 3
		for i := 0; i < nPadding; i++ {
			data = append(data, 0x00)
		}
		if table.tag == "head" {
			if len(data) < 12 {
				return nil, ErrInvalidFontData
			}
			checkSumAdjustment = binary.BigEndian.Uint32(data[8:])
			iCheckSumAdjustment = w.Len() + 8

			// to check checksum for head table, replace the overal checksum with zero and reset it at the end
			binary.BigEndian.PutUint32(data[8:], 0x00000000)
			if calcChecksum(data) != table.origChecksum {
				return nil, fmt.Errorf("%s: bad checksum", table.tag)
			}
		} else if calcChecksum(data) != table.origChecksum {
			return nil, fmt.Errorf("%s: bad checksum", table.tag)
		}

		w.WriteBytes(data)
	}
	if w.Len() != totalSfntSize {
		return nil, ErrInvalidFontData
	}

	if iCheckSumAdjustment == 0 {
		return nil, ErrInvalidFontData
	} else {
		// TODO: (WOFF) overal checksum is off by a little...
		//fmt.Println(binary.BigEndian.Uint32(w.Bytes()[iCheckSumAdjustment:]))
		//checksum := 0xB1B0AFBA - calcChecksum(w.Bytes())
		//if checkSumAdjustment != checksum {
		//return nil, 0, fmt.Errorf("bad checksum")
	}

	// replace overal checksum in head table
	buf := w.Bytes()
	binary.BigEndian.PutUint32(buf[iCheckSumAdjustment:], checkSumAdjustment)
	return buf, nil
}
