package canvas

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"os"
	"reflect"
	"sync"

	"github.com/Seanld/canvas/text"
	"github.com/tdewolff/font"
)

// FontStyle defines the font style to be used for the font. It specifies a boldness with optionally italic, e.g. FontBlack | FontItalic will specify a black boldness (a font-weight of 800 in CSS) and italic.
type FontStyle int

// see FontStyle
const (
	FontRegular    FontStyle = iota // 400
	FontThin                        // 100
	FontExtraLight                  // 200
	FontLight                       // 300
	FontMedium                      // 500
	FontSemiBold                    // 600
	FontBold                        // 700
	FontExtraBold                   // 800
	FontBlack                       // 900
	FontItalic     FontStyle = 1 << 8
)

// Italic returns true if italic.
func (style FontStyle) Italic() bool {
	return style&FontItalic != 0
}

// Weight returns the font weight (FontRegular, FontBold, ...)
func (style FontStyle) Weight() FontStyle {
	return style & 0xFF
}

// CSS returns the CSS boldness value for the font face.
func (style FontStyle) CSS() int {
	switch style.Weight() {
	case FontThin:
		return 100
	case FontExtraLight:
		return 200
	case FontLight:
		return 300
	case FontMedium:
		return 500
	case FontSemiBold:
		return 600
	case FontBold:
		return 700
	case FontExtraBold:
		return 800
	case FontBlack:
		return 900
	}
	return 400
}

// FauxWeight returns the path offset for fake boldness relative to regular style. The offset is multiplied by the font size (in millimeters) for an offset in millimeters.
func (style FontStyle) FauxWeight() float64 {
	switch style.Weight() {
	case FontThin:
		return -0.02
	case FontExtraLight:
		return -0.01
	case FontLight:
		return -0.005
	case FontMedium:
		return 0.005
	case FontSemiBold:
		return 0.01
	case FontBold:
		return 0.02
	case FontExtraBold:
		return 0.03
	case FontBlack:
		return 0.04
	}
	return 0.0
}

func (style FontStyle) String() string {
	var s string
	switch style.Weight() {
	case FontThin:
		s = "Thin"
	case FontExtraLight:
		s = "ExtraLight"
	case FontLight:
		s = "Light"
	case FontRegular:
		s = "Regular"
	case FontMedium:
		s = "Medium"
	case FontSemiBold:
		s = "SemiBold"
	case FontBold:
		s = "Bold"
	case FontExtraBold:
		s = "ExtraBold"
	case FontBlack:
		s = "Black"
	default:
		return "Unknown"
	}
	if style.Italic() {
		s += " Italic"
	}
	return s
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

func (variant FontVariant) String() string {
	switch variant {
	case FontNormal:
		return "Normal"
	case FontSubscript:
		return "Subscript"
	case FontSuperscript:
		return "Superscript"
	case FontSmallcaps:
		return "Smallcaps"
	}
	return "Unknown"
}

////////////////////////////////////////////////////////////////

// FontSubsetter holds a map between original glyph IDs and new glyph IDs in a subsetted font.
type FontSubsetter struct {
	IDs   []uint16          // old glyphIDs for increasing new glyphIDs
	IDMap map[uint16]uint16 // old to new glyphID
}

// NewFontSubsetter returns a new font subsetter.
func NewFontSubsetter() *FontSubsetter {
	return &FontSubsetter{
		IDs:   []uint16{0}, // .notdef should always be at zero
		IDMap: map[uint16]uint16{0: 0},
	}
}

// Get maps a glyphID of the original font to the subsetted font. If the glyphID is not subsetted, it will be added to the map.
func (subsetter *FontSubsetter) Get(glyphID uint16) uint16 {
	if subsetGlyphID, ok := subsetter.IDMap[glyphID]; ok {
		return subsetGlyphID
	}
	subsetGlyphID := uint16(len(subsetter.IDs))
	subsetter.IDs = append(subsetter.IDs, glyphID)
	subsetter.IDMap[glyphID] = subsetGlyphID
	return subsetGlyphID
}

// List returns all subsetted IDs in the order of appearance.
func (subsetter *FontSubsetter) List() []uint16 {
	return subsetter.IDs
}

////////////////////////////////////////////////////////////////

var systemFonts = struct {
	*font.SystemFonts
	sync.Mutex
}{}

// FindLocalFont finds the path to a font from the system's fonts.
func FindLocalFont(name string, style FontStyle) string {
	log.Println("WARNING: github.com/tdewolff/canvas/FindLocalFont is deprecated, please use github.com/tdewolff/canvas/FindSystemFont") // TODO: remove
	filename, _ := FindSystemFont(name, style)
	return filename
}

// CacheSystemFonts will write and load the list of system fonts to the given filename. It scans the given directories for fonts, leave nil to use github.com/tdewolff/font/DefaultFontDirs().
func CacheSystemFonts(filename string, dirs []string) error {
	var fonts *font.SystemFonts
	if info, err := os.Stat(filename); err == nil && info.Mode().IsRegular() {
		fonts, err = font.LoadSystemFonts(filename)
		if err != nil {
			return err
		}
	} else {
		if dirs == nil {
			dirs = font.DefaultFontDirs()
		}
		var err error
		fonts, err = font.FindSystemFonts(dirs)
		if err != nil {
			return err
		}
		if err := fonts.Save(filename); err != nil {
			return err
		}
	}
	systemFonts.Lock()
	systemFonts.SystemFonts = fonts
	systemFonts.Unlock()
	return nil
}

// FindSystemFont finds the path to a font from the system's fonts.
func FindSystemFont(name string, style FontStyle) (string, bool) {
	systemFonts.Lock()
	if systemFonts.SystemFonts == nil {
		systemFonts.SystemFonts, _ = font.FindSystemFonts(font.DefaultFontDirs())
	}
	font, ok := systemFonts.Match(name, font.ParseStyleCSS(style.CSS(), style.Italic()))
	systemFonts.Unlock()
	return font.Filename, ok
}

// Font defines an SFNT font such as TTF or OTF.
type Font struct {
	*font.SFNT
	name       string
	style      FontStyle
	shaper     text.Shaper
	variations string
	features   string
}

// LoadLocalFont loads a font from the system's fonts.
func LoadLocalFont(name string, style FontStyle) (*Font, error) {
	log.Println("WARNING: github.com/tdewolff/canvas/LoadLocalFont is deprecated, please use github.com/tdewolff/canvas/LoadSystemFont") // TODO: remove
	return LoadSystemFont(name, style)
}

// LoadSystemFont loads a font from the system's fonts.
func LoadSystemFont(name string, style FontStyle) (*Font, error) {
	filename, ok := FindSystemFont(name, style)
	if !ok {
		return nil, fmt.Errorf("failed to find font '%s'", name)
	}
	return LoadFontFile(filename, style)
}

// LoadFontFile loads a font from a file.
func LoadFontFile(filename string, style FontStyle) (*Font, error) {
	b, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to load font file '%s': %w", filename, err)
	}
	return LoadFont(b, 0, style)
}

// LoadFontCollection loads a font from a collection file and uses the font at the specified index.
func LoadFontCollection(filename string, index int, style FontStyle) (*Font, error) {
	b, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to load font file '%s': %w", filename, err)
	}
	return LoadFont(b, index, style)
}

var nonameFonts = 0

// LoadFont loads a font from memory.
func LoadFont(b []byte, index int, style FontStyle) (*Font, error) {
	SFNT, err := font.ParseFont(b, index)
	if err != nil {
		return nil, err
	}

	shaper, err := text.NewShaperSFNT(SFNT)
	if err != nil {
		return nil, err
	}

	name := ""
NameLoop:
	for _, id := range []int{6, 4, 1} {
		for _, record := range SFNT.Name.Get(font.NameID(id)) {
			name = record.String()
			break NameLoop
		}
	}
	if name == "" {
		name = fmt.Sprintf("f%d", nonameFonts)
		nonameFonts++
	}

	font := &Font{
		SFNT:   SFNT,
		name:   name,
		style:  style,
		shaper: shaper,
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

// SetVariations sets the font variations (not yet supported).
func (f *Font) SetVariations(variations string) {
	// TODO: support font variations
	f.variations = variations
}

// SetFeatures sets the font features (not yet supported).
func (f *Font) SetFeatures(features string) {
	f.features = features
}

// Face gets the font face given by the font size in points and its style. Fill can be any of Paint, color.Color, or canvas.Pattern.
func (f *Font) Face(size float64, ifill interface{}, deco ...FontDecorator) *FontFace {
	face := &FontFace{}
	face.Font = f
	face.Size = size * mmPerPt
	face.Style = f.style
	face.Variant = FontNormal

	if paint, ok := ifill.(Paint); ok {
		face.Fill = paint
	} else if pattern, ok := ifill.(Pattern); ok {
		face.Fill = Paint{Pattern: pattern}
	} else if gradient, ok := ifill.(Gradient); ok {
		face.Fill = Paint{Gradient: gradient}
	} else if col, ok := ifill.(color.Color); ok {
		face.Fill = Paint{Color: rgbaColor(col)}
	}
	face.Deco = deco
	face.Hinting = font.VerticalHinting
	face.MmPerEm = face.Size / float64(face.Font.Head.UnitsPerEm)
	return face
}

////////////////////////////////////////////////////////////////

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
	log.Println("WARNING: github.com/tdewolff/canvas/FontFamily.LoadLocalFont is deprecated, please use github.com/tdewolff/canvas/FontFamily.LoadSystemFont") // TODO: remove
	return family.LoadSystemFont(name, style)
}

// MustLoadLocalFont loads a font from the system's fonts and panics on error.
func (family *FontFamily) MustLoadLocalFont(name string, style FontStyle) {
	log.Println("WARNING: github.com/tdewolff/canvas/FontFamily.MustLoadLocalFont is deprecated, please use github.com/tdewolff/canvas/FontFamily.MustLoadSystemFont") // TODO: remove
	family.MustLoadSystemFont(name, style)
}

// LoadSystemFont loads a font from the system's fonts.
func (family *FontFamily) LoadSystemFont(name string, style FontStyle) error {
	font, err := LoadSystemFont(name, style)
	if err != nil {
		return err
	}
	family.fonts[style] = font
	font.name = family.name
	return nil
}

// MustLoadSystemFont loads a font from the system's fonts and panics on error.
func (family *FontFamily) MustLoadSystemFont(name string, style FontStyle) {
	if err := family.LoadSystemFont(name, style); err != nil {
		panic(err)
	}
}

// LoadFontFile loads a font from a file.
func (family *FontFamily) LoadFontFile(filename string, style FontStyle) error {
	font, err := LoadFontFile(filename, style)
	if err != nil {
		return err
	}
	family.fonts[style] = font
	font.name = family.name
	return nil
}

// MustLoadFontFile loads a font from a filea and panics on error.
func (family *FontFamily) MustLoadFontFile(filename string, style FontStyle) {
	if err := family.LoadFontFile(filename, style); err != nil {
		panic(err)
	}
}

// LoadFontCollection loads a font from a collection file and uses the font at the specified index.
func (family *FontFamily) LoadFontCollection(filename string, index int, style FontStyle) error {
	font, err := LoadFontCollection(filename, index, style)
	if err != nil {
		return err
	}
	family.fonts[style] = font
	font.name = family.name
	return nil
}

// MustLoadFontCollection loads a font from a collection file and uses the font at the specified index. It panics on error.
func (family *FontFamily) MustLoadFontCollection(filename string, index int, style FontStyle) {
	if err := family.LoadFontCollection(filename, index, style); err != nil {
		panic(err)
	}
}

// LoadFont loads a font from memory.
func (family *FontFamily) LoadFont(b []byte, index int, style FontStyle) error {
	font, err := LoadFont(b, index, style)
	if err != nil {
		return err
	}
	family.fonts[style] = font
	font.name = family.name
	return nil
}

// MustLoadFont loads a font from memory. It panics on error.
func (family *FontFamily) MustLoadFont(b []byte, index int, style FontStyle) {
	if err := family.LoadFont(b, index, style); err != nil {
		panic(err)
	}
}

// Face gets the font face given by the font size in points. Other arguments that can be passed: Paint/Pattern/color.Color (=Black), FontStyle (=FontRegular), FontVariant (=FontNormal), multiple FontDecorator, and Hinting (=VerticalHinting).
func (family *FontFamily) Face(size float64, args ...interface{}) *FontFace {
	if len(family.fonts) == 0 {
		panic("font family is empty")
	}

	face := &FontFace{
		Fill:    Paint{Color: Black},
		Hinting: font.VerticalHinting,
		Size:    size * mmPerPt,
	}
	for _, iarg := range args {
		switch arg := iarg.(type) {
		case Paint:
			face.Fill = arg
		case Pattern:
			face.Fill = Paint{Pattern: arg}
		case Gradient:
			face.Fill = Paint{Gradient: arg}
		case color.Color:
			face.Fill = Paint{Color: rgbaColor(arg)}
		case FontStyle:
			face.Style = arg
		case FontVariant:
			face.Variant = arg
		case FontDecorator:
			face.Deco = append(face.Deco, arg)
		case font.Hinting:
			face.Hinting = arg
		}
	}

	// add weight for sub- and superscript
	if face.Variant == FontSubscript || face.Variant == FontSuperscript {
		switch face.Style.Weight() {
		case FontThin:
			face.Style = face.Style&FontItalic | FontExtraLight
		case FontExtraLight:
			face.Style = face.Style&FontItalic | FontLight
		case FontLight:
			face.Style = face.Style&FontItalic | FontRegular
		case FontRegular:
			face.Style = face.Style&FontItalic | FontSemiBold
		case FontMedium:
			face.Style = face.Style&FontItalic | FontSemiBold
		case FontSemiBold:
			face.Style = face.Style&FontItalic | FontBold
		case FontBold:
			face.Style = face.Style&FontItalic | FontExtraBold
		case FontExtraBold:
			face.Style = face.Style&FontItalic | FontBlack
		default:
			face.FauxBold += 0.02
		}
	}

	// find closest font that matches requested style
	face.Font = family.fonts[face.Style]
	if face.Font == nil {
		minDiff := math.Inf(1.0)
		minStyle := FontRegular
		for style := range family.fonts {
			diff := math.Abs(face.Style.FauxWeight() - style.FauxWeight())
			if face.Style.Italic() != style.Italic() {
				diff += 0.02
			}
			if diff < minDiff {
				minStyle = style
				minDiff = diff
			}
		}
		face.Font = family.fonts[minStyle]
		face.FauxBold += face.Style.FauxWeight() - minStyle.FauxWeight()
		if face.Style.Italic() != minStyle.Italic() {
			sign := 1.0
			if !face.Style.Italic() {
				sign = -1.0
			}
			if face.Font.Post.ItalicAngle != 0 {
				face.FauxItalic = sign * math.Tan(-face.Font.Post.ItalicAngle)
			} else {
				face.FauxItalic = sign * 0.3
			}
		}
	}

	if face.Variant == FontSubscript || face.Variant == FontSuperscript {
		scale := 0.583
		xOffset, yOffset := int16(0), int16(0)
		units := float64(face.Font.Head.UnitsPerEm)
		if face.Variant == FontSubscript {
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
		} else if face.Variant == FontSuperscript {
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
	}
	face.MmPerEm = face.Size / float64(face.Font.Head.UnitsPerEm)
	return face
}

////////////////////////////////////////////////////////////////

// FontFace defines a font face from a given font. It specifies the font size, color, faux styles and font decorations.
type FontFace struct {
	Font *Font

	Size    float64 // in mm
	Style   FontStyle
	Variant FontVariant

	Fill    Paint
	Deco    []FontDecorator
	Hinting font.Hinting

	// faux styles for bold, italic, and sub- and superscript
	FauxBold, FauxItalic float64
	XOffset, YOffset     int32

	Language  string
	Script    text.Script
	Direction text.Direction // TODO: really needed here?

	// letter spacing
	// stroke and stroke color
	// line height
	// shadow

	MmPerEm float64 // millimeters per EM unit!
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
	// TODO: use resolution
	sfnt := face.Font.SFNT
	ascender, descender, lineGap := sfnt.VerticalMetrics()
	return FontMetrics{
		LineHeight: face.MmPerEm * float64(ascender+descender+lineGap),
		Ascent:     face.MmPerEm * float64(ascender),
		Descent:    face.MmPerEm * float64(descender),
		LineGap:    face.MmPerEm * float64(lineGap),
		XHeight:    face.MmPerEm * float64(sfnt.OS2.SxHeight),
		CapHeight:  face.MmPerEm * float64(sfnt.OS2.SCapHeight),
		XMin:       face.MmPerEm * float64(sfnt.Head.XMin),
		YMin:       face.MmPerEm * float64(sfnt.Head.YMin),
		XMax:       face.MmPerEm * float64(sfnt.Head.XMax),
		YMax:       face.MmPerEm * float64(sfnt.Head.YMax),
	}
}

// PPEM returns the pixels-per-EM for a given resolution of the font face.
func (face *FontFace) PPEM(resolution Resolution) uint16 {
	// ppem is for hinting purposes only, this does not influence glyph advances
	return uint16(resolution.DPMM() * face.MmPerEm * float64(face.Font.Head.UnitsPerEm))
}

// LineHeight returns the height (ascent+descent) of a line.
func (face *FontFace) LineHeight() float64 {
	metrics := face.Metrics()
	return metrics.Ascent + metrics.Descent
}

func (face *FontFace) Glyphs(s string) []text.Glyph {
	ppem := face.PPEM(DefaultResolution)
	return face.Font.shaper.Shape(s, ppem, face.Direction, face.Script, face.Language, face.Font.features, face.Font.variations)
}

// TextWidth returns the width of a given string in millimeters.
func (face *FontFace) TextWidth(s string) float64 {
	return face.textWidth(face.Glyphs(s))
}

func (face *FontFace) textWidth(glyphs []text.Glyph) float64 {
	w := int32(0)
	for _, glyph := range glyphs {
		if !glyph.Vertical {
			w += glyph.XAdvance
		} else {
			w -= glyph.YAdvance
		}
	}
	return face.MmPerEm * float64(w)
}

func (face *FontFace) heights(mode WritingMode) (float64, float64, float64, float64) {
	metrics := face.Metrics()
	if mode != HorizontalTB {
		ascent, descent, lineGap, xHeight := metrics.Ascent, metrics.Descent, metrics.LineGap, metrics.XHeight
		ascent -= xHeight / 2.0
		descent += xHeight / 2.0
		if mode == VerticalLR {
			ascent, descent = descent, ascent
		}
		return ascent + lineGap, ascent, descent, descent + lineGap
	}
	return metrics.Ascent + metrics.LineGap, metrics.Ascent, metrics.Descent, metrics.Descent + metrics.LineGap
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
	return face.toPath(face.Glyphs(s), ppem)
}

func (face *FontFace) toPath(glyphs []text.Glyph, ppem uint16) (*Path, float64, error) {
	p := &Path{}
	f := face.MmPerEm
	x, y := face.XOffset, face.YOffset
	for _, glyph := range glyphs {
		err := face.Font.GlyphPath(p, glyph.ID, ppem, f*float64(x+glyph.XOffset), f*float64(y+glyph.YOffset), f, font.NoHinting)
		if err != nil {
			return p, 0.0, err
		}
		x += glyph.XAdvance
		y += glyph.YAdvance
	}

	if face.FauxBold != 0.0 {
		p = p.Offset(face.FauxBold*face.Size, NonZero, Tolerance)
	}
	if face.FauxItalic != 0.0 {
		p = p.Transform(Identity.Shear(face.FauxItalic, 0.0))
	}
	return p, face.MmPerEm * float64(x), nil
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
		r = face.MmPerEm * float64(face.Font.Post.UnderlineThickness)
	}
	if face.Font.Post.UnderlinePosition != 0 {
		y = face.MmPerEm * float64(face.Font.Post.UnderlinePosition)
	}
	y -= r

	p := &Path{}
	p.MoveTo(0.0, y)
	p.LineTo(w, y)
	return p.Stroke(r, ButtCap, BevelJoin, Tolerance)
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
		r = face.MmPerEm * float64(face.Font.Post.UnderlineThickness)
	}
	y -= 0.5 * r

	dx := face.FauxItalic * y
	w += face.FauxItalic * y

	p := &Path{}
	p.MoveTo(dx, y)
	p.LineTo(w, y)
	return p.Stroke(r, ButtCap, BevelJoin, Tolerance)
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
		r = face.MmPerEm * float64(face.Font.OS2.YStrikeoutSize)
	}
	if face.Font.OS2.YStrikeoutPosition != 0 {
		y = face.MmPerEm * float64(face.Font.OS2.YStrikeoutPosition)
	}
	y += 0.5 * r

	dx := face.FauxItalic * y
	w += face.FauxItalic * y

	p := &Path{}
	p.MoveTo(dx, y)
	p.LineTo(w, y)
	return p.Stroke(r, ButtCap, BevelJoin, Tolerance)
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
		r = face.MmPerEm * float64(face.Font.Post.UnderlineThickness)
	}
	if face.Font.Post.UnderlinePosition != 0 {
		y = face.MmPerEm * float64(face.Font.Post.UnderlinePosition)
	}
	y -= r

	p := &Path{}
	p.MoveTo(0.0, y)
	p.LineTo(w, y)
	p.MoveTo(0.0, y-1.5*r)
	p.LineTo(w, y-1.5*r)
	return p.Stroke(r, ButtCap, BevelJoin, Tolerance)
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
		r = 0.5 * face.MmPerEm * float64(face.Font.Post.UnderlineThickness)
	}
	if face.Font.Post.UnderlinePosition != 0 {
		y = face.MmPerEm * float64(face.Font.Post.UnderlinePosition)
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
		r = face.MmPerEm * float64(face.Font.Post.UnderlineThickness)
	}
	if face.Font.Post.UnderlinePosition != 0 {
		y = face.MmPerEm * float64(face.Font.Post.UnderlinePosition)
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
	return p.Stroke(r, ButtCap, BevelJoin, Tolerance)
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
		r = face.MmPerEm * float64(face.Font.Post.UnderlineThickness)
	}
	if face.Font.Post.UnderlinePosition != 0 {
		y = face.MmPerEm * float64(face.Font.Post.UnderlinePosition)
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
	return p.Stroke(r, ButtCap, MiterJoin, Tolerance)
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
		r = face.MmPerEm * float64(face.Font.Post.UnderlineThickness)
	}
	if face.Font.Post.UnderlinePosition != 0 {
		y = face.MmPerEm * float64(face.Font.Post.UnderlinePosition)
	}
	y -= r

	w -= r
	d := 4.0 * r
	n := int(0.5 + w/d)
	if n == 0 {
		return &Path{}
	}
	d = (w - r) / float64(n)

	dx := r
	dh := -face.Size * 0.15
	y += 0.5 * dh
	p := &Path{}
	p.MoveTo(dx, y)
	for i := 0; i < n; i++ {
		if i%2 == 0 {
			p.QuadTo(dx+d*0.5, y-dh, dx+d, y)
		} else {
			p.QuadTo(dx+d*0.5, y+dh, dx+d, y)
		}
		dx += d
	}
	return p.Stroke(r, RoundCap, RoundJoin, Tolerance)
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
		r = face.MmPerEm * float64(face.Font.Post.UnderlineThickness)
	}
	if face.Font.Post.UnderlinePosition != 0 {
		y = face.MmPerEm * float64(face.Font.Post.UnderlinePosition)
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
	return p.Stroke(r, ButtCap, MiterJoin, Tolerance)
}

func (sawtoothUnderline) String() string {
	return "SawtoothUnderline"
}
