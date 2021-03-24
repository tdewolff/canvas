package canvas

import (
	"fmt"
	"math"
	"sort"
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

// Text holds the representation of text using lines and text spans.
type Text struct {
	lines []line
	fonts map[*Font]bool
	Face  *FontFace
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
	x      float64
	Width  float64
	Face   *FontFace
	Text   string
	Glyphs []canvasText.Glyph
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
				ppem := face.PPEM(DefaultDPMM)
				lineWidth := 0.0
				line := line{y: y, spans: []TextSpan{}}
				itemsL, itemsV := itemizeString(s[i:j])
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
						x:      lineWidth,
						Width:  width,
						Face:   face,
						Text:   text,
						Glyphs: glyphs,
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
}

func NewRichText(face *FontFace) *RichText {
	return &RichText{
		Builder: &strings.Builder{},
		locs:    indexer{0},
		faces:   []*FontFace{face},
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

// ToText takes the added text spans and fits them within a given box of certain width and height.
func (rt *RichText) ToText(width, height float64, halign, valign TextAlign, indent, lineSpacing float64) *Text {
	mainDirection := rt.faces[0].Direction
	if mainDirection != canvasText.LeftToRight && mainDirection != canvasText.RightToLeft && mainDirection != canvasText.TopToBottom && mainDirection != canvasText.BottomToTop {
		mainDirection = canvasText.LeftToRight
	}
	if mainDirection == canvasText.TopToBottom || mainDirection == canvasText.BottomToTop {
		for _, face := range rt.faces {
			if face.Direction == canvasText.LeftToRight || face.Direction == canvasText.RightToLeft {
				mainDirection = face.Direction
				break
			}
		}
	}

	vis, mapV2L := canvasText.Bidi(rt.String())

	// itemize string by font face and script
	texts := []string{}
	faces := []*FontFace{}
	i, j := 0, 0 // index into visString
	curFace := 0 // index into rt.faces
	for k, r := range []rune(vis) {
		nextFace := rt.locs.index(mapV2L[k])
		if nextFace != curFace {
			scriptItems := canvasText.ScriptItemizer(vis[i:j])
			texts = append(texts, scriptItems...)
			for _ = range scriptItems {
				faces = append(faces, rt.faces[curFace])
			}
			curFace = nextFace
			i = j
		}
		j += utf8.RuneLen(r)
	}
	if i < j {
		scriptItems := canvasText.ScriptItemizer(vis[i:j])
		texts = append(texts, scriptItems...)
		for _ = range scriptItems {
			faces = append(faces, rt.faces[curFace])
		}
	}

	// shape text into glyphs and keep index into texts and faces
	indexes := indexer{} // indexes glyphs into texts and faces
	rtls := []bool{}
	glyphs := []canvasText.Glyph{}
	for k, text := range texts {
		face := faces[k]
		ppem := face.PPEM(DefaultDPMM)
		direction := face.Direction
		if direction == canvasText.TopToBottom || direction == canvasText.BottomToTop {
			direction = mainDirection
		}
		glyphsString := face.Font.shaper.Shape(text, ppem, direction, face.Script, face.Language, face.Font.features, face.Font.variations)
		for i, _ := range glyphsString {
			glyphsString[i].SFNT = face.Font.SFNT
			glyphsString[i].Size = face.Size * face.XScale
		}
		rtl := 0 < len(glyphsString) && glyphsString[len(glyphsString)-1].Cluster < glyphsString[0].Cluster
		indexes = append(indexes, len(glyphs))
		rtls = append(rtls, rtl)
		glyphs = append(glyphs, glyphsString...)
	}

	// break glyphs into lines following Donald Knuth's line breaking algorithm
	align := canvasText.Left
	if halign == Right {
		align = canvasText.Right
	} else if halign == Center {
		align = canvasText.Centered
	} else if halign == Justify {
		align = canvasText.Justified
	}
	items := canvasText.GlyphsToItems(glyphs, indent, align)
	breaks := canvasText.Linebreak(items, width, 0)

	// build up lines
	t := &Text{
		lines: []line{line{}},
		fonts: map[*Font]bool{},
	}

	i, j = 0, 0 // index into: glyphs, breaks/lines
	atStart := true
	x, y := 0.0, 0.0
	if halign == Right {
		x += width - breaks[j].Width
	}
	for position, item := range items {
		if position == breaks[j].Position {
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

			t.lines[j].y = y - ascent
			y -= ascent + bottom
			if height < -y || position == len(items)-1 {
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
			}
			atStart = true
		} else if item.Type == canvasText.BoxType {
			// find index k into faces/texts
			a := i
			k := indexes.index(i)
			for b := i + 1; b <= i+item.Size; b++ {
				nextK := indexes.index(b)
				if nextK != k || b == i+item.Size {
					var at, bt uint32
					if !rtls[k] {
						at = glyphs[a].Cluster
						bt = uint32(len(texts[k]))
						if b < len(glyphs) {
							bt = glyphs[b].Cluster
						}
					} else {
						at = uint32(0)
						if 0 < b {
							at = glyphs[b-1].Cluster
						}
						bt = uint32(len(texts[k]))
						if 0 < a {
							bt = glyphs[a-1].Cluster
						}
					}
					t.lines[j].spans = append(t.lines[j].spans, TextSpan{
						x:      x,
						Width:  faces[k].textWidth(glyphs[a:b]),
						Face:   faces[k],
						Text:   texts[k][at:bt],
						Glyphs: glyphs[a:b],
					})
					t.fonts[faces[k].Font] = true
					k = nextK
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
		}
		i += item.Size
	}

	_, ascent, descent, bottom := t.lines[j].Heights()
	y += bottom * lineSpacing

	if height < -y+descent {
		// doesn't fit
		t.lines = t.lines[:len(t.lines)-1]
		if 0 < j {
			_, _, descent2, bottom2 := t.lines[j-1].Heights()
			y -= descent2 - (bottom2+ascent)*lineSpacing
		} else {
			// no lines at all
			y = 0.0
		}
	} else {
		y -= descent
	}

	// TODO: test vertical text
	fmt.Println("lines:")
	for j, line := range t.lines {
		fmt.Println(j, line.y)
		for _, span := range line.spans {
			fmt.Printf(" %v %v %v\n", span.x, span.Width, span.Text)
		}
	}

	// vertical align
	if valign == Center || valign == Bottom {
		dy := height + y
		if valign == Center {
			dy /= 2.0
		}
		for j, _ := range t.lines {
			t.lines[j].y -= dy
		}
	} else if valign == Justify {
		ddy := (height + y) / float64(len(t.lines)-1)
		dy := 0.0
		for j, _ := range t.lines {
			t.lines[j].y -= dy
			dy += ddy
		}
	}
	return t
}

//// RichText allows to build up a rich text with text spans of different font faces and by fitting that into a box.
//type RichText struct {
//	spans []TextSpan
//	fonts map[*Font]bool
//	text  string
//}
//
//// NewRichText returns a new RichText.
//func NewRichText() *RichText {
//	return &RichText{
//		fonts: map[*Font]bool{},
//	}
//}
//
//// Add adds a new text span element.
//func (rt *RichText) Add(ff FontFace, s string) *RichText {
//	if 0 < len(s) {
//		rPrev := ' '
//		rNext, size := utf8.DecodeRuneInString(s)
//		if 0 < len(rt.text) {
//			rPrev, _ = utf8.DecodeLastRuneInString(rt.text)
//		}
//		if isWhitespace(rPrev) && isWhitespace(rNext) {
//			s = s[size:]
//		}
//	}
//
//	start := len(rt.text)
//	rt.text += s
//
//	// TODO: can we simplify this? Just merge adjacent spans, don't split at newlines or sentences?
//	i := 0
//	for _, boundary := range calcTextBoundaries(s, 0, len(s)) {
//		if boundary.kind == lineBoundary || boundary.kind == sentenceBoundary || boundary.kind == eofBoundary {
//			j := boundary.pos + boundary.size
//			if i < j {
//				extendPrev := false
//				if i == 0 && boundary.kind != lineBoundary && 0 < len(rt.spans) && rt.spans[len(rt.spans)-1].Face.Equals(ff) {
//					prevSpan := rt.spans[len(rt.spans)-1]
//					if 1 < len(prevSpan.boundaries) {
//						prevBoundaryKind := prevSpan.boundaries[len(prevSpan.boundaries)-2].kind
//						if prevBoundaryKind != lineBoundary && prevBoundaryKind != sentenceBoundary {
//							extendPrev = true
//						}
//					} else {
//						extendPrev = true
//					}
//				}
//
//				if extendPrev {
//					diff := len(rt.spans[len(rt.spans)-1].Text)
//					rt.spans[len(rt.spans)-1] = newTextSpan(ff, rt.text[:start+j], start+i-diff)
//				} else {
//					rt.spans = append(rt.spans, newTextSpan(ff, rt.text[:start+j], start+i))
//				}
//			}
//			i = j
//		}
//	}
//	rt.fonts[ff.Font] = true
//	return rt
//}
//
//func (rt *RichText) valign(lines []line, h, height float64, valign TextAlign) {
//	dy := 0.0
//	extraLineSpacing := 0.0
//	if height != 0.0 && (valign == Bottom || valign == Center || valign == Justify) {
//		if valign == Bottom {
//			dy = height - h
//		} else if valign == Center {
//			dy = (height - h) / 2.0
//		} else if len(lines) > 1 {
//			extraLineSpacing = (height - h) / float64(len(lines)-1)
//		}
//	}
//	for j := range lines {
//		lines[j].y -= dy + float64(j)*extraLineSpacing
//	}
//}
//
//func (rt *RichText) decorate(lines []line) {
//	for j, line := range lines {
//		ff := FontFace{}
//		x0, x1 := 0.0, 0.0
//		for _, span := range line.spans {
//			if 0.0 < x1-x0 && !span.Face.Equals(ff) {
//				if ff.deco != nil {
//					lines[j].decos = append(lines[j].decos, decoSpan{ff, x0, x1})
//				}
//				x0 = x1
//			}
//			ff = span.Face
//			if x0 == x1 {
//				x0 = span.dx // skip space when starting new decoSpan
//			}
//			x1 = span.dx + span.width
//		}
//		if 0.0 < x1-x0 && ff.deco != nil {
//			lines[j].decos = append(lines[j].decos, decoSpan{ff, x0, x1})
//		}
//	}
//}
//
//// ToText takes the added text spans and fits them within a given box of certain width and height.
//func (rt *RichText) ToText(width, height float64, halign, valign TextAlign, indent, lineStretch float64) *Text {
//	if len(rt.spans) == 0 {
//		return &Text{[]line{}, rt.fonts}
//	}
//	spans := []TextSpan{rt.spans[0]}
//
//	k := 0 // index into rt.spans and rt.positions
//	lines := []line{}
//	yoverflow := false
//	y, prevLineSpacing := 0.0, 0.0
//	for k < len(rt.spans) {
//		dx := indent
//		indent = 0.0
//
//		// trim left spaces
//		spans[0] = spans[0].TrimLeft()
//		for spans[0].Text == "" {
//			// TODO: reachable?
//			if k+1 == len(rt.spans) {
//				break
//			}
//			k++
//			spans = []TextSpan{rt.spans[k]}
//			spans[0] = spans[0].TrimLeft()
//		}
//
//		// accumulate line spans for a full line, ie. either split span1 to fit or if it fits retrieve the next span1 and repeat
//		ss := []TextSpan{}
//		for {
//			// space or inter-word splitting
//			if width != 0.0 && len(spans) == 1 {
//				// there is a width limit and we have only one (unsplit) span to process
//				var ok bool
//				spans, ok = spans[0].Split(width - dx)
//				if !ok && len(ss) != 0 {
//					// span couln't fit but this line already has a span, try next line
//					break
//				}
//			}
//
//			// if this span ends with a newline, split off that newline boundary
//			newline := 1 < len(spans[0].boundaries) && spans[0].boundaries[len(spans[0].boundaries)-2].kind == lineBoundary
//			if newline {
//				spans[0], _ = spans[0].split(len(spans[0].boundaries) - 2)
//			}
//
//			spans[0].dx = dx
//			ss = append(ss, spans[0])
//			dx += spans[0].width
//
//			spans = spans[1:]
//			if len(spans) == 0 {
//				k++
//				if k == len(rt.spans) {
//					break
//				}
//				spans = []TextSpan{rt.spans[k]}
//			} else {
//				break // span couldn't fully fit, we have a full line
//			}
//			if newline {
//				break
//			}
//		}
//
//		// trim right spaces
//		for 0 < len(ss) {
//			ss[len(ss)-1] = ss[len(ss)-1].TrimRight()
//			if 1 < len(ss) && ss[len(ss)-1].Text == "" {
//				ss = ss[:len(ss)-1]
//			} else {
//				break
//			}
//		}
//
//		l := line{ss, []decoSpan{}, 0.0}
//		top, ascent, descent, bottom := l.Heights()
//		lineSpacing := math.Max(top-ascent, prevLineSpacing)
//		if len(lines) != 0 {
//			y -= lineSpacing * (1.0 + lineStretch)
//			y -= ascent * lineStretch
//		}
//		y -= ascent
//		l.y = y
//		y -= descent * (1.0 + lineStretch)
//		prevLineSpacing = bottom - descent
//
//		if height != 0.0 && y < -height {
//			yoverflow = true
//			break
//		}
//		lines = append(lines, l)
//	}
//
//	if len(lines) == 0 {
//		return &Text{lines, rt.fonts}
//	}
//
//	// apply horizontal alignment
//	rt.halign(lines, yoverflow, width, halign)
//
//	// apply vertical alignment
//	rt.valign(lines, -y, height, valign)
//
//	// set decorations
//	rt.decorate(lines)
//
//	return &Text{lines, rt.fonts}
//}

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
	return firstLine.y + ascent, -lastLine.y + descent
}

// Bounds returns the bounding rectangle that defines the text box.
func (t *Text) Bounds() Rect {
	if len(t.lines) == 0 || len(t.lines[0].spans) == 0 {
		return Rect{}
	}
	rect := Rect{}
	for _, line := range t.lines {
		for _, span := range line.spans {
			rect = rect.Add(Rect{span.x, line.y - span.Face.Metrics().Descent, span.Width, span.Face.Metrics().Ascent + span.Face.Metrics().Descent})
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

			p, _, _ := span.Face.toPath(span.Glyphs, span.Face.PPEM(DefaultDPMM))
			p = p.Translate(span.x, line.y)
			r.RenderPath(p, style, m)

			if span.Face.HasDecoration() {
				p = span.Face.Decorate(span.Width)
				p = p.Translate(span.x, line.y)
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
			callback(line.y, span.x, span)
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

////////////////////////////////////////////////////////////////

//type decoSpan struct {
//	face   FontFace
//	x0, x1 float64
//}
//
//type TextSpan struct {
//	Face       FontFace
//	Text       string
//	width      float64
//	boundaries []textBoundary
//
//	dx              float64
//	SentenceSpacing float64
//	WordSpacing     float64
//	GlyphSpacing    float64
//}
//
//func newTextSpan(ff FontFace, text string, i int) TextSpan {
//	return TextSpan{
//		Face:            ff,
//		Text:            text[i:],
//		width:           ff.TextWidth(text[i:]),
//		boundaries:      calcTextBoundaries(text, i, len(text)),
//		dx:              0.0,
//		SentenceSpacing: 0.0,
//		WordSpacing:     0.0,
//		GlyphSpacing:    0.0,
//	}
//}
//
//func (span TextSpan) TrimLeft() TextSpan {
//	if 0 < len(span.boundaries) && span.boundaries[0].pos == 0 && span.boundaries[0].kind != lineBoundary {
//		_, span1 := span.split(0)
//		return span1
//	}
//	return span
//}
//
//func (span TextSpan) TrimRight() TextSpan {
//	i := len(span.boundaries) - 2 // the last one is EOF
//	if 1 < len(span.boundaries) && span.boundaries[i].pos+span.boundaries[i].size == len(span.Text) && span.boundaries[i].kind != lineBoundary {
//		span0, _ := span.split(i)
//		return span0
//	}
//	return span
//}
//
//func (span TextSpan) Bounds(width float64) Rect {
//	p, deco, _ := span.ToPath(width)
//	return p.Bounds().Add(deco.Bounds()) // TODO: make more efficient?
//}
//
//func (span TextSpan) split(i int) (TextSpan, TextSpan) {
//	dash := ""
//	if span.boundaries[i].kind == breakBoundary {
//		dash = "-"
//	}
//
//	span0 := TextSpan{}
//	span0.Face = span.Face
//	span0.Text = span.Text[:span.boundaries[i].pos] + dash
//	span0.width = span.Face.TextWidth(span0.Text)
//	span0.boundaries = append(span.boundaries[:i:i], textBoundary{eofBoundary, len(span0.Text), 0})
//	span0.dx = span.dx
//
//	span1 := TextSpan{}
//	span1.Face = span.Face
//	span1.Text = span.Text[span.boundaries[i].pos+span.boundaries[i].size:]
//	span1.width = span.Face.TextWidth(span1.Text)
//	span1.boundaries = make([]textBoundary, len(span.boundaries)-i-1)
//	copy(span1.boundaries, span.boundaries[i+1:])
//	span1.dx = span.dx
//	for j := range span1.boundaries {
//		span1.boundaries[j].pos -= span.boundaries[i].pos + span.boundaries[i].size
//	}
//	return span0, span1
//}
//
//func (span TextSpan) Split(width float64) ([]TextSpan, bool) {
//	if width == 0.0 || span.width <= width {
//		return []TextSpan{span}, true // span fits
//	}
//	for i := len(span.boundaries) - 2; i >= 0; i-- {
//		if span.boundaries[i].pos == 0 {
//			return []TextSpan{span}, false // boundary is at the beginning, do not split
//		}
//
//		span0, span1 := span.split(i)
//		if span0.width <= width {
//			// span fits up to this boundary
//			if span1.width == 0.0 {
//				return []TextSpan{span0}, true // there is no text between the last two boundaries (e.g. space followed by end)
//			}
//			return []TextSpan{span0, span1}, true
//		}
//	}
//	return []TextSpan{span}, false // does not fit, but there are no boundaries to split
//}
//
//// CountGlyphs counts all the glyphs, where ligatures are separated into their constituent parts
//func (span TextSpan) CountGlyphs() int {
//	n := 0
//	for _, _ = range span.Text {
//		//if s, ok := ligatures[r]; ok {
//		//	n += len(s)
//		//} else {
//		n++
//		//}
//	}
//	return n
//}
//
//// ReplaceLigatures replaces all ligatures by their constituent parts
//func (span TextSpan) ReplaceLigatures() TextSpan {
//	//shift := 0
//	//iBoundary := 0
//	//for i, r := range span.Text {
//	//	if span.boundaries[iBoundary].pos == i {
//	//		span.boundaries[iBoundary].pos += shift
//	//		iBoundary++
//	//	} else if s, ok := ligatures[r]; ok {
//	//		span.Text = span.Text[:i] + s + span.Text[i+utf8.RuneLen(r):]
//	//		shift += len(s) - 1
//	//	}
//	//}
//	//span.boundaries[len(span.boundaries)-1].pos = len(span.Text)
//	//span.width = span.Face.TextWidth(span.Text)
//	return span
//}
//
//// TODO: transform to Draw to canvas and cache the glyph rasterizations?
//// TODO: remove width argument and use span.width?
//func (span TextSpan) ToPath(width float64) (*Path, *Path, color.RGBA) {
//	iBoundary := 0
//
//	x := 0.0
//	p := &Path{}
//	var rPrev rune
//	for i, r := range span.Text {
//		if i > 0 {
//			x += span.Face.Kerning(rPrev, r)
//		}
//
//		pr, advance, _ := span.Face.ToPath(string(r))
//		pr = pr.Translate(x, 0.0)
//		p = p.Append(pr)
//
//		x += advance + span.GlyphSpacing
//		if iBoundary < len(span.boundaries) && span.boundaries[iBoundary].pos == i {
//			boundary := span.boundaries[iBoundary]
//			if boundary.kind == sentenceBoundary {
//				x += span.SentenceSpacing
//			} else if boundary.kind == wordBoundary {
//				x += span.WordSpacing
//			}
//			iBoundary++
//		}
//		rPrev = r
//	}
//	return p, span.Face.Decorate(width), span.Face.Color
//}
//
//// Words returns the text of the span, split on wordBoundaries
//func (span TextSpan) Words() []string {
//	var words []string
//	i := 0
//	for _, boundary := range span.boundaries {
//		if boundary.kind != wordBoundary {
//			continue
//		}
//		j := boundary.pos + boundary.size
//		words = append(words, span.Text[i:j])
//		i = j
//	}
//	if i < len(span.Text) {
//		words = append(words, span.Text[i:])
//	}
//	return words
//}
//
//////////////////////////////////////////////////////////////////
//
//type textBoundaryKind int
//
//const (
//	eofBoundary textBoundaryKind = iota
//	lineBoundary
//	sentenceBoundary
//	wordBoundary
//	breakBoundary // zero-width space indicates word boundary
//)
//
//type textBoundary struct {
//	kind textBoundaryKind
//	pos  int
//	size int
//}
//
//func mergeBoundaries(a, b []textBoundary) []textBoundary {
//	if 0 < len(a) && 0 < len(b) && a[len(a)-1].pos+a[len(a)-1].size == b[0].pos {
//		if a[len(a)-1].kind != lineBoundary || b[0].kind != lineBoundary {
//			if b[0].kind < a[len(a)-1].kind {
//				a[len(a)-1].kind = b[0].kind
//			} else if a[len(a)-1].kind < b[0].kind {
//				b[0].kind = a[len(a)-1].kind
//			}
//			a[len(a)-1].size += b[0].size
//			b = b[1:]
//		}
//	}
//	return append(a, b...)
//}
//
//func calcTextBoundaries(s string, a, b int) []textBoundary {
//	boundaries := []textBoundary{}
//	var rPrev, rPrevPrev rune
//	if 0 < a {
//		var size int
//		rPrev, size = utf8.DecodeLastRuneInString(s[:a])
//		if size < a {
//			rPrevPrev, _ = utf8.DecodeLastRuneInString(s[:a-size])
//		}
//	}
//	for i, r := range s[a:b] {
//		size := utf8.RuneLen(r)
//		if isNewline(r) {
//			if r == '\n' && 0 < i && s[i-1] == '\r' {
//				boundaries[len(boundaries)-1].size++
//			} else {
//				boundaries = mergeBoundaries(boundaries, []textBoundary{{lineBoundary, i, size}})
//			}
//		} else if isWhitespace(r) {
//			if (rPrev == '.' && !unicode.IsUpper(rPrevPrev) && !isWhitespace(rPrevPrev)) || rPrev == '!' || rPrev == '?' {
//				boundaries = mergeBoundaries(boundaries, []textBoundary{{sentenceBoundary, i, size}})
//			} else {
//				boundaries = mergeBoundaries(boundaries, []textBoundary{{wordBoundary, i, size}})
//			}
//		} else if r == '\u200b' {
//			boundaries = mergeBoundaries(boundaries, []textBoundary{{breakBoundary, i, size}})
//		}
//		rPrevPrev = rPrev
//		rPrev = r
//	}
//	boundaries = append(boundaries, textBoundary{eofBoundary, b - a, 0})
//	return boundaries
//}
//
//func isNewline(r rune) bool {
//	return r == '\n' || r == '\r' || r == '\f' || r == '\v' || r == '\u2028' || r == '\u2029'
//}
//
//func isWhitespace(r rune) bool {
//	// see https://unicode.org/reports/tr14/#Properties
//	return unicode.IsSpace(r) || r == '\t' || r == '\u2028' || r == '\u2029'
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
