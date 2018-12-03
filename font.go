package canvas

import (
	"fmt"
	"io/ioutil"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

var ErrNotFound = fmt.Errorf("Font name not found")

type FontStyle int

const (
	Regular FontStyle = 0
	Bold    FontStyle = 1 << iota
	Italic
)

type Font struct {
	mimetype string
	raw      []byte

	font  *truetype.Font
	style FontStyle
}

type Fonts struct {
	fonts map[string]Font
	cache map[string]map[float64]font.Face
}

func NewFonts() *Fonts {
	return &Fonts{
		fonts: make(map[string]Font),
		cache: make(map[string]map[float64]font.Face),
	}
}

func (f *Fonts) Add(name string, style FontStyle, path string) error {
	fontBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	font, err := truetype.Parse(fontBytes)
	if err != nil {
		return err
	}
	f.fonts[name] = Font{"font/truetype", fontBytes, font, style}
	return nil
}

// Get gets the font face associated with the give font name and font size.
// Font size is in mm.
func (f *Fonts) Get(name string, size float64) (FontFace, error) {
	if _, ok := f.fonts[name]; !ok {
		return FontFace{}, ErrNotFound
	}

	if f.cache[name] == nil {
		f.cache[name] = make(map[float64]font.Face)
	}
	face, ok := f.cache[name][size]
	if !ok {
		face = truetype.NewFace(f.fonts[name].font, &truetype.Options{
			Size: size / 0.352778, // to pt
		})
		f.cache[name][size] = face
	}

	return FontFace{
		font:  f.fonts[name].font,
		face:  face,
		name:  name,
		size:  size,
		style: f.fonts[name].style,
	}, nil
}

type FontFace struct {
	font  *truetype.Font
	face  font.Face
	name  string
	size  float64
	style FontStyle
}

// LineHeight returns line height in mm.
func (f FontFace) LineHeight() float64 {
	return FromI26_6(f.face.Metrics().Height) * 0.352778
}

// TextWidth returns the width of a given string in mm.
func (f FontFace) TextWidth(s string) float64 {
	return FromI26_6(font.MeasureString(f.face, s)) * 0.352778
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
		start, on := FromTTPoint(ps[0])
		if on {
			others = ps[1:]
		} else {
			last, on := FromTTPoint(ps[len(ps)-1])
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
			q, on := FromTTPoint(p)
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
	return path, FromI26_6(glyphBuf.AdvanceWidth)
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
