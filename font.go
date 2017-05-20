package canvas

import (
	"io/ioutil"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
)

type FontStyle int

const (
	Regular           = 0
	Bold    FontStyle = 1 << iota
	Italic
)

type Font struct {
	*truetype.Font
	style FontStyle
}

type Fonts struct {
	fonts map[string]Font

	font      string
	fontsize  float64
	fontstyle FontStyle
	fontface  font.Face
}

func NewFonts() *Fonts {
	return &Fonts{map[string]Font{}, "", 0.0, 0, nil}
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
	f.fonts[name] = Font{font, style}
	return nil
}

func (f *Fonts) SetFont(name string, size float64) {
	f.font = name
	f.fontsize = size
	f.fontstyle = f.fonts[name].style
	f.fontface = truetype.NewFace(f.fonts[name].Font, &truetype.Options{
		Size: size,
	})
}

func (f *Fonts) LineHeight() float64 {
	return fromI26_6(f.fontface.Metrics().Height) * 0.352778
}

func (f *Fonts) TextWidth(s string) float64 {
	return fromI26_6(font.MeasureString(f.fontface, s)) * 0.352778
}