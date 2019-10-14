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

func ParseWOFF(b []byte) ([]byte, uint32, error) {
	if len(b) < 44 {
		return nil, 0, fmt.Errorf("invalid WOFF data")
	}

	r := newBinaryReader(b)
	signature := r.ReadString(4)
	if signature != "wOFF" {
		return nil, 0, fmt.Errorf("invalid WOFF data")
	}
	flavor := r.ReadUint32()
	_ = r.ReadUint32() // length
	numTables := r.ReadUint16()
	_ = r.ReadUint16() // reserved
	_ = r.ReadUint32() // totalSfntSize
	_ = r.ReadUint16() // majorVersion
	_ = r.ReadUint16() // minorVersion
	_ = r.ReadUint32() // metaOffset
	_ = r.ReadUint32() // metaLength
	_ = r.ReadUint32() // metaOrigLength
	_ = r.ReadUint32() // privOffset
	_ = r.ReadUint32() // privLength

	tables := []woffTable{}
	sfntLength := uint32(12 + 16*int(numTables))
	for i := 0; i < int(numTables); i++ {
		tag := uint32ToString(r.ReadUint32())
		offset := r.ReadUint32()
		compLength := r.ReadUint32()
		origLength := r.ReadUint32()
		origChecksum := r.ReadUint32()
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

	w := newBinaryWriter(make([]byte, sfntLength))
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

	for _, table := range tables {
		data := b[table.offset : table.offset+table.length]
		if table.length != table.origLength {
			var buf bytes.Buffer
			r, _ := zlib.NewReader(bytes.NewReader(data))
			io.Copy(&buf, r)
			r.Close()
			data = buf.Bytes()
		}

		if len(data) != int(table.origLength) {
			return nil, 0, fmt.Errorf("font data size mismatch")
		}

		// TODO: (WOFF) check checksum

		w.WriteBytes(data)
		nPadding := (4 - len(data)&3) & 3
		for i := 0; i < nPadding; i++ {
			w.WriteByte(0x00)
		}
	}
	return w.Bytes(), flavor, nil
}
