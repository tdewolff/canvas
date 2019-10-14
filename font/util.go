package font

import "encoding/binary"

func uint32ToString(v uint32) string {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return string(b)
}

type binaryReader struct {
	buf []byte
	pos uint32
}

func newBinaryReader(buf []byte) *binaryReader {
	return &binaryReader{buf, 0}
}

func (r *binaryReader) ReadBytes(n uint32) []byte {
	buf := r.buf[r.pos : r.pos+n]
	r.pos += n
	return buf
}

func (r *binaryReader) ReadByte() byte {
	return r.ReadBytes(1)[0]
}

func (r *binaryReader) ReadString(n uint32) string {
	return string(r.ReadBytes(n))
}

func (r *binaryReader) ReadUint16() uint16 {
	return binary.BigEndian.Uint16(r.ReadBytes(2))
}

func (r *binaryReader) ReadUint32() uint32 {
	return binary.BigEndian.Uint32(r.ReadBytes(4))
}

func (r *binaryReader) ReadInt16() int16 {
	return int16(r.ReadUint16())
}

func (r *binaryReader) Len() uint32 {
	return uint32(len(r.buf)) - r.pos
}

type bitmapReader struct {
	buf []byte
	pos uint32
	bit int
}

func newBitmapReader(buf []byte) *bitmapReader {
	return &bitmapReader{buf, 0, 0}
}

func (r *bitmapReader) Read() bool {
	bit := r.buf[r.pos]&(1<<r.bit) == 1
	r.bit++
	if r.bit == 8 {
		r.bit = 0
		r.pos++
	}
	return bit
}

func (r *bitmapReader) Len() uint32 {
	return (uint32(len(r.buf))-r.pos)*8 - uint32(r.bit)
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
