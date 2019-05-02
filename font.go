package canvas

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"math"
	"strings"

	findfont "github.com/flopp/go-findfont"
	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

var sfntBuffer sfnt.Buffer

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

	usedGlyphs map[rune]bool
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
	return Font{
		mimetype:   mimetype,
		raw:        b,
		sfnt:       sfnt,
		name:       name,
		style:      style,
		usedGlyphs: map[rune]bool{},
	}, nil
}

// Face gets the font face associated with the give font name and font size (in mm).
func (f *Font) Face(size float64) FontFace {
	// TODO: add hinting
	return FontFace{
		f:       f,
		ppem:    toI26_6(size),
		hinting: font.HintingNone,
	}
}

func (f *Font) MarkUsed(s string) {
	for _, r := range s {
		f.usedGlyphs[r] = true
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

func (f *Font) ToSVG() string {
	sb := strings.Builder{}
	sb.WriteString("<font>")
	sb.WriteString("<font-face font-family=\"")
	sb.WriteString(f.name)
	if f.style&Italic != 0 {
		sb.WriteString("\" font-style=\"italic")
	}
	if f.style&Bold != 0 {
		sb.WriteString("\" font-weight=\"bold")
	}
	sb.WriteString("\" units-per-em=\"1000\">")

	ff := f.Face(1000.0)
	for r, _ := range f.usedGlyphs {
		glyph, advance := ff.ToPath(r)
		sb.WriteString("<glyph unicode=\"")
		sb.WriteRune(r) // TODO: use XML character ref for non-ASCII
		sb.WriteString("\" horiz-adv-x=\"")
		fmt.Fprintf(&sb, "%.0g", advance)
		sb.WriteString("\" d=\"")
		sb.WriteString(glyph.ToSVG())
		sb.WriteString("\">")
	}

	for r0, _ := range f.usedGlyphs {
		for r1, _ := range f.usedGlyphs {
			sb.WriteString("<hkern g1=\"")
			sb.WriteRune(r0) // TODO: use XML character ref for non-ASCII
			sb.WriteString("\" g2=\"")
			sb.WriteRune(r1) // TODO: use XML character ref for non-ASCII
			sb.WriteString("\" k=\"")
			fmt.Fprintf(&sb, "%.0g", ff.Kerning(r0, r1))
			sb.WriteString("\">")
		}
	}

	sb.WriteString("</font>")
	return sb.String()
}

type Metrics struct {
	Size       float64
	LineHeight float64
	Ascent     float64
	Descent    float64
	XHeight    float64
	CapHeight  float64
}

type FontFace struct {
	f       *Font
	ppem    fixed.Int26_6
	hinting font.Hinting
}

// Info returns the font name, style and size.
func (ff FontFace) Info() (name string, style FontStyle, size float64) {
	return ff.f.name, ff.f.style, fromI26_6(ff.ppem)
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

	var start0, end Point
	for i, segment := range segments {
		switch segment.Op {
		case sfnt.SegmentOpMoveTo:
			if i != 0 && start0.Equals(end) {
				p.Close()
			}
			end = fromP26_6(segment.Args[0])
			p.MoveTo(end.X, -end.Y)
			start0 = end
		case sfnt.SegmentOpLineTo:
			end = fromP26_6(segment.Args[0])
			p.LineTo(end.X, -end.Y)
		case sfnt.SegmentOpQuadTo:
			c := fromP26_6(segment.Args[0])
			end = fromP26_6(segment.Args[1])
			p.QuadTo(c.X, -c.Y, end.X, -end.Y)
		case sfnt.SegmentOpCubeTo:
			c0 := fromP26_6(segment.Args[0])
			c1 := fromP26_6(segment.Args[1])
			end = fromP26_6(segment.Args[2])
			p.CubeTo(c0.X, -c0.Y, c1.X, -c1.Y, end.X, -end.Y)
		}
	}
	if !p.Empty() && start0.Equals(end) {
		p.Close()
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
