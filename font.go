package canvas

import (
	"image/color"
	"io/ioutil"
	"math"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"

	findfont "github.com/flopp/go-findfont"
	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

var sfntBuffer sfnt.Buffer

// TypographicOptions are the options that can be enabled to make typographic or ligature substitutions automatically.
type TypographicOptions int

const (
	NoTypography TypographicOptions = 2 << iota
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

// FontStyle defines the font style to be used for the font. Note that Subscript/Inferior and Superscript/Superior might be the same for a given font.
type FontStyle int

const (
	Regular FontStyle = 0
	Bold    FontStyle = 1 << iota
	Italic
	Subscript
	Superscript
	Inferior
	Superior
)

// Font defines a font of type TTF or OTF which which a FontFace can be generated for use in text drawing operations.
type Font struct {
	mimetype string
	raw      []byte

	sfnt  *sfnt.Font
	name  string
	style FontStyle

	typographicOptions     TypographicOptions
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

// Use specifies which typographic options shall be used, ie. whether to use common typographic substitutions and which ligatures classes to use.
func (f *Font) Use(typographicOptions TypographicOptions) {
	f.typographicOptions = typographicOptions
}

// Face gets the font face given by the font size (in pt).
func (f *Font) Face(size float64) FontFace {
	// TODO: add hinting
	return FontFace{
		f:        f,
		ppemOrig: toI26_6(size * MmPerPt),
		ppem:     toI26_6(size * MmPerPt),
		hinting:  font.HintingNone,
	}
}

// Raw returns the mimetype and raw binary data of the font.
func (f *Font) Raw() (string, []byte) {
	// TODO: generate new raw with only used characters
	return f.mimetype, f.raw
}

// FontFace defines a font face from a given font. It allows setting the font size, its color, faux styles and font decorations.
type FontFace struct {
	f              *Font
	ppemOrig, ppem fixed.Int26_6
	hinting        font.Hinting

	color                        color.RGBA
	fauxStyle                    FontStyle
	offset, fauxBold, fauxItalic float64
	decoration                   []FontDecorator
}

// Equals returns true when two font face are equal. In particular this allows two adjacent text spans that use the same decoration to allow the decoration to span both elements instead of two separately.
func (ff FontFace) Equals(other FontFace) bool {
	return ff.f == other.f && ff.ppemOrig == other.ppemOrig && ff.color == other.color && ff.fauxStyle == other.fauxStyle && reflect.DeepEqual(ff.decoration, other.decoration)
}

// Color sets the color to be used and returns a new FontFace.
func (ff FontFace) Color(col color.Color) FontFace {
	r, g, b, a := col.RGBA()
	ff.color = color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
	return ff
}

// Faux sets the faux styles to be used and returns a new FontFace.
func (ff FontFace) Faux(style FontStyle) FontFace {
	// TODO: use font provided subscript etc, or use suggested values for subscript position and size
	metricsOrig := ff.Metrics()
	ff.offset = 0.0
	ff.fauxBold = 0.0
	ff.fauxItalic = 0.0
	ff.ppem = ff.ppemOrig
	if style&Bold != 0 {
		ff.fauxBold = 0.02
	}
	if style&Italic != 0 {
		ff.fauxItalic = 0.07
	}
	if style&Subscript != 0 || style&Superscript != 0 || style&Inferior != 0 || style&Superior != 0 {
		ff.ppem = ff.ppem.Mul(toI26_6(0.583))
		ff.fauxBold += 0.02
	}
	if style&Subscript != 0 {
		ff.offset = -0.33 * fromI26_6(ff.ppemOrig)
	}
	if style&Superscript != 0 {
		ff.offset = 0.33 * fromI26_6(ff.ppemOrig)
	}
	if style&Superior != 0 {
		ff.offset = metricsOrig.XHeight * (1.0 - 0.583)
	}
	ff.fauxBold *= fromI26_6(ff.ppem)
	ff.fauxItalic *= fromI26_6(ff.ppem)
	ff.fauxStyle = style
	return ff
}

// Decoration sets the decorations to be used and returns a new FontFace.
func (ff FontFace) Decoration(decorators ...FontDecorator) FontFace {
	if ff.decoration == nil {
		ff.decoration = []FontDecorator{}
	}
	for _, deco := range decorators {
		ff.decoration = append(ff.decoration, deco)
	}
	return ff
}

// Decorate will return a path from the decorations specified in the FontFace over a given width in mm.
func (ff FontFace) Decorate(width float64) *Path {
	p := &Path{}
	if ff.decoration != nil {
		for _, deco := range ff.decoration {
			p = p.Append(deco.Decorate(ff, width))
		}
	}
	return p
}

// Info returns the font name, size and style.
func (ff FontFace) Info() (name string, size float64, style FontStyle) {
	return ff.f.name, fromI26_6(ff.ppem), ff.f.style
}

// FontMetrics contains a number of metrics that define a font face.
type FontMetrics struct {
	Size       float64
	LineHeight float64
	Ascent     float64
	Descent    float64
	XHeight    float64
	CapHeight  float64
}

// Metrics returns the font metrics. See https://developer.apple.com/library/archive/documentation/TextFonts/Conceptual/CocoaTextArchitecture/Art/glyph_metrics_2x.png for an explaination of the different metrics.
func (ff FontFace) Metrics() FontMetrics {
	m, _ := ff.f.sfnt.Metrics(&sfntBuffer, ff.ppem, ff.hinting)
	return FontMetrics{
		Size:       fromI26_6(ff.ppem),
		LineHeight: math.Abs(fromI26_6(m.Height)),
		Ascent:     math.Abs(fromI26_6(m.Ascent)),
		Descent:    math.Abs(fromI26_6(m.Descent)),
		XHeight:    math.Abs(fromI26_6(m.XHeight)),
		CapHeight:  math.Abs(fromI26_6(m.CapHeight)),
	}
}

// TextWidth returns the width of a given string in mm.
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

// ToPath converts a string to a path and also returns its advance in mm.
func (ff FontFace) ToPath(s string) (*Path, float64) {
	// TODO: use caching if performance suffers
	p := &Path{}
	x := 0.0
	for _, r := range s {
		index, err := ff.f.sfnt.GlyphIndex(&sfntBuffer, r)
		if err != nil {
			return p, 0.0
		}

		segments, err := ff.f.sfnt.LoadGlyph(&sfntBuffer, index, ff.ppem, nil)
		if err != nil {
			return p, 0.0
		}

		var start0, end Point
		for i, segment := range segments {
			switch segment.Op {
			case sfnt.SegmentOpMoveTo:
				if i != 0 && start0.Equals(end) {
					p.Close()
				}
				end = fromP26_6(segment.Args[0])
				end.X += ff.fauxItalic * -end.Y
				p.MoveTo(x+end.X, ff.offset-end.Y)
				start0 = end
			case sfnt.SegmentOpLineTo:
				end = fromP26_6(segment.Args[0])
				end.X += ff.fauxItalic * -end.Y
				p.LineTo(x+end.X, ff.offset-end.Y)
			case sfnt.SegmentOpQuadTo:
				cp := fromP26_6(segment.Args[0])
				end = fromP26_6(segment.Args[1])
				cp.X += ff.fauxItalic * -cp.Y
				end.X += ff.fauxItalic * -end.Y
				p.QuadTo(x+cp.X, ff.offset-cp.Y, x+end.X, ff.offset-end.Y)
			case sfnt.SegmentOpCubeTo:
				cp1 := fromP26_6(segment.Args[0])
				cp2 := fromP26_6(segment.Args[1])
				end = fromP26_6(segment.Args[2])
				cp1.X += ff.fauxItalic * -cp1.Y
				cp2.X += ff.fauxItalic * -cp2.Y
				end.X += ff.fauxItalic * -end.Y
				p.CubeTo(x+cp1.X, ff.offset-cp1.Y, x+cp2.X, ff.offset-cp2.Y, x+end.X, ff.offset-end.Y)
			}
		}
		if !p.Empty() && start0.Equals(end) {
			p.Close()
		}
		if ff.fauxBold != 0.0 {
			p = p.Offset(ff.fauxBold)
		}

		advance, err := ff.f.sfnt.GlyphAdvance(&sfntBuffer, index, ff.ppem, ff.hinting)
		if err == nil {
			x += fromI26_6(advance)
		}
	}
	return p, x
}

// Kerning returns the kerning between two runes in mm (ie. the adjustment on the advance).
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

////////////////////////////////////////////////////////////////

// FontDecorator is an interface that returns a path given a font face and a width in mm.
type FontDecorator interface {
	Decorate(FontFace, float64) *Path
}

const underlineDistance = 0.15
const underlineThickness = 0.075

// Underline is a font decoration that draws a line under the text at the base line.
var Underline FontDecorator = underline{}

type underline struct{}

func (underline) Decorate(ff FontFace, w float64) *Path {
	r := ff.Metrics().Size * underlineThickness
	y := -ff.Metrics().Size * underlineDistance

	p := &Path{}
	p.MoveTo(0.0, y)
	p.LineTo(w, y)
	return p.Stroke(r, ButtCapper, BevelJoiner)
}

// Overline is a font decoration that draws a line over the text at the X-Height line.
var Overline FontDecorator = overline{}

type overline struct{}

func (overline) Decorate(ff FontFace, w float64) *Path {
	r := ff.Metrics().Size * underlineThickness
	y := ff.Metrics().XHeight + ff.Metrics().Size*underlineDistance

	dx := ff.fauxItalic * y
	w += ff.fauxItalic * y

	p := &Path{}
	p.MoveTo(dx, y)
	p.LineTo(w, y)
	return p.Stroke(r, ButtCapper, BevelJoiner)
}

// Strikethrough is a font decoration that draws a line through the text in the middle between the base and X-Height line.
var Strikethrough FontDecorator = strikethrough{}

type strikethrough struct{}

func (strikethrough) Decorate(ff FontFace, w float64) *Path {
	r := ff.Metrics().Size * underlineThickness
	y := ff.Metrics().XHeight / 2.0

	dx := ff.fauxItalic * y
	w += ff.fauxItalic * y

	p := &Path{}
	p.MoveTo(dx, y)
	p.LineTo(w, y)
	return p.Stroke(r, ButtCapper, BevelJoiner)
}

// DoubleUnderline is a font decoration that draws two lines at the base line.
var DoubleUnderline FontDecorator = doubleUnderline{}

type doubleUnderline struct{}

func (doubleUnderline) Decorate(ff FontFace, w float64) *Path {
	r := ff.Metrics().Size * underlineThickness
	y := -ff.Metrics().Size * underlineDistance * 0.75

	p := &Path{}
	p.MoveTo(0.0, y)
	p.LineTo(w, y)
	p.MoveTo(0.0, y-r*2.0)
	p.LineTo(w, y-r*2.0)
	return p.Stroke(r, ButtCapper, BevelJoiner)
}

// DottedUnderline is a font decoration that draws a dotted line at the base line.
var DottedUnderline FontDecorator = dottedUnderline{}

type dottedUnderline struct{}

func (dottedUnderline) Decorate(ff FontFace, w float64) *Path {
	r := ff.Metrics().Size * underlineThickness * 0.8
	w -= r

	y := -ff.Metrics().Size * underlineDistance
	d := 15.0 * underlineThickness
	n := int((w-r)/d) + 1
	d = (w - r) / float64(n-1)

	p := &Path{}
	for i := 0; i < n; i++ {
		p = p.Append(Circle(r).Translate(r+float64(i)*d, y))
	}
	return p
}

// DashedUnderline is a font decoration that draws a dashed line at the base line.
var DashedUnderline FontDecorator = dashedUnderline{}

type dashedUnderline struct{}

func (dashedUnderline) Decorate(ff FontFace, w float64) *Path {
	r := ff.Metrics().Size * underlineThickness
	y := -ff.Metrics().Size * underlineDistance
	d := 12.0 * underlineThickness
	n := int(w / (2.0 * d))
	d = w / float64(2*n-1)

	p := &Path{}
	p.MoveTo(0.0, y)
	p.LineTo(w, y)
	p = p.Dash(d).Stroke(r, ButtCapper, BevelJoiner)
	return p
}

// SineUnderline is a font decoration that draws a wavy sine path at the base line.
var SineUnderline FontDecorator = sineUnderline{}

type sineUnderline struct{}

func (sineUnderline) Decorate(ff FontFace, w float64) *Path {
	r := ff.Metrics().Size * underlineThickness
	w -= r

	dh := -ff.Metrics().Size * 0.15
	y := -ff.Metrics().Size * underlineDistance
	d := 12.0 * underlineThickness
	n := int(0.5 + w/d)
	d = (w - r) / float64(n)

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

// Sawtooth is a font decoration that draws a wavy sawtooth path at the base line.
var SawtoothUnderline FontDecorator = sawtoothUnderline{}

type sawtoothUnderline struct{}

func (sawtoothUnderline) Decorate(ff FontFace, w float64) *Path {
	r := ff.Metrics().Size * underlineThickness
	dx := 0.707 * r
	w -= 2.0 * dx

	dh := -ff.Metrics().Size * 0.15
	y := -ff.Metrics().Size * underlineDistance
	d := 8.0 * underlineThickness
	n := int(0.5 + w/d)
	d = w / float64(n)

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
	return p.Stroke(r, ButtCapper, MiterJoiner)
}

////////////////////////////////////////////////////////////////

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
	if f.typographicOptions&NoRequiredLigatures == 0 {
		for _, transformation := range f.requiredLigatures {
			s = strings.ReplaceAll(s, transformation[0], transformation[1])
		}
	}
	if f.typographicOptions&CommonLigatures != 0 {
		for _, transformation := range f.commonLigatures {
			if replaceCombinations || utf8.RuneCountInString(transformation[0]) == 1 {
				s = strings.ReplaceAll(s, transformation[0], transformation[1])
			}
		}
	}
	if f.typographicOptions&DiscretionaryLigatures != 0 {
		for _, transformation := range f.discretionaryLigatures {
			if replaceCombinations || utf8.RuneCountInString(transformation[0]) == 1 {
				s = strings.ReplaceAll(s, transformation[0], transformation[1])
			}
		}
	}
	if f.typographicOptions&HistoricalLigatures != 0 {
		for _, transformation := range f.historicalLigatures {
			if replaceCombinations || utf8.RuneCountInString(transformation[0]) == 1 {
				s = strings.ReplaceAll(s, transformation[0], transformation[1])
			}
		}
	}
	// TODO: make sure unicode points exist in font
	if f.typographicOptions&NoTypography == 0 {
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
