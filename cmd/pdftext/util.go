package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/Seanld/canvas"
	"github.com/tdewolff/minify/v2"
)

func isWhiteSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f' || b == 0
}

func isDelimiter(b byte) bool {
	return b == '(' || b == ')' || b == '<' || b == '>' || b == '[' || b == ']' || b == '{' || b == '}' || b == '/' || b == '%'
}

func isRegular(b byte) bool {
	return !isWhiteSpace(b) && !isDelimiter(b)
}

func moveWhiteSpace(b []byte, i int) int {
Again:
	for i < len(b) && isWhiteSpace(b[i]) {
		i++
	}
	if i < len(b) && b[i] == '%' {
		for i < len(b) {
			if b[i] == '\r' || b[i] == '\n' {
				break
			}
			i++
		}
		goto Again
	}
	return i
}

func parseName(b []byte) ([]byte, int, error) {
	var s []byte
	j := 0 // start in b
	i := 0 // position
	for i < len(b) && '!' <= b[i] && b[i] <= '~' {
		if isDelimiter(b[i]) {
			break
		} else if b[i] == '#' {
			s = append(s, b[j:i]...)
			if i+2 < len(b) {
				s = append(s, 0)
				_, err := hex.Decode(s[len(s)-1:], b[i+1:i+3])
				if err != nil {
					return nil, 0, fmt.Errorf("bad name")
				}
				i += 2
			} else {
				return nil, 0, fmt.Errorf("bad name")
			}
		}
		i++
	}
	return append(s, b[j:i]...), i, nil
}

func parseTextString(b []byte) string {
	if len(b)%2 == 0 && len(b) != 0 && b[0] == 254 && b[1] == 255 {
		s := []rune{}
		for i := 2; i+2 <= len(b); i += 2 {
			s = append(s, rune(binary.BigEndian.Uint16(b[i:i+2])))
		}
		return string(s)
	}
	return string(b)
}

func parseDate(b []byte) time.Time {
	t, _ := time.Parse("D:20060102150405Z07'00'"[:len(b)], string(b))
	return t
}

type lineReader struct {
	b      []byte
	offset int
}

func newLineReader(b []byte, offset int) *lineReader {
	return &lineReader{b, offset}
}

func (r *lineReader) Pos() int {
	return r.offset
}

func (r *lineReader) Next() []byte {
	for i := r.offset; i < len(r.b); i++ {
		if r.b[i] == '\r' || r.b[i] == '\n' {
			line := r.b[r.offset:i]
			if i+1 < len(r.b) && r.b[i] == '\r' && r.b[i+1] == '\n' {
				i++
			}
			i++
			r.offset = i
			return line
		}
	}
	if r.offset < len(r.b) {
		line := r.b[r.offset:]
		r.offset = len(r.b)
		return line
	}
	return nil
}

type lineReaderReverse struct {
	b      []byte
	offset int
}

func newLineReaderReverse(b []byte, offset int) *lineReaderReverse {
	// skip final empty line
	if len(b) < offset {
		offset = len(b)
	}
	if 0 <= offset-1 && b[offset-1] == 0 {
		offset-- // some PDFs end in 0x00
	}
	if 0 <= offset-1 && (b[offset-1] == '\r' || b[offset-1] == '\n') {
		if 0 <= offset-2 && b[offset-1] == '\n' && b[offset-2] == '\r' {
			offset--
		}
		offset--
	}
	return &lineReaderReverse{b, offset}
}

func (r *lineReaderReverse) Pos() int {
	return r.offset
}

func (r *lineReaderReverse) Next() []byte {
	for i := r.offset - 1; 0 <= i; i-- {
		if r.b[i] == '\r' || r.b[i] == '\n' {
			line := r.b[i+1 : r.offset]
			if 0 < i && r.b[i] == '\n' && r.b[i-1] == '\r' {
				i--
			}
			r.offset = i
			return line
		}
	}
	if 0 < r.offset {
		line := r.b[:r.offset]
		r.offset = 0
		return line
	}
	return nil
}

type dec float64

func (f dec) String() string {
	s := fmt.Sprintf("%.*f", canvas.Precision, f)
	s = string(minify.Decimal([]byte(s), canvas.Precision))
	if dec(math.MaxInt32) < f || f < dec(math.MinInt32) {
		if i := strings.IndexByte(s, '.'); i == -1 {
			s += ".0"
		}
	}
	return s
}

func readNumberLE(b []byte, n int) uint32 {
	num := uint32(0)
	for i := 0; i < n; i++ {
		num <<= 8
		num += uint32(b[i])
	}
	return num
}
