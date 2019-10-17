package font

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
)

type woffTable struct {
	tag          string
	offset       uint32
	length       uint32
	origLength   uint32
	origChecksum uint32
}

// ParseWOFF parses the WOFF font format and returns its contained SFNT font format (TTF or OTF).
// See https://www.w3.org/TR/WOFF/
func ParseWOFF(b []byte) ([]byte, uint32, error) {
	if len(b) < 44 {
		return nil, 0, fmt.Errorf("invalid data")
	}

	r := newBinaryReader(b)
	signature := r.ReadString(4)
	if signature != "wOFF" {
		return nil, 0, fmt.Errorf("invalid data")
	}
	flavor := r.ReadUint32()
	_ = r.ReadUint32() // length
	numTables := r.ReadUint16()
	reserved := r.ReadUint16()      // reserved
	totalSfntSize := r.ReadUint32() // totalSfntSize
	_ = r.ReadUint16()              // majorVersion
	_ = r.ReadUint16()              // minorVersion
	_ = r.ReadUint32()              // metaOffset
	_ = r.ReadUint32()              // metaLength
	_ = r.ReadUint32()              // metaOrigLength
	_ = r.ReadUint32()              // privOffset
	_ = r.ReadUint32()              // privLength

	frontSize := uint32(12 + 16*int(numTables))
	if r.EOF() || numTables == 0 || r.Len() < frontSize || reserved != 0 {
		return nil, 0, ErrInvalidFontData
	}

	tables := []woffTable{}
	for i := 0; i < int(numTables); i++ {
		// EOF already checked above
		tag := uint32ToString(r.ReadUint32())
		offset := r.ReadUint32()
		compLength := r.ReadUint32()
		origLength := r.ReadUint32()
		origChecksum := r.ReadUint32()
		if uint32(len(b)) < offset+compLength {
			return nil, 0, ErrInvalidFontData // table extends beyond file
		}
		if 0 < i && tag < tables[i-1].tag {
			return nil, 0, ErrInvalidFontData // tables not sorted alphabetically
		}
		if origLength < compLength {
			return nil, 0, ErrInvalidFontData
		}
		tables = append(tables, woffTable{
			tag:          tag,
			offset:       offset,
			length:       compLength,
			origLength:   origLength,
			origChecksum: origChecksum,
		})
	}

	// TODO: (WOFF) check table overlap and padding

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

	sfntOffset := uint32(12 + 16*int(numTables))
	for _, table := range tables {
		w.WriteUint32(binary.BigEndian.Uint32([]byte(table.tag)))
		w.WriteUint32(table.origChecksum)
		w.WriteUint32(sfntOffset)
		w.WriteUint32(table.origLength)
		sfntOffset += table.origLength
		sfntOffset = (sfntOffset + 3) & 0xFFFFFFFC // add padding
	}

	var iCheckSumAdjustment uint32
	var checkSumAdjustment uint32
	for _, table := range tables {
		data := b[table.offset : table.offset+table.length : table.offset+table.length]
		if table.length != table.origLength {
			var buf bytes.Buffer
			r, err := zlib.NewReader(bytes.NewReader(data))
			if err != nil {
				return nil, 0, fmt.Errorf("%s: %w", table.tag, err)
			}
			if _, err = io.Copy(&buf, r); err != nil {
				return nil, 0, fmt.Errorf("%s: %w", table.tag, err)
			}
			if err = r.Close(); err != nil {
				return nil, 0, fmt.Errorf("%s: %w", table.tag, err)
			}
			data = buf.Bytes()
		}

		if len(data) != int(table.origLength) {
			return nil, 0, ErrInvalidFontData
		}

		// add padding
		nPadding := (4 - len(data)&3) & 3
		for i := 0; i < nPadding; i++ {
			data = append(data, 0x00)
		}

		if table.tag == "head" {
			if len(data) < 12 {
				return nil, 0, ErrInvalidFontData
			}
			checkSumAdjustment = binary.BigEndian.Uint32(data[8:])
			iCheckSumAdjustment = w.Len() + 8

			// to check checksum for head table, replace the overal checksum with zero and reset it at the end
			binary.BigEndian.PutUint32(data[8:], 0x00000000)
			if calcChecksum(data) != table.origChecksum {
				return nil, 0, fmt.Errorf("%s: bad checksum", table.tag)
			}
		} else if calcChecksum(data) != table.origChecksum {
			return nil, 0, fmt.Errorf("%s: bad checksum", table.tag)
		}

		w.WriteBytes(data)
	}
	if w.Len() != totalSfntSize {
		return nil, 0, ErrInvalidFontData
	}

	if iCheckSumAdjustment == 0 {
		return nil, 0, ErrInvalidFontData
	} else {
		// TODO: (WOFF) overal checksum is off...
		//checksum := 0xB1B0AFBA - calcChecksum(w.Bytes())
		//fmt.Println(checkSumAdjustment, checksum)
		//return nil, 0, fmt.Errorf("bad checksum")
	}

	// replace overal checksum in head table
	buf := w.Bytes()
	binary.BigEndian.PutUint32(buf[iCheckSumAdjustment:], checkSumAdjustment) // TODO: (WOFF) check overal checksum
	return buf, flavor, nil
}

func calcChecksum(b []byte) uint32 {
	if len(b)%4 != 0 {
		panic("data not multiple of four bytes")
	}
	var sum uint32
	r := newBinaryReader(b)
	for !r.EOF() {
		sum += r.ReadUint32()
	}
	return sum
}
