package font

import (
	"encoding/binary"
	"fmt"
)

var ErrInvalidFontData = fmt.Errorf("invalid font data")

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
	return &binaryReader{buf, 0, false}
}

func (r *binaryReader) ReadBytes(n uint32) []byte {
	if r.eof || uint32(len(r.buf)) < r.pos+n {
		r.eof = true
		return nil
	}
	buf := r.buf[r.pos : r.pos+n]
	r.pos += n
	return buf
}

func (r *binaryReader) ReadByte() byte {
	b := r.ReadBytes(1)
	if b == nil {
		return 0
	}
	return b[0]
}

func (r *binaryReader) ReadString(n uint32) string {
	return string(r.ReadBytes(n))
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

func (r *binaryReader) ReadInt16() int16 {
	return int16(r.ReadUint16())
}

func (r *binaryReader) EOF() bool {
	return r.eof
}

func (r *binaryReader) Len() uint32 {
	return uint32(len(r.buf)) - r.pos
}

type bitmapReader struct {
	buf []byte
	pos uint32
	bit int
	eof bool
}

func newBitmapReader(buf []byte) *bitmapReader {
	return &bitmapReader{buf, 0, 0, false}
}

func (r *bitmapReader) Read() bool {
	if r.eof || uint32(len(r.buf)) <= r.pos {
		r.eof = true
		return false
	}
	bit := r.buf[r.pos]&(1<<r.bit) == 1
	r.bit++
	if r.bit == 8 {
		r.bit = 0
		r.pos++
	}
	return bit
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

func (w *binaryWriter) WriteByte(v byte) {
	w.WriteBytes([]byte{v})
}

func (w *binaryWriter) WriteString(v string) {
	w.WriteBytes([]byte(v))
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

func (w *binaryWriter) WriteInt16(v int16) {
	w.WriteUint16(uint16(v))
}

func (w *binaryWriter) Len() uint32 {
	return uint32(len(w.buf))
}
