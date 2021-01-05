package canvas

import (
	"fmt"
	"sort"
	"unicode"

	canvasFont "github.com/tdewolff/canvas/font"
	canvasText "github.com/tdewolff/canvas/text"
	"golang.org/x/image/font/sfnt"
)

func StringPath(sfnt *canvasFont.SFNT, text string, size float64) (*Path, error) {
	fontShaping, err := canvasText.NewFont(sfnt.Data, 0)
	if err != nil {
		return nil, err
	}
	defer fontShaping.Destroy()

	f := size / float64(sfnt.Head.UnitsPerEm)

	p := &Path{}
	var x, y int32
	glyphs := fontShaping.Shape(text, size, canvasText.LeftToRight, canvasText.Latin)
	for _, glyph := range glyphs {
		path, err := GlyphPath(sfnt, glyph.ID, size, float64(x+glyph.XOffset)*f, float64(y+glyph.YOffset)*f)
		if err != nil {
			return p, err
		}
		if path != nil {
			p = p.Append(path)
		}
		x += glyph.XAdvance
		y += glyph.YAdvance
	}
	return p, nil
}

func GlyphPath(sfnt *canvasFont.SFNT, glyphID uint16, size, x, y float64) (*Path, error) {
	if sfnt.IsTrueType {
		contour, err := sfnt.GlyphContour(glyphID)
		if err != nil || contour == nil {
			return nil, err
		}

		f := size / float64(sfnt.Head.UnitsPerEm)
		p := &Path{}
		var i uint16
		for _, endPoint := range contour.EndPoints {
			j := i
			first := true
			firstOff := false
			prevOff := false
			for ; i <= endPoint; i++ {
				if first {
					if contour.OnCurve[i] {
						p.MoveTo(x+float64(contour.XCoordinates[i])*f, y+float64(contour.YCoordinates[i])*f)
						first = false
					} else if !prevOff {
						// first point is off
						firstOff = true
						prevOff = true
					} else {
						// first and second point are off
						xMid := float64(contour.XCoordinates[i-1]+contour.XCoordinates[i]) / 2.0
						yMid := float64(contour.YCoordinates[i-1]+contour.YCoordinates[i]) / 2.0
						p.MoveTo(x+xMid*f, y+yMid*f)
					}
				} else if !prevOff {
					if contour.OnCurve[i] {
						p.LineTo(x+float64(contour.XCoordinates[i])*f, y+float64(contour.YCoordinates[i])*f)
					} else {
						prevOff = true
					}
				} else {
					if contour.OnCurve[i] {
						p.QuadTo(x+float64(contour.XCoordinates[i-1])*f, y+float64(contour.YCoordinates[i-1])*f, x+float64(contour.XCoordinates[i])*f, y+float64(contour.YCoordinates[i])*f)
						prevOff = false
					} else {
						xMid := float64(contour.XCoordinates[i-1]+contour.XCoordinates[i]) / 2.0
						yMid := float64(contour.YCoordinates[i-1]+contour.YCoordinates[i]) / 2.0
						p.QuadTo(x+float64(contour.XCoordinates[i-1])*f, y+float64(contour.YCoordinates[i-1])*f, x+xMid*f, y+yMid*f)
					}
				}
			}
			start := p.StartPos()
			if firstOff {
				if prevOff {
					xMid := float64(contour.XCoordinates[i-1]+contour.XCoordinates[j]) / 2.0
					yMid := float64(contour.YCoordinates[i-1]+contour.YCoordinates[j]) / 2.0
					p.QuadTo(x+xMid*f, y+yMid*f, start.X, start.Y)
				} else {
					p.QuadTo(x+float64(contour.XCoordinates[i-1])*f, y+float64(contour.YCoordinates[i-1])*f, start.X, start.Y)
				}
			} else if prevOff {
				p.QuadTo(x+float64(contour.XCoordinates[i-1])*f, y+float64(contour.YCoordinates[i-1])*f, start.X, start.Y)
			}
			p.Close()
		}
		return p, nil
	} else {
		return nil, fmt.Errorf("CFF not supported")
	}
}

// Font defines a font of type TTF or OTF which which a FontFace can be generated for use in text drawing operations.
type Font struct {
	name    string
	sfnt    *sfnt.Font
	SFNT    *canvasFont.SFNT
	usedIDs map[uint16]bool
}

func parseFont(name string, b []byte) (*Font, error) {
	sfntFont, err := canvasFont.ParseFont(b)
	if err != nil {
		return nil, err
	}

	SFNT, err := canvasFont.ParseSFNT(b)
	if err != nil {
		return nil, err
	}

	font := &Font{
		name:    name,
		sfnt:    (*sfnt.Font)(sfntFont),
		SFNT:    SFNT,
		usedIDs: map[uint16]bool{},
	}
	return font, nil
}

// Name returns the name of the font.
func (font *Font) Name() string {
	return font.name
}

// Kerning returns the horizontal adjustment for the rune pair. A positive kern means to move the glyphs further apart.
// Returns 0 if there is an error.
func (font *Font) Kerning(left, right rune, ppem float64) float64 {
	f := ppem / float64(font.SFNT.Head.UnitsPerEm)
	return f * float64(font.SFNT.Kerning(font.SFNT.GlyphIndex(left), font.SFNT.GlyphIndex(right)))
}

// FontMetrics contains a number of metrics that define a font face.
// See https://developer.apple.com/library/archive/documentation/TextFonts/Conceptual/CocoaTextArchitecture/Art/glyph_metrics_2x.png for an explanation of the different metrics.
type FontMetrics struct {
	LineHeight float64
	Ascent     float64
	Descent    float64
	LineGap    float64
	XHeight    float64
	CapHeight  float64

	XMin, XMax float64
	YMin, YMax float64
}

func (font *Font) Metrics(ppem float64) FontMetrics {
	f := ppem / float64(font.SFNT.Head.UnitsPerEm)
	return FontMetrics{
		LineHeight: f * float64(font.SFNT.Hhea.Ascender-font.SFNT.Hhea.Descender+font.SFNT.Hhea.LineGap),
		Ascent:     f * float64(font.SFNT.Hhea.Ascender),
		Descent:    f * float64(-font.SFNT.Hhea.Descender),
		LineGap:    f * float64(font.SFNT.Hhea.LineGap),
		XHeight:    f * float64(font.SFNT.OS2.SxHeight),
		CapHeight:  f * float64(font.SFNT.OS2.SCapHeight),
		XMin:       f * float64(font.SFNT.Head.XMin),
		XMax:       f * float64(font.SFNT.Head.XMax),
		YMin:       f * float64(font.SFNT.Head.YMin),
		YMax:       f * float64(font.SFNT.Head.YMax),
	}
}

func (font *Font) Widths(indices []uint16, ppem float64) []float64 {
	f := ppem / float64(font.SFNT.Head.UnitsPerEm)
	widths := []float64{}
	for i := uint16(0); i < font.SFNT.Maxp.NumGlyphs; i++ {
		widths = append(widths, f*float64(font.SFNT.GlyphAdvance(i)))
	}
	return widths
}

func (font *Font) IndicesOf(s string) []uint16 {
	rs := []rune(s)
	indices := make([]uint16, len(rs))
	for i, r := range rs {
		index := font.SFNT.GlyphIndex(r)
		indices[i] = uint16(index)
		font.usedIDs[index] = true
	}
	return indices
}

func (font *Font) UsedIndices() []uint16 {
	glyphIDs := make([]uint16, len(font.usedIDs))
	for glyphID, _ := range font.usedIDs {
		glyphIDs = append(glyphIDs, glyphID)
	}
	sort.Sort(canvasFont.GlyphIDList(glyphIDs))
	return glyphIDs
}

func isWordBoundary(r rune) bool {
	return r == 0 || isSpace(r) || isPunct(r)
}

func isSpace(r rune) bool {
	return unicode.IsSpace(r)
}

func isPunct(r rune) bool {
	for _, punct := range "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~" {
		if r == punct {
			return true
		}
	}
	return false
}
