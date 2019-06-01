package canvas

import (
	"encoding/base64"
	"io/ioutil"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

	findfont "github.com/flopp/go-findfont"
	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

var sfntBuffer sfnt.Buffer

type TransformationOptions int

const (
	NoTypography TransformationOptions = 2 << iota
	NoRequiredLigatures
	CommonLigatures
	DiscretionaryLigatures
	HistoricalLigatures
)

// TODO: read from liga tables in OpenType (clig, dlig, hlig) with rlig default enabled
var commonLigatures = [][2]string{
	{"ffi", "\uFB03"},
	{"ffl", "\uFB04"},
	{"ff", "\uFB00"},
	{"fi", "\uFB01"},
	{"fl", "\uFB02"},
}

type FontStyle int

const (
	Regular FontStyle = 0
	Bold    FontStyle = 1 << iota
	Italic
)

type Font struct {
	mimetype string
	raw      []byte

	sfnt  *sfnt.Font
	name  string
	style FontStyle

	transformationOptions  TransformationOptions
	requiredLigatures      [][2]string
	commonLigatures        [][2]string
	discretionaryLigatures [][2]string
	historicalLigatures    [][2]string
}

// LoadLocalFont loads a font from the system fonts location.
func LoadLocalFont(name string, style FontStyle) (Font, error) {
	fontPath, err := findfont.Find(name)
	if err != nil {
		return Font{}, err
	}
	return LoadFontFile(name, style, fontPath)
}

// LoadFontFile loads a font from a file.
func LoadFontFile(name string, style FontStyle, filename string) (Font, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return Font{}, err
	}
	return LoadFont(name, style, b)
}

// LoadFont loads a font from memory.
func LoadFont(name string, style FontStyle, b []byte) (Font, error) {
	mimetype, sfnt, err := parseFont(b)
	if err != nil {
		return Font{}, err
	}

	// TODO: extract from liga tables
	clig := [][2]string{}
	for _, transformation := range commonLigatures {
		var err error
		for _, r := range []rune(transformation[1]) {
			_, err = sfnt.GlyphIndex(&sfntBuffer, r)
			if err != nil {
				continue
			}
		}
		if err == nil {
			clig = append(clig, transformation)
		}
	}

	return Font{
		mimetype:        mimetype,
		raw:             b,
		sfnt:            sfnt,
		name:            name,
		style:           style,
		commonLigatures: clig,
	}, nil
}

func (f *Font) Use(transformationOptions TransformationOptions) {
	f.transformationOptions = transformationOptions
}

// Face gets the font face associated with the give font name and font size (in pt).
func (f *Font) Face(size float64) FontFace {
	// TODO: add hinting
	return FontFace{
		f:       f,
		ppem:    toI26_6(size * MmPerPt),
		hinting: font.HintingNone,
	}
}

func (f *Font) ToDataURI() string {
	sb := strings.Builder{}
	sb.WriteString("data:")
	sb.WriteString(f.mimetype)
	sb.WriteString(";base64,")
	encoder := base64.NewEncoder(base64.StdEncoding, &sb)
	encoder.Write(f.raw)
	encoder.Close()
	return sb.String()
}

// TODO: generate new raw with only used characters
func (f *Font) Raw() (string, []byte) {
	return f.mimetype, f.raw
}

type Metrics struct {
	Size       float64
	LineHeight float64
	Ascent     float64
	Descent    float64
	XHeight    float64
	CapHeight  float64
	Offset     float64
}

// TODO: dashed underline, dotted underline, overline, strikethrough
type FontDecoration func(Metrics, float64) *Path

const underlineDistance = 0.15
const underlineThickness = 0.1

var Underline = func(metrics Metrics, w float64) *Path {
	r := metrics.Size * underlineThickness
	w -= r

	y := -metrics.Size * underlineDistance

	p := &Path{}
	p.MoveTo(r, y)
	p.LineTo(w, y)
	return p.Stroke(r, RoundCapper, RoundJoiner)
}

var Overline = func(metrics Metrics, w float64) *Path {
	r := metrics.Size * underlineThickness
	w -= r
	y := metrics.XHeight + metrics.Size*underlineDistance

	p := &Path{}
	p.MoveTo(r, y)
	p.LineTo(w, y)
	return p.Stroke(r, RoundCapper, RoundJoiner)
}

var Strikethrough = func(metrics Metrics, w float64) *Path {
	r := metrics.Size * underlineThickness
	w -= r

	y := metrics.XHeight / 2.0

	p := &Path{}
	p.MoveTo(r, y)
	p.LineTo(w, y)
	return p.Stroke(r, RoundCapper, RoundJoiner)
}

var DoubleUnderline = func(metrics Metrics, w float64) *Path {
	r := metrics.Size * underlineThickness
	w -= r

	y := -metrics.Size * underlineDistance * 0.75

	p := &Path{}
	p.MoveTo(r, y)
	p.LineTo(w, y)
	p.MoveTo(r, y-r*1.5)
	p.LineTo(w, y-r*1.5)
	return p.Stroke(r, RoundCapper, RoundJoiner)
}

var DottedUnderline = func(metrics Metrics, w float64) *Path {
	r := metrics.Size * underlineThickness / 2.0
	w -= r

	y := -metrics.Size * underlineDistance
	d := 10.0 * underlineThickness
	n := int((w-r)/d) + 1
	d = (w - r) / float64(n-1)

	p := &Path{}
	for i := 0; i < n; i++ {
		p.Append(Circle(r+float64(i)*d, y, r))
	}
	return p
}

var DashedUnderline = func(metrics Metrics, w float64) *Path {
	r := metrics.Size * underlineThickness
	w -= r

	y := -metrics.Size * underlineDistance
	d := 6.0 * underlineThickness
	n := int((w-r)/(3.0*d)) + 1
	d = (w - r + 2.0*d) / float64(3*n-3)

	p := &Path{}
	p.MoveTo(r, y)
	p.LineTo(w, y)
	p = p.Dash(d, d*2.0).Stroke(r, RoundCapper, RoundJoiner)
	return p
}

var SineUnderline = func(metrics Metrics, w float64) *Path {
	r := metrics.Size * underlineThickness
	w -= r

	dh := -metrics.Size * 0.15
	y := -metrics.Size * underlineDistance
	d := 10.0 * underlineThickness
	n := int(0.5 + w/d)
	dx := r

	p := &Path{}
	p.MoveTo(dx, y)
	for i := 0; i < n; i++ {
		if i%2 == 0 {
			p.CubeTo(dx+d*0.3642, y, dx+d*0.6358, y+dh, dx+d, y+dh)
		} else {
			p.CubeTo(dx+d*0.3642, y+dh, dx+d*0.6358, y, dx+d, y)
		}
		dx += d
	}
	return p.Stroke(r, RoundCapper, RoundJoiner)
}

var SawtoothUnderline = func(metrics Metrics, w float64) *Path {
	r := metrics.Size * underlineThickness
	w -= r

	dh := -metrics.Size * 0.15
	y := -metrics.Size * underlineDistance
	d := 6.0 * underlineThickness
	n := int(0.5 + w/d)
	dx := r

	p := &Path{}
	p.MoveTo(dx, y)
	for i := 0; i < n; i++ {
		if i%2 == 0 {
			p.LineTo(dx+d, y+dh)
		} else {
			p.LineTo(dx+d, y)
		}
		dx += d
	}
	return p.Stroke(r, RoundCapper, RoundJoiner)
}

// TODO: use font provided subscript etc, or use suggested values for subscript position and size
// same for underlining
type FontFace struct {
	f                            *Font
	ppem                         fixed.Int26_6
	hinting                      font.Hinting
	offset, fauxBold, fauxItalic float64 // relative to ppem
	decoration                   *FontDecoration
}

func (ff FontFace) Subscript() FontFace {
	ff.ppem = ff.ppem.Mul(toI26_6(0.583))
	ff.offset = -0.33 / 0.583
	ff.fauxBold = 0.1
	return ff
}

func (ff FontFace) Superscript() FontFace {
	ff.ppem = ff.ppem.Mul(toI26_6(0.583))
	ff.offset = 0.33 / 0.583
	ff.fauxBold = 0.1
	return ff
}

func (ff FontFace) FauxBold() FontFace {
	ff.fauxBold = 0.02
	return ff
}

func (ff FontFace) FauxItalic() FontFace {
	ff.fauxItalic = 0.07
	return ff
}

func (ff FontFace) Decoration(decoration FontDecoration) FontFace {
	ff.decoration = &decoration
	return ff
}

func (ff FontFace) ToPathDecoration(width float64) *Path {
	if ff.decoration == nil {
		return &Path{}
	}
	return (*ff.decoration)(ff.Metrics(), width)
}

// Info returns the font name, style and size.
func (ff FontFace) Info() (name string, size float64, style FontStyle, decoration *FontDecoration) {
	return ff.f.name, fromI26_6(ff.ppem), ff.f.style, ff.decoration
}

// Metrics returns the font metrics. See https://developer.apple.com/library/archive/documentation/TextFonts/Conceptual/CocoaTextArchitecture/Art/glyph_metrics_2x.png for an explaination of the different metrics.
func (ff FontFace) Metrics() Metrics {
	m, _ := ff.f.sfnt.Metrics(&sfntBuffer, ff.ppem, ff.hinting)
	return Metrics{
		Size:       fromI26_6(ff.ppem),
		LineHeight: math.Abs(fromI26_6(m.Height)),
		Ascent:     math.Abs(fromI26_6(m.Ascent)),
		Descent:    math.Abs(fromI26_6(m.Descent)),
		XHeight:    math.Abs(fromI26_6(m.XHeight)),
		CapHeight:  math.Abs(fromI26_6(m.CapHeight)),
		Offset:     ff.offset * fromI26_6(ff.ppem),
	}
}

// textWidth returns the width of a given string in mm.
func (ff FontFace) TextWidth(s string) float64 {
	w := 0.0
	var prevIndex sfnt.GlyphIndex
	for i, r := range s {
		index, err := ff.f.sfnt.GlyphIndex(&sfntBuffer, r)
		if err != nil {
			continue
		}

		if i != 0 {
			kern, err := ff.f.sfnt.Kern(&sfntBuffer, prevIndex, index, ff.ppem, ff.hinting)
			if err == nil {
				w += fromI26_6(kern)
			}
		}
		advance, err := ff.f.sfnt.GlyphAdvance(&sfntBuffer, index, ff.ppem, ff.hinting)
		if err == nil {
			w += fromI26_6(advance)
		}
		prevIndex = index
	}
	return w
}

// ToPath converts a rune to a path and its advance.
func (ff FontFace) ToPath(r rune) (*Path, float64) {
	p := &Path{}
	index, err := ff.f.sfnt.GlyphIndex(&sfntBuffer, r)
	if err != nil {
		return p, 0.0
	}

	segments, err := ff.f.sfnt.LoadGlyph(&sfntBuffer, index, ff.ppem, nil)
	if err != nil {
		return p, 0.0
	}

	offset := ff.offset * fromI26_6(ff.ppem)
	fauxBold := ff.fauxBold * fromI26_6(ff.ppem)
	fauxItalic := ff.fauxItalic * fromI26_6(ff.ppem)

	var start0, end Point
	for i, segment := range segments {
		switch segment.Op {
		case sfnt.SegmentOpMoveTo:
			if i != 0 && start0.Equals(end) {
				p.Close()
			}
			end = fromP26_6(segment.Args[0])
			end.X += fauxItalic * -end.Y
			p.MoveTo(end.X, offset-end.Y)
			start0 = end
		case sfnt.SegmentOpLineTo:
			end = fromP26_6(segment.Args[0])
			end.X += fauxItalic * -end.Y
			p.LineTo(end.X, offset-end.Y)
		case sfnt.SegmentOpQuadTo:
			cp := fromP26_6(segment.Args[0])
			end = fromP26_6(segment.Args[1])
			cp.X += fauxItalic * -cp.Y
			end.X += fauxItalic * -end.Y
			p.QuadTo(cp.X, offset-cp.Y, end.X, offset-end.Y)
		case sfnt.SegmentOpCubeTo:
			cp1 := fromP26_6(segment.Args[0])
			cp2 := fromP26_6(segment.Args[1])
			end = fromP26_6(segment.Args[2])
			cp1.X += fauxItalic * -cp1.Y
			cp2.X += fauxItalic * -cp2.Y
			end.X += fauxItalic * -end.Y
			p.CubeTo(cp1.X, offset-cp1.Y, cp2.X, offset-cp2.Y, end.X, offset-end.Y)
		}
	}
	if !p.Empty() && start0.Equals(end) {
		p.Close()
	}
	if fauxBold != 0.0 {
		p = p.Offset(fauxBold)
	}

	dx := 0.0
	advance, err := ff.f.sfnt.GlyphAdvance(&sfntBuffer, index, ff.ppem, ff.hinting)
	if err == nil {
		dx = fromI26_6(advance)
	}
	return p, dx
}

func (ff FontFace) Kerning(rPrev, rNext rune) float64 {
	prevIndex, err := ff.f.sfnt.GlyphIndex(&sfntBuffer, rPrev)
	if err != nil {
		return 0.0
	}

	nextIndex, err := ff.f.sfnt.GlyphIndex(&sfntBuffer, rNext)
	if err != nil {
		return 0.0
	}

	kern, err := ff.f.sfnt.Kern(&sfntBuffer, prevIndex, nextIndex, ff.ppem, ff.hinting)
	if err == nil {
		return fromI26_6(kern)
	}
	return 0.0
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

func isWordBoundary(r rune) bool {
	return r == 0 || isspace(r) || ispunct(r)
}

func stringReplace(s string, i, n int, target string) (string, int) {
	s = s[:i] + target + s[i+n:]
	return s, len(target)
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

func (f *Font) transform(s string, replaceCombinations bool) string {
	s = strings.ReplaceAll(s, "\u200b", "")
	if f.transformationOptions&NoRequiredLigatures == 0 {
		for _, transformation := range f.requiredLigatures {
			s = strings.ReplaceAll(s, transformation[0], transformation[1])
		}
	}
	if f.transformationOptions&CommonLigatures != 0 {
		for _, transformation := range f.commonLigatures {
			if replaceCombinations || utf8.RuneCountInString(transformation[0]) == 1 {
				s = strings.ReplaceAll(s, transformation[0], transformation[1])
			}
		}
	}
	if f.transformationOptions&DiscretionaryLigatures != 0 {
		for _, transformation := range f.discretionaryLigatures {
			if replaceCombinations || utf8.RuneCountInString(transformation[0]) == 1 {
				s = strings.ReplaceAll(s, transformation[0], transformation[1])
			}
		}
	}
	if f.transformationOptions&HistoricalLigatures != 0 {
		for _, transformation := range f.historicalLigatures {
			if replaceCombinations || utf8.RuneCountInString(transformation[0]) == 1 {
				s = strings.ReplaceAll(s, transformation[0], transformation[1])
			}
		}
	}
	// TODO: make sure unicode points exist in font
	if f.transformationOptions&NoTypography == 0 {
		var inSingleQuote, inDoubleQuote bool
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

			var rNext rune
			// quotes
			if i+1 < len(s) {
				rNext, _ = utf8.DecodeRuneInString(s[i+1:])
			}
			if s[i] == '"' {
				s, size = quoteReplace(s, i, rPrev, r, rNext, &inDoubleQuote)
				continue
			} else if s[i] == '\'' {
				s, size = quoteReplace(s, i, rPrev, r, rNext, &inSingleQuote)
				continue
			}

			// fractions
			if i+3 < len(s) {
				rNext, _ = utf8.DecodeRuneInString(s[i+3:])
			}
			if i+2 < len(s) && s[i+1] == '/' && isWordBoundary(rPrev) && rPrev != '/' && isWordBoundary(rNext) && rNext != '/' {
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
	return s
}
