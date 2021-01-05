package canvas

import (
	"fmt"

	canvasFont "github.com/tdewolff/canvas/font"
	canvasText "github.com/tdewolff/canvas/text"
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
		if err != nil {
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
	SFNT    *canvasFont.SFNT
	usedIDs map[uint16]bool
}

func parseFont(name string, b []byte, index int) (*Font, error) {
	SFNT, err := canvasFont.ParseFont(b, index)
	if err != nil {
		return nil, err
	}

	font := &Font{
		name:    name,
		SFNT:    SFNT,
		usedIDs: map[uint16]bool{},
	}
	return font, nil
}

// Name returns the name of the font.
func (font *Font) Name() string {
	return font.name
}

func (font *Font) Use(glyphID uint16) {
	font.usedIDs[glyphID] = true
}

func (font *Font) UsedIndices() []uint16 {
	glyphIDs := make([]uint16, len(font.usedIDs))
	for glyphID, _ := range font.usedIDs {
		glyphIDs = append(glyphIDs, glyphID)
	}
	return glyphIDs
}
