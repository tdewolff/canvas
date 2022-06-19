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
	r := NewBinaryReader(b)
	for !r.EOF() {
		sum += r.ReadUint32()
	}
	return sum
}

// Uint8ToFlags converts a uint8 in 8 booleans from least to most significant.
func Uint8ToFlags(v uint8) (flags [8]bool) {
	for i := 0; i < 8; i++ {
		flags[i] = v&(1<<i) != 0
	}
	return
}

// Uint16ToFlags converts a uint16 in 16 booleans from least to most significant.
func Uint16ToFlags(v uint16) (flags [16]bool) {
	for i := 0; i < 16; i++ {
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

// BinaryReader is a binary big endian file format reader.
type BinaryReader struct {
	buf []byte
	pos uint32
	eof bool
}

// NewBinaryReader returns a big endian binary file format reader.
func NewBinaryReader(buf []byte) *BinaryReader {
	if math.MaxUint32 < uint(len(buf)) {
		return &BinaryReader{nil, 0, true}
	}
	return &BinaryReader{buf, 0, false}
}

// Seek set the reader position in the buffer.
func (r *BinaryReader) Seek(pos uint32) {
	if uint32(len(r.buf)) < pos {
		r.eof = true
		return
	}
	r.pos = pos
	r.eof = false
}

// Pos returns the reader's position.
func (r *BinaryReader) Pos() uint32 {
	return r.pos
}

// Len returns the remaining length of the buffer.
func (r *BinaryReader) Len() uint32 {
	return uint32(len(r.buf)) - r.pos
}

// EOF returns true if we reached the end-of-file.
func (r *BinaryReader) EOF() bool {
	return r.eof
}

// ReadBytes reads n bytes.
func (r *BinaryReader) ReadBytes(n uint32) []byte {
	if r.eof || uint32(len(r.buf))-r.pos < n {
		r.eof = true
		return nil
	}
	buf := r.buf[r.pos : r.pos+n]
	r.pos += n
	return buf
}

// ReadString reads a string of length n.
func (r *BinaryReader) ReadString(n uint32) string {
	return string(r.ReadBytes(n))
}

// ReadByte reads a single byte.
func (r *BinaryReader) ReadByte() byte {
	b := r.ReadBytes(1)
	if b == nil {
		return 0
	}
	return b[0]
}

// ReadUint8 reads a uint8.
func (r *BinaryReader) ReadUint8() uint8 {
	return r.ReadByte()
}

// ReadUint16 reads a uint16.
func (r *BinaryReader) ReadUint16() uint16 {
	b := r.ReadBytes(2)
	if b == nil {
		return 0
	}
	return binary.BigEndian.Uint16(b)
}

// ReadUint32 reads a uint32.
func (r *BinaryReader) ReadUint32() uint32 {
	b := r.ReadBytes(4)
	if b == nil {
		return 0
	}
	return binary.BigEndian.Uint32(b)
}

// ReadUint64 reads a uint64.
func (r *BinaryReader) ReadUint64() uint64 {
	b := r.ReadBytes(8)
	if b == nil {
		return 0
	}
	return binary.BigEndian.Uint64(b)
}

// ReadInt8 reads a int8.
func (r *BinaryReader) ReadInt8() int8 {
	return int8(r.ReadByte())
}

// ReadInt16 reads a int16.
func (r *BinaryReader) ReadInt16() int16 {
	return int16(r.ReadUint16())
}

// ReadInt32 reads a int32.
func (r *BinaryReader) ReadInt32() int32 {
	return int32(r.ReadUint32())
}

// ReadInt64 reads a int64.
func (r *BinaryReader) ReadInt64() int64 {
	return int64(r.ReadUint64())
}

// ReadUint16LE reads a uint16 little endian.
func (r *BinaryReader) ReadUint16LE() uint16 {
	b := r.ReadBytes(2)
	if b == nil {
		return 0
	}
	return binary.LittleEndian.Uint16(b)
}

// ReadUint32LE reads a uint32 little endian.
func (r *BinaryReader) ReadUint32LE() uint32 {
	b := r.ReadBytes(4)
	if b == nil {
		return 0
	}
	return binary.LittleEndian.Uint32(b)
}

// ReadInt16LE reads a int16 little endian.
func (r *BinaryReader) ReadInt16LE() int16 {
	return int16(r.ReadUint16LE())
}

// BitmapReader is a binary bitmap reader.
type BitmapReader struct {
	buf []byte
	pos uint32
	eof bool
}

// NewBitmapReader returns a binary bitmap reader.
func NewBitmapReader(buf []byte) *BitmapReader {
	if math.MaxUint32 < uint(len(buf)) {
		return &BitmapReader{nil, 0, true}
	}
	return &BitmapReader{buf, 0, false}
}

// Pos returns the current bit position.
func (r *BitmapReader) Pos() uint32 {
	return r.pos
}

// EOF returns if we reached the buffer's end-of-file.
func (r *BitmapReader) EOF() bool {
	return r.eof
}

// Read reads the next bit.
func (r *BitmapReader) Read() bool {
	if r.eof || uint32(len(r.buf)) <= (r.pos+1)/8 {
		r.eof = true
		return false
	}
	bit := r.buf[r.pos>>3]&(0x80>>(r.pos&7)) != 0
	r.pos += 1
	return bit
}

// BinaryWriter is a big endian binary file format writer.
type BinaryWriter struct {
	buf []byte
}

// NewBinaryWriter returns a big endian binary file format writer.
func NewBinaryWriter(buf []byte) *BinaryWriter {
	return &BinaryWriter{buf[:0]}
}

// Len returns the buffer's length in bytes.
func (w *BinaryWriter) Len() uint32 {
	return uint32(len(w.buf))
}

// Bytes returns the buffer's bytes.
func (w *BinaryWriter) Bytes() []byte {
	return w.buf
}

// WriteBytes writes the given bytes to the buffer.
func (w *BinaryWriter) WriteBytes(v []byte) {
	pos := len(w.buf)
	w.buf = append(w.buf, make([]byte, len(v))...)
	copy(w.buf[pos:], v)
}

// WriteString writes the given string to the buffer.
func (w *BinaryWriter) WriteString(v string) {
	w.WriteBytes([]byte(v))
}

// WriteByte writes the given byte to the buffer.
func (w *BinaryWriter) WriteByte(v byte) {
	w.WriteBytes([]byte{v})
}

// WriteUint8 writes the given uint8 to the buffer.
func (w *BinaryWriter) WriteUint8(v uint8) {
	w.WriteByte(v)
}

// WriteUint16 writes the given uint16 to the buffer.
func (w *BinaryWriter) WriteUint16(v uint16) {
	pos := len(w.buf)
	w.buf = append(w.buf, make([]byte, 2)...)
	binary.BigEndian.PutUint16(w.buf[pos:], v)
}

// WriteUint32 writes the given uint32 to the buffer.
func (w *BinaryWriter) WriteUint32(v uint32) {
	pos := len(w.buf)
	w.buf = append(w.buf, make([]byte, 4)...)
	binary.BigEndian.PutUint32(w.buf[pos:], v)
}

// WriteUint64 writes the given uint64 to the buffer.
func (w *BinaryWriter) WriteUint64(v uint64) {
	pos := len(w.buf)
	w.buf = append(w.buf, make([]byte, 8)...)
	binary.BigEndian.PutUint64(w.buf[pos:], v)
}

// WriteInt8 writes the given int8 to the buffer.
func (w *BinaryWriter) WriteInt8(v int8) {
	w.WriteUint8(uint8(v))
}

// WriteInt16 writes the given int16 to the buffer.
func (w *BinaryWriter) WriteInt16(v int16) {
	w.WriteUint16(uint16(v))
}

// WriteInt32 writes the given int32 to the buffer.
func (w *BinaryWriter) WriteInt32(v int32) {
	w.WriteUint32(uint32(v))
}

// WriteInt64 writes the given int64 to the buffer.
func (w *BinaryWriter) WriteInt64(v int64) {
	w.WriteUint64(uint64(v))
}
