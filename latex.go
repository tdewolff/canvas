// +build !latex

package canvas

import (
	"fmt"
	"unicode/utf8"

	"github.com/go-fonts/dejavu/dejavusans"
	"github.com/go-fonts/dejavu/dejavusansbold"
	"github.com/go-fonts/dejavu/dejavusansoblique"
	"github.com/go-fonts/latin-modern/lmroman12bold"
	"github.com/go-fonts/latin-modern/lmroman12italic"
	"github.com/go-fonts/latin-modern/lmroman12regular"
	"github.com/go-fonts/liberation/liberationsansbold"
	"github.com/go-fonts/liberation/liberationsansitalic"
	"github.com/go-fonts/liberation/liberationsansregular"
	"github.com/go-fonts/stix/stix2textbold"
	"github.com/go-fonts/stix/stix2textitalic"
	"github.com/go-fonts/stix/stix2textregular"
	"github.com/go-latex/latex/font"
	"github.com/go-latex/latex/mtex"
	"github.com/go-latex/latex/tex"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/goregular"
)

type LaTeXFonts int

const (
	LatinModernFonts LaTeXFonts = iota
	STIXFonts
	LiberationFonts
	DejaVuFonts
	GoFonts
)

func ParseLaTeX(s string, fontsize float64) (*Path, error) {
	fonts := LatinModernFonts

	backend, err := newBackend(fonts)
	if err != nil {
		return nil, err
	}

	box, err := mtex.Parse(s, fontsize, 1.0, backend)
	if err != nil {
		return nil, fmt.Errorf("could not parse expression: %w", err)
	}
	backend.SetHeight(box.Height())

	var sh tex.Ship
	sh.Call(0, 0, box.(tex.Tree))
	return backend.p, nil
}

type backend struct {
	p     *Path
	h     float64
	fonts map[string]*FontFamily
}

func newBackend(latexFonts LaTeXFonts) (*backend, error) {
	var err error
	fonts := make(map[string]*FontFamily)
	switch latexFonts {
	case STIXFonts:
		f := NewFontFamily("stix")
		if err = f.LoadFont(stix2textregular.TTF, 0, FontRegular); err != nil {
			return nil, err
		}
		if err = f.LoadFont(stix2textitalic.TTF, 0, FontItalic); err != nil {
			return nil, err
		}
		if err = f.LoadFont(stix2textbold.TTF, 0, FontBold); err != nil {
			return nil, err
		}
		fonts["default"] = f
	case LiberationFonts:
		f := NewFontFamily("liberation")
		if err = f.LoadFont(liberationsansregular.TTF, 0, FontRegular); err != nil {
			return nil, err
		}
		if err = f.LoadFont(liberationsansitalic.TTF, 0, FontItalic); err != nil {
			return nil, err
		}
		if err = f.LoadFont(liberationsansbold.TTF, 0, FontBold); err != nil {
			return nil, err
		}
		fonts["default"] = f
	case DejaVuFonts:
		f := NewFontFamily("dejavu")
		if err = f.LoadFont(dejavusans.TTF, 0, FontRegular); err != nil {
			return nil, err
		}
		if err = f.LoadFont(dejavusansoblique.TTF, 0, FontItalic); err != nil {
			return nil, err
		}
		if err = f.LoadFont(dejavusansbold.TTF, 0, FontBold); err != nil {
			return nil, err
		}
		fonts["default"] = f
	case GoFonts:
		f := NewFontFamily("go")
		if err = f.LoadFont(goregular.TTF, 0, FontRegular); err != nil {
			return nil, err
		}
		if err = f.LoadFont(goitalic.TTF, 0, FontItalic); err != nil {
			return nil, err
		}
		if err = f.LoadFont(gobold.TTF, 0, FontBold); err != nil {
			return nil, err
		}
		fonts["default"] = f
	default:
		f := NewFontFamily("lm")
		if err = f.LoadFont(lmroman12regular.TTF, 0, FontRegular); err != nil {
			return nil, err
		}
		if err = f.LoadFont(lmroman12italic.TTF, 0, FontItalic); err != nil {
			return nil, err
		}
		if err = f.LoadFont(lmroman12bold.TTF, 0, FontBold); err != nil {
			return nil, err
		}
		fonts["default"] = f
	}

	return &backend{
		p:     &Path{},
		fonts: fonts,
	}, nil
}

func (b *backend) SetHeight(h float64) {
	b.h = h
}

func (b *backend) getFace(ft font.Font) *FontFace {
	// font names: circled, default, cal, bf, regular, tt, scr, sf, frak, rm, it, bb
	// font types: default, regular, rm, it, bf
	style := FontRegular
	if ft.Type == "it" {
		style = FontItalic
	} else if ft.Type == "bf" {
		style = FontBold
	}
	return b.fonts["default"].Face(ft.Size, Black, style, FontNormal)
}

func (b *backend) RenderGlyph(x, y float64, ft font.Font, symbol string, dpi float64) {
	face := b.getFace(ft)
	p, _, err := face.ToPath(symbol)
	if err != nil {
		panic(err)
	}
	b.p = b.p.Append(p.Translate(x, b.h-y))
}

func (b *backend) RenderRectFilled(x1, y1, x2, y2 float64) {
	b.p.MoveTo(x1, b.h-y1)
	b.p.LineTo(x2, b.h-y1)
	b.p.LineTo(x2, b.h-y2)
	b.p.LineTo(x1, b.h-y2)
	b.p.Close()
}

func (b *backend) Kern(ft1 font.Font, sym1 string, ft2 font.Font, sym2 string, dpi float64) float64 {
	left, _ := utf8.DecodeRuneInString(sym1)
	right, _ := utf8.DecodeRuneInString(sym2)
	face1 := b.getFace(ft1)
	face2 := b.getFace(ft2)

	mmPerEm1 := face1.Size / float64(face1.Font.Head.UnitsPerEm)
	mmPerEm2 := face2.Size / float64(face2.Font.Head.UnitsPerEm)
	kern1 := mmPerEm1 * float64(face1.Font.Kerning(face1.Font.GlyphIndex(left), face1.Font.GlyphIndex(right)))
	kern2 := mmPerEm2 * float64(face2.Font.Kerning(face2.Font.GlyphIndex(left), face2.Font.GlyphIndex(right)))
	if kern1 < kern2 {
		return kern2
	}
	return kern1
}

func (b *backend) Metrics(symbol string, ft font.Font, dpi float64, math bool) font.Metrics {
	face := b.getFace(ft)
	r, _ := utf8.DecodeRuneInString(symbol)
	gid := face.Font.GlyphIndex(r)

	advance := face.Font.GlyphAdvance(gid)
	xmin, ymin, xmax, ymax, _ := face.Font.GlyphBounds(gid)

	mmPerEm := face.Size / float64(face.Font.Head.UnitsPerEm)
	return font.Metrics{
		Advance: mmPerEm * float64(advance),
		Width:   mmPerEm * float64(xmax-xmin),
		Height:  mmPerEm * float64(ymax-ymin),
		XMin:    mmPerEm * float64(xmin),
		YMin:    mmPerEm * float64(ymin),
		XMax:    mmPerEm * float64(xmax),
		YMax:    mmPerEm * float64(ymax),
		Iceberg: mmPerEm * float64(ymax),
		Slanted: ft.Type == "it",
	}
}

func (b *backend) XHeight(ft font.Font, dpi float64) float64 {
	face := b.getFace(ft)
	return face.Size / float64(face.Font.Head.UnitsPerEm) * face.Metrics().XHeight
}

func (b *backend) UnderlineThickness(ft font.Font, dpi float64) float64 {
	return ft.Size * mmPerPt * 0.05
}
