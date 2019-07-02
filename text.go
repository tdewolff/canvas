package canvas

import (
	"fmt"
	"image/color"
	"io"
	"math"
	"unicode"
	"unicode/utf8"
)

const MaxSentenceSpacing = 3.0 // times width of space
const MaxWordSpacing = 2.5     // times width of space
const MaxGlyphSpacing = 0.5    // times x-height

// TextAlign specifies how the text should align or whether it should be justified.
type TextAlign int

const (
	Left TextAlign = iota
	Right
	Center
	Top
	Bottom
	Justify
)

type line struct {
	lineSpans []lineSpan
	decoSpans []decoSpan
	y         float64
}

func (l line) Heights() (float64, float64, float64, float64) {
	top, ascent, descent, bottom := 0.0, 0.0, 0.0, 0.0
	for _, ls := range l.lineSpans {
		spanAscent, spanDescent, lineSpacing := ls.span.Heights()
		top = math.Max(top, spanAscent+lineSpacing)
		ascent = math.Max(ascent, spanAscent)
		descent = math.Max(descent, spanDescent)
		bottom = math.Max(bottom, spanDescent+lineSpacing)
	}
	return top, ascent, descent, bottom
}

type lineSpan struct {
	span
	dx float64
	w  float64
}

type decoSpan struct {
	ff     FontFace
	x0, x1 float64
}

type span interface {
	Color() color.RGBA // TODO: incorporate with ToPath? Or remove when replace ToPath with Rasterize
	Bounds(float64) Rect
	WidthRange() (float64, float64)       // min-width and max-width
	Heights() (float64, float64, float64) // ascent, descent, line spacing
	Split(float64) (span, span)
	ToPath(float64) *Path
}

////////////////////////////////////////////////////////////////

// Text holds the representation of text using lines and text spans.
type Text struct {
	lines []line
	fonts map[*Font]bool
	ff    FontFace
}

func NewTextLine(ff FontFace, s string, halign TextAlign) *Text {
	// TODO: use halign
	ss := splitNewlines(s)
	y := 0.0
	lines := []line{}
	for _, s := range ss {
		span := lineSpan{newTextSpan(ff, s), 0.0, 0.0}
		line := line{[]lineSpan{span}, []decoSpan{}, y}
		if ff.deco != nil {
			line.decoSpans = append(line.decoSpans, decoSpan{ff, 0.0, ff.TextWidth(s)})
		}
		lines = append(lines, line)

		ascent, descent, spacing := span.Heights()
		y -= spacing + ascent + descent + spacing
	}
	return &Text{lines, map[*Font]bool{ff.font: true}, ff}
}

func NewTextBox(ff FontFace, s string, width, height float64, halign, valign TextAlign, indent, lineStretch float64) *Text {
	return NewRichText(ff).Add(ff, s).ToText(width, height, halign, valign, indent, lineStretch)
}

// RichText allows to build up a rich text with text spans of different font faces and by fitting that into a box.
type RichText struct {
	spans      []span
	positions  []int
	boundaries []textBoundary
	fonts      map[*Font]bool
	ff         FontFace
	text       string
}

func NewRichText(ff FontFace) *RichText {
	// TODO: allow for default font and use font face modifiers (color, style, faux style, decorations)
	return &RichText{
		fonts: map[*Font]bool{},
		ff:    ff,
	}
}

// Add adds a new text span element.
func (rt *RichText) Add(ff FontFace, s string) *RichText {
	start := len(rt.text)
	rt.text += s

	// split at all whitespace and add as separate spans
	i := 0
	boundaries := calcTextBoundaries(ff, rt.text, start, start+len(s))
	for _, boundary := range boundaries {
		j := boundary.pos - start
		if i < j {
			rt.spans = append(rt.spans, newTextSpan(ff, s[i:j]))
			rt.positions = append(rt.positions, start+i)
		}
		i = j + boundary.size
	}
	if i < len(s) {
		rt.spans = append(rt.spans, newTextSpan(ff, s[i:]))
		rt.positions = append(rt.positions, start+i)
	}
	rt.boundaries = mergeBoundaries(rt.boundaries, boundaries)
	rt.fonts[ff.font] = true
	return rt
}

// ToText takes the added text spans and fits them within a given box of certain width and height.
func (rt *RichText) ToText(width, height float64, halign, valign TextAlign, indent, lineStretch float64) *Text {
	var span0, span1 span
	if 0 < len(rt.spans) {
		span1 = rt.spans[0]
	}

	k := 0 // index into rt.spans and rt.positions
	lines := []line{}
	kSpans := [][]int{} // indices (k) per line, per span
	iBoundary := 0
	y, prevLineSpacing := 0.0, 0.0
	for k < len(rt.spans) {
		dx := indent
		indent = 0.0

		// accumulate line spans for a full line, ie. either split span1 to fit or if it fits retrieve the next span1 and repeat
		lss := []lineSpan{}
		kSpans = append(kSpans, []int{})
		for {
			if iBoundary < len(rt.boundaries) && rt.boundaries[iBoundary].pos < rt.positions[k] {
				boundary := rt.boundaries[iBoundary]
				iBoundary++
				if boundary.kind == lineBoundary {
					break
				} else if boundary.kind == sentenceBoundary || boundary.kind == wordBoundary {
					dx += boundary.width
				}
			}

			// inter-word splitting
			if width == 0.0 {
				span0, span1 = span1.Split(0.0)
			} else {
				span0, span1 = span1.Split(width - dx)
			}
			if span0 == nil {
				// span couln't fit
				if len(lss) == 0 {
					// but we have no choice as it's the only span on the line
					span0 = span1
					span1 = nil
				} else {
					break
				}
			}

			spanWidth, _ := span0.WidthRange()
			lss = append(lss, lineSpan{span0, dx, spanWidth})
			kSpans[len(lines)] = append(kSpans[len(lines)], k)
			dx += spanWidth
			if span1 != nil {
				break // span couldn't fully fit, we have a full line
			} else {
				k++
				if k == len(rt.spans) {
					break
				}
				span1 = rt.spans[k]
			}
		}

		l := line{lss, nil, 0.0}
		top, ascent, descent, bottom := l.Heights()
		lineSpacing := math.Max(top-ascent, prevLineSpacing)
		if len(lines) != 0 {
			y -= lineSpacing * (1.0 + lineStretch)
			y -= ascent * lineStretch
		}
		y -= ascent
		l.y = y
		y -= descent * (1.0 + lineStretch)
		prevLineSpacing = bottom - descent

		if height != 0.0 && y < -height {
			break
		}
		lines = append(lines, l)
	}

	if 0 < len(lines) {
		// apply horizontal alignment
		if halign == Right || halign == Center {
			for _, l := range lines {
				firstLineSpan := l.lineSpans[0]
				lastLineSpan := l.lineSpans[len(l.lineSpans)-1]
				dx := width - lastLineSpan.dx - lastLineSpan.w - firstLineSpan.dx
				if halign == Center {
					dx /= 2.0
				}
				for i := range l.lineSpans {
					l.lineSpans[i].dx += dx
				}
			}
		} else if 0.0 < width && halign == Justify {
			iBoundary := 0
			for j, l := range lines[:len(lines)-1] {
				firstPos := rt.positions[kSpans[j][0]]
				if 0 < j && 0 < len(rt.boundaries) && rt.boundaries[iBoundary].pos < firstPos {
					iBoundary++ // skip first boundary on line for all but the first line
				}

				// find boundaries on this line and their width range (word and sentence boundaries can expand to fit line)
				iBoundaryLine := iBoundary
				lastPos := rt.positions[kSpans[j][len(kSpans[j])-1]] // position of last word (first character)
				minBoundaryWidth, maxBoundaryWidth := 0.0, 0.0
				for ; iBoundary < len(rt.boundaries) && rt.boundaries[iBoundary].pos < lastPos; iBoundary++ {
					boundary := rt.boundaries[iBoundary]
					minBoundaryWidth += boundary.width
					if boundary.kind == sentenceBoundary {
						maxBoundaryWidth += boundary.width + boundary.width*MaxSentenceSpacing
					} else if boundary.kind == wordBoundary {
						maxBoundaryWidth += boundary.width + boundary.width*MaxWordSpacing
					}
				}

				// get the width range of our spans (eg. for text width can increase with extra character spacing)
				minTextWidth, maxTextWidth := 0.0, 0.0
				for i, ls := range l.lineSpans {
					spanWidth, spanMaxWidth := ls.span.WidthRange()
					if i == 0 {
						minTextWidth += ls.dx
						maxTextWidth += ls.dx
					}
					minTextWidth += spanWidth
					maxTextWidth += spanMaxWidth
				}

				// only expand if we can reach the line width
				minWidth := minTextWidth + minBoundaryWidth
				maxWidth := maxTextWidth + maxBoundaryWidth
				if minWidth < width && width < maxWidth {
					dx := 0.0

					// see if boundary expanding alone is enough to fit the line, otherwise also expand spans
					boundaryFactor, spanFactor := 0.0, 0.0
					if width < minTextWidth+maxBoundaryWidth {
						boundaryFactor = (width - minTextWidth - minBoundaryWidth) / (maxBoundaryWidth - minBoundaryWidth)
					} else {
						boundaryFactor = 1.0
						spanFactor = (width - minTextWidth - maxBoundaryWidth) / (maxTextWidth - minTextWidth)
					}
					for i, ls := range l.lineSpans {
						if iBoundaryLine < len(rt.boundaries) && rt.boundaries[iBoundaryLine].pos < rt.positions[kSpans[j][i]] {
							boundary := rt.boundaries[iBoundaryLine]
							iBoundaryLine++
							var boundaryExpansion float64
							if boundary.kind == sentenceBoundary {
								boundaryExpansion = boundary.width * MaxSentenceSpacing
							} else if boundary.kind == wordBoundary {
								boundaryExpansion = boundary.width * MaxWordSpacing
							}
							dx += boundaryExpansion * boundaryFactor
						}

						spanWidth, spanMaxWidth := ls.span.WidthRange()
						w := spanWidth + (spanMaxWidth-spanWidth)*spanFactor
						l.lineSpans[i].dx += dx
						dx += w - ls.w
						l.lineSpans[i].w = w
					}
				}
			}
		}

		// apply vertical alignment
		dy := 0.0
		h := -y
		extraLineSpacing := 0.0
		if height != 0.0 && (valign == Bottom || valign == Center || valign == Justify) {
			if valign == Bottom {
				dy = height - h
			} else if valign == Center {
				dy = (height - h) / 2.0
			} else if len(lines) > 1 {
				extraLineSpacing = (height - h) / float64(len(lines)-1)
			}
		}
		for j := range lines {
			lines[j].y -= dy + float64(j)*extraLineSpacing
		}

		// set decorations
		for j, line := range lines {
			color := Black
			ff := FontFace{}
			x0, x1 := 0.0, 0.0
			for _, ls := range line.lineSpans {
				ts, ok := ls.span.(textSpan)
				if 0.0 < x1-x0 && (!ok || ts.color != color || !ts.ff.Equals(ff)) {
					if ff.deco != nil {
						lines[j].decoSpans = append(lines[j].decoSpans, decoSpan{ff, x0, x1})
					}
					x0 = x1
				}
				if ok {
					ff = ts.ff
					color = ts.color
				}
				if x0 == x1 {
					x0 = ls.dx // skip space when starting new decoSpan
				}
				x1 = ls.dx + ls.w
			}
			if 0.0 < x1-x0 && ff.deco != nil {
				lines[j].decoSpans = append(lines[j].decoSpans, decoSpan{ff, x0, x1})
			}
		}
	}
	return &Text{lines, rt.fonts, rt.ff}
}

// Bounds returns the rectangle that contains the entire text box.
func (t *Text) Bounds() Rect {
	if len(t.lines) == 0 || len(t.lines[0].lineSpans) == 0 {
		return Rect{}
	}
	r := Rect{}
	fmt.Println(len(t.lines))
	for _, line := range t.lines {
		fmt.Println(len(line.lineSpans))
		for _, ls := range line.lineSpans {
			fmt.Println(len(line.lineSpans))
			spanBounds := ls.span.Bounds(ls.w)
			spanBounds = spanBounds.Move(Point{ls.dx, line.y})
			r = r.Add(spanBounds)
		}
		fmt.Println(len(line.decoSpans))
		for _, ds := range line.decoSpans {
			spanBounds := ds.ff.Decorate(ds.x1 - ds.x0).Bounds()
			spanBounds = spanBounds.Move(Point{ds.x0, line.y})
			r = r.Add(spanBounds)
		}
	}
	return r
}

// iterateSpans groups by type and then equality of spans. It also splits on (larger) sentence spacings.
func (t *Text) iterateSpans(cb func([]span)) {
}

// ToPath makes a path out of the text, with x,y the top-left point of the rectangle that fits the text (ie. y is not the text base)
func (t *Text) ToPaths() ([]*Path, []color.RGBA) {
	paths := []*Path{}
	colors := []color.RGBA{}
	for _, line := range t.lines {
		for _, ls := range line.lineSpans {
			span := ls.span.ToPath(ls.w)
			span = span.Translate(ls.dx, line.y)
			paths = append(paths, span)
			colors = append(colors, ls.Color())
		}
		for _, ds := range line.decoSpans {
			deco := ds.ff.Decorate(ds.x1 - ds.x0)
			deco = deco.Translate(ds.x0, line.y)
			paths = append(paths, deco)
			colors = append(colors, ds.ff.color)
		}
	}
	return paths, colors
}

// WriteSVG will write out the text in the SVG file format.
func (t *Text) WriteSVG(w io.Writer, x, y, rot float64) {
	fmt.Fprintf(w, `<text x="%g" y="%g`, x, y)
	if rot != 0.0 {
		fmt.Fprintf(w, `" transform="rotate(%g,%g,%g)`, -rot, x, y)
	}
	// TODO: use default font
	fmt.Fprintf(w, `">`)
	decorations := []pathLayer{}
	for _, line := range t.lines {
		for _, ls := range line.lineSpans {
			switch span := ls.span.(type) {
			case textSpan:
				name, size, style, variant := span.ff.Info()
				glyphSpacing := span.getGlyphSpacing(ls.w)
				offset := span.ff.voffset // supscript and superscript
				smallScript := span.ff.variant&FontSubscript != 0 || span.ff.variant&FontSuperscript != 0

				fmt.Fprintf(w, `<tspan x="%g" y="%g`, x+ls.dx, y-line.y-offset)
				if glyphSpacing > 0.0 {
					fmt.Fprintf(w, `" textLength="%g`, span.textWidth+float64(utf8.RuneCountInString(span.s))*glyphSpacing)
				}
				fmt.Fprintf(w, `" style="font:`)
				if style&FontItalic != 0 {
					fmt.Fprintf(w, ` italic`)
				}

				boldness := 400
				if style&FontExtraLight == FontExtraLight {
					boldness = 100
				} else if style&FontLight == FontLight {
					boldness = 200
				} else if style&FontBook == FontBook {
					boldness = 300
				} else if style&FontMedium == FontMedium {
					boldness = 500
				} else if style&FontSemibold == FontSemibold {
					boldness = 600
				} else if style&FontBold == FontBold {
					boldness = 700
				} else if style&FontBlack == FontBlack {
					boldness = 800
				} else if style&FontExtraBlack == FontExtraBlack {
					boldness = 900
				}
				if smallScript {
					boldness += 300
					if 1000 < boldness {
						boldness = 1000
					}
				}
				if boldness != 400 {
					fmt.Fprintf(w, ` %d`, boldness)
				}

				if variant&FontSmallcaps != 0 {
					fmt.Fprintf(w, ` small-caps`)
				}

				fmt.Fprintf(w, ` %gpx %s`, size, name)
				if span.color != Black {
					fmt.Fprintf(w, `;fill:%s`, toCSSColor(span.color))
				}
				fmt.Fprintf(w, `">%s</tspan>`, span.ff.font.substitute(span.s, glyphSpacing == 0.0))
			default:
				panic("unsupported span type")
			}
		}
		for _, ds := range line.decoSpans {
			deco := ds.ff.Decorate(ds.x1 - ds.x0)
			deco = deco.Transform(Identity.Translate(x+ds.x0, -y+line.y).RotateAt(rot, x, -y))
			decorations = append(decorations, pathLayer{deco, drawState{fillColor: ds.ff.color}})
		}
	}
	fmt.Fprintf(w, `</text>`)

	for _, l := range decorations {
		l.WriteSVG(w, 0.0)
	}
}

// WritePDF will write out the text in the PDF file format.
func (t *Text) WritePDF(w *PDFPageWriter) {
	// TODO: Text.WritePDF
}

// TODO: Text.WriteEPS
// TODO: Text.WriteImage

////////////////////////////////////////////////////////////////

type textSpan struct {
	ff             FontFace
	color          color.RGBA
	s              string
	textWidth      float64
	glyphSpacings  int
	wordBoundaries []textBoundary
}

// TODO: proper transformation of typographic elements, ie. including surrounding text
func newTextSpan(ff FontFace, s string) textSpan {
	textWidth := ff.TextWidth(s)
	wordBoundaries, glyphSpacings := calcWordBoundaries(s)
	return textSpan{
		ff:             ff,
		s:              s,
		textWidth:      textWidth,
		glyphSpacings:  glyphSpacings,
		wordBoundaries: wordBoundaries,
	}
}

func (ts textSpan) Color() color.RGBA {
	return ts.ff.color
}

func (ts textSpan) Bounds(width float64) Rect {
	return ts.ToPath(width).Bounds() // TODO: make more efficient?
}

func (ts textSpan) WidthRange() (float64, float64) {
	return ts.textWidth, ts.textWidth + float64(ts.glyphSpacings)*ts.ff.Metrics().XHeight*MaxGlyphSpacing
}

func (ts textSpan) Heights() (float64, float64, float64) {
	return ts.ff.Metrics().Ascent, ts.ff.Metrics().Descent, ts.ff.Metrics().LineHeight - ts.ff.Metrics().Ascent - ts.ff.Metrics().Descent
}

func (ts textSpan) Split(width float64) (span, span) {
	if width == 0.0 || ts.textWidth <= width {
		return ts, nil
	}
	for i := len(ts.wordBoundaries) - 1; i >= 0; i-- {
		boundary := ts.wordBoundaries[i]
		s0 := ts.s[:boundary.pos] + "-"
		if ts.ff.TextWidth(s0) <= width {
			s1 := ts.s[boundary.pos+boundary.size:]
			if boundary.pos == 0 {
				return nil, ts
			}
			return newTextSpan(ts.ff, s0), newTextSpan(ts.ff, s1)
		}
	}
	return nil, ts
}

func (ts textSpan) getGlyphSpacing(width float64) float64 {
	glyphSpacing := 0.0
	maxGlyphSpacing := ts.ff.Metrics().XHeight * MaxGlyphSpacing
	if 0 < ts.glyphSpacings && ts.textWidth < width && width < ts.textWidth+float64(ts.glyphSpacings)*maxGlyphSpacing {
		glyphSpacing = (width - ts.textWidth) / float64(ts.glyphSpacings)
	}
	return glyphSpacing
}

// TODO: transform to Draw to canvas and cache the glyph rasterizations?
func (ts textSpan) ToPath(width float64) *Path {
	glyphSpacing := ts.getGlyphSpacing(width)
	s := ts.ff.font.substitute(ts.s, glyphSpacing == 0.0)

	x := 0.0
	p := &Path{}
	var rPrev rune
	for i, r := range s {
		if i > 0 {
			x += ts.ff.Kerning(rPrev, r)
		}

		pr, advance := ts.ff.ToPath(string(r))
		pr = pr.Translate(x, 0.0)
		p = p.Append(pr)
		x += advance

		x += glyphSpacing
		rPrev = r
	}
	return p
}

////////////////////////////////////////////////////////////////

type textBoundaryKind int

const (
	lineBoundary textBoundaryKind = iota
	sentenceBoundary
	wordBoundary
	breakBoundary // zero-width space indicates word boundary
)

type textBoundary struct {
	kind  textBoundaryKind
	pos   int
	size  int
	width float64
}

func mergeBoundaries(a, b []textBoundary) []textBoundary {
	if 0 < len(a) && 0 < len(b) && a[len(a)-1].pos+a[len(a)-1].size == b[0].pos {
		if b[0].kind < a[len(a)-1].kind {
			a[len(a)-1].kind = b[0].kind
		}
		a[len(a)-1].size += b[0].size
		a[len(a)-1].width += b[0].width
		b = b[1:]
	}
	return append(a, b...)
}

func calcWordBoundaries(s string) ([]textBoundary, int) {
	boundaries := []textBoundary{}
	glyphSpacings := 0
	for i, r := range s {
		size := utf8.RuneLen(r)
		if r == '\u200b' {
			boundaries = append(boundaries, textBoundary{breakBoundary, i, size, 0.0})
		} else {
			glyphSpacings++
		}
	}
	return boundaries, glyphSpacings
}

func calcTextBoundaries(ff FontFace, s string, a, b int) []textBoundary {
	width := ff.TextWidth(" ")

	boundaries := []textBoundary{}
	var rPrev, rPrevPrev rune
	if 0 < a {
		var size int
		rPrev, size = utf8.DecodeLastRuneInString(s[:a])
		if size < a {
			rPrevPrev, _ = utf8.DecodeLastRuneInString(s[:a-size])
		}
	}
	for i, r := range s[a:b] {
		size := utf8.RuneLen(r)
		if isNewline(r) {
			if r == '\n' && 0 < i && s[i-1] == '\r' {
				boundaries[len(boundaries)-1].size++
			} else {
				boundaries = mergeBoundaries(boundaries, []textBoundary{{lineBoundary, a + i, size, 0.0}})
			}
		} else if isWhitespace(r) {
			if (rPrev == '.' && !unicode.IsUpper(rPrevPrev) && !isWhitespace(rPrevPrev)) || rPrev == '!' || rPrev == '?' {
				boundaries = mergeBoundaries(boundaries, []textBoundary{{sentenceBoundary, a + i, size, width}})
			} else {
				boundaries = mergeBoundaries(boundaries, []textBoundary{{wordBoundary, a + i, size, width}})
			}
		}
		rPrevPrev = rPrev
		rPrev = r
	}
	return boundaries
}

func splitNewlines(s string) []string {
	ss := []string{}
	i := 0
	for j, r := range s {
		if r == '\n' || r == '\r' || r == '\f' || r == '\v' || r == '\u2028' || r == '\u2029' {
			if r == '\n' && 0 < j && s[j-1] == '\r' {
				i++
				continue
			}
			ss = append(ss, s[i:j])
			i = j + utf8.RuneLen(r)
		}
	}
	ss = append(ss, s[i:])
	return ss
}

func isNewline(r rune) bool {
	return r == '\n' || r == '\r' || r == '\f' || r == '\v' || r == '\u2028' || r == '\u2029'
}

func isWhitespace(r rune) bool {
	// TODO: add breaking spaces such as en quad, em space, hair space, ...
	// see https://unicode.org/reports/tr14/#Properties
	return r == ' ' || r == '\t' || isNewline(r)
}
