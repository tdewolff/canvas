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
		if table.offsets[i]-table.offsets[i-1] < table.lengths[i-1] {
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
	flavor := r.ReadString(4)
	if flavor == "ttcf" {
		return nil, fmt.Errorf("collections are unsupported")
	}
	length := r.ReadUint32()        // length
	numTables := r.ReadUint16()     // numTables
	reserved := r.ReadUint16()      // reserved
	totalSfntSize := r.ReadUint32() // totalSfntSize
	_ = r.ReadUint16()              // majorVersion
	_ = r.ReadUint16()              // minorVersion
	_ = r.ReadUint32()              // metaOffset
	_ = r.ReadUint32()              // metaLength
	_ = r.ReadUint32()              // metaOrigLength
	_ = r.ReadUint32()              // privOffset
	_ = r.ReadUint32()              // privLength

	frontSize := 44 + 20*uint32(numTables) // can never exceed uint32 as numTables is uint16
	if length <= frontSize {
		// table directory is bigger or equal to the file size
		return nil, ErrInvalidFontData
	} else if length != uint32(len(b)) {
		return nil, fmt.Errorf("length in header must match file size")
	} else if numTables == 0 {
		return nil, fmt.Errorf("numTables in header must not be zero")
	} else if reserved != 0 {
		return nil, fmt.Errorf("reserved in header must be zero")
	}

	tables := []woffTable{}
	tablePos := &tablePositions{[]uint32{}, []uint32{}}
	tablePos.Add(0, frontSize)
	sfntOffset := 12 + 16*uint32(numTables) // can never exceed uint32 as numTables is uint16
	for i := 0; i < int(numTables); i++ {
		// EOF already checked above
		tag := r.ReadString(4)
		offset := r.ReadUint32()
		compLength := r.ReadUint32()
		origLength := r.ReadUint32()
		origChecksum := r.ReadUint32()
		if length-compLength < offset {
			return nil, fmt.Errorf("table extends beyond file size")
		} else if origLength < compLength {
			return nil, fmt.Errorf("compressed table size is larger than decompressed size")
		} else if 0 < i && tag < tables[i-1].tag {
			return nil, fmt.Errorf("tables are not sorted alphabetically")
		}
		padding := (4 - origLength&3) & 3
		if math.MaxUint32-origLength < padding || math.MaxUint32-origLength-padding < sfntOffset {
			// both origLength and sfntOffset can overflow, check for both
			return nil, ErrInvalidFontData
		}
		sfntOffset += origLength + padding

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
		return nil, fmt.Errorf("totalSfntSize is incorrect")
	}
	if tablePos.HasOverlap() {
		return nil, fmt.Errorf("tables can not overlap")
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

	// write offset table
	if MaxMemory < totalSfntSize {
		return nil, ErrExceedsMemory
	}
	w := newBinaryWriter(make([]byte, totalSfntSize))
	w.WriteString(flavor)
	w.WriteUint16(numTables)
	w.WriteUint16(searchRange)
	w.WriteUint16(entrySelector)
	w.WriteUint16(rangeShift)

	// write table record entries
	sfntOffset = 12 + 16*uint32(numTables) // can never exceed uint32 as numTables is uint16
	for _, table := range tables {
		w.WriteUint32(binary.BigEndian.Uint32([]byte(table.tag)))
		w.WriteUint32(table.origChecksum)
		w.WriteUint32(sfntOffset) // offset already verified
		w.WriteUint32(table.origLength)
		sfntOffset += (table.origLength + 3) & 0xFFFFFFFC // add padding
	}

	// write tables
	var checksumAdjustment, checksumAdjustmentPos uint32
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

		dataLength := uint32(len(data))
		if dataLength != table.origLength {
			return nil, fmt.Errorf("decompressed table length must be equal to origLength")
		}

		// add padding
		nPadding := (4 - dataLength&3) & 3
		for i := 0; i < int(nPadding); i++ {
			data = append(data, 0x00)
		}
		if table.tag == "head" {
			if dataLength < 12 {
				return nil, ErrInvalidFontData
			}
			checksumAdjustment = binary.BigEndian.Uint32(data[8:])
			checksumAdjustmentPos = w.Len() + 8

			// to check checksum for head table, replace the overal checksum with zero and reset it at the end
			binary.BigEndian.PutUint32(data[8:], 0x00000000)
		}
		if calcChecksum(data) != table.origChecksum {
			return nil, fmt.Errorf("%s: bad checksum", table.tag)
		}

		w.WriteBytes(data)
	}
	if w.Len() != totalSfntSize {
		return nil, ErrInvalidFontData
	} else if checksumAdjustmentPos == 0 {
		return nil, ErrInvalidFontData
	}

	checksum := 0xB1B0AFBA - calcChecksum(w.Bytes())
	// TODO: (WOFF) master checksum seems right, but we don't throw an error if it is off
	//if checkSumAdjustment != checksum {
	//	return nil, fmt.Errorf("bad checksum")
	//}
	checksumAdjustment = checksum

	// replace overal checksum in head table
	buf := w.Bytes()
	binary.BigEndian.PutUint32(buf[checksumAdjustmentPos:], checksumAdjustment)
	return buf, nil
}
