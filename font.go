package canvas

import (
	"io/ioutil"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
)

type FontStyle int

const (
	Regular FontStyle = 0
	Bold    FontStyle = 1 << iota
	Italic
)

type Font struct {
	mimetype string
	raw      []byte

	*truetype.Font
	style FontStyle
}

type Fonts struct {
	fonts map[string]Font
}

func NewFonts() *Fonts {
	return &Fonts{make(map[string]Font)}
}

func (f *Fonts) AddFont(name string, style FontStyle, path string) error {
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

type CanvasFonts struct {
	fonts *Fonts

	font      string
	fontsize  float64
	fontstyle FontStyle
	fontface  font.Face
}

func NewCanvasFonts(fonts *Fonts) *CanvasFonts {
	return &CanvasFonts{fonts, "", 0.0, Regular, nil}
}

func (f *CanvasFonts) SetFont(name string, size float64) {
	f.font = name
	f.fontsize = size
	f.fontstyle = f.fonts.fonts[name].style
	f.fontface = truetype.NewFace(f.fonts.fonts[name].Font, &truetype.Options{
		Size: size,
	})
}

func (f *CanvasFonts) LineHeight() float64 {
	return FromI26_6(f.fontface.Metrics().Height) * 0.352778
}

func (f *CanvasFonts) TextWidth(s string) float64 {
	return FromI26_6(font.MeasureString(f.fontface, s)) * 0.352778
}
