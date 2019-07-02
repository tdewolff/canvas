package canvas

import (
	"encoding/binary"
	"unicode"
)

func uint32ToString(v uint32) string {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return string(b)
}

type popper struct {
	b []byte
	i int
}

func newPopper(b []byte) *popper {
	return &popper{b, 0}
}

func (p *popper) pop(n int) []byte {
	b := p.b[p.i : p.i+n]
	p.i += n
	return b
}

func (p *popper) pops(n int) string {
	return string(p.pop(n))
}

func (p *popper) pop8() byte {
	return p.pop(1)[0]
}

func (p *popper) pop16() uint16 {
	return binary.BigEndian.Uint16(p.pop(2))
}

func (p *popper) pop32() uint32 {
	return binary.BigEndian.Uint32(p.pop(4))
}

func (p *popper) popBase128() uint32 {
	var accum uint32
	for i := 0; i < 5; i++ {
		dataByte := p.b[p.i+i]
		if i == 0 && dataByte == 0x80 {
			return 0
		}
		if (accum & 0xFE000000) != 0 {
			return 0
		}
		accum = (accum << 7) | uint32(dataByte&0x7F)
		if (dataByte & 0x80) == 0 {
			p.i += i + 1
			return accum
		}
	}
	return 0
}

type pusher struct {
	b []byte
	i int
}

func newPusher(b []byte) *pusher {
	return &pusher{b, 0}
}

func (p *pusher) push(v []byte) {
	p.i += copy(p.b[p.i:], v)
}

func (p *pusher) pushs(v string) {
	p.push([]byte(v))
}

func (p *pusher) push16(v uint16) {
	binary.BigEndian.PutUint16(p.b[p.i:], v)
	p.i += 2
}

func (p *pusher) push32(v uint32) {
	binary.BigEndian.PutUint32(p.b[p.i:], v)
	p.i += 4
}

// from https://github.com/russross/blackfriday/blob/11635eb403ff09dbc3a6b5a007ab5ab09151c229/smartypants.go#L42
func quoteReplace(s string, i int, prev, quote, next rune, isOpen *bool) (string, int) {
	switch {
	case prev == 0 && next == 0:
		// context is not any help here, so toggle
		*isOpen = !*isOpen
	case isspace(prev) && next == 0:
		// [ "] might be [ "<code>foo...]
		*isOpen = true
	case ispunct(prev) && next == 0:
		// [!"] hmm... could be [Run!"] or [("<code>...]
		*isOpen = false
	case /* isnormal(prev) && */ next == 0:
		// [a"] is probably a close
		*isOpen = false
	case prev == 0 && isspace(next):
		// [" ] might be [...foo</code>" ]
		*isOpen = false
	case isspace(prev) && isspace(next):
		// [ " ] context is not any help here, so toggle
		*isOpen = !*isOpen
	case ispunct(prev) && isspace(next):
		// [!" ] is probably a close
		*isOpen = false
	case /* isnormal(prev) && */ isspace(next):
		// [a" ] this is one of the easy cases
		*isOpen = false
	case prev == 0 && ispunct(next):
		// ["!] hmm... could be ["$1.95] or [</code>"!...]
		*isOpen = false
	case isspace(prev) && ispunct(next):
		// [ "!] looks more like [ "$1.95]
		*isOpen = true
	case ispunct(prev) && ispunct(next):
		// [!"!] context is not any help here, so toggle
		*isOpen = !*isOpen
	case /* isnormal(prev) && */ ispunct(next):
		// [a"!] is probably a close
		*isOpen = false
	case prev == 0 /* && isnormal(next) */ :
		// ["a] is probably an open
		*isOpen = true
	case isspace(prev) /* && isnormal(next) */ :
		// [ "a] this is one of the easy cases
		*isOpen = true
	case ispunct(prev) /* && isnormal(next) */ :
		// [!"a] is probably an open
		*isOpen = true
	default:
		// [a'b] maybe a contraction?
		*isOpen = false
	}

	if quote == '"' {
		if *isOpen {
			return stringReplace(s, i, 1, "\u201C")
		} else {
			return stringReplace(s, i, 1, "\u201D")
		}
	} else if quote == '\'' {
		if *isOpen {
			return stringReplace(s, i, 1, "\u2018")
		} else {
			return stringReplace(s, i, 1, "\u2019")
		}
	}
	return s, 1
}

func stringReplace(s string, i, n int, target string) (string, int) {
	s = s[:i] + target + s[i+n:]
	return s, len(target)
}

func isWordBoundary(r rune) bool {
	return r == 0 || isspace(r) || ispunct(r)
}

func isspace(r rune) bool {
	return unicode.IsSpace(r)
}

func ispunct(r rune) bool {
	for _, punct := range "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~" {
		if r == punct {
			return true
		}
	}
	return false
}
