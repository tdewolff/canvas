package font

import (
	"encoding/binary"
	"fmt"
	"math"
)

// MaxMemory is the maximum memory that can be allocated by a font.
var MaxMemory uint32 = 30 * 1024 * 1024

// ErrExceedsMemory is returned if the font is malformed.
var ErrExceedsMemory = fmt.Errorf("memory limit exceded")

// ErrInvalidFontData is returned if the font is malformed.
var ErrInvalidFontData = fmt.Errorf("invalid font data")

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

func uint16ToFlags(v uint16) (flags [16]bool) {
	for i := 0; i < 16; i++ {
		flags[i] = v&(1<<i) != 0
	}
	return
}

func uint8ToFlags(v uint8) (flags [8]bool) {
	for i := 0; i < 8; i++ {
		flags[i] = v&(1<<i) != 0
	}
	return
}

func flagsToUint8(flags [8]bool) (v uint8) {
	for i := 0; i < 8; i++ {
		if flags[i] {
			v |= 1 << i
		}
	}
	return
}

func uint32ToString(v uint32) string {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return string(b)
}

type binaryReader struct {
	buf []byte
	pos uint32
	eof bool
}

func newBinaryReader(buf []byte) *binaryReader {
	if math.MaxUint32 < uint64(len(buf)) {
		return &binaryReader{nil, 0, true}
	}
	return &binaryReader{buf, 0, false}
}

func (r *binaryReader) ReadBytes(n uint32) []byte {
	if r.eof || uint32(len(r.buf))-r.pos < n {
		r.eof = true
		return nil
	}
	buf := r.buf[r.pos : r.pos+n]
	r.pos += n
	return buf
}

func (r *binaryReader) ReadString(n uint32) string {
	return string(r.ReadBytes(n))
}

func (r *binaryReader) ReadByte() byte {
	b := r.ReadBytes(1)
	if b == nil {
		return 0
	}
	return b[0]
}

func (r *binaryReader) ReadUint8() uint8 {
	return r.ReadByte()
}

func (r *binaryReader) ReadUint16() uint16 {
	b := r.ReadBytes(2)
	if b == nil {
		return 0
	}
	return binary.BigEndian.Uint16(b)
}

func (r *binaryReader) ReadUint32() uint32 {
	b := r.ReadBytes(4)
	if b == nil {
		return 0
	}
	return binary.BigEndian.Uint32(b)
}

func (r *binaryReader) ReadUint64() uint64 {
	b := r.ReadBytes(8)
	if b == nil {
		return 0
	}
	return binary.BigEndian.Uint64(b)
}

func (r *binaryReader) ReadInt8() int8 {
	return int8(r.ReadByte())
}

func (r *binaryReader) ReadInt16() int16 {
	return int16(r.ReadUint16())
}

func (r *binaryReader) ReadUint16LE() uint16 {
	b := r.ReadBytes(2)
	if b == nil {
		return 0
	}
	return binary.LittleEndian.Uint16(b)
}

func (r *binaryReader) ReadUint32LE() uint32 {
	b := r.ReadBytes(4)
	if b == nil {
		return 0
	}
	return binary.LittleEndian.Uint32(b)
}

func (r *binaryReader) ReadInt16LE() int16 {
	return int16(r.ReadUint16LE())
}

func (r *binaryReader) Seek(pos uint32) {
	if uint32(len(r.buf)) < pos {
		r.eof = true
		return
	}
	r.pos = pos
	r.eof = false
}

func (r *binaryReader) Pos() uint32 {
	return r.pos
}

func (r *binaryReader) Len() uint32 {
	return uint32(len(r.buf)) - r.pos
}

func (r *binaryReader) EOF() bool {
	return r.eof
}

type bitmapReader struct {
	buf []byte
	pos uint32
	eof bool
}

func newBitmapReader(buf []byte) *bitmapReader {
	if math.MaxUint32 < uint64(len(buf)) {
		return &bitmapReader{nil, 0, true}
	}
	return &bitmapReader{buf, 0, false}
}

func (r *bitmapReader) Read() bool {
	if r.eof || uint32(len(r.buf)) <= (r.pos+1)/8 {
		r.eof = true
		return false
	}
	bit := r.buf[r.pos>>3]&(0x80>>(r.pos&7)) != 0
	r.pos += 1
	return bit
}

func (r *bitmapReader) Pos() uint32 {
	return r.pos
}

func (r *bitmapReader) EOF() bool {
	return r.eof
}

type binaryWriter struct {
	buf []byte
}

func newBinaryWriter(buf []byte) *binaryWriter {
	return &binaryWriter{buf[:0]}
}

func (w *binaryWriter) Bytes() []byte {
	return w.buf
}

func (w *binaryWriter) WriteBytes(v []byte) {
	pos := len(w.buf)
	w.buf = append(w.buf, make([]byte, len(v))...)
	copy(w.buf[pos:], v)
}

func (w *binaryWriter) WriteString(v string) {
	w.WriteBytes([]byte(v))
}

func (w *binaryWriter) WriteByte(v byte) {
	w.WriteBytes([]byte{v})
}

func (w *binaryWriter) WriteUint8(v uint8) {
	w.WriteByte(v)
}

func (w *binaryWriter) WriteUint16(v uint16) {
	pos := len(w.buf)
	w.buf = append(w.buf, make([]byte, 2)...)
	binary.BigEndian.PutUint16(w.buf[pos:], v)
}

func (w *binaryWriter) WriteUint32(v uint32) {
	pos := len(w.buf)
	w.buf = append(w.buf, make([]byte, 4)...)
	binary.BigEndian.PutUint32(w.buf[pos:], v)
}

func (w *binaryWriter) WriteUint64(v uint64) {
	pos := len(w.buf)
	w.buf = append(w.buf, make([]byte, 8)...)
	binary.BigEndian.PutUint64(w.buf[pos:], v)
}

func (w *binaryWriter) WriteInt16(v int16) {
	w.WriteUint16(uint16(v))
}

func (w *binaryWriter) WriteInt64(v int64) {
	w.WriteUint64(uint64(v))
}

func (w *binaryWriter) Len() uint32 {
	return uint32(len(w.buf))
}
