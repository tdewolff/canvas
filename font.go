package canvas

import (
	"fmt"
	"io/ioutil"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
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
		face:  face,
		name:  name,
		size:  size,
		style: f.fonts[name].style,
	}, nil
}

type FontFace struct {
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
