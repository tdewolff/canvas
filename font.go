package canvas

import (
	"fmt"
	"io/ioutil"
	"math"

	"github.com/flopp/go-findfont"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

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

func (f *Fonts) AddLocalFont(name string, style FontStyle) error {
	fontPath, err := findfont.Find(name)
	if err != nil {
		return err
	}
	return f.AddFontFile(name, style, fontPath)
}

func (f *Fonts) AddFontFile(name string, style FontStyle, path string) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return f.AddFont(name, style, b)
}

func (f *Fonts) AddFont(name string, style FontStyle, b []byte) error {
	ttf, err := truetype.Parse(b)
	if err != nil {
		return err
	}
	f.fonts[name] = Font{
		mimetype: "font/truetype",
		raw:      b,
		faces:    map[float64]font.Face{},
		font:     ttf,
		name:     name,
		style:    style,
		dpi:      f.dpi,
	}
	return nil
}

func (f *Fonts) Font(name string) (Font, error) {
	if _, ok := f.fonts[name]; !ok {
		return Font{}, ErrNotFound
	}
	return f.fonts[name], nil
}

type Font struct {
	mimetype string
	raw      []byte
	faces    map[float64]font.Face

	font  *truetype.Font
	name  string
	style FontStyle
	dpi   float64
}

// Get gets the font face associated with the give font name and font size.
// Font size is in mm.
func (f *Font) Face(size float64) FontFace {
	face, ok := f.faces[size]
	if !ok {
		face = truetype.NewFace(f.font, &truetype.Options{
			Size: size / 0.352778, // to pt
			DPI:  f.dpi,
		})
		f.faces[size] = face
	}

	return FontFace{
		font:  f.font,
		name:  f.name,
		size:  size,
		style: f.style,
		face:  face,
	}
}

type FontFace struct {
	font  *truetype.Font
	name  string
	size  float64
	style FontStyle

	face font.Face
}

// LineHeight returns line height in mm, same as font size.
func (f FontFace) LineHeight() float64 {
	return fromI26_6(f.face.Metrics().Height) * 0.352778
}

func (f FontFace) Ascent() float64 {
	return fromI26_6(f.face.Metrics().Ascent) * 0.352778
}

func (f FontFace) Descent() float64 {
	return fromI26_6(f.face.Metrics().Descent) * 0.352778
}

// TextWidth returns the width of a given string in mm.
func (f FontFace) TextWidth(s string) float64 {
	return fromI26_6(font.MeasureString(f.face, s)) * 0.352778
}

func (f FontFace) BBox(s string) (float64, float64) {
	w := 0.0
	newlines := 1

	i := 0
	for j := 0; j < len(s); j++ {
		d := 0
		c := s[j]
		if c == '\n' {
			d = 1
		} else if c == '\r' && j+1 < len(s) && s[j+1] == '\n' {
			d = 2
		} else if c == 0xE2 && j+2 < len(s) && s[j+1] == 0x80 && (s[j+2] == 0xA8 || s[j+2] == 0xA9) {
			d = 3
		}

		if d > 0 {
			w = math.Max(w, f.TextWidth(s[i:j]))
			newlines++
			i = j + d
			j += d - 1
		}
	}
	if i < len(s) {
		w = math.Max(w, f.TextWidth(s[i:]))
	}
	return w, float64(newlines) * f.LineHeight()
}

// mostly takes from github.com/golang/freetype/truetype/face.go
func (f FontFace) glyphToPath(r rune) (*Path, float64) {
	glyphBuf := truetype.GlyphBuf{}
	glyphBuf.Load(f.font, fixed.Int26_6((f.size*64)+0.5), f.font.Index(r), font.HintingNone)

	path := &Path{}
	e0 := 0
	for _, e1 := range glyphBuf.Ends {
		ps := glyphBuf.Points[e0:e1]
		if len(ps) == 0 {
			continue
		}

		var others []truetype.Point
		start, on := fromTTPoint(ps[0])
		if on {
			others = ps[1:]
		} else {
			last, on := fromTTPoint(ps[len(ps)-1])
			if on {
				start = last
				others = ps[:len(ps)-1]
			} else {
				start = Point{(start.X + last.X) / 2.0, (start.Y + last.Y) / 2.0}
				others = ps
			}
		}
		path.MoveTo(start.X, start.Y)
		q0, on0 := start, true
		for _, p := range others {
			q, on := fromTTPoint(p)
			if on {
				if on0 {
					path.LineTo(q.X, q.Y)
				} else {
					path.QuadTo(q0.X, q0.Y, q.X, q.Y)
				}
			} else {
				if on0 {
					// No-op
				} else {
					mid := Point{(q0.X + q.X) / 2.0, (q0.Y + q.Y) / 2.0}
					path.QuadTo(q0.X, q0.Y, mid.X, mid.Y)
				}
			}
			q0, on0 = q, on
		}
		if !on0 {
			path.QuadTo(q0.X, q0.Y, start.X, start.Y)
		}
		path.Close()

		e0 = e1
	}
	return path, fromI26_6(glyphBuf.AdvanceWidth)
}

func (f FontFace) ToPath(s string) *Path {
	x := 0.0
	p := &Path{}
	for _, r := range s {
		pRune, advance := f.glyphToPath(r)
		pRune = pRune.Translate(x, 0.0)
		p.Append(pRune)
		x += advance
	}
	return p
}
