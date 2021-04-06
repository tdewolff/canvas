package canvas

import (
	"fmt"
	"image/color"
	"io/ioutil"
	"math"
	"os/exec"
	"reflect"

	"github.com/tdewolff/canvas/font"
	"github.com/tdewolff/canvas/text"
)

// FontStyle defines the font style to be used for the font.
type FontStyle int

// see FontStyle
const (
	FontRegular    FontStyle = 0 // 400
	FontItalic     FontStyle = 1
	FontExtraLight FontStyle = 2 << iota // 100
	FontLight                            // 200
	FontBook                             // 300
	FontMedium                           // 500
	FontSemibold                         // 600
	FontBold                             // 700
	FontBlack                            // 800
	FontExtraBlack                       // 900
)

// FontVariant defines the font variant to be used for the font, such as subscript or smallcaps.
type FontVariant int

// see FontVariant
const (
	FontNormal FontVariant = 2 << iota
	FontSubscript
	FontSuperscript
	FontSmallcaps
)

// Font defines a font of type TTF or OTF which which a FontFace can be generated for use in text drawing operations.
type Font struct {
	*font.SFNT
	name        string
	subsetIDs   []uint16          // old glyphIDs for increasing new glyphIDs
	subsetIDMap map[uint16]uint16 // old to new glyphID
	shaper      text.Shaper
	variations  string
	features    string
}

func parseFont(name string, b []byte, index int) (*Font, error) {
	SFNT, err := font.ParseFont(b, index)
	if err != nil {
		return nil, err
	}

	shaper, err := text.NewShaperSFNT(SFNT)
	if err != nil {
		return nil, err
	}

	font := &Font{
		SFNT:        SFNT,
		name:        name,
		subsetIDs:   []uint16{0}, // .notdef should always be at zero
		subsetIDMap: map[uint16]uint16{0: 0},
		shaper:      shaper,
	}
	return font, nil
}

func (f *Font) Destroy() {
	f.shaper.Destroy()
}

// Name returns the name of the font.
func (f *Font) Name() string {
	return f.name
}

func (f *Font) SubsetID(glyphID uint16) uint16 {
	if subsetGlyphID, ok := f.subsetIDMap[glyphID]; ok {
		return subsetGlyphID
	}
	subsetGlyphID := uint16(len(f.subsetIDs))
	f.subsetIDs = append(f.subsetIDs, glyphID)
	f.subsetIDMap[glyphID] = subsetGlyphID
	return subsetGlyphID
}

func (f *Font) SubsetIDs() []uint16 {
	return f.subsetIDs
}

func (f *Font) SetVariations(variations string) {
	f.variations = variations
}

func (f *Font) SetFeatures(features string) {
	f.features = features
}

// FontFamily contains a family of fonts (bold, italic, ...). Selecting an italic style will pick the native italic font or use faux italic if not present.
type FontFamily struct {
	name  string
	fonts map[FontStyle]*Font
}

// NewFontFamily returns a new FontFamily.
func NewFontFamily(name string) *FontFamily {
	return &FontFamily{
		name:  name,
		fonts: map[FontStyle]*Font{},
	}
}

func (family *FontFamily) Destroy() {
	for _, font := range family.fonts {
		font.Destroy()
	}
}

func (family *FontFamily) SetVariations(variations string) {
	for _, font := range family.fonts {
		font.SetVariations(variations)
	}
}

func (family *FontFamily) SetFeatures(features string) {
	for _, font := range family.fonts {
		font.SetFeatures(features)
	}
}

// LoadLocalFont loads a font from the system fonts location.
func (family *FontFamily) LoadLocalFont(name string, style FontStyle) error {
	match := name
	if style&FontItalic == FontItalic {
		match += ":italic"
	}
	if style&FontExtraLight == FontExtraLight {
		match += ":weight=40"
	} else if style&FontLight == FontLight {
		match += ":weight=50"
	} else if style&FontBook == FontBook {
		match += ":weight=75"
	} else if style&FontMedium == FontMedium {
		match += ":weight=100"
	} else if style&FontSemibold == FontSemibold {
		match += ":weight=180"
	} else if style&FontBold == FontBold {
		match += ":weight=200"
	} else if style&FontBlack == FontBlack {
		match += ":weight=205"
	} else if style&FontExtraBlack == FontExtraBlack {
		match += ":weight=210"
	}
	b, err := exec.Command("fc-match", "--format=%{file}", match).Output()
	if err != nil {
		return err
	}
	return family.LoadFontFile(string(b), style)
}

// LoadFontFile loads a font from a file.
func (family *FontFamily) LoadFontFile(filename string, style FontStyle) error {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to load font file '%s': %w", filename, err)
	}
	return family.LoadFont(b, 0, style)
}

// LoadFontCollection loads a font from a collection file and uses the font at the specified index.
func (family *FontFamily) LoadFontCollection(filename string, index int, style FontStyle) error {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to load font file '%s': %w", filename, err)
	}
	return family.LoadFont(b, index, style)
}

// LoadFont loads a font from memory.
func (family *FontFamily) LoadFont(b []byte, index int, style FontStyle) error {
	font, err := parseFont(family.name, b, index)
	if err != nil {
		return err
	}
	family.fonts[style] = font
	return nil
}

// Face gets the font face given by the font size (in pt).
func (family *FontFamily) Face(size float64, col color.Color, style FontStyle, variant FontVariant, deco ...FontDecorator) *FontFace {
	face := &FontFace{}
	face.Font = family.fonts[style]
	face.Size = size * mmPerPt
	face.Style = style
	face.Variant = variant

	r, g, b, a := col.RGBA()
	face.Color = color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
	face.Deco = deco

	if face.Font == nil {
		face.Font = family.fonts[FontRegular]
		if face.Font == nil {
			panic("requested font style not found")
		}
		if style&FontItalic != 0 {
			face.FauxItalic = 0.3 // TODO: use post table
		}
		if style&FontExtraLight == FontExtraLight {
			face.FauxBold = -0.02 * face.Size
		} else if style&FontLight == FontLight {
			face.FauxBold = -0.01 * face.Size
		} else if style&FontBook == FontBook {
			face.FauxBold = -0.005 * face.Size
		} else if style&FontMedium == FontMedium {
			face.FauxBold = 0.005 * face.Size
		} else if style&FontSemibold == FontSemibold {
			face.FauxBold = 0.01 * face.Size
		} else if style&FontBold == FontBold {
			face.FauxBold = 0.02 * face.Size
		} else if style&FontBlack == FontBlack {
			face.FauxBold = 0.03 * face.Size
		} else if style&FontExtraBlack == FontExtraBlack {
			face.FauxBold = 0.04 * face.Size
		}
	}

	units := float64(face.Font.Head.UnitsPerEm)
	if variant&FontSubscript != 0 {
		face.XScale = float64(face.Font.OS2.YSubscriptXSize) / units
		face.YScale = float64(face.Font.OS2.YSubscriptYSize) / units
		face.XOffset = int32(face.Font.OS2.YSubscriptXOffset)
		face.YOffset = int32(face.Font.OS2.YSubscriptYOffset)
	} else if variant&FontSuperscript != 0 {
		face.XScale = float64(face.Font.OS2.YSuperscriptXSize) / units
		face.YScale = float64(face.Font.OS2.YSuperscriptYSize) / units
		face.XOffset = int32(face.Font.OS2.YSuperscriptXOffset)
		face.YOffset = -int32(face.Font.OS2.YSuperscriptYOffset)
	} else {
		face.XScale = 1.0
		face.YScale = 1.0
	}
	return face
}

// FontFace defines a font face from a given font. It allows setting the font size, its color, faux styles and font decorations.
type FontFace struct {
	Font *Font

	Size    float64 // in pt
	Style   FontStyle
	Variant FontVariant

	Color color.RGBA
	Deco  []FontDecorator

	// faux styles for bold, italic, and sub- and superscript
	FauxBold, FauxItalic float64
	XScale, YScale       float64
	XOffset, YOffset     int32

	Language  string
	Script    text.Script
	Direction text.Direction

	// letter spacing
	// stroke and stroke color
	// line height
	// shadow
}

// Equals returns true when two font face are equal. In particular this allows two adjacent text spans that use the same decoration to allow the decoration to span both elements instead of two separately.
func (face *FontFace) Equals(other *FontFace) bool {
	return reflect.DeepEqual(face, other)
	//return face.Font == other.Font && face.Size == other.Size && face.Style == other.Style && face.Variant == other.Variant && face.Color == other.Color && reflect.DeepEqual(face.Deco, other.Deco) && face.Language == other.Language && face.Script == other.Script && face.Direction == other.Direction
}

// Name returns the name of the underlying font
func (face *FontFace) Name() string {
	return face.Font.name
}

func (face *FontFace) HasDecoration() bool {
	return 0 < len(face.Deco)
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

	XMin, YMin float64
	XMax, YMax float64
}

// Metrics returns the font metrics. See https://developer.apple.com/library/archive/documentation/TextFonts/Conceptual/CocoaTextArchitecture/Art/glyph_metrics_2x.png for an explanation of the different metrics.
func (face *FontFace) Metrics() FontMetrics {
	sfnt := face.Font.SFNT
	fx := face.Size * face.XScale / float64(sfnt.Head.UnitsPerEm)
	fy := face.Size * face.YScale / float64(sfnt.Head.UnitsPerEm)
	return FontMetrics{
		LineHeight: fy * float64(sfnt.Hhea.Ascender-sfnt.Hhea.Descender+sfnt.Hhea.LineGap),
		Ascent:     fy * float64(sfnt.Hhea.Ascender),
		Descent:    fy * float64(-sfnt.Hhea.Descender),
		LineGap:    fy * float64(sfnt.Hhea.LineGap),
		XHeight:    fy * float64(sfnt.OS2.SxHeight),
		CapHeight:  fy * float64(sfnt.OS2.SCapHeight),
		XMin:       fx * float64(sfnt.Head.XMin),
		YMin:       fy * float64(sfnt.Head.YMin),
		XMax:       fx * float64(sfnt.Head.XMax),
		YMax:       fy * float64(sfnt.Head.YMax),
	}
}

func (face *FontFace) PPEM(resolution Resolution) uint16 {
	// ppem is for hinting purposes only, this does not influence glyph advances
	return uint16(resolution.DPMM() * face.Size * math.Min(face.XScale, face.YScale))
}

// Kerning returns the eventual kerning between two runes in mm (ie. the adjustment on the advance).
func (face *FontFace) Kerning(left, right rune) float64 {
	sfnt := face.Font.SFNT
	fx := face.Size * face.XScale / float64(sfnt.Head.UnitsPerEm)
	return fx * float64(sfnt.Kerning(sfnt.GlyphIndex(left), sfnt.GlyphIndex(right)))
}

// TextWidth returns the width of a given string in mm.
func (face *FontFace) TextWidth(s string) float64 {
	ppem := face.PPEM(DefaultResolution)
	glyphs := face.Font.shaper.Shape(s, ppem, face.Direction, face.Script, face.Language, face.Font.features, face.Font.variations)
	return face.textWidth(glyphs)
}

func (face *FontFace) textWidth(glyphs []text.Glyph) float64 {
	sfnt := face.Font.SFNT
	fx := face.Size * face.XScale / float64(sfnt.Head.UnitsPerEm)

	w := int32(0)
	for i, glyph := range glyphs {
		if i != 0 {
			w += int32(sfnt.Kerning(glyphs[i-1].ID, glyph.ID))
		}
		w += int32(sfnt.GlyphAdvance(glyph.ID))
	}
	return fx * float64(w)
}

// Decorate will return a path from the decorations specified in the FontFace over a given width in mm.
func (face *FontFace) Decorate(width float64) *Path {
	p := &Path{}
	if face.Deco != nil {
		for _, deco := range face.Deco {
			p = p.Append(deco.Decorate(face, width))
		}
	}
	return p
}

func (face *FontFace) ToPath(s string, resolution Resolution) (*Path, float64, error) {
	ppem := face.PPEM(resolution)
	glyphs := face.Font.shaper.Shape(s, ppem, face.Direction, face.Script, face.Language, face.Font.features, face.Font.variations)
	return face.toPath(glyphs, ppem)
}

func (face *FontFace) toPath(glyphs []text.Glyph, ppem uint16) (*Path, float64, error) {
	sfnt := face.Font.SFNT
	fx := face.Size * face.XScale / float64(sfnt.Head.UnitsPerEm)
	fy := face.Size * face.YScale / float64(sfnt.Head.UnitsPerEm)

	p := &Path{}
	x, y := face.XOffset, face.YOffset
	for _, glyph := range glyphs {
		err := face.Font.GlyphToPath(p, glyph.ID, ppem, x+glyph.XOffset, y+glyph.YOffset, fx, fy, font.NoHinting)
		if err != nil {
			return p, 0.0, err
		}
		x += glyph.XAdvance
		y += glyph.YAdvance
	}

	if face.FauxBold != 0.0 {
		p = p.Offset(face.FauxBold, NonZero)
	}
	if face.FauxItalic != 0.0 {
		p = p.Transform(Identity.Shear(face.FauxItalic, 0.0))
	}
	return p, fx * float64(x), nil
}

func (face *FontFace) Boldness() int {
	boldness := 400
	if face.Style&FontExtraLight == FontExtraLight {
		boldness = 100
	} else if face.Style&FontLight == FontLight {
		boldness = 200
	} else if face.Style&FontBook == FontBook {
		boldness = 300
	} else if face.Style&FontMedium == FontMedium {
		boldness = 500
	} else if face.Style&FontSemibold == FontSemibold {
		boldness = 600
	} else if face.Style&FontBold == FontBold {
		boldness = 700
	} else if face.Style&FontBlack == FontBlack {
		boldness = 800
	} else if face.Style&FontExtraBlack == FontExtraBlack {
		boldness = 900
	}
	if face.Variant&FontSubscript != 0 || face.Variant&FontSuperscript != 0 {
		boldness += 300
		if 1000 < boldness {
			boldness = 1000
		}
	}
	return boldness
}

////////////////////////////////////////////////////////////////

// FontDecorator is an interface that returns a path given a font face and a width in mm.
type FontDecorator interface {
	Decorate(*FontFace, float64) *Path
}

const underlineDistance = 0.15
const underlineThickness = 0.075

// FontUnderline is a font decoration that draws a line under the text at the base line.
var FontUnderline FontDecorator = underline{}

type underline struct{}

func (underline) Decorate(face *FontFace, w float64) *Path {
	// TODO: use post table
	r := face.Size * underlineThickness
	y := -face.Size * underlineDistance

	p := &Path{}
	p.MoveTo(0.0, y)
	p.LineTo(w, y)
	return p.Stroke(r, ButtCap, BevelJoin)
}

// FontOverline is a font decoration that draws a line over the text at the X-Height line.
var FontOverline FontDecorator = overline{}

type overline struct{}

func (overline) Decorate(face *FontFace, w float64) *Path {
	r := face.Size * underlineThickness
	y := face.Metrics().XHeight + face.Size*underlineDistance

	dx := face.FauxItalic * y
	w += face.FauxItalic * y

	p := &Path{}
	p.MoveTo(dx, y)
	p.LineTo(w, y)
	return p.Stroke(r, ButtCap, BevelJoin)
}

// FontStrikethrough is a font decoration that draws a line through the text in the middle between the base and X-Height line.
var FontStrikethrough FontDecorator = strikethrough{}

type strikethrough struct{}

func (strikethrough) Decorate(face *FontFace, w float64) *Path {
	// TODO: use OS/2 table
	r := face.Size * underlineThickness
	y := face.Metrics().XHeight / 2.0

	dx := face.FauxItalic * y
	w += face.FauxItalic * y

	p := &Path{}
	p.MoveTo(dx, y)
	p.LineTo(w, y)
	return p.Stroke(r, ButtCap, BevelJoin)
}

// FontDoubleUnderline is a font decoration that draws two lines at the base line.
var FontDoubleUnderline FontDecorator = doubleUnderline{}

type doubleUnderline struct{}

func (doubleUnderline) Decorate(face *FontFace, w float64) *Path {
	r := face.Size * underlineThickness
	y := -face.Size * underlineDistance * 0.75

	p := &Path{}
	p.MoveTo(0.0, y)
	p.LineTo(w, y)
	p.MoveTo(0.0, y-r*2.0)
	p.LineTo(w, y-r*2.0)
	return p.Stroke(r, ButtCap, BevelJoin)
}

// FontDottedUnderline is a font decoration that draws a dotted line at the base line.
var FontDottedUnderline FontDecorator = dottedUnderline{}

type dottedUnderline struct{}

func (dottedUnderline) Decorate(face *FontFace, w float64) *Path {
	r := face.Size * underlineThickness * 0.8
	w -= r

	y := -face.Size * underlineDistance
	d := 15.0 * underlineThickness
	n := int((w-r)/d) + 1
	d = (w - r) / float64(n-1)

	p := &Path{}
	for i := 0; i < n; i++ {
		p = p.Append(Circle(r).Translate(r+float64(i)*d, y))
	}
	return p
}

// FontDashedUnderline is a font decoration that draws a dashed line at the base line.
var FontDashedUnderline FontDecorator = dashedUnderline{}

type dashedUnderline struct{}

func (dashedUnderline) Decorate(face *FontFace, w float64) *Path {
	r := face.Size * underlineThickness
	y := -face.Size * underlineDistance
	d := 12.0 * underlineThickness
	n := int(w / (2.0 * d))
	d = w / float64(2*n-1)

	p := &Path{}
	p.MoveTo(0.0, y)
	p.LineTo(w, y)
	p = p.Dash(d).Stroke(r, ButtCap, BevelJoin)
	return p
}

// FontSineUnderline is a font decoration that draws a wavy sine path at the base line.
var FontSineUnderline FontDecorator = sineUnderline{}

type sineUnderline struct{}

func (sineUnderline) Decorate(face *FontFace, w float64) *Path {
	r := face.Size * underlineThickness
	w -= r

	dh := -face.Size * 0.15
	y := -face.Size * underlineDistance
	d := 12.0 * underlineThickness
	n := int(0.5 + w/d)
	d = (w - r) / float64(n)

	dx := r
	p := &Path{}
	p.MoveTo(dx, y)
	for i := 0; i < n; i++ {
		if i%2 == 0 {
			p.CubeTo(dx+d*0.3642, y, dx+d*0.6358, y+dh, dx+d, y+dh)
		} else {
			p.CubeTo(dx+d*0.3642, y+dh, dx+d*0.6358, y, dx+d, y)
		}
		dx += d
	}
	return p.Stroke(r, RoundCap, RoundJoin)
}

// FontSawtoothUnderline is a font decoration that draws a wavy sawtooth path at the base line.
var FontSawtoothUnderline FontDecorator = sawtoothUnderline{}

type sawtoothUnderline struct{}

func (sawtoothUnderline) Decorate(face *FontFace, w float64) *Path {
	r := face.Size * underlineThickness
	dx := 0.707 * r
	w -= 2.0 * dx

	dh := -face.Size * 0.15
	y := -face.Size * underlineDistance
	d := 8.0 * underlineThickness
	n := int(0.5 + w/d)
	d = w / float64(n)

	p := &Path{}
	p.MoveTo(dx, y)
	for i := 0; i < n; i++ {
		if i%2 == 0 {
			p.LineTo(dx+d, y+dh)
		} else {
			p.LineTo(dx+d, y)
		}
		dx += d
	}
	return p.Stroke(r, ButtCap, MiterJoin)
}
