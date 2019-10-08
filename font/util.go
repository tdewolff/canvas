package font

import "encoding/binary"

func uint32ToString(v uint32) string {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return string(b)
}

type binaryReader struct {
	b []byte
	i int
}

func newBinaryReader(b []byte) *binaryReader {
	return &binaryReader{b, 0}
}

func (r *binaryReader) ReadBytes(n int) []byte {
	b := r.b[r.i : r.i+n]
	r.i += n
	return b
}

func (r *binaryReader) ReadByte() byte {
	return r.ReadBytes(1)[0]
}

func (r *binaryReader) ReadString(n int) string {
	return string(r.ReadBytes(n))
}

func (r *binaryReader) ReadUint16() uint16 {
	return binary.BigEndian.Uint16(r.ReadBytes(2))
}

func (r *binaryReader) ReadUint32() uint32 {
	return binary.BigEndian.Uint32(r.ReadBytes(4))
}

func (r *binaryReader) ReadBase128() uint32 {
	var accum uint32
	for i := 0; i < 5; i++ {
		dataByte := r.b[r.i+i]
		if i == 0 && dataByte == 0x80 {
			return 0
		}
		if (accum & 0xFE000000) != 0 {
			return 0
		}
		accum = (accum << 7) | uint32(dataByte&0x7F)
		if (dataByte & 0x80) == 0 {
			r.i += i + 1
			return accum
		}
	}
	return 0
}

type binaryWriter struct {
	b []byte
	i int
}

func newBinaryWriter(b []byte) *binaryWriter {
	return &binaryWriter{b, 0}
}

func (w *binaryWriter) Bytes() []byte {
	return w.b
}

func (w *binaryWriter) WriteBytes(v []byte) {
	w.i += copy(w.b[w.i:], v)
}

func (w *binaryWriter) WriteByte(v byte) {
	w.WriteBytes([]byte{v})
}

func (w *binaryWriter) WriteString(v string) {
	w.WriteBytes([]byte(v))
}

func (w *binaryWriter) WriteUint16(v uint16) {
	binary.BigEndian.PutUint16(w.b[w.i:], v)
	w.i += 2
}

func (w *binaryWriter) WriteUint32(v uint32) {
	binary.BigEndian.PutUint32(w.b[w.i:], v)
	w.i += 4
}
