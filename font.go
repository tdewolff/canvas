package canvas

import (
	"fmt"
	"image/color"
	"io/ioutil"
	"math"
	"reflect"

	"github.com/adrg/sysfont"
	"github.com/tdewolff/canvas/font"
	"github.com/tdewolff/canvas/text"
)

// FontStyle defines the font style to be used for the font. It specifies a boldness with optionally italic, e.g. FontBlack | FontItalic will specify a black boldness (a font-weight of 800 in CSS) and italic.
type FontStyle int

// see FontStyle
const (
	FontRegular    FontStyle = iota // 400
	FontExtraLight                  // 100
	FontLight                       // 200
	FontBook                        // 300
	FontMedium                      // 500
	FontSemibold                    // 600
	FontBold                        // 700
	FontBlack                       // 800
	FontExtraBlack                  // 900
	FontItalic     FontStyle = 1 << 8
)

// Weight returns the font weight (FontRegular, FontBold, ...)
func (style FontStyle) Weight() FontStyle {
	return style & 0xFF
}

// Italic returns true if italic.
func (style FontStyle) Italic() bool {
	return style&FontItalic != 0
}

// CSS returns the CSS boldness value for the font face.
func (style FontStyle) CSS() int {
	switch style.Weight() {
	case FontExtraLight:
		return 100
	case FontLight:
		return 200
	case FontBook:
		return 300
	case FontMedium:
		return 500
	case FontSemibold:
		return 600
	case FontBold:
		return 700
	case FontBlack:
		return 800
	case FontExtraBlack:
		return 900
	}
	return 400
}

// FontVariant defines the font variant to be used for the font, such as subscript or smallcaps.
type FontVariant int

// see FontVariant
const (
	FontNormal FontVariant = iota
	FontSubscript
	FontSuperscript
	FontSmallcaps
)

// Font defines an SFNT font such as TTF or OTF.
type Font struct {
	*font.SFNT
	name        string
	style       FontStyle
	subsetIDs   []uint16          // old glyphIDs for increasing new glyphIDs
	subsetIDMap map[uint16]uint16 // old to new glyphID
	shaper      text.Shaper
	variations  string
	features    string
}

func parseFont(name string, style FontStyle, b []byte, index int) (*Font, error) {
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
		style:       style,
		subsetIDs:   []uint16{0}, // .notdef should always be at zero
		subsetIDMap: map[uint16]uint16{0: 0},
		shaper:      shaper,
	}
	return font, nil
}

// Destroy should be called when using HarfBuzz to free the C resources.
func (f *Font) Destroy() {
	f.shaper.Destroy()
}

// Name returns the name of the font.
func (f *Font) Name() string {
	return f.name
}

// Style returns the style of the font.
func (f *Font) Style() FontStyle {
	return f.style
}

// SubsetID maps a glyphID of the original font to the subsetted font. If the glyphID is not subsetted, it will be added to the map.
func (f *Font) SubsetID(glyphID uint16) uint16 {
	if subsetGlyphID, ok := f.subsetIDMap[glyphID]; ok {
		return subsetGlyphID
	}
	subsetGlyphID := uint16(len(f.subsetIDs))
	f.subsetIDs = append(f.subsetIDs, glyphID)
	f.subsetIDMap[glyphID] = subsetGlyphID
	return subsetGlyphID
}

// SubsetIDs returns all subsetted IDs in the order of appearance.
func (f *Font) SubsetIDs() []uint16 {
	return f.subsetIDs
}

// SetVariations sets the font variations (not yet supported).
func (f *Font) SetVariations(variations string) {
	// TODO: support font variations
	f.variations = variations
}

// SetFeatures sets the font features (not yet supported).
func (f *Font) SetFeatures(features string) {
	// TODO: support font features
	f.features = features
}

// FontFamily contains a family of fonts (bold, italic, ...). Allowing to select an italic style as the native italic font or to use faux italic if not present.
type FontFamily struct {
	name  string
	fonts map[FontStyle]*Font
}

// NewFontFamily returns a new font family.
func NewFontFamily(name string) *FontFamily {
	return &FontFamily{
		name:  name,
		fonts: map[FontStyle]*Font{},
	}
}

// Destroy should be called when using HarfBuzz to free the C resources.
func (family *FontFamily) Destroy() {
	for _, font := range family.fonts {
		font.Destroy()
	}
}

// Name returns the name of the font family.
func (family *FontFamily) Name() string {
	return family.name
}

// SetVariations sets the font variations (not yet supported).
func (family *FontFamily) SetVariations(variations string) {
	for _, font := range family.fonts {
		font.SetVariations(variations)
	}
}

// SetFeatures sets the font features (not yet supported).
func (family *FontFamily) SetFeatures(features string) {
	for _, font := range family.fonts {
		font.SetFeatures(features)
	}
}

// LoadLocalFont loads a font from the system's fonts.
func (family *FontFamily) LoadLocalFont(name string, style FontStyle) error {
	// TODO: use style to match font
	finder := sysfont.NewFinder(&sysfont.FinderOpts{
		Extensions: []string{".ttf", ".otf", ".ttc", ".woff", ".woff2", ".eot"},
	})
	font := finder.Match(name)
	return family.LoadFontFile(font.Filename, style)

	//match := name
	//if style&FontExtraLight == FontExtraLight {
	//	match += ":weight=40"
	//} else if style&FontLight == FontLight {
	//	match += ":weight=50"
	//} else if style&FontBook == FontBook {
	//	match += ":weight=75"
	//} else if style&FontMedium == FontMedium {
	//	match += ":weight=100"
	//} else if style&FontSemibold == FontSemibold {
	//	match += ":weight=180"
	//} else if style&FontBold == FontBold {
	//	match += ":weight=200"
	//} else if style&FontBlack == FontBlack {
	//	match += ":weight=205"
	//} else if style&FontExtraBlack == FontExtraBlack {
	//	match += ":weight=210"
	//}
	//if style&FontItalic == FontItalic {
	//	match += ":italic"
	//}
	//b, err := exec.Command("fc-match", "--format=%{file}", match).Output()
	//if err != nil {
	//	return err
	//}
	//return family.LoadFontFile(string(b), style)
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
	font, err := parseFont(family.name, style, b, index)
	if err != nil {
		return err
	}
	family.fonts[style] = font
	return nil
}

// Face gets the font face given by the font size in points and its style.
func (family *FontFamily) Face(size float64, col color.Color, style FontStyle, variant FontVariant, deco ...FontDecorator) *FontFace {
	face := &FontFace{}
	face.Font = family.fonts[style]
	face.Size = size * mmPerPt
	face.Style = style
	face.Variant = variant

	r, g, b, a := col.RGBA()
	face.Color = color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
	face.Deco = deco

	if variant == FontSubscript || variant == FontSuperscript {
		scale := 0.583
		xOffset, yOffset := int16(0), int16(0)
		units := float64(face.Font.Head.UnitsPerEm)
		if variant == FontSubscript {
			if face.Font.OS2.YSubscriptXSize != 0 {
				scale = float64(face.Font.OS2.YSubscriptXSize) / units
			}
			if face.Font.OS2.YSubscriptXOffset != 0 {
				xOffset = face.Font.OS2.YSubscriptXOffset
			}
			yOffset = int16(0.33 * units)
			if face.Font.OS2.YSubscriptYOffset != 0 {
				yOffset = -face.Font.OS2.YSubscriptYOffset
			}
		} else if variant == FontSuperscript {
			if face.Font.OS2.YSuperscriptXSize != 0 {
				scale = float64(face.Font.OS2.YSuperscriptXSize) / units
			}
			if face.Font.OS2.YSuperscriptXOffset != 0 {
				xOffset = face.Font.OS2.YSuperscriptXOffset
			}
			yOffset = int16(-0.33 * units)
			if face.Font.OS2.YSuperscriptYOffset != 0 {
				yOffset = face.Font.OS2.YSuperscriptYOffset
			}
		}
		face.Size *= scale
		face.XOffset = int32(float64(xOffset) / scale)
		face.YOffset = int32(float64(yOffset) / scale)
		if style&0xFF == FontExtraLight {
			style = style&0x100 | FontLight
		} else if style&0xFF == FontLight || style&0xFF == FontBook {
			style = style & 0x100
		} else if style&0xFF == FontRegular {
			style = style&0x100 | FontSemibold
		} else if style&0xFF == FontMedium || style&0xFF == FontSemibold {
			style = style&0x100 | FontBold
		} else if style&0xFF == FontBold {
			style = style&0x100 | FontBlack
		} else if style&0xFF == FontBlack {
			style = style&0x100 | FontExtraBlack
		} else {
			face.FauxBold += 0.02
		}
		face.Font = family.fonts[style]
		face.Style = style
	}

	if face.Font == nil {
		face.Font = family.fonts[FontRegular]
		if face.Font == nil {
			panic("requested font style not found")
		}
		if style&0xFF == FontExtraLight {
			face.FauxBold += -0.02
		} else if style&0xFF == FontLight {
			face.FauxBold += -0.01
		} else if style&0xFF == FontBook {
			face.FauxBold += -0.005
		} else if style&0xFF == FontMedium {
			face.FauxBold += 0.005
		} else if style&0xFF == FontSemibold {
			face.FauxBold += 0.01
		} else if style&0xFF == FontBold {
			face.FauxBold += 0.02
		} else if style&0xFF == FontBlack {
			face.FauxBold += 0.03
		} else if style&0xFF == FontExtraBlack {
			face.FauxBold += 0.04
		}
		if style&FontItalic != 0 {
			if face.Font.Post.ItalicAngle != 0 {
				face.FauxItalic = math.Tan(float64(-face.Font.Post.ItalicAngle))
			} else {
				face.FauxItalic = 0.3
			}
		}
	}
	face.mmPerEm = face.Size / float64(face.Font.Head.UnitsPerEm)
	return face
}

// FontFace defines a font face from a given font. It specifies the font size, color, faux styles and font decorations.
type FontFace struct {
	Font *Font

	Size    float64 // in pt
	Style   FontStyle
	Variant FontVariant

	Color color.RGBA
	Deco  []FontDecorator

	// faux styles for bold, italic, and sub- and superscript
	FauxBold, FauxItalic float64
	XOffset, YOffset     int32

	Language  string
	Script    text.Script
	Direction text.Direction

	// letter spacing
	// stroke and stroke color
	// line height
	// shadow

	mmPerEm float64
}

// Equals returns true when two font face are equal.
func (face *FontFace) Equals(other *FontFace) bool {
	return reflect.DeepEqual(face, other)
}

// Name returns the name of the underlying font.
func (face *FontFace) Name() string {
	return face.Font.name
}

// HasDecoration returns true if the font face has decorations enabled.
func (face *FontFace) HasDecoration() bool {
	return 0 < len(face.Deco)
}

// FontMetrics contains a number of metrics that define a font face. See https://developer.apple.com/library/archive/documentation/TextFonts/Conceptual/CocoaTextArchitecture/Art/glyph_metrics_2x.png for an explanation of the different metrics.
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

func (m FontMetrics) String() string {
	return fmt.Sprintf("{LineHeight: %v, Ascent: %v, Descent: %v, LineGap: %v, XHeight: %v, CapHeight: %v}", m.LineHeight, m.Ascent, m.Descent, m.LineGap, m.XHeight, m.CapHeight)
}

// Metrics returns the font metrics. See https://developer.apple.com/library/archive/documentation/TextFonts/Conceptual/CocoaTextArchitecture/Art/glyph_metrics_2x.png for an explanation of the different metrics.
func (face *FontFace) Metrics() FontMetrics {
	sfnt := face.Font.SFNT
	return FontMetrics{
		LineHeight: face.mmPerEm * float64(sfnt.Hhea.Ascender-sfnt.Hhea.Descender+sfnt.Hhea.LineGap),
		Ascent:     face.mmPerEm * float64(sfnt.Hhea.Ascender),
		Descent:    face.mmPerEm * float64(-sfnt.Hhea.Descender),
		LineGap:    face.mmPerEm * float64(sfnt.Hhea.LineGap),
		XHeight:    face.mmPerEm * float64(sfnt.OS2.SxHeight),
		CapHeight:  face.mmPerEm * float64(sfnt.OS2.SCapHeight),
		XMin:       face.mmPerEm * float64(sfnt.Head.XMin),
		YMin:       face.mmPerEm * float64(sfnt.Head.YMin),
		XMax:       face.mmPerEm * float64(sfnt.Head.XMax),
		YMax:       face.mmPerEm * float64(sfnt.Head.YMax),
	}
}

// PPEM returns the pixels-per-EM for a given resolution of the font face.
func (face *FontFace) PPEM(resolution Resolution) uint16 {
	// ppem is for hinting purposes only, this does not influence glyph advances
	return uint16(resolution.DPMM() * face.Size)
}

// TextWidth returns the width of a given string in millimeters.
func (face *FontFace) TextWidth(s string) float64 {
	ppem := face.PPEM(DefaultResolution)
	glyphs := face.Font.shaper.Shape(s, ppem, face.Direction, face.Script, face.Language, face.Font.features, face.Font.variations)
	return face.textWidth(glyphs)
}

func (face *FontFace) textWidth(glyphs []text.Glyph) float64 {
	sfnt := face.Font.SFNT
	w := int32(0)
	for i, glyph := range glyphs {
		if i != 0 {
			w += int32(sfnt.Kerning(glyphs[i-1].ID, glyph.ID))
		}
		w += int32(sfnt.GlyphAdvance(glyph.ID))
	}
	return face.mmPerEm * float64(w)
}

// Decorate will return the decoration path over a given width in millimeters.
func (face *FontFace) Decorate(width float64) *Path {
	p := &Path{}
	if face.Deco != nil {
		for _, deco := range face.Deco {
			p = p.Append(deco.Decorate(face, width))
		}
	}
	return p
}

// ToPath converts a string to its glyph paths.
func (face *FontFace) ToPath(s string) (*Path, float64, error) {
	ppem := face.PPEM(DefaultResolution)
	glyphs := face.Font.shaper.Shape(s, ppem, face.Direction, face.Script, face.Language, face.Font.features, face.Font.variations)
	return face.toPath(glyphs, ppem)
}

func (face *FontFace) toPath(glyphs []text.Glyph, ppem uint16) (*Path, float64, error) {
	p := &Path{}
	x, y := face.XOffset, face.YOffset
	for _, glyph := range glyphs {
		err := face.Font.GlyphPath(p, glyph.ID, ppem, x+glyph.XOffset, y+glyph.YOffset, face.mmPerEm, font.NoHinting)
		if err != nil {
			return p, 0.0, err
		}
		x += glyph.XAdvance
		y += glyph.YAdvance
	}

	if face.FauxBold != 0.0 {
		p = p.Offset(face.FauxBold*face.Size, NonZero)
	}
	if face.FauxItalic != 0.0 {
		p = p.Transform(Identity.Shear(face.FauxItalic, 0.0))
	}
	return p, face.mmPerEm * float64(x), nil
}

////////////////////////////////////////////////////////////////

// FontDecorator is an interface that returns a path given a font face and a width in millimeters.
type FontDecorator interface {
	Decorate(*FontFace, float64) *Path
}

const underlineDistance = 0.075
const underlineThickness = 0.05

// FontUnderline is a font decoration that draws a line under the text.
var FontUnderline FontDecorator = underline{}

type underline struct{}

func (underline) Decorate(face *FontFace, w float64) *Path {
	r := face.Size * underlineThickness
	y := -face.Size * underlineDistance
	if face.Font.Post.UnderlineThickness != 0 {
		r = face.mmPerEm * float64(face.Font.Post.UnderlineThickness)
	}
	if face.Font.Post.UnderlinePosition != 0 {
		y = face.mmPerEm * float64(face.Font.Post.UnderlinePosition)
	}
	y -= r

	p := &Path{}
	p.MoveTo(0.0, y)
	p.LineTo(w, y)
	return p.Stroke(r, ButtCap, BevelJoin)
}

func (underline) String() string {
	return "Underline"
}

// FontOverline is a font decoration that draws a line over the text.
var FontOverline FontDecorator = overline{}

type overline struct{}

func (overline) Decorate(face *FontFace, w float64) *Path {
	r := face.Size * underlineThickness
	y := face.Metrics().Ascent
	if face.Font.Post.UnderlineThickness != 0 {
		r = face.mmPerEm * float64(face.Font.Post.UnderlineThickness)
	}
	y -= 0.5 * r

	dx := face.FauxItalic * y
	w += face.FauxItalic * y

	p := &Path{}
	p.MoveTo(dx, y)
	p.LineTo(w, y)
	return p.Stroke(r, ButtCap, BevelJoin)
}

func (overline) String() string {
	return "Overline"
}

// FontStrikethrough is a font decoration that draws a line through the text.
var FontStrikethrough FontDecorator = strikethrough{}

type strikethrough struct{}

func (strikethrough) Decorate(face *FontFace, w float64) *Path {
	r := face.Size * underlineThickness
	y := face.Metrics().XHeight / 2.0
	if face.Font.OS2.YStrikeoutSize != 0 {
		r = face.mmPerEm * float64(face.Font.OS2.YStrikeoutSize)
	}
	if face.Font.OS2.YStrikeoutPosition != 0 {
		y = face.mmPerEm * float64(face.Font.OS2.YStrikeoutPosition)
	}
	y += 0.5 * r

	dx := face.FauxItalic * y
	w += face.FauxItalic * y

	p := &Path{}
	p.MoveTo(dx, y)
	p.LineTo(w, y)
	return p.Stroke(r, ButtCap, BevelJoin)
}

func (strikethrough) String() string {
	return "Strikethrough"
}

// FontDoubleUnderline is a font decoration that draws two lines under the text.
var FontDoubleUnderline FontDecorator = doubleUnderline{}

type doubleUnderline struct{}

func (doubleUnderline) Decorate(face *FontFace, w float64) *Path {
	r := face.Size * underlineThickness
	y := -face.Size * underlineDistance
	if face.Font.Post.UnderlineThickness != 0 {
		r = face.mmPerEm * float64(face.Font.Post.UnderlineThickness)
	}
	if face.Font.Post.UnderlinePosition != 0 {
		y = face.mmPerEm * float64(face.Font.Post.UnderlinePosition)
	}
	y -= r

	p := &Path{}
	p.MoveTo(0.0, y)
	p.LineTo(w, y)
	p.MoveTo(0.0, y-1.5*r)
	p.LineTo(w, y-1.5*r)
	return p.Stroke(r, ButtCap, BevelJoin)
}

func (doubleUnderline) String() string {
	return "DoubleUnderline"
}

// FontDottedUnderline is a font decoration that draws a dotted line under the text.
var FontDottedUnderline FontDecorator = dottedUnderline{}

type dottedUnderline struct{}

func (dottedUnderline) Decorate(face *FontFace, w float64) *Path {
	r := 0.5 * face.Size * underlineThickness
	y := -face.Size * underlineDistance
	if face.Font.Post.UnderlineThickness != 0 {
		r = 0.5 * face.mmPerEm * float64(face.Font.Post.UnderlineThickness)
	}
	if face.Font.Post.UnderlinePosition != 0 {
		y = face.mmPerEm * float64(face.Font.Post.UnderlinePosition)
	}
	y -= 2.0 * r
	w -= 2.0 * r
	if w < 0.0 {
		return &Path{}
	}

	d := 4.0 * r
	n := int(w/d) + 1
	p := &Path{}
	if n == 1 {
		return p.Append(Circle(r).Translate(r+w/2.0, y))
	}

	d = w / float64(n-1)
	for i := 0; i < n; i++ {
		p = p.Append(Circle(r).Translate(r+float64(i)*d, y))
	}
	return p
}

func (dottedUnderline) String() string {
	return "DottedUnderline"
}

// FontDashedUnderline is a font decoration that draws a dashed line under the text.
var FontDashedUnderline FontDecorator = dashedUnderline{}

type dashedUnderline struct{}

func (dashedUnderline) Decorate(face *FontFace, w float64) *Path {
	r := face.Size * underlineThickness
	y := -face.Size * underlineDistance
	if face.Font.Post.UnderlineThickness != 0 {
		r = face.mmPerEm * float64(face.Font.Post.UnderlineThickness)
	}
	if face.Font.Post.UnderlinePosition != 0 {
		y = face.mmPerEm * float64(face.Font.Post.UnderlinePosition)
	}
	y -= r

	d := 3.0 * r
	n := 2*int((w-d)/(2.0*d)) + 1

	p := &Path{}
	p.MoveTo(0.0, y)
	p.LineTo(w, y)
	if 2 < n {
		d = w / float64(n)
		p = p.Dash(0.0, d)
	}
	return p.Stroke(r, ButtCap, BevelJoin)
}

func (dashedUnderline) String() string {
	return "DashedUnderline"
}

// FontWavyUnderline is a font decoration that draws a wavy path under the text.
var FontWavyUnderline FontDecorator = wavyUnderline{}

type wavyUnderline struct{}

func (wavyUnderline) Decorate(face *FontFace, w float64) *Path {
	r := face.Size * underlineThickness
	y := -face.Size * underlineDistance
	if face.Font.Post.UnderlineThickness != 0 {
		r = face.mmPerEm * float64(face.Font.Post.UnderlineThickness)
	}
	if face.Font.Post.UnderlinePosition != 0 {
		y = face.mmPerEm * float64(face.Font.Post.UnderlinePosition)
	}
	y -= r

	dx := 0.707 * r
	w -= 2.0 * dx
	dh := -face.Size * 0.15
	d := 5.0 * r
	n := int(0.5 + w/d)
	if n == 0 {
		return &Path{}
	}
	d = w / float64(n)

	p := &Path{}
	p.MoveTo(dx, y)
	for i := 0; i < n; i++ {
		if i%2 == 0 {
			p.LineTo(dx+d/3.0, y)
			p.LineTo(dx+d, y+dh)
		} else {
			p.LineTo(dx+d/3.0, y+dh)
			p.LineTo(dx+d, y)
		}
		dx += d
	}
	return p.Stroke(r, ButtCap, MiterJoin)
}

func (wavyUnderline) String() string {
	return "WavyUnderline"
}

// FontSineUnderline is a font decoration that draws a wavy sine path under the text.
var FontSineUnderline FontDecorator = sineUnderline{}

type sineUnderline struct{}

func (sineUnderline) Decorate(face *FontFace, w float64) *Path {
	r := face.Size * underlineThickness
	y := -face.Size * underlineDistance
	if face.Font.Post.UnderlineThickness != 0 {
		r = face.mmPerEm * float64(face.Font.Post.UnderlineThickness)
	}
	if face.Font.Post.UnderlinePosition != 0 {
		y = face.mmPerEm * float64(face.Font.Post.UnderlinePosition)
	}
	y -= r

	w -= r
	dh := -face.Size * 0.15
	d := 4.0 * r
	n := int(0.5 + w/d)
	if n == 0 {
		return &Path{}
	}
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

func (sineUnderline) String() string {
	return "SineUnderline"
}

// FontSawtoothUnderline is a font decoration that draws a wavy sawtooth path under the text.
var FontSawtoothUnderline FontDecorator = sawtoothUnderline{}

type sawtoothUnderline struct{}

func (sawtoothUnderline) Decorate(face *FontFace, w float64) *Path {
	r := face.Size * underlineThickness
	y := -face.Size * underlineDistance
	if face.Font.Post.UnderlineThickness != 0 {
		r = face.mmPerEm * float64(face.Font.Post.UnderlineThickness)
	}
	if face.Font.Post.UnderlinePosition != 0 {
		y = face.mmPerEm * float64(face.Font.Post.UnderlinePosition)
	}
	y -= r

	dx := 0.707 * r
	w -= 2.0 * dx
	dh := -face.Size * 0.15
	d := 4.0 * r
	n := int(0.5 + w/d)
	if n == 0 {
		return &Path{}
	}
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

func (sawtoothUnderline) String() string {
	return "SawtoothUnderline"
}
