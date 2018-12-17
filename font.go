package canvas

import (
	"fmt"
	"io/ioutil"
	"math"
	"path"
	"unicode/utf8"

	findfont "github.com/flopp/go-findfont"
	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

var sfntBuffer sfnt.Buffer

var ErrNotFound = fmt.Errorf("font not found")

type FontStyle int

const (
	Regular FontStyle = 0
	Bold    FontStyle = 1 << iota
	Italic
)

type Fonts struct {
	fonts map[string]Font
	dpi   float64
}

func NewFonts(dpi float64) *Fonts {
	return &Fonts{
		fonts: map[string]Font{},
		dpi:   dpi,
	}
}

// AddLocalFont adds a font from the system fonts location.
func (fs *Fonts) AddLocalFont(name string, style FontStyle) error {
	fontPath, err := findfont.Find(name)
	if err != nil {
		return err
	}
	return fs.AddFontFile(name, style, fontPath)
}

// AddFontFile adds a font file given by a file path.
func (fs *Fonts) AddFontFile(name string, style FontStyle, filename string) error {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	mimetype := ""
	switch path.Ext(filename) {
	case ".ttf":
		mimetype = "font/truetype"
	case ".otf":
		mimetype = "font/opentype"
	case ".woff":
		mimetype = "font/woff"
	case ".woff2":
		mimetype = "font/woff2"
	}

	return fs.AddFont(name, style, mimetype, b)
}

// AddFont adds a font from its raw bytes.
func (fs *Fonts) AddFont(name string, style FontStyle, mimetype string, b []byte) error {
	f, err := NewFont(name, style, fs.dpi, mimetype, b)
	if err != nil {
		return err
	}
	fs.fonts[name] = f
	return nil
}

// Font returns a previously added font.
func (fs *Fonts) Font(name string) (Font, error) {
	if _, ok := fs.fonts[name]; !ok {
		return Font{}, ErrNotFound
	}
	return fs.fonts[name], nil
}

type Font struct {
	mimetype string
	raw      []byte

	font  *sfnt.Font
	name  string
	style FontStyle
	dpi   float64
}

func NewFont(name string, style FontStyle, dpi float64, mimetype string, b []byte) (Font, error) {
	// TODO: get mimetype from header
	f, err := sfnt.Parse(b)
	if err != nil {
		return Font{}, err
	}

	return Font{
		mimetype: mimetype,
		raw:      b,
		font:     f,
		name:     name,
		style:    style,
		dpi:      dpi,
	}, nil
}

// Face gets the font face associated with the give font name and font size (in mm).
func (f *Font) Face(size float64) FontFace {
	return FontFace{
		font:    f.font,
		name:    f.name,
		style:   f.style,
		size:    size,
		ppem:    toI26_6(size * (f.dpi / 72.0)),
		hinting: font.HintingNone,
	}
}

type Metrics struct {
	LineHeight float64
	Ascent     float64
	Descent    float64
	XHeight    float64
	CapHeight  float64
}

type FontFace struct {
	font    *sfnt.Font
	name    string
	style   FontStyle
	size    float64
	ppem    fixed.Int26_6
	hinting font.Hinting
}

// Info returns the font name, style and size.
func (ff FontFace) Info() (name string, style FontStyle, size float64) {
	return ff.name, ff.style, ff.size
}

// Metrics returns the font metrics. See https://developer.apple.com/library/archive/documentation/TextFonts/Conceptual/CocoaTextArchitecture/Art/glyph_metrics_2x.png for an explaination of the different metrics.
func (ff FontFace) Metrics() Metrics {
	m, _ := ff.font.Metrics(&sfntBuffer, ff.ppem, ff.hinting)
	return Metrics{
		LineHeight: math.Abs(fromI26_6(m.Height)),
		Ascent:     math.Abs(fromI26_6(m.Ascent)),
		Descent:    math.Abs(fromI26_6(m.Descent)),
		XHeight:    math.Abs(fromI26_6(m.XHeight)),
		CapHeight:  math.Abs(fromI26_6(m.CapHeight)),
	}
}

func splitNewlines(s string) []string {
	ss := []string{}
	i := 0
	for j, r := range s {
		if r == '\n' || r == '\r' || r == '\u2028' || r == '\u2029' {
			if r == '\n' && j > 0 && s[j-1] == '\r' {
				i++
				continue
			}
			ss = append(ss, s[i:j])
			i = j + utf8.RuneLen(r)
		}
	}
	ss = append(ss, s[i:])
	return ss
}

// textWidth returns the width of a given string in mm.
func (ff FontFace) textWidth(s string) float64 {
	x := 0.0
	var prevIndex sfnt.GlyphIndex
	for i, r := range s {
		index, err := ff.font.GlyphIndex(&sfntBuffer, r)
		if err != nil {
			continue
		}

		if i != 0 {
			kern, err := ff.font.Kern(&sfntBuffer, prevIndex, index, ff.ppem, ff.hinting)
			if err == nil {
				x += fromI26_6(kern)
			}
		}
		advance, err := ff.font.GlyphAdvance(&sfntBuffer, index, ff.ppem, ff.hinting)
		if err == nil {
			x += fromI26_6(advance)
		}
		prevIndex = index
	}
	return x
}

// Bounds returns the bounding box (width and height) of a string.
func (ff FontFace) Bounds(s string) (w float64, h float64) {
	ss := splitNewlines(s)
	for _, s := range ss {
		w = math.Max(w, ff.textWidth(s))
	}
	h = ff.Metrics().CapHeight + float64(len(ss)-1)*ff.Metrics().LineHeight
	return w, h
}

// ToPath converts a string to a path.
func (ff FontFace) ToPath(s string) *Path {
	p := &Path{}
	x := 0.0
	y := 0.0
	for _, s := range splitNewlines(s) {
		var prevIndex sfnt.GlyphIndex
		for i, r := range s {
			index, err := ff.font.GlyphIndex(&sfntBuffer, r)
			if err != nil {
				continue
			}

			if i > 0 {
				kern, err := ff.font.Kern(&sfntBuffer, prevIndex, index, ff.ppem, ff.hinting)
				if err == nil {
					x += fromI26_6(kern)
				}
			}

			segments, err := ff.font.LoadGlyph(&sfntBuffer, index, ff.ppem, nil)
			if err != nil {
				continue
			}

			var start0, end Point
			for i, segment := range segments {
				switch segment.Op {
				case sfnt.SegmentOpMoveTo:
					if i != 0 && start0.Equals(end) {
						p.Close()
					}
					end = fromP26_6(segment.Args[0])
					p.MoveTo(x+end.X, y+end.Y)
					start0 = end
				case sfnt.SegmentOpLineTo:
					end = fromP26_6(segment.Args[0])
					p.LineTo(x+end.X, y+end.Y)
				case sfnt.SegmentOpQuadTo:
					c := fromP26_6(segment.Args[0])
					end = fromP26_6(segment.Args[1])
					p.QuadTo(x+c.X, y+c.Y, x+end.X, y+end.Y)
				case sfnt.SegmentOpCubeTo:
					c0 := fromP26_6(segment.Args[0])
					c1 := fromP26_6(segment.Args[1])
					end = fromP26_6(segment.Args[2])
					p.CubeTo(x+c0.X, y+c0.Y, x+c1.X, y+c1.Y, x+end.X, y+end.Y)
				}
			}
			if !p.Empty() && start0.Equals(end) {
				p.Close()
			}

			advance, err := ff.font.GlyphAdvance(&sfntBuffer, index, ff.ppem, ff.hinting)
			if err == nil {
				x += fromI26_6(advance)
			}
			prevIndex = index
		}
		x = 0.0
		y += ff.Metrics().LineHeight
	}
	return p
}
