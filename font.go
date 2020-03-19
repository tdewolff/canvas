package canvas

import (
	"strings"
	"unicode"
	"unicode/utf8"

	canvasFont "github.com/tdewolff/canvas/font"
	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
)

// TypographicOptions are the options that can be enabled to make typographic or ligature substitutions automatically.
type TypographicOptions int

// see TypographicOptions
const (
	NoTypography TypographicOptions = 2 << iota
	NoRequiredLigatures
	CommonLigatures
	DiscretionaryLigatures
	HistoricalLigatures
)

// Font defines a font of type TTF or OTF which which a FontFace can be generated for use in text drawing operations.
type Font struct {
	// TODO: extend to fully read in sfnt data and read liga tables, generate Raw font data (base on used glyphs), etc
	name     string
	mimetype string
	raw      []byte
	sfnt     *sfnt.Font

	// TODO: use sub/superscript Unicode transformations in ToPath etc. if they exist
	typography  bool
	ligatures   []textSubstitution
	superscript []textSubstitution
	subscript   []textSubstitution
}

func parseFont(name string, b []byte) (*Font, error) {
	mimetype, err := canvasFont.Mimetype(b)
	if err != nil {
		return nil, err
	}

	sfntFont, err := canvasFont.ParseFont(b)
	if err != nil {
		return nil, err
	}

	f := &Font{
		name:     name,
		mimetype: mimetype,
		raw:      b,
		sfnt:     (*sfnt.Font)(sfntFont),
	}
	f.superscript = f.supportedSubstitutions(superscriptSubstitutes)
	f.subscript = f.supportedSubstitutions(subscriptSubstitutes)
	f.Use(0)
	return f, nil
}

// Name returns the name of the font.
func (f *Font) Name() string {
	return f.name
}

// Raw returns the mimetype and raw binary data of the font.
func (f *Font) Raw() (string, []byte) {
	return f.mimetype, f.raw
}

func (f *Font) pdfInfo() (Rect, float64, float64, float64, float64, []int) {
	buffer := &sfnt.Buffer{}
	units := float64(f.sfnt.UnitsPerEm())

	bounds := Rect{}
	rect, err := f.sfnt.Bounds(buffer, toI26_6(units), font.HintingNone)
	if err == nil {
		x0, y0 := fromI26_6(rect.Min.X)*1000.0/units, fromI26_6(rect.Min.Y)*1000.0/units
		x1, y1 := fromI26_6(rect.Max.X)*1000.0/units, fromI26_6(rect.Max.Y)*1000.0/units
		bounds = Rect{x0, y0, x1 - x0, y1 - y0}
	}

	italicAngle := 0.0
	if f.sfnt.PostTable() != nil {
		italicAngle = f.sfnt.PostTable().ItalicAngle
	}

	ascent, descent, capHeight := 0.0, 0.0, 0.0
	metrics, err := f.sfnt.Metrics(buffer, toI26_6(units), font.HintingNone)
	if err == nil {
		ascent = fromI26_6(metrics.Ascent) * 1000.0 / units
		descent = fromI26_6(metrics.Descent) * 1000.0 / units
		capHeight = fromI26_6(metrics.CapHeight) * 1000.0 / units
	}

	widths := []int{}
	for i := 0; i < f.sfnt.NumGlyphs(); i++ {
		index := sfnt.GlyphIndex(i)
		advance, err := f.sfnt.GlyphAdvance(buffer, index, toI26_6(units), font.HintingNone)
		if err == nil {
			widths = append(widths, int(fromI26_6(advance)*1000.0/units+0.5))
		}
	}
	return bounds, italicAngle, ascent, descent, capHeight, widths
}

func (f *Font) toIndices(s string) []uint16 {
	buffer := &sfnt.Buffer{}
	runes := []rune(s)
	indices := make([]uint16, len(runes))
	for i, r := range runes {
		index, err := f.sfnt.GlyphIndex(buffer, r)
		if err == nil {
			indices[i] = uint16(index)
		}
	}
	return indices
}

type textSubstitution struct {
	src string
	dst rune
}

// TODO: read from liga tables in OpenType (clig, dlig, hlig) with rlig default enabled
var commonLigatures = []textSubstitution{
	{"ffi", '\uFB03'},
	{"ffl", '\uFB04'},
	{"ff", '\uFB00'},
	{"fi", '\uFB01'},
	{"fl", '\uFB02'},
}

var ligatures = map[rune]string{
	'\u00C6': "AE",
	'\u00DF': "ſz",
	'\u00E6': "ae",
	'\u0152': "OE",
	'\u0153': "oe",
	'\u01F6': "Hv",
	'\u0195': "hv",
	'\u2114': "lb",
	'\u1D6B': "ue",
	'\u1E9E': "ſs",
	'\u1EFA': "lL",
	'\u1EFB': "ll",
	'\uA6B2': "ɔe",
	'\uAB63': "uo",
	'\uA728': "TZ",
	'\uA729': "tz",
	'\uA732': "AA",
	'\uA733': "aa",
	'\uA734': "AO",
	'\uA735': "ao",
	'\uA736': "AU",
	'\uA737': "au",
	'\uA738': "AV",
	'\uA739': "av",
	'\uA73A': "AV",
	'\uA73B': "av",
	'\uA73C': "AY",
	'\uA73D': "ay",
	'\uA74E': "OO",
	'\uA74F': "oo",
	'\uA760': "VY",
	'\uA761': "vy",
	'\uAB31': "aə",
	'\uAB41': "əø",
	'\uFB00': "ff",
	'\uFB01': "fi",
	'\uFB02': "fl",
	'\uFB03': "ffi",
	'\uFB04': "ffl",
	'\uFB05': "ſt",
	'\uFB06': "st",
}

var superscriptSubstitutes = []textSubstitution{
	{"0", '\u2070'},
	{"i", '\u2071'},
	{"2", '\u00B2'},
	{"3", '\u00B3'},
	{"4", '\u2074'},
	{"5", '\u2075'},
	{"6", '\u2076'},
	{"7", '\u2077'},
	{"8", '\u2078'},
	{"9", '\u2079'},
	{"+", '\u207A'},
	{"-", '\u207B'},
	{"=", '\u207C'},
	{"(", '\u207D'},
	{")", '\u207E'},
	{"n", '\u207F'},
}

var subscriptSubstitutes = []textSubstitution{
	{"0", '\u2080'},
	{"1", '\u2081'},
	{"2", '\u2082'},
	{"3", '\u2083'},
	{"4", '\u2084'},
	{"5", '\u2085'},
	{"6", '\u2086'},
	{"7", '\u2087'},
	{"8", '\u2088'},
	{"9", '\u2089'},
	{"+", '\u208A'},
	{"-", '\u208B'},
	{"=", '\u208C'},
	{"(", '\u208D'},
	{")", '\u208E'},
	{"a", '\u2090'},
	{"e", '\u2091'},
	{"o", '\u2092'},
	{"x", '\u2093'},
	{"h", '\u2095'},
	{"k", '\u2096'},
	{"l", '\u2097'},
	{"m", '\u2098'},
	{"n", '\u2099'},
	{"p", '\u209A'},
	{"s", '\u209B'},
	{"t", '\u209C'},
}

func (f *Font) supportedSubstitutions(substitutions []textSubstitution) []textSubstitution {
	buffer := &sfnt.Buffer{}
	supported := []textSubstitution{}
	for _, stn := range substitutions {
		if _, err := f.sfnt.GlyphIndex(buffer, stn.dst); err == nil {
			supported = append(supported, stn)
		}
	}
	return supported
}

// Use enables typographic options on the font such as ligatures.
func (f *Font) Use(options TypographicOptions) {
	if options&NoTypography == 0 {
		f.typography = true
	}

	f.ligatures = []textSubstitution{}
	if options&CommonLigatures != 0 {
		f.ligatures = append(f.ligatures, f.supportedSubstitutions(commonLigatures)...)
	}
}

func (f *Font) substituteLigatures(s string) string {
	for _, stn := range f.ligatures {
		s = strings.ReplaceAll(s, stn.src, string(stn.dst))
	}
	return s
}

func (f *Font) substituteTypography(s string, inSingleQuote, inDoubleQuote bool) (string, bool, bool) {
	// TODO: typography substitution should maybe not be part of this package (or of Font)
	if f.typography {
		var rPrev, r rune
		var i, size int
		for {
			rPrev = r
			i += size
			if i >= len(s) {
				break
			}

			r, size = utf8.DecodeRuneInString(s[i:])
			if i+2 < len(s) && s[i] == '.' && s[i+1] == '.' && s[i+2] == '.' {
				s, size = stringReplace(s, i, 3, "\u2026") // ellipsis
				continue
			} else if i+4 < len(s) && s[i] == '.' && s[i+1] == ' ' && s[i+2] == '.' && s[i+3] == ' ' && s[i+4] == '.' {
				s, size = stringReplace(s, i, 5, "\u2026") // ellipsis
				continue
			} else if i+2 < len(s) && s[i] == '-' && s[i+1] == '-' && s[i+2] == '-' {
				s, size = stringReplace(s, i, 3, "\u2014") // em-dash
				continue
			} else if i+1 < len(s) && s[i] == '-' && s[i+1] == '-' {
				s, size = stringReplace(s, i, 2, "\u2013") // en-dash
				continue
			} else if i+2 < len(s) && s[i] == '(' && s[i+1] == 'c' && s[i+2] == ')' {
				s, size = stringReplace(s, i, 3, "\u00A9") // copyright
				continue
			} else if i+2 < len(s) && s[i] == '(' && s[i+1] == 'r' && s[i+2] == ')' {
				s, size = stringReplace(s, i, 3, "\u00AE") // registered
				continue
			} else if i+3 < len(s) && s[i] == '(' && s[i+1] == 't' && s[i+2] == 'm' && s[i+3] == ')' {
				s, size = stringReplace(s, i, 4, "\u2122") // trademark
				continue
			}

			// quotes
			if s[i] == '"' || s[i] == '\'' {
				var rNext rune
				if i+1 < len(s) {
					rNext, _ = utf8.DecodeRuneInString(s[i+1:])
				}
				if s[i] == '"' {
					s, size = quoteReplace(s, i, rPrev, r, rNext, &inDoubleQuote)
					continue
				} else {
					s, size = quoteReplace(s, i, rPrev, r, rNext, &inSingleQuote)
					continue
				}
			}

			// fractions
			if i+2 < len(s) && s[i+1] == '/' && isWordBoundary(rPrev) && rPrev != '/' {
				var rNext rune
				if i+3 < len(s) {
					rNext, _ = utf8.DecodeRuneInString(s[i+3:])
				}
				if isWordBoundary(rNext) && rNext != '/' {
					if s[i] == '1' && s[i+2] == '2' {
						s, size = stringReplace(s, i, 3, "\u00BD") // 1/2
						continue
					} else if s[i] == '1' && s[i+2] == '4' {
						s, size = stringReplace(s, i, 3, "\u00BC") // 1/4
						continue
					} else if s[i] == '3' && s[i+2] == '4' {
						s, size = stringReplace(s, i, 3, "\u00BE") // 3/4
						continue
					} else if s[i] == '+' && s[i+2] == '-' {
						s, size = stringReplace(s, i, 3, "\u00B1") // +/-
						continue
					}
				}
			}
		}
	}
	return s, inSingleQuote, inDoubleQuote
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
		}
		return stringReplace(s, i, 1, "\u201D")
	} else if quote == '\'' {
		if *isOpen {
			return stringReplace(s, i, 1, "\u2018")
		}
		return stringReplace(s, i, 1, "\u2019")
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
