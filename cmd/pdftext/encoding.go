package main

import (
	"encoding/binary"
	"fmt"
	"io"

	"golang.org/x/text/encoding/charmap"
)

type encoding interface {
	Encode([]byte) ([]byte, error)
	Decode([]byte) ([]byte, error)
}

type winAnsiEncoding struct{}

func (_ winAnsiEncoding) Encode(b []byte) ([]byte, error) {
	return charmap.Windows1252.NewEncoder().Bytes(b)
}

func (_ winAnsiEncoding) Decode(b []byte) ([]byte, error) {
	return charmap.Windows1252.NewDecoder().Bytes(b)
}

type macRomanEncoding struct{}

func (_ macRomanEncoding) Encode(b []byte) ([]byte, error) {
	return charmap.Macintosh.NewEncoder().Bytes(b)
}

func (_ macRomanEncoding) Decode(b []byte) ([]byte, error) {
	return charmap.Macintosh.NewDecoder().Bytes(b)
}

type buildinEncoding struct{}

func (_ buildinEncoding) Encode(b []byte) ([]byte, error) {
	return b, nil
}

func (_ buildinEncoding) Decode(b []byte) ([]byte, error) {
	return b, nil
}

type pdfFont struct {
	diffs    map[byte]rune
	revDiffs map[rune]byte

	encoding encoding

	bytes      int
	begin, end uint16
	mapping    map[uint16][]byte
	//revMapping map[[]byte]uint16
}

func (f pdfFont) ToUnicode(b []byte) []byte {
	s := make([]byte, 0, len(b))
	if f.bytes == 1 {
		for _, c := range b {
			if dst, ok := f.mapping[uint16(c)]; ok {
				s = append(s, dst...)
			} else {
				s = append(s, c)
			}
		}
	} else if f.bytes == 2 {
		for i := 0; i+2 <= len(b); i += 2 {
			if dst, ok := f.mapping[binary.BigEndian.Uint16(b[i:i+2])]; ok {
				s = append(s, dst...)
			} else {
				if b[i] != 0 {
					s = append(s, b[i])
				}
				s = append(s, b[i+1])
			}
		}
	} else {
		s = b
	}
	if f.encoding != nil {
		s, _ = f.encoding.Decode(s)
	}
	if f.diffs != nil {
		runes := make([]rune, len(s))
		for i, c := range s {
			if replacement, ok := f.diffs[c]; ok {
				runes[i] = replacement
			}
		}
		s = []byte(string(runes))
	}
	return s
}

func (f pdfFont) FromUnicode(b []byte) []byte {
	return b
	//if f.revDiffs != nil {
	//	s := make([]byte, utf8.RuneCount(b))
	//	for i, r := range string(b) {
	//		if replacement, ok := f.revDiffs[r]; ok {
	//			s[i] = replacement
	//		}
	//		b = s
	//	}
	//}
	//if f.encoding != nil {
	//	b, _ = f.encoding.Encode(b)
	//}
	//return b

	//s := make([]byte, 0, len(b))
	//if f.bytes == 1 {
	//	for _, c := range b {
	//		if dst, ok := f.revMapping[uint16(c)]; ok {
	//			s = append(s, dst...)
	//		} else {
	//			s = append(s, c)
	//		}
	//	}
	//} else if f.bytes == 2 {
	//	for i := 0; i+2 <= len(b); i += 2 {
	//		if dst, ok := f.mapping[binary.BigEndian.Uint16(b[i:i+2])]; ok {
	//			s = append(s, dst...)
	//		} else {
	//			if b[i] != 0 {
	//				s = append(s, b[i])
	//			}
	//			s = append(s, b[i+1])
	//		}
	//	}
	//} else {
	//	s = b
	//}
	//return s
}

func (r *pdfReader) GetFont(index int, name pdfName) (pdfFont, error) {
	page, _, err := r.GetPage(index)
	if err != nil {
		return pdfFont{}, err
	}
	resources, err := r.GetDict(page["Resources"])
	if err != nil {
		return pdfFont{}, err
	}
	fonts, err := r.GetDict(resources["Font"])
	if err != nil {
		return pdfFont{}, err
	}
	ifont, ok := fonts[string(name)]
	if !ok {
		return pdfFont{}, fmt.Errorf("unknown font %v", name)
	}
	font, err := r.GetDict(ifont)
	if err != nil {
		return pdfFont{}, err
	}

	fmt.Println(font)

	nonSymbolic := false
	if fontDescriptor, err := r.GetDict(font["FontDescriptor"]); err == nil {
		if flags, err := r.GetInt(fontDescriptor["Flags"]); err == nil {
			nonSymbolic = (flags & 0x20) != 0
		}
	}

	f := pdfFont{
		mapping: map[uint16][]byte{},
		//revMapping: map[[]byte]uint16{},
	}
	if _, ok := font["Encoding"]; ok {
		if encoding, err := r.GetName(font["Encoding"]); err == nil {
			fmt.Println("has encoding")
			if encoding == "WinAnsiEncoding" {
				f.encoding = winAnsiEncoding{}
			} else if encoding == "MacRomanEncoding" {
				f.encoding = macRomanEncoding{}
			} else if encoding != "StandardEncoding" {
				fmt.Println("WARNING: unsupported encoding", encoding)
			}
		} else if encoding, err := r.GetDict(font["Encoding"]); err == nil {
			if _, ok := encoding["BaseEncoding"]; ok {
				fmt.Println("has encoding")
				baseEncoding, _ := r.GetName(encoding["BaseEncoding"])
				if baseEncoding == "WinAnsiEncoding" {
					f.encoding = winAnsiEncoding{}
				} else if baseEncoding == "MacRomanEncoding" {
					f.encoding = macRomanEncoding{}
				} else if baseEncoding != "StandardEncoding" {
					fmt.Println("WARNING: unsupported encoding", encoding)
				}
			} else if nonSymbolic {
				// standard encoding
			} else {
				// TODO: buildin
				fmt.Println("WARNING: unsupported buildin encoding")
			}

			if _, ok := encoding["Differences"]; ok {
				fmt.Println("has differences")
				differences, err := r.GetArray(encoding["Differences"])
				if err != nil {
					return pdfFont{}, err
				}
				f.diffs = map[byte]rune{}
				f.revDiffs = map[rune]byte{}
				for i := 0; i < len(differences); {
					code, ok := differences[i].(int)
					if !ok || code < 0 || 256 <= code {
						return pdfFont{}, fmt.Errorf("bad font encoding differences")
					}
					i++
					for i < len(differences) {
						name, ok := differences[i].(pdfName)
						if !ok {
							break
						}
						r, ok := charset[string(name)]
						if !ok {
							return pdfFont{}, fmt.Errorf("character name doesn't exist: %v", name)
						}
						f.diffs[byte(code)] = r
						f.revDiffs[r] = byte(code)
						code++
						i++
					}
				}
			}
		}
	} else {
		// TODO: buildin
		fmt.Println("WARNING: unsupported buildin encoding")
	}

	if _, ok := font["ToUnicode"]; ok {
		fmt.Println("has unicode")
		toUnicode, err := r.GetStream(font["ToUnicode"])
		if err != nil {
			return pdfFont{}, err
		}
		toUnicode, err = toUnicode.Decompress()
		if err != nil {
			return pdfFont{}, err
		}

		stream := newPDFStreamReader(toUnicode.data)
		for {
			op, vals, err := stream.Next()
			if err == io.EOF {
				break
			} else if err != nil {
				return pdfFont{}, err
			}

			if op == "endcodespacerange" {
				if len(vals) != 2 || f.bytes != 0 {
					return pdfFont{}, fmt.Errorf("bad CMap")
				}
				begin, ok := vals[0].([]byte)
				if !ok {
					return pdfFont{}, fmt.Errorf("bad CMap")
				}
				end, ok := vals[1].([]byte)
				if !ok {
					return pdfFont{}, fmt.Errorf("bad CMap")
				}
				if len(begin) != len(end) || len(begin) != 1 && len(begin) != 2 {
					return pdfFont{}, fmt.Errorf("bad CMap")
				}
				f.bytes = len(begin)
				if f.bytes == 1 {
					f.begin = uint16(begin[0])
					f.end = uint16(end[0])
				} else {
					f.begin = binary.BigEndian.Uint16(begin)
					f.end = binary.BigEndian.Uint16(end)
				}
			} else if op == "endbfchar" {
				if len(vals)%2 != 0 || len(vals) == 0 {
					return pdfFont{}, fmt.Errorf("bad CMap")
				}
				for i := 0; i < len(vals); i += 2 {
					src, ok := vals[i].([]byte)
					if !ok || len(src) != f.bytes {
						return pdfFont{}, fmt.Errorf("bad CMap")
					}
					dst, ok := vals[i+1].([]byte)
					if !ok || len(dst) != 1 && len(dst) != 2 {
						return pdfFont{}, fmt.Errorf("bad CMap")
					}
					if f.bytes == 1 {
						f.mapping[uint16(src[0])] = dst
						//f.revMapping[dst] = uint16(src[0])
					} else {
						f.mapping[binary.BigEndian.Uint16(src)] = dst
						//f.revMapping[dst] = binary.BigEndian.Uint16(src)
					}
				}
			} else if op == "endbfrange" {
				if len(vals)%3 != 0 || len(vals) == 0 {
					return pdfFont{}, fmt.Errorf("bad CMap")
				}
				for i := 0; i < len(vals); i += 3 {
					src0, ok := vals[i].([]byte)
					if !ok || len(src0) != f.bytes {
						return pdfFont{}, fmt.Errorf("bad CMap")
					}
					src1, ok := vals[i+1].([]byte)
					if !ok || len(src1) != f.bytes {
						return pdfFont{}, fmt.Errorf("bad CMap")
					}
					var begin, end uint16
					if f.bytes == 1 {
						begin = uint16(src0[0])
						end = uint16(src1[0]) + 1
					} else {
						begin = binary.BigEndian.Uint16(src0)
						end = binary.BigEndian.Uint16(src1) + 1
					}
					if array, ok := vals[i+2].(pdfArray); ok && len(array) == int(end-begin) {
						for i := begin; i < end; i++ {
							dst, ok := array[i-begin].([]byte)
							if !ok || len(dst) != 1 && len(dst) != 2 {
								return pdfFont{}, fmt.Errorf("bad CMap")
							}
							f.mapping[i] = dst
							//f.revMapping[dst] = i
						}
					} else if dst, ok := vals[i+2].([]byte); ok && (len(dst) == 1 || len(dst) == 2) {
						for i := begin; i < end; i++ {
							f.mapping[i] = dst
							//f.revMapping[dst] = i

							i := len(dst) - 1
							dst = append([]byte{}, dst...)
							for 0 <= i {
								if dst[i] < 255 {
									dst[i]++
									break
								}
								dst[i] = 0
								i--
							}
							if i == -1 {
								dst = append([]byte{1}, dst...)
							}
						}
					} else {
						return pdfFont{}, fmt.Errorf("bad CMap")
					}
				}
			}
		}
	}
	// TODO: add other ways
	return f, nil // identity encoding
}

var charset = map[string]rune{
	"A":              'A',
	"AE":             'Æ',
	"Aacute":         'Á',
	"Acircumflex":    'Â',
	"Adieresis":      'Ä',
	"Agrave":         'À',
	"Aring":          'Å',
	"Atilde":         'Ã',
	"B":              'B',
	"C":              'C',
	"Ccedilla":       'Ç',
	"D":              'D',
	"E":              'E',
	"Eacute":         'É',
	"Ecircumflex":    'Ê',
	"Edieresis":      'Ë',
	"Egrave":         'È',
	"Eth":            'Ð',
	"Euro":           '€',
	"F":              'F',
	"G":              'G',
	"H":              'H',
	"I":              'I',
	"Iacute":         'Í',
	"Icircumflex":    'Î',
	"Idieresis":      'Ï',
	"Igrave":         'Ì',
	"J":              'J',
	"K":              'K',
	"L":              'L',
	"Lslash":         'Ł',
	"M":              'M',
	"N":              'N',
	"Ntilde":         'Ñ',
	"O":              'O',
	"OE":             'Œ',
	"Oacute":         'Ó',
	"Ocircumflex":    'Ô',
	"Odieresis":      'Ö',
	"Ograve":         'Ò',
	"Oslash":         'Ø',
	"Otilde":         'Õ',
	"P":              'P',
	"Q":              'Q',
	"R":              'R',
	"S":              'S',
	"Scaron":         'Š',
	"T":              'T',
	"Thorn":          'Þ',
	"U":              'U',
	"Uacute":         'Ú',
	"Ucircumflex":    'Û',
	"Udieresis":      'Ü',
	"Ugrave":         'Ù',
	"V":              'V',
	"W":              'W',
	"X":              'X',
	"Y":              'Y',
	"Yacute":         'Ý',
	"Ydieresis":      'Ÿ',
	"Z":              'Z',
	"Zcaron":         'Ž',
	"a":              'a',
	"aacute":         'á',
	"acircumflex":    'â',
	"acute":          '´',
	"adieresis":      'ä',
	"ae":             'æ',
	"agrave":         'à',
	"ampersand":      '&',
	"aring":          'å',
	"asciicircum":    '^',
	"asciitilde":     '~',
	"asterisk":       '*',
	"at":             '@',
	"atilde":         'ã',
	"b":              'b',
	"backslash":      '\\',
	"bar":            '|',
	"braceleft":      '{',
	"braceright":     '}',
	"bracketleft":    '[',
	"bracketright":   ']',
	"breve":          '˘',
	"brokenbar":      '¦',
	"bullet":         '•',
	"c":              'c',
	"caron":          'ˇ',
	"ccedilla":       'ç',
	"cedilla":        '¸',
	"cent":           '¢',
	"circumflex":     'ˆ',
	"colon":          ':',
	"comma":          ',',
	"copyright":      '©',
	"currency":       '¤',
	"d":              'd',
	"dagger":         '†',
	"daggerdbl":      '‡',
	"degree":         '°',
	"dieresis":       '¨',
	"divide":         '÷',
	"dollar":         '$',
	"dotaccent":      '˙',
	"dotlessi":       'ı',
	"e":              'e',
	"eacute":         'é',
	"ecircumflex":    'ê',
	"edieresis":      'ë',
	"egrave":         'è',
	"eight":          '8',
	"ellipsis":       '…',
	"emdash":         '—',
	"endash":         '–',
	"equal":          '=',
	"eth":            'ð',
	"exclam":         '!',
	"exclamdown":     '¡',
	"f":              'f',
	"fi":             'ﬁ',
	"five":           '5',
	"fl":             'ﬂ',
	"florin":         'ƒ',
	"four":           '4',
	"fraction":       '⁄',
	"g":              'g',
	"germandbls":     'ß',
	"grave":          '`',
	"greater":        '>',
	"guillemotleft":  '«',
	"guillemotright": '»',
	"guilsinglleft":  '‹',
	"guilsinglright": '›',
	"h":              'h',
	"hungarumlaut":   '˝',
	"hyphen":         '-',
	"i":              'i',
	"iacute":         'í',
	"icircumflex":    'î',
	"idieresis":      'ï',
	"igrave":         'ì',
	"j":              'j',
	"k":              'k',
	"l":              'l',
	"less":           '<',
	"logicalnot":     '¬',
	"lslash":         'ł',
	"m":              'm',
	"macron":         '¯',
	"minus":          '−',
	"mu":             'μ',
	"multiply":       '×',
	"n":              'n',
	"nine":           '9',
	"ntilde":         'ñ',
	"numbersign":     '#',
	"o":              'o',
	"oacute":         'ó',
	"ocircumflex":    'ô',
	"odieresis":      'ö',
	"oe":             'œ',
	"ogonek":         '˛',
	"ograve":         'ò',
	"one":            '1',
	"onehalf":        '½',
	"onequarter":     '¼',
	"onesuperior":    '¹',
	"ordfeminine":    'ª',
	"ordmasculine":   'º',
	"oslash":         'ø',
	"otilde":         'õ',
	"p":              'p',
	"paragraph":      '¶',
	"parenleft":      '(',
	"parenright":     ')',
	"percent":        '%',
	"period":         '.',
	"periodcentered": '·',
	"perthousand":    '‰',
	"plus":           '+',
	"plusminus":      '±',
	"q":              'q',
	"question":       '?',
	"questiondown":   '¿',
	"quotedbl":       '"',
	"quotedblbase":   '„',
	"quotedblleft":   '“',
	"quotedblright":  '”',
	"quoteleft":      '‘',
	"quoteright":     '’',
	"quotesinglbase": '‚',
	"quotesingle":    '\'',
	"r":              'r',
	"registered":     '®',
	"ring":           '˚',
	"s":              's',
	"scaron":         'š',
	"section":        '§',
	"semicolon":      ';',
	"seven":          '7',
	"six":            '6',
	"slash":          '/',
	"space":          ' ',
	"sterling":       '£',
	"t":              't',
	"thorn":          'þ',
	"three":          '3',
	"threequarters":  '¾',
	"threesuperior":  '³',
	"tilde":          '˜',
	"trademark":      '™',
	"two":            '2',
	"twosuperior":    '²',
	"u":              'u',
	"uacute":         'ú',
	"ucircumflex":    'û',
	"udieresis":      'ü',
	"ugrave":         'ù',
	"underscore":     '_',
	"v":              'v',
	"w":              'w',
	"x":              'x',
	"y":              'y',
	"yacute":         'ý',
	"ydieresis":      'ÿ',
	"yen":            '¥',
	"z":              'z',
	"zcaron":         'ž',
	"zero":           '0',
}
