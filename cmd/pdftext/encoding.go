package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/tdewolff/font"
	"golang.org/x/text/encoding/charmap"
)

type encoding interface {
	Has(rune) bool
	Encode([]byte) ([]byte, error)
	Decode([]byte) ([]byte, error)
}

type standardEncoding struct{}

func (_ standardEncoding) Has(r rune) bool {
	entry, ok := charset[r]
	return ok && entry.std != 0
}

func (_ standardEncoding) Encode(b []byte) ([]byte, error) {
	return b, nil
}

func (_ standardEncoding) Decode(b []byte) ([]byte, error) {
	return b, nil
}

type winAnsiEncoding struct{}

func (_ winAnsiEncoding) Has(r rune) bool {
	entry, ok := charset[r]
	return ok && entry.win != 0
}

func (_ winAnsiEncoding) Encode(b []byte) ([]byte, error) {
	return charmap.Windows1252.NewEncoder().Bytes(b)
}

func (_ winAnsiEncoding) Decode(b []byte) ([]byte, error) {
	return charmap.Windows1252.NewDecoder().Bytes(b)
}

type macRomanEncoding struct{}

func (_ macRomanEncoding) Has(r rune) bool {
	entry, ok := charset[r]
	return ok && entry.mac != 0
}

func (_ macRomanEncoding) Encode(b []byte) ([]byte, error) {
	return charmap.Macintosh.NewEncoder().Bytes(b)
}

func (_ macRomanEncoding) Decode(b []byte) ([]byte, error) {
	return charmap.Macintosh.NewDecoder().Bytes(b)
}

type builtinEncoding struct {
	bytes int
	sfnt  *font.SFNT
}

func (e builtinEncoding) Has(r rune) bool {
	return true
}

func (e builtinEncoding) Encode(b []byte) ([]byte, error) {
	s := []byte{}
	for _, r := range string(b) {
		gid := e.sfnt.Cmap.Get(r)
		if e.bytes == 1 {
			if 256 <= gid {
				return nil, fmt.Errorf("invalid GID %v for character '%v' (0x%X)\n", gid, string(r), r)
			}
			s = append(s, byte(gid))
		} else {
			s = binary.BigEndian.AppendUint16(s, gid)
		}
	}
	return s, nil
}

func (e builtinEncoding) Decode(b []byte) ([]byte, error) {
	runes := []rune{}
	if e.bytes == 1 {
		for _, c := range b {
			runes = append(runes, e.sfnt.Cmap.ToUnicode(uint16(c)))
		}
	} else {
		for i := 0; i+2 <= len(b); i += 2 {
			c := binary.BigEndian.Uint16(b[i : i+2])
			runes = append(runes, e.sfnt.Cmap.ToUnicode(c))
		}
	}
	return []byte(string(runes)), nil
}

type pdfFont interface {
	Bytes() int
	ToUnicode([]byte) string
	FromUnicode(string) []byte
}

type pdfFontUnicode struct {
	bytes      int
	maxRunes   int               // maximum rune count in string
	mapping    map[uint16]string // character code to UTF8 strings (e.g. to "ffl")
	mappingRev map[string]uint16
}

func (f *pdfFontUnicode) Bytes() int {
	return f.bytes
}

func (f *pdfFontUnicode) ToUnicode(b []byte) string {
	s := &strings.Builder{}
	if f.bytes == 1 {
		for _, c := range b {
			if dst, ok := f.mapping[uint16(c)]; ok {
				s.WriteString(dst)
			} else {
				s.WriteByte(c)
			}
		}
	} else {
		for i := 0; i+2 <= len(b); i += 2 {
			c := binary.BigEndian.Uint16(b[i : i+2])
			if dst, ok := f.mapping[c]; ok {
				s.WriteString(dst)
			} else {
				s.WriteRune(utf16.Decode([]uint16{c})[0])
			}
		}
	}
	return s.String()
}

func (f *pdfFontUnicode) FromUnicode(s string) []byte {
	b := []byte{}
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		n := f.maxRunes
		if len(runes) < i+f.maxRunes {
			n = len(runes) - i
		}
		for {
			if c, ok := f.mappingRev[string(runes[i:i+n])]; ok {
				if f.bytes == 1 {
					b = append(b, byte(c))
				} else {
					b = binary.BigEndian.AppendUint16(b, c)
				}
				break
			} else if n == 1 {
				// not in explicit tmapping
				r := runes[i]
				if f.bytes == 1 {
					if 256 <= runes[i] {
						fmt.Printf("WARNING: encoding doesn't allow character '%v' (0x%X)\n", string(r), r)
						b = append(b, 0)
					} else {
						b = append(b, byte(r))
					}
				} else {
					if utf16.IsSurrogate(r) {
						fmt.Printf("WARNING: encoding doesn't allow character '%v' (0x%X)\n", string(r), r)
						b = append(b, 0, 0)
					} else {
						c := utf16.Encode([]rune{r})[0]
						b = binary.BigEndian.AppendUint16(b, c)
					}
				}
				break
			}
			n--
		}
	}
	return b
}

type pdfFontType1 struct {
	encoding encoding
	diffs    map[byte]rune
	diffsRev map[rune]byte
}

func (f *pdfFontType1) Bytes() int {
	return 1
}

func (f *pdfFontType1) ToUnicode(b []byte) string {
	s, _ := f.encoding.Decode(b)
	if f.diffs != nil {
		runes := []rune(string(s))
		for i, c := range b {
			if replacement, ok := f.diffs[c]; ok {
				runes[i] = replacement
			}
		}
		return string(runes)
	}
	return string(s)
}

func (f *pdfFontType1) FromUnicode(s string) []byte {
	b, _ := f.encoding.Encode([]byte(s))
	if f.diffsRev != nil {
		i := 0
		for _, r := range s {
			n := utf8.RuneLen(r)
			if replacement, ok := f.diffsRev[r]; ok {
				b[i] = replacement
				if 1 < n {
					b = append(b[:i+1], b[i+n:]...)
					n = 1
				}
			} else if !f.encoding.Has(r) {
				fmt.Printf("WARNING: encoding doesn't allow character '%v' (0x%X)\n", string(r), r)
				b[i] = 0
			}
			i += n
		}
	}
	return b
}

func (r *pdfReader) GetFont(dict pdfDict, name pdfName) (pdfFont, error) {
	resources, err := r.GetDict(dict["Resources"])
	if err != nil {
		return nil, err
	}
	fonts, err := r.GetDict(resources["Font"])
	if err != nil {
		return nil, err
	}
	ifont, ok := fonts[string(name)]
	if !ok {
		return nil, fmt.Errorf("unknown font %v", name)
	}
	font, err := r.GetDict(ifont)
	if err != nil {
		return nil, err
	}
	subtype, err := r.GetName(font["Subtype"])
	if err != nil {
		return nil, fmt.Errorf("bad font subtype: %w", err)
	}

	if _, ok := font["ToUnicode"]; ok {
		toUnicode, err := r.GetStream(font["ToUnicode"])
		if err != nil {
			return nil, err
		}

		f := &pdfFontUnicode{
			mapping:    map[uint16]string{},
			mappingRev: map[string]uint16{},
		}
		if subtype == "Type0" {
			f.bytes = 2
		} else {
			f.bytes = 1
		}
		stream := newPDFStreamReader(toUnicode.data)
		for {
			op, vals, err := stream.Next()
			if err == io.EOF {
				break
			} else if err != nil {
				return nil, err
			}

			//if op == "endcodespacerange" {
			//	if len(vals) != 2 || f.bytes != 0 {
			//		return nil, fmt.Errorf("bad CMap")
			//	}
			//	begin, ok := vals[0].([]byte)
			//	if !ok {
			//		return nil, fmt.Errorf("bad CMap")
			//	}
			//	end, ok := vals[1].([]byte)
			//	if !ok {
			//		return nil, fmt.Errorf("bad CMap")
			//	}
			//	if len(begin) != len(end) || len(begin) != 1 && len(begin) != 2 {
			//		return nil, fmt.Errorf("bad CMap")
			//	}
			//	f.bytes = len(begin)
			//	if f.bytes == 1 {
			//		f.begin = uint16(begin[0])
			//		f.end = uint16(end[0])
			//	} else {
			//		f.begin = binary.BigEndian.Uint16(begin)
			//		f.end = binary.BigEndian.Uint16(end)
			//	}
			//} else
			if op == "endbfchar" {
				if len(vals)%2 != 0 || len(vals) == 0 {
					return nil, fmt.Errorf("bad CMap")
				}
				for i := 0; i < len(vals); i += 2 {
					src, ok := vals[i].([]byte)
					if !ok || len(src) != 1 && len(src) != 2 {
						return nil, fmt.Errorf("bad CMap")
					}
					dst8, ok := vals[i+1].([]byte)
					if !ok || len(dst8)%2 != 0 {
						return nil, fmt.Errorf("bad CMap")
					}
					dst16 := make([]uint16, len(dst8)/2)
					for i := 0; i < len(dst8); i += 2 {
						dst16[i/2] = binary.BigEndian.Uint16(dst8[i : i+2])
					}
					if f.maxRunes < len(dst16) {
						f.maxRunes = len(dst16)
					}
					dst := string(utf16.Decode(dst16))
					if len(src) == 2 {
						f.mapping[binary.BigEndian.Uint16(src)] = dst
						f.mappingRev[dst] = binary.BigEndian.Uint16(src)
					} else {
						f.mapping[uint16(src[0])] = dst
						f.mappingRev[dst] = uint16(src[0])
					}
				}
			} else if op == "endbfrange" {
				if len(vals)%3 != 0 || len(vals) == 0 {
					return nil, fmt.Errorf("bad CMap")
				}
				for i := 0; i < len(vals); i += 3 {
					src0, ok := vals[i].([]byte)
					if !ok || len(src0) != 1 && len(src0) != 2 {
						return nil, fmt.Errorf("bad CMap")
					}
					src1, ok := vals[i+1].([]byte)
					if !ok || len(src1) != 1 && len(src1) != 2 {
						return nil, fmt.Errorf("bad CMap")
					}
					var begin, end uint16
					if len(src0) == 2 {
						begin = binary.BigEndian.Uint16(src0)
					} else {
						begin = uint16(src0[0])
					}
					if len(src1) == 2 {
						end = binary.BigEndian.Uint16(src1) + 1
					} else {
						end = uint16(src1[0]) + 1
					}
					if array, ok := vals[i+2].(pdfArray); ok && len(array) == int(end-begin) {
						for i := begin; i < end; i++ {
							dst8, ok := array[i-begin].([]byte)
							if !ok || len(dst8)%2 != 0 {
								return nil, fmt.Errorf("bad CMap")
							}
							dst16 := make([]uint16, len(dst8)/2)
							for i := 0; i < len(dst8); i += 2 {
								dst16[i/2] = binary.BigEndian.Uint16(dst8[i : i+2])
							}
							if f.maxRunes < len(dst16) {
								f.maxRunes = len(dst16)
							}
							dst := string(utf16.Decode(dst16))
							f.mapping[i] = dst
							f.mappingRev[dst] = i
						}
					} else if dst8, ok := vals[i+2].([]byte); ok && len(dst8)%2 == 0 {
						dst16 := make([]uint16, len(dst8)/2)
						for i := 0; i < len(dst8); i += 2 {
							dst16[i/2] = binary.BigEndian.Uint16(dst8[i : i+2])
						}
						for i := begin; i < end; i++ {
							if f.maxRunes < len(dst16) {
								f.maxRunes = len(dst16)
							}
							dst := string(utf16.Decode(dst16))
							f.mapping[i] = dst
							f.mappingRev[dst] = i

							i := len(dst16) - 1
							dst16 = append([]uint16{}, dst16...) // copy
							for ; 0 <= i; i-- {
								if dst16[i] < 65535 {
									dst16[i]++
									break
								}
								dst16[i] = 0
							}
							if i == -1 {
								dst16 = append([]uint16{1}, dst16...)
							}
						}
					} else {
						return nil, fmt.Errorf("bad CMap")
					}
				}
			}
		}
		return f, nil
	}

	if subtype == "Type1" || subtype == "TrueType" {
		nonSymbolic := false
		fontDescriptor, err := r.GetDict(font["FontDescriptor"])
		if err == nil {
			if flags, err := r.GetInt(fontDescriptor["Flags"]); err == nil {
				nonSymbolic = (flags & 0x20) != 0
			}
		}

		f := &pdfFontType1{
			encoding: standardEncoding{},
		}
		if _, ok := font["Encoding"]; ok {
			if encoding, err := r.GetName(font["Encoding"]); err == nil {
				if encoding == "WinAnsiEncoding" {
					f.encoding = winAnsiEncoding{}
				} else if encoding == "MacRomanEncoding" {
					f.encoding = macRomanEncoding{}
				} else if encoding != "StandardEncoding" {
					fmt.Println("WARNING: unsupported encoding", encoding)
				}
			} else if encoding, err := r.GetDict(font["Encoding"]); err == nil {
				if _, ok := encoding["BaseEncoding"]; ok {
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
					f.encoding, err = r.getBuiltinEncoding(subtype, fontDescriptor)
					if err != nil {
						return nil, err
					}
				}

				if _, ok := encoding["Differences"]; ok && subtype == "Type1" {
					differences, err := r.GetArray(encoding["Differences"])
					if err != nil {
						return nil, err
					}
					f.diffs = map[byte]rune{}
					f.diffsRev = map[rune]byte{}
					for i := 0; i < len(differences); {
						code, ok := differences[i].(int)
						if !ok || code < 0 || 256 <= code {
							return nil, fmt.Errorf("bad font encoding differences")
						}
						i++
						for i < len(differences) {
							name, ok := differences[i].(pdfName)
							if !ok {
								break
							}
							r, ok := charsetName[string(name)]
							if !ok {
								return nil, fmt.Errorf("character name doesn't exist: %v", name)
							}
							f.diffs[byte(code)] = r
							if _, ok := f.diffsRev[r]; !ok {
								f.diffsRev[r] = byte(code)
							}
							code++
							i++
						}
					}
				}
			} else {
				fmt.Println("WARNING: unsupported encoding")
			}
		} else {
			f.encoding, err = r.getBuiltinEncoding(subtype, fontDescriptor)
			if err != nil {
				return nil, err
			}
		}
		return f, nil
	} else if subtype == "Type0" {
		// TODO: Type0
	} else if subtype == "CIDFontType0" || subtype == "CIDFontType2" {
	}
	return nil, fmt.Errorf("unsupported font subtype: %v", subtype)
}

func (r *pdfReader) getFontProgram(fontDescriptor pdfDict) (*font.SFNT, error) {
	if ifontFile2, ok := fontDescriptor["FontFile2"]; ok {
		fontFile2, err := r.GetStream(ifontFile2)
		if err != nil {
			return nil, err
		}
		return font.ParseEmbeddedSFNT(fontFile2.data, 0)
	} else if ifontFile3, ok := fontDescriptor["FontFile3"]; ok {
		if fontFile3, err := r.GetStream(ifontFile3); err != nil {
			return nil, err
		} else if subtype, err := r.GetName(fontFile3.dict["Subtype"]); err != nil {
			return nil, err
		} else if subtype == pdfName("OpenType") || subtype == pdfName("Type1C") || subtype == pdfName("CIDFOntType0C") {
			fontFile, err := r.GetStream(fontFile3)
			if err != nil {
				return nil, err
			}
			if subtype == pdfName("OpenType") {
				return font.ParseEmbeddedSFNT(fontFile.data, 0)
			}
			return font.ParseCFF(fontFile.data)
		} else {
			return nil, fmt.Errorf("invalid subtype for FontFile3: %v", subtype)
		}
	} else if _, ok := fontDescriptor["FontFile"]; ok {
		return nil, fmt.Errorf("unsupported FontFile type")
	}
	return nil, nil // standard font
}

func (r *pdfReader) getBuiltinEncoding(fontType pdfName, fontDescriptor pdfDict) (encoding, error) {
	sfnt, err := r.getFontProgram(fontDescriptor)
	if err != nil {
		return builtinEncoding{}, err
	} else if sfnt == nil {
		return standardEncoding{}, nil // standard font
	}
	bytes := 1
	if fontType == pdfName("CIDFontType0") || fontType == pdfName("CIDFontType2") {
		bytes = 2
	}
	return builtinEncoding{bytes, sfnt}, nil
}

type charsetEntry struct {
	name               string
	std, mac, win, pdf int
}

var charset = map[rune]charsetEntry{
	'A':  {"A", 0101, 0101, 0101, 0101},
	'Æ':  {"AE", 0341, 0256, 0306, 0306},
	'Á':  {"Aacute", 0, 0347, 0301, 0301},
	'Â':  {"Acircumflex", 0, 0345, 0302, 0302},
	'Ä':  {"Adieresis", 0, 0200, 0304, 0304},
	'À':  {"Agrave", 0, 0313, 0300, 0300},
	'Å':  {"Aring", 0, 0201, 0305, 0305},
	'Ã':  {"Atilde", 0, 0314, 0303, 0303},
	'B':  {"B", 0102, 0102, 0102, 0102},
	'C':  {"C", 0103, 0103, 0103, 0103},
	'Ç':  {"Ccedilla", 0, 0202, 0307, 0307},
	'D':  {"D", 0104, 0104, 0104, 0104},
	'E':  {"E", 0105, 0105, 0105, 0105},
	'É':  {"Eacute", 0, 0203, 0311, 0311},
	'Ê':  {"Ecircumflex", 0, 0346, 0312, 0312},
	'Ë':  {"Edieresis", 0, 0350, 0313, 0313},
	'È':  {"Egrave", 0, 0351, 0310, 0310},
	'Ð':  {"Eth", 0, 0, 0320, 0320},
	'€':  {"Euro", 0, 0, 0200, 0240},
	'F':  {"F", 0106, 0106, 0106, 0106},
	'G':  {"G", 0107, 0107, 0107, 0107},
	'H':  {"H", 0110, 0110, 0110, 0110},
	'I':  {"I", 0111, 0111, 0111, 0111},
	'Í':  {"Iacute", 0, 0352, 0315, 0315},
	'Î':  {"Icircumflex", 0, 0353, 0316, 0316},
	'Ï':  {"Idieresis", 0, 0354, 0317, 0317},
	'Ì':  {"Igrave", 0, 0355, 0314, 0314},
	'J':  {"J", 0112, 0112, 0112, 0112},
	'K':  {"K", 0113, 0113, 0113, 0113},
	'L':  {"L", 0114, 0114, 0114, 0114},
	'Ł':  {"Lslash", 0350, 0, 0, 0225},
	'M':  {"M", 0115, 0115, 0115, 0115},
	'N':  {"N", 0116, 0116, 0116, 0116},
	'Ñ':  {"Ntilde", 0, 0204, 0321, 0321},
	'O':  {"O", 0117, 0117, 0117, 0117},
	'Œ':  {"OE", 0352, 0316, 0214, 0226},
	'Ó':  {"Oacute", 0, 0356, 0323, 0323},
	'Ô':  {"Ocircumflex", 0, 0357, 0324, 0324},
	'Ö':  {"Odieresis", 0, 0205, 0326, 0326},
	'Ò':  {"Ograve", 0, 0361, 0322, 0322},
	'Ø':  {"Oslash", 0351, 0257, 0330, 0330},
	'Õ':  {"Otilde", 0, 0315, 0325, 0325},
	'P':  {"P", 0120, 0120, 0120, 0120},
	'Q':  {"Q", 0121, 0121, 0121, 0121},
	'R':  {"R", 0122, 0122, 0122, 0122},
	'S':  {"S", 0123, 0123, 0123, 0123},
	'Š':  {"Scaron", 0, 0, 0212, 0227},
	'T':  {"T", 0124, 0124, 0124, 0124},
	'Þ':  {"Thorn", 0, 0, 0336, 0336},
	'U':  {"U", 0125, 0125, 0125, 0125},
	'Ú':  {"Uacute", 0, 0362, 0332, 0332},
	'Û':  {"Ucircumflex", 0, 0363, 0333, 0333},
	'Ü':  {"Udieresis", 0, 0206, 0334, 0334},
	'Ù':  {"Ugrave", 0, 0364, 0331, 0331},
	'V':  {"V", 0126, 0126, 0126, 0126},
	'W':  {"W", 0127, 0127, 0127, 0127},
	'X':  {"X", 0130, 0130, 0130, 0130},
	'Y':  {"Y", 0131, 0131, 0131, 0131},
	'Ý':  {"Yacute", 0, 0, 0335, 0335},
	'Ÿ':  {"Ydieresis", 0, 0331, 0237, 0230},
	'Z':  {"Z", 0132, 0132, 0132, 0132},
	'Ž':  {"Zcaron", 0, 0, 0216, 0231},
	'a':  {"a", 0141, 0141, 0141, 0141},
	'á':  {"aacute", 0, 0207, 0341, 0341},
	'â':  {"acircumflex", 0, 0211, 0342, 0342},
	'´':  {"acute", 0302, 0253, 0264, 0264},
	'ä':  {"adieresis", 0, 0212, 0344, 0344},
	'æ':  {"ae", 0361, 0276, 0346, 0346},
	'à':  {"agrave", 0, 0210, 0340, 0340},
	'&':  {"ampersand", 0046, 0046, 0046, 0046},
	'å':  {"aring", 0, 0214, 0345, 0345},
	'^':  {"asciicircum", 0136, 0136, 0136, 0136},
	'~':  {"asciitilde", 0176, 0176, 0176, 0176},
	'*':  {"asterisk", 0052, 0052, 0052, 0052},
	'@':  {"at", 0100, 0100, 0100, 0100},
	'ã':  {"atilde", 0, 0213, 0343, 0343},
	'b':  {"b", 0142, 0142, 0142, 0142},
	'\\': {"backslash", 0134, 0134, 0134, 0134},
	'|':  {"bar", 0174, 0174, 0174, 0174},
	'{':  {"braceleft", 0173, 0173, 0173, 0173},
	'}':  {"braceright", 0175, 0175, 0175, 0175},
	'[':  {"bracketleft", 0133, 0133, 0133, 0133},
	']':  {"bracketright", 0135, 0135, 0135, 0135},
	'˘':  {"breve", 0306, 0371, 0, 0030},
	'¦':  {"brokenbar", 0, 0, 0246, 0246},
	'•':  {"bullet", 0267, 0245, 0225, 0200},
	'c':  {"c", 0143, 0143, 0143, 0143},
	'ˇ':  {"caron", 0317, 0377, 0, 0031},
	'ç':  {"ccedilla", 0, 0215, 0347, 0347},
	'¸':  {"cedilla", 0313, 0374, 0270, 0270},
	'¢':  {"cent", 0242, 0242, 0242, 0242},
	'ˆ':  {"circumflex", 0303, 0366, 0210, 0032},
	':':  {"colon", 0072, 0072, 0072, 0072},
	',':  {"comma", 0054, 0054, 0054, 0054},
	'©':  {"copyright", 0, 0251, 0251, 0251},
	'¤':  {"currency", 0250, 0333, 0244, 0244},
	'd':  {"d", 0144, 0144, 0144, 0144},
	'†':  {"dagger", 0262, 0240, 0206, 0201},
	'‡':  {"daggerdbl", 0263, 0340, 0207, 0202},
	'°':  {"degree", 0, 0241, 0260, 0260},
	'¨':  {"dieresis", 0310, 0254, 0250, 0250},
	'÷':  {"divide", 0, 0326, 0367, 0367},
	'$':  {"dollar", 0044, 0044, 0044, 0044},
	'˙':  {"dotaccent", 0307, 0372, 0, 0033},
	'ı':  {"dotlessi", 0365, 0365, 0, 0232},
	'e':  {"e", 0145, 0145, 0145, 0145},
	'é':  {"eacute", 0, 0216, 0351, 0351},
	'ê':  {"ecircumflex", 0, 0220, 0352, 0352},
	'ë':  {"edieresis", 0, 0221, 0353, 0353},
	'è':  {"egrave", 0, 0217, 0350, 0350},
	'8':  {"eight", 0070, 0070, 0070, 0070},
	'…':  {"ellipsis", 0274, 0311, 0205, 0203},
	'—':  {"emdash", 0320, 0321, 0227, 0204},
	'–':  {"endash", 0261, 0320, 0226, 0205},
	'=':  {"equal", 0075, 0075, 0075, 0075},
	'ð':  {"eth", 0, 0, 0360, 0360},
	'!':  {"exclam", 0041, 0041, 0041, 0041},
	'¡':  {"exclamdown", 0241, 0301, 0241, 0241},
	'f':  {"f", 0146, 0146, 0146, 0146},
	'ﬁ':  {"fi", 0256, 0336, 0, 0223},
	'5':  {"five", 0065, 0065, 0065, 0065},
	'ﬂ':  {"fl", 0257, 0337, 0, 0224},
	'ƒ':  {"florin", 0246, 0304, 0203, 0206},
	'4':  {"four", 0064, 0064, 0064, 0064},
	'⁄':  {"fraction", 0244, 0332, 0, 0207},
	'g':  {"g", 0147, 0147, 0147, 0147},
	'ß':  {"germandbls", 0373, 0247, 0337, 0337},
	'`':  {"grave", 0301, 0140, 0140, 0140},
	'>':  {"greater", 0076, 0076, 0076, 0076},
	'«':  {"guillemotleft", 0253, 0307, 0253, 0253},
	'»':  {"guillemotright", 0273, 0310, 0273, 0273},
	'‹':  {"guilsinglleft", 0254, 0334, 0213, 0210},
	'›':  {"guilsinglright", 0255, 0335, 0233, 0211},
	'h':  {"h", 0150, 0150, 0150, 0150},
	'˝':  {"hungarumlaut", 0315, 0375, 0, 0034},
	'-':  {"hyphen", 0055, 0055, 0055, 0055},
	'i':  {"i", 0151, 0151, 0151, 0151},
	'í':  {"iacute", 0, 0222, 0355, 0355},
	'î':  {"icircumflex", 0, 0224, 0356, 0356},
	'ï':  {"idieresis", 0, 0225, 0357, 0357},
	'ì':  {"igrave", 0, 0223, 0354, 0354},
	'j':  {"j", 0152, 0152, 0152, 0152},
	'k':  {"k", 0153, 0153, 0153, 0153},
	'l':  {"l", 0154, 0154, 0154, 0154},
	'<':  {"less", 0074, 0074, 0074, 0074},
	'¬':  {"logicalnot", 0, 0302, 0254, 0254},
	'ł':  {"lslash", 0370, 0, 0, 0233},
	'm':  {"m", 0155, 0155, 0155, 0155},
	'¯':  {"macron", 0305, 0370, 0257, 0257},
	'−':  {"minus", 0, 0, 0, 0212},
	'μ':  {"mu", 0, 0265, 0265, 0265},
	'×':  {"multiply", 0, 0, 0327, 0327},
	'n':  {"n", 0156, 0156, 0156, 0156},
	'9':  {"nine", 0071, 0071, 0071, 0071},
	'ñ':  {"ntilde", 0, 0226, 0361, 0361},
	'#':  {"numbersign", 0043, 0043, 0043, 0043},
	'o':  {"o", 0157, 0157, 0157, 0157},
	'ó':  {"oacute", 0, 0227, 0363, 0363},
	'ô':  {"ocircumflex", 0, 0231, 0364, 0364},
	'ö':  {"odieresis", 0, 0232, 0366, 0366},
	'œ':  {"oe", 0372, 0317, 0234, 0234},
	'˛':  {"ogonek", 0316, 0376, 0, 0035},
	'ò':  {"ograve", 0, 0230, 0362, 0362},
	'1':  {"one", 0061, 0061, 0061, 0061},
	'½':  {"onehalf", 0, 0, 0275, 0275},
	'¼':  {"onequarter", 0, 0, 0274, 0274},
	'¹':  {"onesuperior", 0, 0, 0271, 0271},
	'ª':  {"ordfeminine", 0343, 0273, 0252, 0252},
	'º':  {"ordmasculine", 0353, 0274, 0272, 0272},
	'ø':  {"oslash", 0371, 0277, 0370, 0370},
	'õ':  {"otilde", 0, 0233, 0365, 0365},
	'p':  {"p", 0160, 0160, 0160, 0160},
	'¶':  {"paragraph", 0266, 0246, 0266, 0266},
	'(':  {"parenleft", 0050, 0050, 0050, 0050},
	')':  {"parenright", 0051, 0051, 0051, 0051},
	'%':  {"percent", 0045, 0045, 0045, 0045},
	'.':  {"period", 0056, 0056, 0056, 0056},
	'·':  {"periodcentered", 0264, 0341, 0267, 0267},
	'‰':  {"perthousand", 0275, 0344, 0211, 0213},
	'+':  {"plus", 0053, 0053, 0053, 0053},
	'±':  {"plusminus", 0, 0261, 0261, 0261},
	'q':  {"q", 0161, 0161, 0161, 0161},
	'?':  {"question", 0077, 0077, 0077, 0077},
	'¿':  {"questiondown", 0277, 0300, 0277, 0277},
	'"':  {"quotedbl", 0042, 0042, 0042, 0042},
	'„':  {"quotedblbase", 0271, 0343, 0204, 0214},
	'“':  {"quotedblleft", 0252, 0322, 0223, 0215},
	'”':  {"quotedblright", 0272, 0323, 0224, 0216},
	'‘':  {"quoteleft", 0140, 0324, 0221, 0217},
	'’':  {"quoteright", 0047, 0325, 0222, 0220},
	'‚':  {"quotesinglbase", 0270, 0342, 0202, 0221},
	'\'': {"quotesingle", 0251, 0047, 0047, 0047},
	'r':  {"r", 0162, 0162, 0162, 0162},
	'®':  {"registered", 0, 0250, 0256, 0256},
	'˚':  {"ring", 0312, 0373, 0, 0036},
	's':  {"s", 0163, 0163, 0163, 0163},
	'š':  {"scaron", 0, 0, 0232, 0235},
	'§':  {"section", 0247, 0244, 0247, 0247},
	';':  {"semicolon", 0073, 0073, 0073, 0073},
	'7':  {"seven", 0067, 0067, 0067, 0067},
	'6':  {"six", 0066, 0066, 0066, 0066},
	'/':  {"slash", 0057, 0057, 0057, 0057},
	' ':  {"space", 0040, 0040, 0040, 0040},
	'£':  {"sterling", 0243, 0243, 0243, 0243},
	't':  {"t", 0164, 0164, 0164, 0164},
	'þ':  {"thorn", 0, 0, 0376, 0376},
	'3':  {"three", 0063, 0063, 0063, 0063},
	'¾':  {"threequarters", 0, 0, 0276, 0276},
	'³':  {"threesuperior", 0, 0, 0263, 0263},
	'˜':  {"tilde", 0304, 0367, 0230, 0037},
	'™':  {"trademark", 0, 0252, 0231, 0222},
	'2':  {"two", 0062, 0062, 0062, 0062},
	'²':  {"twosuperior", 0, 0, 0262, 0262},
	'u':  {"u", 0165, 0165, 0165, 0165},
	'ú':  {"uacute", 0, 0234, 0372, 0372},
	'û':  {"ucircumflex", 0, 0236, 0373, 0373},
	'ü':  {"udieresis", 0, 0237, 0374, 0374},
	'ù':  {"ugrave", 0, 0235, 0371, 0371},
	'_':  {"underscore", 0137, 0137, 0137, 0137},
	'v':  {"v", 0166, 0166, 0166, 0166},
	'w':  {"w", 0167, 0167, 0167, 0167},
	'x':  {"x", 0170, 0170, 0170, 0170},
	'y':  {"y", 0171, 0171, 0171, 0171},
	'ý':  {"yacute", 0, 0, 0375, 0375},
	'ÿ':  {"ydieresis", 0, 0330, 0377, 0377},
	'¥':  {"yen", 0245, 0264, 0245, 0245},
	'z':  {"z", 0172, 0172, 0172, 0172},
	'ž':  {"zcaron", 0, 0, 0236, 0236},
	'0':  {"zero", 0060, 0060, 0060, 0060},
}
