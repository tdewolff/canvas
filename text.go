package canvas

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/tdewolff/canvas/text"
	"github.com/tdewolff/font"
)

// TextAlign specifies how the text should align or whether it should be justified.
type TextAlign int

// see TextAlign
const (
	Left TextAlign = iota
	Right
	Center
	Middle
	Top
	Bottom
	Justify
)

func (ta TextAlign) String() string {
	switch ta {
	case Left:
		return "Left"
	case Right:
		return "Right"
	case Center:
		return "Center"
	case Middle:
		return "Middle"
	case Top:
		return "Top"
	case Bottom:
		return "Bottom"
	case Justify:
		return "Justify"
	}
	return "Invalid(" + strconv.Itoa(int(ta)) + ")"
}

// VerticalAlign specifies how the object should align vertically when embedded in text.
type VerticalAlign int

// see VerticalAlign
const (
	Baseline VerticalAlign = iota
	FontTop
	FontMiddle
	FontBottom
)

func (valign VerticalAlign) String() string {
	switch valign {
	case Baseline:
		return "Baseline"
	case FontTop:
		return "FontTop"
	case FontMiddle:
		return "FontMiddle"
	case FontBottom:
		return "FontBottom"
	}
	return "Invalid(" + strconv.Itoa(int(valign)) + ")"
}

// WritingMode specifies how the text lines should be laid out.
type WritingMode int

// see WritingMode
const (
	HorizontalTB WritingMode = iota
	VerticalRL
	VerticalLR
)

func (wm WritingMode) String() string {
	switch wm {
	case HorizontalTB:
		return "HorizontalTB"
	case VerticalRL:
		return "VerticalRL"
	case VerticalLR:
		return "VerticalLR"
	}
	return "Invalid(" + strconv.Itoa(int(wm)) + ")"
}

// TextOrientation specifies how horizontal text should be oriented within vertical text, or how vertical-only text should be laid out in horizontal text.
type TextOrientation int

// see TextOrientation
const (
	Natural TextOrientation = iota // turn horizontal text 90deg clockwise for VerticalRL, and counter clockwise for VerticalLR
	Upright                        // split characters and lay them out upright
)

func (orient TextOrientation) String() string {
	switch orient {
	case Natural:
		return "Natural"
	case Upright:
		return "Upright"
	}
	return "Invalid(" + strconv.Itoa(int(orient)) + ")"
}

// Text holds the representation of a text object.
type Text struct {
	lines []line
	fonts map[*Font]bool
	WritingMode
	TextOrientation
	Width, Height float64
	Text          string
	Overflows     bool // true if lines stick out of the box
}

type line struct {
	y     float64
	spans []TextSpan
}

// Heights returns the maximum top, ascent, descent, and bottom heights of the line, where top and bottom are equal to ascent and descent respectively with added line spacing.
func (l line) Heights(mode WritingMode) (float64, float64, float64, float64) {
	top, ascent, descent, bottom := 0.0, 0.0, 0.0, 0.0
	if mode == HorizontalTB {
		for _, span := range l.spans {
			if span.IsText() {
				spanTop, spanAscent, spanDescent, spanBottom := span.Face.heights(mode)
				top = math.Max(top, spanTop)
				ascent = math.Max(ascent, spanAscent)
				descent = math.Max(descent, spanDescent)
				bottom = math.Max(bottom, spanBottom)
			} else {
				for _, obj := range span.Objects {
					spanAscent, spanDescent := obj.Heights(span.Face)
					lineSpacing := span.Face.Metrics().LineGap
					top = math.Max(top, spanAscent+lineSpacing)
					ascent = math.Max(ascent, spanAscent)
					descent = math.Max(descent, spanDescent)
					bottom = math.Max(bottom, spanDescent+lineSpacing)
				}
			}
		}
	} else {
		width := 0.0
		for _, span := range l.spans {
			if span.IsText() {
				for _, glyph := range span.Glyphs {
					if glyph.Vertical {
						width = math.Max(width, 1.2*span.Face.MmPerEm*float64(glyph.SFNT.GlyphAdvance(glyph.ID))) // TODO: what left/right padding should upright characters in a vertical layout have?
					} else {
						spanTop, spanAscent, spanDescent, spanBottom := span.Face.heights(mode)
						top = math.Max(top, spanTop)
						ascent = math.Max(ascent, spanAscent)
						descent = math.Max(descent, spanDescent)
						bottom = math.Max(bottom, spanBottom)
					}
				}
			} else {
				for _, obj := range span.Objects {
					width = math.Max(width, obj.Width)
				}
			}
		}
		top = math.Max(top, width/2.0)
		ascent = math.Max(ascent, width/2.0)
		descent = math.Max(descent, width/2.0)
		bottom = math.Max(bottom, width/2.0)
	}
	return top, ascent, descent, bottom
}

// TextSpan is a span of text.
type TextSpan struct {
	X         float64
	Width     float64
	Face      *FontFace
	Text      string
	Glyphs    []text.Glyph
	Direction text.Direction
	Rotation  text.Rotation
	Level     int

	Objects []TextSpanObject
}

// IsText returns true if the text span is text and not objects (such as images or paths).
func (span *TextSpan) IsText() bool {
	return len(span.Objects) == 0
}

// TextSpanObject is an object that can be used within a text span. It is a wrapper around Canvas and can thus draw anything to be mixed with text, such as images (emoticons) or paths (symbols).
type TextSpanObject struct {
	*Canvas
	X, Y          float64
	Width, Height float64
	VAlign        VerticalAlign
}

// Heights returns the ascender and descender values of the span object.
func (obj TextSpanObject) Heights(face *FontFace) (float64, float64) {
	switch obj.VAlign {
	case FontTop:
		ascent := face.Metrics().Ascent
		return ascent, -(ascent - obj.Height)
	case FontMiddle:
		ascent, descent := face.Metrics().Ascent, face.Metrics().Descent
		return (ascent - descent + obj.Height) / 2.0, -(ascent - descent - obj.Height) / 2.0
	case FontBottom:
		descent := face.Metrics().Descent
		return -descent + obj.Height, descent
	}
	return obj.Height, 0.0 // Baseline
}

// View returns the object's view to be placed within the text line.:
func (obj TextSpanObject) View(x, y float64, face *FontFace) Matrix {
	_, bottom := obj.Heights(face)
	return Identity.Translate(x+obj.X, y+obj.Y-bottom)
}

////////////////////////////////////////////////////////////////

func itemizeString(log string) []text.ScriptItem {
	logRunes := []rune(log)
	embeddingLevels := text.EmbeddingLevels(logRunes)
	return text.ScriptItemizer(logRunes, embeddingLevels)
}

func scriptDirection(mode WritingMode, orient TextOrientation, script text.Script, level int, direction text.Direction) (text.Direction, text.Rotation) {
	// override text direction for given writing mode
	// script and level come from ScriptItemizer
	// direction is the explicit direction set on the face
	vertical := false
	rotation := text.NoRotation
	if mode == VerticalLR || mode == VerticalRL {
		if !text.IsVerticalScript(script) && orient == Natural {
			// horizontal script with natural orientation
			rotation = text.CW
		} else if rot := text.ScriptRotation(script); rot != text.NoRotation {
			// rotated horizontal script for vertical mode (such as Mongolian)
			rotation = rot
		} else {
			// horizontal script with upright orientation or vertical script
			vertical = true
		}
	}

	if !vertical {
		if direction != text.LeftToRight && direction != text.RightToLeft {
			if (level % 2) == 1 {
				direction = text.RightToLeft
			} else {
				direction = text.LeftToRight
			}
		}
	} else {
		if direction != text.TopToBottom && direction != text.BottomToTop {
			if (level % 2) == 1 {
				direction = text.BottomToTop
			} else {
				direction = text.TopToBottom
			}
		}
	}
	return direction, rotation
}

func reorderSpans(spans []TextSpan) {
	// find runs of a certain level and deeper (including nested)
	// and reverse order for each level
	// e.g. [0 1 2 2 1 0] would first reverse order of [1 2 2 1], and then again of [2 2]
	prevLevel := 0
	for first := 0; first < len(spans); first++ {
		level := spans[first].Level
		if prevLevel < level { // every boundary of increased level
			last := first + 1
			for ; last <= len(spans); last++ {
				if last == len(spans) || spans[last].Level < level {
					if 1 < last-first {
						// reverse position of spans
						var x float64
						if (level % 2) == 1 {
							x = spans[first].X
						} else {
							x = spans[last-1].X
						}
						for i := last - 1; first <= i; i-- {
							spans[i].X = x
							x += spans[i].Width
						}
					}
					break
				}
			}
		}
		prevLevel = level
	}
}

// NewTextLine is a simple text line using a single font face, a string (supporting new lines) and horizontal alignment (Left, Center, Right). The text's baseline will be drawn on the current coordinate.
func NewTextLine(face *FontFace, s string, halign TextAlign) *Text {
	t := &Text{
		fonts: map[*Font]bool{face.Font: true},
		Text:  s,
	}

	ascent, descent, spacing := face.Metrics().Ascent, face.Metrics().Descent, face.Metrics().LineGap

	i := 0
	y := 0.0
	skipNext := false
	for j, r := range s + "\n" {
		if text.IsParagraphSeparator(r) {
			if skipNext {
				skipNext = false
				i++
				continue
			}
			if i < j {
				x := 0.0
				ppem := face.PPEM(DefaultResolution)
				line := line{y: y, spans: []TextSpan{}}
				for _, item := range itemizeString(s[i:j]) {
					direction, _ := scriptDirection(HorizontalTB, Natural, item.Script, item.Level, face.Direction)
					glyphs := face.Font.shaper.Shape(item.Text, ppem, direction, face.Script, face.Language, face.Font.features, face.Font.variations)
					width := face.textWidth(glyphs)
					line.spans = append(line.spans, TextSpan{
						X:         x,
						Width:     width,
						Face:      face,
						Text:      item.Text,
						Glyphs:    glyphs,
						Direction: direction,
						Level:     item.Level,
					})
					x += width
				}
				if halign == Center || halign == Middle {
					for k := range line.spans {
						line.spans[k].X = -x / 2.0
					}
				} else if halign == Right {
					for k := range line.spans {
						line.spans[k].X = -x
					}
				}

				// reorder runs of RTL text
				reorderSpans(line.spans)

				t.lines = append(t.lines, line)
			}
			y += ascent + descent + spacing
			i = j + utf8.RuneLen(r)
			skipNext = r == '\r' && j+1 < len(s) && s[j+1] == '\n'
		}
	}
	return t
}

// NewTextBox is an advanced text formatter that will format text placement based on the settings. It takes a single font face, a string, the width or height of the box (can be zero to disable), horizontal and vertical alignment (Left, Center, Right, Top, Bottom or Justify), text indentation for the first line and line stretch (percentage to stretch the line based on the line height).
func NewTextBox(face *FontFace, s string, width, height float64, halign, valign TextAlign, indent, lineStretch float64) *Text {
	rt := NewRichText(face)
	rt.WriteString(s)
	return rt.ToText(width, height, halign, valign, indent, lineStretch)
}

type indexer []int

func (indexer indexer) index(loc int) int {
	for index, start := range indexer {
		if loc < start {
			return index - 1
		}
	}
	return len(indexer) - 1
}

// RichText allows to build up a rich text with text spans of different font faces and fitting that into a box using Donald Knuth's line breaking algorithm.
type RichText struct {
	*strings.Builder
	locs   indexer // faces locations in string by number of runes
	faces  []*FontFace
	mode   WritingMode
	orient TextOrientation

	defaultFace *FontFace
	objects     []TextSpanObject
}

// NewRichText returns a new rich text with the given default font face.
func NewRichText(face *FontFace) *RichText {
	return &RichText{
		Builder:     &strings.Builder{},
		locs:        indexer{0},
		faces:       []*FontFace{face},
		mode:        HorizontalTB,
		orient:      Natural,
		defaultFace: face,
	}
}

// Reset resets the rich text to its initial state.
func (rt *RichText) Reset() {
	rt.Builder.Reset()
	rt.locs = rt.locs[:1]
	rt.faces = rt.faces[:1]
}

// SetWritingMode sets the writing mode.
func (rt *RichText) SetWritingMode(mode WritingMode) {
	rt.mode = mode
}

// SetTextOrientation sets the text orientation of non-CJK between CJK.
func (rt *RichText) SetTextOrientation(orient TextOrientation) {
	rt.orient = orient
}

// SetFace sets the font face.
func (rt *RichText) SetFace(face *FontFace) {
	if face == rt.faces[len(rt.faces)-1] {
		return
	}
	prevLoc := rt.locs[len(rt.locs)-1]
	if rt.Len()-prevLoc == 0 {
		rt.locs = rt.locs[:len(rt.locs)-1]
		rt.faces = rt.faces[:len(rt.faces)-1]
	}
	rt.locs = append(rt.locs, len([]rune(rt.String())))
	rt.faces = append(rt.faces, face)
}

// SetFaceSpan sets the font face between start and end measured in bytes.
func (rt *RichText) SetFaceSpan(face *FontFace, start, end int) {
	fmt.Println("WARNING: deprecated RichText.SetFaceSpan") // TODO: remove
	if end <= start || rt.Len() <= start {
		return
	} else if rt.Len() < end {
		end = rt.Len()
	}

	k := 0
	i, j := 0, len(rt.locs)-1
	for k < len(rt.locs) {
		if rt.locs[k] < start {
			i = k
		}
		if end <= rt.locs[k] {
			j = k - 1
			break
		}
		k++
	}
	rt.locs[j] = len([]rune(rt.String()[:end]))
	rt.locs = append(rt.locs[:i], append(indexer{len([]rune(rt.String()[:start]))}, rt.locs[j:]...)...)
	rt.faces = append(rt.faces[:i], append([]*FontFace{face}, rt.faces[j:]...)...)
}

// WriteFace writes a string with a given font face.
func (rt *RichText) WriteFace(face *FontFace, text string) {
	origFace := rt.faces[len(rt.faces)-1]
	rt.SetFace(face)
	rt.WriteString(text)
	rt.SetFace(origFace)
}

// WriteCanvas writes an inline canvas object.
func (rt *RichText) WriteCanvas(c *Canvas, valign VerticalAlign) {
	width, height := c.Size()
	rt.WriteRune('\uFFFC') // object replacement character
	rt.objects = append(rt.objects, TextSpanObject{
		Canvas: c,
		Width:  width,
		Height: height,
		VAlign: valign,
	})
}

// WritePath writes an inline path.
func (rt *RichText) WritePath(path *Path, col color.RGBA, valign VerticalAlign) {
	style := DefaultStyle
	style.Fill.Color = col
	bounds := path.Bounds()
	c := New(bounds.X+bounds.W, bounds.Y+bounds.H)
	c.RenderPath(path, style, Identity)
	rt.WriteCanvas(c, valign)
}

// WriteImage writes an inline image.
func (rt *RichText) WriteImage(img image.Image, res Resolution, valign VerticalAlign) {
	bounds := img.Bounds().Size()
	c := New(float64(bounds.X)/res.DPMM(), float64(bounds.Y)/res.DPMM())
	c.RenderImage(img, Identity.Scale(1.0/res.DPMM(), 1.0/res.DPMM()))
	rt.WriteCanvas(c, valign)
}

// WriteLaTeX writes an inline LaTeX formula.
func (rt *RichText) WriteLaTeX(s string) error {
	p, err := ParseLaTeX(s)
	if err != nil {
		return err
	}
	rt.WritePath(p, Black, Baseline)
	return nil
}

func (rt *RichText) Add(face *FontFace, text string) *RichText {
	fmt.Println("WARNING: deprecated RichText.Add, use RichText.WriteFace") // TODO: remove
	rt.WriteFace(face, text)
	return rt
}

func (rt *RichText) AddCanvas(c *Canvas, valign VerticalAlign) *RichText {
	fmt.Println("WARNING: deprecated RichText.AddCanvas, use RichText.WriteCanvas") // TODO: remove
	rt.WriteCanvas(c, valign)
	return rt
}

func (rt *RichText) AddPath(path *Path, col color.RGBA, valign VerticalAlign) *RichText {
	fmt.Println("WARNING: deprecated RichText.AddPath, use RichText.WritePath") // TODO: remove
	rt.WritePath(path, col, valign)
	return rt
}

func (rt *RichText) AddImage(img image.Image, res Resolution, valign VerticalAlign) *RichText {
	fmt.Println("WARNING: deprecated RichText.AddImage, use RichText.WriteImage") // TODO: remove
	rt.WriteImage(img, res, valign)
	return rt
}

func (rt *RichText) AddLaTeX(s string) *RichText {
	fmt.Println("WARNING: deprecated RichText.AddLaTeX, use RichText.WriteLaTeX") // TODO: remove
	rt.WriteLaTeX(s)
	return rt
}

type textRun struct {
	Text      string
	Level     int
	Face      *FontFace
	Script    text.Script
	Direction text.Direction
	Rotation  text.Rotation
}

// ToText takes the added text spans and fits them within a given box of certain width and height using Donald Knuth's line breaking algorithm.
func (rt *RichText) ToText(width, height float64, halign, valign TextAlign, indent, lineStretch float64) *Text {
	log := rt.String()
	logRunes := []rune(log)
	embeddingLevels := text.EmbeddingLevels(logRunes)

	// itemize string by font face and script
	// this also splits on embedding level boundaries and runs of U+FFFC (object replacement)
	i := 0       // index into logRunes
	curFace := 0 // index into rt.faces
	runs := []textRun{}
	for j := range append(logRunes, 0) {
		nextFace := rt.locs.index(j)
		if nextFace != curFace || j == len(logRunes) {
			items := text.ScriptItemizer(logRunes[i:j], embeddingLevels[i:j])
			for _, item := range items {
				direction, rotation := scriptDirection(rt.mode, rt.orient, item.Script, item.Level, rt.faces[curFace].Direction)
				runs = append(runs, textRun{
					Text:      item.Text,
					Level:     item.Level,
					Face:      rt.faces[curFace],
					Script:    item.Script,
					Direction: direction,
					Rotation:  rotation,
				})
			}
			curFace = nextFace
			i = j
		}
	}

	// shape text into glyphs and keep index into runs
	objectOffset := 0
	clusterOffset := uint32(0)
	glyphIndices := indexer{} // indexes glyphs into runs
	glyphs := make([]text.Glyph, 0, len(logRunes))
	for _, run := range runs {
		ppem := run.Face.PPEM(DefaultResolution)
		glyphRun := run.Face.Font.shaper.Shape(run.Text, ppem, run.Direction, run.Script, run.Face.Language, run.Face.Font.features, run.Face.Font.variations)
		for i, glyph := range glyphRun {
			glyphRun[i].SFNT = run.Face.Font.SFNT
			glyphRun[i].Size = run.Face.Size
			glyphRun[i].Script = run.Script
			glyphRun[i].Cluster += clusterOffset
			if glyph.Text == '\uFFFC' {
				// path/image objects
				obj := rt.objects[objectOffset]
				ppem := float64(run.Face.Font.SFNT.Head.UnitsPerEm)
				xadv, yadv := obj.Width, obj.Height
				if rt.mode != HorizontalTB {
					yadv = -yadv
				}
				glyphRun[i].Vertical = rt.mode != HorizontalTB
				glyphRun[i].XAdvance = int32(xadv * ppem / run.Face.Size)
				glyphRun[i].YAdvance = int32(yadv * ppem / run.Face.Size)
				objectOffset++
			} else {
				glyphRun[i].Vertical = run.Direction == text.TopToBottom || run.Direction == text.BottomToTop
				if rt.mode != HorizontalTB {
					if run.Script == text.Mongolian {
						glyphRun[i].YOffset += int32(run.Face.Font.SFNT.Hhea.Descender)
					} else if run.Rotation != text.NoRotation {
						// center horizontal text by x-height when rotated in vertical layout
						glyphRun[i].YOffset -= int32(run.Face.Font.SFNT.OS2.SxHeight) / 2
					} else if rt.orient == Upright && run.Rotation == text.NoRotation && !text.IsVerticalScript(run.Script) {
						// center horizontal text vertically when upright in vertical layout
						glyphRun[i].YOffset = -(int32(run.Face.Font.SFNT.Head.UnitsPerEm) + int32(run.Face.Font.SFNT.OS2.SxHeight)) / 2
					}
				}
			}
		}

		if run.Direction == text.RightToLeft || run.Direction == text.BottomToTop {
			// shaping puts characters in visual order, go back to logical order for line breaking
			for i := 0; i < len(glyphRun)/2; i++ {
				glyphRun[i], glyphRun[len(glyphRun)-1-i] = glyphRun[len(glyphRun)-1-i], glyphRun[i]
			}
		}

		glyphIndices = append(glyphIndices, len(glyphs))
		glyphs = append(glyphs, glyphRun...)
		clusterOffset += uint32(len(run.Text))
	}

	// interchange width/height and halign/valign for vertical text
	if rt.mode != HorizontalTB {
		width, height = height, width
		halign, valign = valign, halign
		if halign == Top {
			halign = Left
		} else if halign == Bottom {
			halign = Right
		}
		if valign == Left {
			valign = Top
		} else if valign == Right {
			valign = Bottom
		}
	}

	// break glyphs into lines following Donald Knuth's line breaking algorithm
	looseness := 0
	align := text.Left
	if halign == Justify {
		align = text.Justified
	}
	items := text.GlyphsToItems(glyphs, indent, align)

	var breaks []*text.Breakpoint
	var overflows bool
	if 0 < len(items) {
		if width != 0.0 {
			var ok bool
			breaks, ok = text.Linebreak(items, width, looseness)
			overflows = !ok
		} else {
			lineWidth := 0.0
			for i, item := range items {
				if item.Type != text.PenaltyType {
					lineWidth += item.Width
				} else if item.Penalty <= -text.Infinity {
					breaks = append(breaks, &text.Breakpoint{Position: i, Width: lineWidth})
					lineWidth = 0.0
				}
			}
		}
	}

	// build up lines
	t := &Text{
		fonts:           map[*Font]bool{},
		WritingMode:     rt.mode,
		TextOrientation: rt.orient,
		Width:           width,
		Height:          height,
		Text:            log,
		Overflows:       overflows,
	}
	glyphs = append(glyphs, text.Glyph{Cluster: uint32(len(log))}) // makes indexing easier

	y := 0.0
	ai, ag := 0, 0   // index into items and glyphs
	objectOffset = 0 // index into objects
	lineSpacing := 1.0 + lineStretch
	for j := range breaks {
		// j is the current line
		// [ai,bi) is the range of items
		// [ag,bg) is the range of glyphs
		eolSkip := 0 // number of glyphs after the last box

		// skip glues/penalties with no glyphs
		for ai < breaks[j].Position && items[ai].Type != text.BoxType {
			ag += items[ai].Size
			ai++
		}
		bi, bg := breaks[j].Position, ag

		// apply stretching or shrinking of glue (whitespace)
		// find run of glue/penalty and sum the width, stretch, and shrink values
		// then calculate the final stretch/shrink factor and apply to all glyphs in the run
		if breaks[j].Ratio != 0.0 {
			ag2, bg2 := ag, ag
			width, stretch, shrink := 0.0, 0.0, 0.0
			for i := ai; i <= bi; i++ {
				if i == bi || items[i].Type == text.BoxType {
					if 0.0 < width {
						adv := 0.0
						if 0.0 < breaks[j].Ratio && !math.IsInf(stretch, 0.0) {
							adv = breaks[j].Ratio * stretch
						} else if breaks[j].Ratio < 0.0 && !math.IsInf(shrink, 0.0) {
							adv = breaks[j].Ratio * shrink
						}
						breaks[j].Width += adv
						adv /= width // stretch/shrink factor
						for g := ag2; g < bg2; g++ {
							glyphs[g].XAdvance += int32(adv*float64(glyphs[g].XAdvance) + 0.5)
						}
					}
					if i == bi {
						break
					}
					width, stretch, shrink = 0.0, 0.0, 0.0
					ag2 = bg2 + items[i].Size
				} else if items[i].Type == text.GlueType {
					width += items[i].Width
					stretch += items[i].Stretch
					shrink += items[i].Shrink
				}
				bg2 += items[i].Size
			}
		}

		// skip glue/hyphens at end of line (before breakpoint)
		for _, item := range items[ai:bi] {
			if item.Type == text.BoxType {
				eolSkip = 0
			} else {
				eolSkip += item.Size
			}
			bg += item.Size
		}

		// handle breakpoint
		if items[bi].Type == text.PenaltyType && items[bi].Size == 1 && glyphs[bg].Text == '\u00AD' {
			// hyphenate at breakpoint
			// TODO: hyphen depends on script
			id := glyphs[bg].SFNT.GlyphIndex('-')
			glyphs[bg].ID = id
			glyphs[bg].XAdvance = int32(glyphs[bg].SFNT.GlyphAdvance(id))
			glyphs[bg].Text = '-'
		} else {
			eolSkip += items[bi].Size
		}
		bg += items[bi].Size
		bi++

		// absorb whitespace after breakpoint
		for bi < len(items) && items[bi].Type == text.GlueType {
			eolSkip += items[bi].Size
			bg += items[bi].Size
			bi++
		}

		// build text spans of line
		x := 0.0
		if halign == Right {
			x += width - breaks[j].Width
		} else if halign == Center || halign == Middle {
			x += (width - breaks[j].Width) / 2.0
		}
		if j == 0 {
			x += indent
		}

		line := line{}
		a := ag
		k := glyphIndices.index(a) // index into runs
		for b := a + 1; b <= bg-eolSkip; b++ {
			nextK := glyphIndices.index(b)
			if nextK != k || b == bg-eolSkip {
				run := runs[k]

				var w float64
				var objects []TextSpanObject
				if glyphs[a].Text == '\uFFFC' {
					// path/image objects
					n := b - a
					objects = make([]TextSpanObject, n)
					for i := 0; i < n; i++ {
						var obj TextSpanObject
						if run.Direction == text.RightToLeft || run.Direction == text.BottomToTop {
							// logical to visual order
							obj = rt.objects[objectOffset+(n-1-i)]
						} else {
							obj = rt.objects[objectOffset+i]
						}
						if rt.mode == HorizontalTB {
							obj.X = w
							w += obj.Width
						} else {
							obj.X = -obj.Width / 2.0
							obj.Y = -w - obj.Height
							w += obj.Height
						}
						objects[i] = obj
					}
					objectOffset += n
				} else {
					if run.Direction == text.RightToLeft || run.Direction == text.BottomToTop {
						// logical to visual order
						// this undoes the previous reversal after shaping for line breaking
						for i := 0; i < (b-a)/2; i++ {
							glyphs[a+i], glyphs[b-1-i] = glyphs[b-1-i], glyphs[a+i]
						}
					}
					w = run.Face.textWidth(glyphs[a:b])
					t.fonts[run.Face.Font] = true
				}

				ac, bc := glyphs[a].Cluster, glyphs[b].Cluster
				line.spans = append(line.spans, TextSpan{
					X:         x,
					Width:     w,
					Face:      run.Face,
					Text:      log[ac:bc],
					Objects:   objects,
					Glyphs:    glyphs[a:b],
					Direction: run.Direction,
					Rotation:  run.Rotation,
					Level:     run.Level,
				})

				k = nextK
				x += w
				a = b
			}
		}

		// set y position of line
		/*var ascent, descent, bottom float64
		if len(line.spans) == 0 {
			_, ascent, descent, bottom = runs[glyphIndices.index(i)].Face.heights(rt.mode)
		} else {
			_, ascent, descent, bottom = line.Heights(rt.mode)
		}
		if 0 < j {
			ascent *= lineSpacing
		}
		bottom *= lineSpacing
		if height != 0.0 && height < y+ascent+descent {
			// line doesn't fit
			t.Text = log[:glyphs[a].Cluster]
			break
		}*/
		faceSize := runs[glyphIndices.index(i)].Face.Size
		lineHeight := faceSize * lineSpacing * 1.13
		line.y = y + lineHeight

		// add line
		t.lines = append(t.lines, line)
		y += lineHeight

		ai, ag = bi, bg
	}

	// reorder from logical to visual order of text spans in line
	for _, line := range t.lines {
		reorderSpans(line.spans)
	}

	if 0 < len(t.lines) {
		// remove line gap of last line
		_, _, descent, bottom := t.lines[len(t.lines)-1].Heights(rt.mode)
		y += -bottom*lineSpacing + descent
	}

	// vertical align
	if rt.mode == VerticalRL {
		if valign == Top {
			valign = Bottom
		} else if valign == Bottom {
			valign = Top
		}
	}
	if valign == Center || valign == Middle || valign == Bottom {
		dy := height - y
		if valign == Center || valign == Middle {
			dy /= 2.0
		}
		for j := range t.lines {
			t.lines[j].y += dy
		}
	} else if valign == Justify {
		ddy := (height - y) / float64(len(t.lines)-1)
		dy := 0.0
		for j := range t.lines {
			t.lines[j].y += dy
			dy += ddy
		}
	}
	if rt.mode == VerticalRL {
		for j := range t.lines {
			t.lines[j].y = height - t.lines[j].y
		}
	}
	return t
}

// String returns the content of the text box.
func (t *Text) String() string {
	return t.Text
}

// Empty returns true if there are no text lines or text spans.
func (t *Text) Empty() bool {
	for _, line := range t.lines {
		if len(line.spans) != 0 {
			return false
		}
	}
	return true
}

// Lines returns the number of text lines of the text box.
func (t *Text) Lines() int {
	return len(t.lines)
}

// Size returns the width and height of a text box. Either can be zero when unspecified.
func (t *Text) Size() (float64, float64) {
	return t.Width, t.Height
}

// Heights returns the top and bottom position of the first and last line respectively.
func (t *Text) Heights() (float64, float64) {
	if len(t.lines) == 0 {
		return 0.0, 0.0
	}
	firstLine := t.lines[0]
	lastLine := t.lines[len(t.lines)-1]
	_, ascent, _, _ := firstLine.Heights(t.WritingMode)
	_, _, descent, _ := lastLine.Heights(t.WritingMode)
	return -firstLine.y + ascent, lastLine.y + descent
}

// Bounds returns the bounding rectangle that defines the text box.
func (t *Text) Bounds() Rect {
	if t.Empty() {
		return Rect{}
	}
	rect := Rect{}
	for _, line := range t.lines {
		for _, span := range line.spans {
			// TODO: vertical text
			rect = rect.Add(Rect{span.X, -line.y - span.Face.Metrics().Descent, span.Width, span.Face.Metrics().Ascent + span.Face.Metrics().Descent})
		}
	}
	return rect
}

// OutlineBounds returns the rectangle that contains the entire text box, i.e. the glyph outlines (slow).
func (t *Text) OutlineBounds() Rect {
	if t.Empty() {
		return Rect{}
	}
	r := Rect{}
	for _, line := range t.lines {
		for _, span := range line.spans {
			// TODO: vertical text
			p, _, err := span.Face.toPath(span.Glyphs, span.Face.PPEM(DefaultResolution))
			if err != nil {
				panic(err)
			}
			spanBounds := p.Bounds()
			spanBounds = spanBounds.Move(Point{span.X, -line.y})
			r = r.Add(spanBounds)
		}
	}
	t.WalkDecorations(func(_ Paint, p *Path) {
		r = r.Add(p.Bounds())
	})
	return r
}

// Fonts returns the list of fonts used.
func (t *Text) Fonts() []*Font {
	fonts := []*Font{}
	fontNames := []string{}
	fontMap := map[string]*Font{}
	for font := range t.fonts {
		name := font.Name()
		fontNames = append(fontNames, name)
		fontMap[name] = font
	}
	sort.Strings(fontNames)
	for _, name := range fontNames {
		fonts = append(fonts, fontMap[name])
	}
	return fonts
}

// MostCommonFontFace returns the most common FontFace of the text.
func (t *Text) MostCommonFontFace() *FontFace {
	fonts := map[*Font]int{}
	sizes := map[float64]int{}
	styles := map[FontStyle]int{}
	variants := map[FontVariant]int{}
	colors := map[color.RGBA]int{}
	for _, line := range t.lines {
		for _, span := range line.spans {
			fonts[span.Face.Font]++
			sizes[span.Face.Size]++
			styles[span.Face.Style]++
			variants[span.Face.Variant]++
			if span.Face.Fill.IsColor() {
				colors[span.Face.Fill.Color]++ // TODO: also for patterns or other fill paints
			}
		}
	}
	if len(fonts) == 0 {
		return nil
	}

	font, size, style, variant, col := (*Font)(nil), 0.0, FontRegular, FontNormal, Black
	for key, val := range fonts {
		if fonts[font] < val {
			font = key
		}
	}
	for key, val := range sizes {
		if sizes[size] < val {
			size = key
		}
	}
	for key, val := range styles {
		if styles[style] < val {
			style = key
		}
	}
	for key, val := range variants {
		if variants[variant] < val {
			variant = key
		}
	}
	for key, val := range colors {
		if colors[col] < val {
			col = key
		}
	}

	face := font.Face(size*ptPerMm, col)
	face.Style = style
	face.Variant = variant
	return face
}

type decorationSpan struct {
	deco   FontDecorator
	fill   Paint
	x0, x1 float64
	face   *FontFace // biggest face
}

// WalkDecorations calls the callback for each color of decoration used per line.
func (t *Text) WalkDecorations(callback func(fill Paint, deco *Path)) {
	// TODO: vertical text
	// accumulate paths and fill paints for all lines
	ps := []*Path{}
	fs := []Paint{}
	for _, line := range t.lines {
		// track active decorations, when finished draw and append to accumulated paths
		active := []decorationSpan{}
		for k, span := range line.spans {
			foundActive := make([]bool, len(active))
			for _, spanDeco := range span.Face.Deco {
				found := false
				for i, deco := range active {
					if reflect.DeepEqual(span.Face.Fill, deco.fill) && reflect.DeepEqual(deco.deco, spanDeco) {
						// extend decoration
						active[i].x0 = math.Min(active[i].x0, span.X)
						active[i].x1 = math.Max(active[i].x1, span.X+span.Width)
						if active[i].face.Size < span.Face.Size {
							active[i].face = span.Face
						}
						foundActive[i] = true
						found = true
						break
					}
				}
				if !found {
					// add new decoration
					active = append(active, decorationSpan{
						deco: spanDeco,
						fill: span.Face.Fill,
						x0:   span.X,
						x1:   span.X + span.Width,
						face: span.Face,
					})
				}
			}

			if k == len(line.spans)-1 {
				foundActive = make([]bool, len(active))
			}

			di := 0
			for i, found := range foundActive {
				if !found {
					// remove active decoration and draw it
					decoSpan := active[i-di]
					xOffset := span.Face.MmPerEm * float64(span.Face.XOffset)
					yOffset := span.Face.MmPerEm * float64(span.Face.YOffset)
					p := decoSpan.deco.Decorate(decoSpan.face, decoSpan.x1-decoSpan.x0)
					p = p.Translate(decoSpan.x0+xOffset, -line.y+yOffset)

					foundFill := false
					for j, fill := range fs {
						if reflect.DeepEqual(fill, decoSpan.fill) {
							ps[j] = ps[j].Append(p)
							foundFill = true
						}
					}
					if !foundFill {
						fs = append(fs, decoSpan.fill)
						ps = append(ps, p)
					}

					active = append(active[:i-di], active[i-di+1:]...)
					di++
				}
			}
		}
	}

	for i := 0; i < len(ps); i++ {
		callback(fs[i], ps[i])
	}
}

// WalkLines calls the callback for each text line.
func (t *Text) WalkLines(callback func(float64, []TextSpan)) {
	for _, line := range t.lines {
		callback(-line.y, line.spans)
	}
}

// WalkSpans calls the callback for each text span per line.
func (t *Text) WalkSpans(callback func(float64, float64, TextSpan)) {
	for _, line := range t.lines {
		for _, span := range line.spans {
			xOffset := span.Face.MmPerEm * float64(span.Face.XOffset)
			yOffset := span.Face.MmPerEm * float64(span.Face.YOffset)
			if t.WritingMode == HorizontalTB {
				callback(span.X+xOffset, -line.y+yOffset, span)
			} else {
				callback(line.y+xOffset, -span.X+yOffset, span)
			}
		}
	}
}

// RenderAsPath renders the text and its decorations converted to paths, calling r.RenderPath.
func (t *Text) RenderAsPath(r Renderer, m Matrix, resolution Resolution) {
	t.WalkDecorations(func(paint Paint, p *Path) {
		style := DefaultStyle
		style.Fill = paint
		r.RenderPath(p, style, m)
	})

	for _, line := range t.lines {
		for _, span := range line.spans {
			x, y := span.X, -line.y
			if t.WritingMode != HorizontalTB {
				x, y = line.y, -span.X
			}

			if span.IsText() {
				style := DefaultStyle
				style.Fill = span.Face.Fill
				p, _, err := span.Face.toPath(span.Glyphs, span.Face.PPEM(resolution))
				if err != nil {
					panic(err)
				}
				if span.Rotation != 0.0 {
					p = p.Transform(Identity.Rotate(float64(span.Rotation)))
				}
				if resolution != 0.0 && span.Face.Hinting != font.NoHinting && span.Rotation == text.NoRotation {
					// grid-align vertically on pixel raster, this improves font sharpness
					_, dy := m.Pos()
					dy += y
					y += float64(int(dy*resolution.DPMM()+0.5))/resolution.DPMM() - dy
				}
				p = p.Translate(x, y)
				r.RenderPath(p, style, m)
			} else {
				for _, obj := range span.Objects {
					obj.RenderViewTo(r, m.Mul(obj.View(x, y, span.Face)))
				}
			}
		}
	}
}
