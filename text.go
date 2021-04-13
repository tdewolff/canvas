package canvas

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	canvasText "github.com/tdewolff/canvas/text"
)

// TextAlign specifies how the text should align or whether it should be justified.
type TextAlign int

// see TextAlign
const (
	Left TextAlign = iota
	Right
	Center
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
	case Top:
		return "Top"
	case Bottom:
		return "Bottom"
	case Justify:
		return "Justify"
	}
	return "Invalid(" + strconv.Itoa(int(ta)) + ")"
}

// WritingMode specifies how the text should be layed out.
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

// Text holds the representation of text using lines and text spans.
type Text struct {
	lines []line
	fonts map[*Font]bool
	Face  *FontFace
	Mode  WritingMode
}

type line struct {
	y     float64
	spans []TextSpan
}

func (l line) Heights() (float64, float64, float64, float64) {
	top, ascent, descent, bottom := 0.0, 0.0, 0.0, 0.0
	for _, span := range l.spans {
		spanAscent, spanDescent, lineSpacing := span.Face.Metrics().Ascent, span.Face.Metrics().Descent, span.Face.Metrics().LineGap
		top = math.Max(top, spanAscent+lineSpacing)
		ascent = math.Max(ascent, spanAscent)
		descent = math.Max(descent, spanDescent)
		bottom = math.Max(bottom, spanDescent+lineSpacing)
	}
	return top, ascent, descent, bottom
}

type TextSpan struct {
	x         float64
	Width     float64
	Face      *FontFace
	Text      string
	Glyphs    []canvasText.Glyph
	Direction canvasText.Direction
}

////////////////////////////////////////////////////////////////

// NewTextLine is a simple text line using a font face, a string (supporting new lines) and horizontal alignment (Left, Center, Right).
func NewTextLine(face *FontFace, s string, halign TextAlign) *Text {
	t := &Text{
		fonts: map[*Font]bool{face.Font: true},
		Face:  face,
	}

	ascent, descent, spacing := face.Metrics().Ascent, face.Metrics().Descent, face.Metrics().LineGap

	i := 0
	y := 0.0
	skipNext := false
	for j, r := range s + "\n" {
		if canvasText.IsParagraphSeparator(r) {
			if skipNext {
				skipNext = false
				i++
				continue
			}
			if i < j {
				ppem := face.PPEM(DefaultResolution)
				lineWidth := 0.0
				line := line{y: y, spans: []TextSpan{}}
				itemsL, itemsV := itemizeString(s[i:j]) // TODO: use same as RichText?
				for k := 0; k < len(itemsL); k++ {
					glyphs := face.Font.shaper.Shape(itemsV[k], ppem, face.Direction, face.Script, face.Language, face.Font.features, face.Font.variations)
					width := face.textWidth(glyphs)
					text := itemsL[k]
					if face.Direction == canvasText.BottomToTop {
						length := len([]rune(text))
						reverseText := make([]rune, length)
						for pos, r := range []rune(text) {
							reverseText[length-pos-1] = r
						}
						text = string(reverseText)
					}
					line.spans = append(line.spans, TextSpan{
						x:         lineWidth,
						Width:     width,
						Face:      face,
						Text:      text,
						Glyphs:    glyphs,
						Direction: face.Direction,
					})
					lineWidth += width
				}
				if halign == Center {
					for k, _ := range line.spans {
						line.spans[k].x = -lineWidth / 2.0
					}
				} else if halign == Right {
					for k, _ := range line.spans {
						line.spans[k].x = -lineWidth
					}
				}
				t.lines = append(t.lines, line)
			}
			y -= ascent + descent + spacing
			i = j + utf8.RuneLen(r)
			skipNext = r == '\r' && j+1 < len(s) && s[j+1] == '\n'
		}
	}
	return t
}

// NewTextBox is an advanced text formatter that will calculate text placement based on the settings. It takes a font face, a string, the width or height of the box (can be zero for no limit), horizontal and vertical alignment (Left, Center, Right, Top, Bottom or Justify), text indentation for the first line and line stretch (percentage to stretch the line based on the line height).
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

// RichText allows to build up a rich text with text spans of different font faces and by fitting that into a box.
type RichText struct {
	*strings.Builder
	locs  indexer // faces locations ino string by number of runes
	faces []*FontFace
	mode  WritingMode
}

func NewRichText(face *FontFace) *RichText {
	return &RichText{
		Builder: &strings.Builder{},
		locs:    indexer{0},
		faces:   []*FontFace{face},
		mode:    HorizontalTB,
	}
}

func (rt *RichText) Reset() {
	rt.Builder.Reset()
	rt.locs = rt.locs[:0]
	rt.faces = rt.faces[:0]
}

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

func (rt *RichText) SetFaceSpan(face *FontFace, start, end int) {
	// TODO: optimize when face already is on (part of) the span
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

func (rt *RichText) Add(face *FontFace, text string) *RichText {
	rt.SetFace(face)
	rt.WriteString(text)
	return rt
}

func (rt *RichText) SetWritingMode(mode WritingMode) {
	rt.mode = mode
}

func writingModeDirection(mode WritingMode, direction canvasText.Direction) canvasText.Direction {
	if direction == canvasText.TopToBottom || direction == canvasText.BottomToTop {
		if mode == HorizontalTB {
			return canvasText.LeftToRight
		} else {
			return canvasText.TopToBottom
		}
	} else if mode != HorizontalTB {
		// unknown, left to right, right to left
		return canvasText.TopToBottom
	}
	return direction
}

// ToText takes the added text spans and fits them within a given box of certain width and height.
func (rt *RichText) ToText(width, height float64, halign, valign TextAlign, indent, lineStretch float64) *Text {
	log := rt.String()
	vis, mapV2L := canvasText.Bidi(log)
	logRunes := []rune(log)

	// itemize string by font face and script
	texts := []string{}
	faces := []*FontFace{}
	i, j := 0, 0 // index into vis
	curFace := 0 // index into rt.faces
	for k, r := range []rune(vis) {
		nextFace := rt.locs.index(mapV2L[k])
		if nextFace != curFace {
			scriptItems := canvasText.ScriptItemizer(vis[i:j])
			texts = append(texts, scriptItems...)
			for _, s := range scriptItems {
				faces = append(faces, rt.faces[curFace])
				i += len(s)
			}
			curFace = nextFace
			i = j
		}
		j += utf8.RuneLen(r)
	}
	if i < j {
		scriptItems := canvasText.ScriptItemizer(vis[i:j])
		texts = append(texts, scriptItems...)
		for _, s := range scriptItems {
			faces = append(faces, rt.faces[curFace])
			i += len(s)
		}
	}

	// shape text into glyphs and keep index into texts and faces
	clusterOffset := uint32(0)
	glyphIndices := indexer{} // indexes glyphs into texts and faces
	glyphs := []canvasText.Glyph{}
	for k, text := range texts {
		face := faces[k]
		ppem := face.PPEM(DefaultResolution)
		direction := writingModeDirection(rt.mode, face.Direction)
		glyphsString := face.Font.shaper.Shape(text, ppem, direction, face.Script, face.Language, face.Font.features, face.Font.variations)
		for i, _ := range glyphsString {
			glyphsString[i].SFNT = face.Font.SFNT
			glyphsString[i].Size = face.Size * face.XScale
			glyphsString[i].Cluster += clusterOffset
		}
		glyphIndices = append(glyphIndices, len(glyphs))
		glyphs = append(glyphs, glyphsString...)
		clusterOffset += uint32(len(text))
	}

	// break glyphs into lines following Donald Knuth's line breaking algorithm
	align := canvasText.Left
	if halign == Justify {
		align = canvasText.Justified
	}
	vertical := rt.mode != HorizontalTB
	looseness := 0
	items := canvasText.GlyphsToItems(glyphs, indent, align, vertical)
	breaks := canvasText.Linebreak(items, width, looseness)

	// build up lines
	t := &Text{
		lines: []line{line{}},
		fonts: map[*Font]bool{},
		Face:  faces[0],
		Mode:  rt.mode,
	}
	glyphs = append(glyphs, canvasText.Glyph{Cluster: uint32(len(vis))}) // makes indexing easier

	i, j = 0, 0 // index into: glyphs, breaks/lines
	atStart := true
	x, y := 0.0, 0.0 // both positive toward the bottom right
	lineSpacing := 1.0 + lineStretch
	if halign == Right {
		x += width - breaks[j].Width
	}
	for position, item := range items {
		if position == breaks[j].Position {
			// add spaces to previous span
			for _, glyph := range glyphs[i : i+item.Size] {
				if glyph.Text != "\u200B" {
					t.lines[j].spans[len(t.lines[j].spans)-1].Text += glyph.Text
				}
			}

			if item.Type == canvasText.PenaltyType && item.Flagged && item.Width != 0.0 {
				if 0 < len(t.lines[j].spans) {
					span := &t.lines[j].spans[len(t.lines[j].spans)-1]
					id := span.Face.Font.GlyphIndex('-')
					glyph := canvasText.Glyph{
						SFNT:     span.Face.Font.SFNT,
						Size:     span.Face.Size * span.Face.XScale,
						ID:       id,
						XAdvance: int32(span.Face.Font.GlyphAdvance(id)),
						Text:     "-",
					}
					span.Glyphs = append(span.Glyphs, glyph)
					span.Width += span.Face.textWidth([]canvasText.Glyph{glyph})
					span.Text += "-"
				}
			}

			_, ascent, _, bottom := t.lines[j].Heights()
			if 0 < j {
				ascent *= lineSpacing
			}
			bottom *= lineSpacing

			t.lines[j].y = y + ascent
			y += ascent + bottom
			if height < y || position == len(items)-1 {
				// doesn't fit or at the end of items
				break
			}

			t.lines = append(t.lines, line{})
			if j+1 < len(breaks) {
				j++
			}
			x = 0.0
			if halign == Right {
				x += width - breaks[j].Width
			} else if halign == Center {
				x += (width - breaks[j].Width) / 2.0
			}
			atStart = true
		} else if item.Type == canvasText.BoxType {
			// find index k into faces/texts
			a := i
			dx := 0.0
			k := glyphIndices.index(i)
			for b := i + 1; b <= i+item.Size; b++ {
				nextK := glyphIndices.index(b)
				if nextK != k || b == i+item.Size {
					ac, bc := glyphs[a].Cluster, glyphs[b].Cluster
					if glyphs[a+1].Cluster < ac {
						// right-to-left
						ac = glyphs[b-1].Cluster
						bc = uint32(len(vis))
						if 0 < a {
							bc = glyphs[a-1].Cluster
						}
					}
					ar := utf8.RuneCountInString(vis[:ac])
					br := utf8.RuneCountInString(vis[:bc])

					s := string(logRunes[ar:br])
					w := faces[k].textWidth(glyphs[a:b])
					t.lines[j].spans = append(t.lines[j].spans, TextSpan{
						x:         x + dx,
						Width:     w,
						Face:      faces[k],
						Text:      s,
						Glyphs:    glyphs[a:b],
						Direction: writingModeDirection(rt.mode, faces[k].Direction),
					})
					t.fonts[faces[k].Font] = true
					k = nextK

					a = b
					dx += w
				}
			}
			atStart = false
			x += item.Width
		} else if item.Type == canvasText.GlueType && !atStart {
			width := item.Width
			if 0.0 <= breaks[j].Ratio {
				if !math.IsInf(item.Stretch, 0.0) {
					width += breaks[j].Ratio * item.Stretch
				}
			} else if !math.IsInf(item.Shrink, 0.0) {
				width += breaks[j].Ratio * item.Shrink
			}
			x += width

			// add spaces to previous span
			for _, glyph := range glyphs[i : i+item.Size] {
				t.lines[j].spans[len(t.lines[j].spans)-1].Text += glyph.Text
			}
		}
		i += item.Size
	}

	_, ascent, descent, bottom := t.lines[j].Heights()
	y -= bottom * lineSpacing

	if height < y+descent {
		// doesn't fit
		t.lines = t.lines[:len(t.lines)-1]
		if 0 < j {
			_, _, descent2, bottom2 := t.lines[j-1].Heights()
			y += descent2 - (bottom2+ascent)*lineSpacing
		} else {
			// no lines at all
			y = 0.0
		}
	} else {
		y += descent
	}

	// vertical align
	if valign == Center || valign == Bottom {
		dy := height - y
		if valign == Center {
			dy /= 2.0
		}
		for j, _ := range t.lines {
			t.lines[j].y += dy
		}
	} else if valign == Justify {
		ddy := (height - y) / float64(len(t.lines)-1)
		dy := 0.0
		for j, _ := range t.lines {
			t.lines[j].y += dy
			dy += ddy
		}
	}
	if rt.mode == VerticalRL {
		for j, _ := range t.lines {
			t.lines[j].y = width - t.lines[j].y
		}
	}
	return t
}

// Empty is true if there are no text lines or no text spans.
func (t *Text) Empty() bool {
	for _, line := range t.lines {
		if len(line.spans) != 0 {
			return false
		}
	}
	return true
}

// Height returns the height of the text using the font metrics, this is usually more than the bounds of the glyph outlines.
func (t *Text) Heights() (float64, float64) {
	if len(t.lines) == 0 {
		return 0.0, 0.0
	}
	firstLine := t.lines[0]
	lastLine := t.lines[len(t.lines)-1]
	_, ascent, _, _ := firstLine.Heights()
	_, _, descent, _ := lastLine.Heights()
	return -firstLine.y + ascent, lastLine.y + descent
}

// Bounds returns the bounding rectangle that defines the text box.
func (t *Text) Bounds() Rect {
	if len(t.lines) == 0 || len(t.lines[0].spans) == 0 {
		return Rect{}
	}
	rect := Rect{}
	for _, line := range t.lines {
		for _, span := range line.spans {
			rect = rect.Add(Rect{span.x, -line.y - span.Face.Metrics().Descent, span.Width, span.Face.Metrics().Ascent + span.Face.Metrics().Descent})
		}
	}
	return rect
}

// OutlineBounds returns the rectangle that contains the entire text box, ie. the glyph outlines (slow).
//func (t *Text) OutlineBounds() Rect {
//	if len(t.lines) == 0 || len(t.lines[0].spans) == 0 {
//		return Rect{}
//	}
//	r := Rect{}
//	for _, line := range t.lines {
//		for _, span := range line.spans {
//			spanBounds := span.Bounds(span.w)
//			spanBounds = spanBounds.Move(Point{span.x, line.y})
//			r = r.Add(spanBounds)
//		}
//	}
//	return r
//}

// Fonts returns list of fonts used.
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

// MostCommonFontFace returns the most common FontFace of the text
//func (t *Text) MostCommonFontFace() FontFace {
//	families := map[*FontFamily]int{}
//	sizes := map[float64]int{}
//	styles := map[FontStyle]int{}
//	variants := map[FontVariant]int{}
//	colors := map[color.RGBA]int{}
//	for _, line := range t.lines {
//		for _, span := range line.spans {
//			families[span.Face.family]++
//			sizes[span.Face.Size]++
//			styles[span.Face.Style]++
//			variants[span.Face.Variant]++
//			colors[span.Face.Color]++
//		}
//	}
//	if len(families) == 0 {
//		return FontFace{}
//	}
//
//	family, size, style, variant, col := (*FontFamily)(nil), 0.0, FontRegular, FontNormal, Black
//	for key, val := range families {
//		if families[family] < val {
//			family = key
//		}
//	}
//	for key, val := range sizes {
//		if sizes[size] < val {
//			size = key
//		}
//	}
//	for key, val := range styles {
//		if styles[style] < val {
//			style = key
//		}
//	}
//	for key, val := range variants {
//		if variants[variant] < val {
//			variant = key
//		}
//	}
//	for key, val := range colors {
//		if colors[col] < val {
//			col = key
//		}
//	}
//	return family.Face(size*ptPerMm, col, style, variant)
//}

// RenderAsPath renders the text (and its decorations) converted to paths (calling r.RenderPath)
func (t *Text) RenderAsPath(r Renderer, m Matrix) {
	style := DefaultStyle
	for _, line := range t.lines {
		for _, span := range line.spans {
			style.FillColor = span.Face.Color

			p, _, err := span.Face.toPath(span.Glyphs, span.Face.PPEM(DefaultResolution))
			if err != nil {
				panic(err)
			}
			if t.Mode == HorizontalTB {
				p = p.Translate(span.x, -line.y)
			} else {
				p = p.Translate(line.y, -span.x)
			}
			r.RenderPath(p, style, m)

			if span.Face.HasDecoration() {
				p = span.Face.Decorate(span.Width)
				if t.Mode == HorizontalTB {
					p = p.Translate(span.x, -line.y)
				} else {
					p = p.Translate(line.y, -span.x)
				}
				r.RenderPath(p, style, m)
			}
		}
	}
}

// RenderDecoration renders the text decorations using the RenderPath method of the Renderer.
// TODO: check text decoration z-positions when text lines are overlapping https://github.com/tdewolff/canvas/pull/40#pullrequestreview-400951503
// TODO: check compliance with https://drafts.csswg.org/css-text-decor-4/#text-line-constancy
//func (t *Text) RenderDecoration(r Renderer, m Matrix) {
//	style := DefaultStyle
//	for _, line := range t.lines {
//		for _, deco := range line.decos {
//			p := deco.face.Decorate(deco.x1 - deco.x0)
//			p = p.Translate(deco.x0, line.y+deco.face.Voffset)
//			style.FillColor = deco.face.Color
//			r.RenderPath(p, style, m)
//		}
//	}
//}

func (t *Text) WalkSpans(callback func(y, x float64, span TextSpan)) {
	for _, line := range t.lines {
		for _, span := range line.spans {
			if t.Mode == HorizontalTB {
				callback(-line.y, span.x, span)
			} else {
				callback(-span.x, line.y, span)
			}
		}
	}
}

//func (t *Text) WalkLines(spanCallback func(y, dx float64, span TextSpan), renderDeco func(path *Path, style Style, m Matrix), m Matrix) {
//	decoStyle := DefaultStyle
//	for _, line := range t.lines {
//		for _, span := range line.spans {
//			spanCallback(line.y, span.dx, span)
//		}
//		for _, deco := range line.decos {
//			p := deco.face.Decorate(deco.x1 - deco.x0)
//			p = p.Translate(deco.x0, line.y+deco.face.Voffset)
//			decoStyle.FillColor = deco.face.Color
//			renderDeco(p, decoStyle, m)
//		}
//	}
//}

func itemizeString(log string) ([]string, []string) {
	offset := 0
	vis, mapV2L := canvasText.Bidi(log)
	itemsV := canvasText.ScriptItemizer(vis)
	itemsL := make([]string, 0, len(itemsV))
	for _, item := range itemsV {
		itemV := []rune(item)
		itemL := make([]rune, len(itemV))
		for i := 0; i < len(itemV); i++ {
			itemL[mapV2L[offset+i]-offset] = itemV[i]
		}
		itemsL = append(itemsL, string(itemL))
		offset += len(itemV)
	}
	return itemsL, itemsV
}
