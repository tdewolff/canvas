package canvas

import (
	"bytes"
	"fmt"
	"image/color"
	"io"
	"math"
	"strings"
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
	spans []textSpan
	//decoSpans []decoSpan
	y float64
}

func (l line) Heights() (float64, float64, float64, float64) {
	top, ascent, descent, bottom := 0.0, 0.0, 0.0, 0.0
	for _, s := range l.spans {
		spanAscent, spanDescent, lineSpacing := s.Heights()
		top = math.Max(top, spanAscent+lineSpacing)
		ascent = math.Max(ascent, spanAscent)
		descent = math.Max(descent, spanDescent)
		bottom = math.Max(bottom, spanDescent+lineSpacing)
	}
	return top, ascent, descent, bottom
}

////////////////////////////////////////////////////////////////

// Text holds the representation of text using lines and text spans.
type Text struct {
	lines []line
	fonts map[*Font]bool
}

func NewTextLine(ff FontFace, s string, halign TextAlign) *Text {
	// TODO: use halign
	ss := splitNewlines(s)
	y := 0.0
	lines := []line{}
	for _, s := range ss {
		span := newTextSpan(ff, s)
		line := line{[]textSpan{span}, y}
		lines = append(lines, line)

		ascent, descent, spacing := span.Heights()
		y -= spacing + ascent + descent + spacing
	}
	return &Text{lines, map[*Font]bool{ff.font: true}}
}

func NewTextBox(ff FontFace, s string, width, height float64, halign, valign TextAlign, indent, lineStretch float64) *Text {
	return NewRichText().Add(ff, s).ToText(width, height, halign, valign, indent, lineStretch)
}

// RichText allows to build up a rich text with text spans of different font faces and by fitting that into a box.
type RichText struct {
	spans      []textSpan
	positions  []int
	boundaries []textBoundary
	fonts      map[*Font]bool
	text       string
}

func NewRichText() *RichText {
	// TODO: allow for default font and use font face modifiers (color, style, faux style, decorations)
	return &RichText{
		fonts: map[*Font]bool{},
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
	var spans []textSpan
	if 0 < len(rt.spans) {
		spans = []textSpan{rt.spans[0]}
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
		ss := []textSpan{}
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
			var ok bool
			if width == 0.0 {
				spans, ok = spans[0].Split(0.0)
			} else {
				spans, ok = spans[0].Split(width - dx)
			}
			if !ok && len(ss) != 0 {
				// span couln't fit, but we have no choice as it's the only span on the line
				break
			}

			spanWidth, _ := spans[0].WidthRange()
			spans[0].dx = dx
			spans[0].w = spanWidth
			ss = append(ss, spans[0])
			kSpans[len(lines)] = append(kSpans[len(lines)], k)
			dx += spanWidth

			spans = spans[1:]
			if len(spans) == 0 {
				k++
				if k == len(rt.spans) {
					break
				}
				spans = []textSpan{rt.spans[k]}
			} else {
				break // span couldn't fully fit, we have a full line
			}
		}

		l := line{ss, 0.0}
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
				firstSpan := l.spans[0]
				lastSpan := l.spans[len(l.spans)-1]
				dx := width - lastSpan.dx - lastSpan.w - firstSpan.dx
				if halign == Center {
					dx /= 2.0
				}
				for i := range l.spans {
					l.spans[i].dx += dx
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
				for i, span := range l.spans {
					spanWidth, spanMaxWidth := span.WidthRange()
					if i == 0 {
						minTextWidth += span.dx
						maxTextWidth += span.dx
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
					for i, span := range l.spans {
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

						spanWidth, spanMaxWidth := span.WidthRange()
						w := spanWidth + (spanMaxWidth-spanWidth)*spanFactor
						l.spans[i].dx += dx
						dx += w - span.w
						l.spans[i].w = w
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
		//for j, line := range lines {
		//	ff := FontFace{}
		//	x0, x1 := 0.0, 0.0
		//	for _, ls := range line.lineSpans {
		//		ts, ok := ls.span.(textSpan)
		//		if 0.0 < x1-x0 && (!ok || !ts.ff.Equals(ff)) {
		//			if ff.deco != nil {
		//				lines[j].decoSpans = append(lines[j].decoSpans, decoSpan{ff, x0, x1})
		//			}
		//			x0 = x1
		//		}
		//		if ok {
		//			ff = ts.ff
		//		}
		//		if x0 == x1 {
		//			x0 = ls.dx // skip space when starting new decoSpan
		//		}
		//		x1 = ls.dx + ls.w
		//	}
		//	if 0.0 < x1-x0 && ff.deco != nil {
		//		lines[j].decoSpans = append(lines[j].decoSpans, decoSpan{ff, x0, x1})
		//	}
		//}
	}
	return &Text{lines, rt.fonts}
}

// Bounds returns the rectangle that contains the entire text box.
func (t *Text) Bounds() Rect {
	if len(t.lines) == 0 || len(t.lines[0].spans) == 0 {
		return Rect{}
	}
	r := Rect{}
	for _, line := range t.lines {
		for _, span := range line.spans {
			spanBounds := span.Bounds(span.w)
			spanBounds = spanBounds.Move(Point{span.dx, line.y})
			r = r.Add(spanBounds)
		}
	}
	return r
}

func (t *Text) mostCommonFontFace() FontFace {
	families := map[*FontFamily]int{}
	sizes := map[float64]int{}
	styles := map[FontStyle]int{}
	variants := map[FontVariant]int{}
	colors := map[color.RGBA]int{}
	for _, line := range t.lines {
		for _, span := range line.spans {
			families[span.ff.family]++
			sizes[span.ff.size]++
			styles[span.ff.style]++
			variants[span.ff.variant]++
			colors[span.ff.color]++
		}
	}
	if len(families) == 0 {
		return FontFace{}
	}

	family, size, style, variant, col := (*FontFamily)(nil), 0.0, FontRegular, FontNormal, Black
	for key, val := range families {
		if families[family] < val {
			family = key
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
	return family.Face(size*PtPerMm, col, style, variant)
}

// iterateSpans groups by equality of font face. It also splits on (larger) sentence spacings.
func (t *Text) iterate(callback func(textSpan, float64)) {
	callbackGroup := func(spans []textSpan, y float64) {
		ff := spans[0].ff
		dx := spans[0].dx
		w := spans[len(spans)-1].dx + spans[len(spans)-1].w - dx

		s := ""
		textWidth := 0.0
		glyphSpacings := 0
		wordBoundaries := []textBoundary{}
		for _, span := range spans {
			s += span.text
			textWidth += span.textWidth
			glyphSpacings += span.glyphSpacings
			wordBoundaries = mergeBoundaries(wordBoundaries, span.wordBoundaries)
		}
		callback(textSpan{dx, w, nil, s, ff, textWidth, glyphSpacings, wordBoundaries}, y)
	}

	for _, line := range t.lines {
		i := 0
		for j, span := range line.spans {
			// TODO: path span elements
			if i < j && !span.ff.Equals(line.spans[i].ff) {
				callbackGroup(line.spans[i:j], line.y)
				i = j
			}
		}
		callbackGroup(line.spans[i:], line.y)
	}
}

// ToPath makes a path out of the text, with x,y the top-left point of the rectangle that fits the text (ie. y is not the text base)
func (t *Text) ToPaths() ([]*Path, []color.RGBA) {
	paths := []*Path{}
	colors := []color.RGBA{}
	for _, line := range t.lines {
		for _, span := range line.spans {
			p, deco, col := span.ToPath(span.w)
			p = p.Translate(span.dx, line.y)
			deco = deco.Translate(span.dx, line.y)
			paths = append(paths, p, deco)
			colors = append(colors, col, col)
		}
	}
	return paths, colors
}

// WriteSVG will write out the text in the SVG file format.
func (t *Text) WriteSVG(w io.Writer, x, y, rot float64) {
	writeStyle := func(ff, ffMain FontFace) {
		boldness := ff.boldness()
		differences := 0
		if ff.style&FontItalic != ffMain.style&FontItalic {
			differences++
		}
		if boldness != ffMain.boldness() {
			differences++
		}
		if ff.variant&FontSmallcaps != ffMain.variant&FontSmallcaps {
			differences++
		}
		if ff.color != ffMain.color {
			differences++
		}
		if ff.font.name != ffMain.font.name || ff.size*ff.scale != ffMain.size || differences == 3 {
			fmt.Fprintf(w, `" style="font:`)
			if ff.style&FontItalic != ffMain.style&FontItalic {
				fmt.Fprintf(w, ` italic`)
			}

			if boldness != ffMain.boldness() {
				fmt.Fprintf(w, ` %d`, boldness)
			}

			if ff.variant&FontSmallcaps != ffMain.variant&FontSmallcaps {
				fmt.Fprintf(w, ` small-caps`)
			}

			fmt.Fprintf(w, ` %gpx %s`, ff.size*ff.scale, ff.font.name)
			if ff.color != ffMain.color {
				fmt.Fprintf(w, `;fill:%s`, toCSSColor(ff.color))
			}
		} else if differences == 1 && ff.color != ffMain.color {
			fmt.Fprintf(w, `" fill="%s`, toCSSColor(ff.color))
		} else if 0 < differences {
			fmt.Fprintf(w, `" style="`)
			buf := &bytes.Buffer{}
			if ff.style&FontItalic != ffMain.style&FontItalic {
				fmt.Fprintf(buf, `;font-style:italic`)
			}
			if boldness != ffMain.boldness() {
				fmt.Fprintf(buf, `;font-weight:%d`, boldness)
			}
			if ff.variant&FontSmallcaps != ffMain.variant&FontSmallcaps {
				fmt.Fprintf(buf, `;font-variant:small-caps`)
			}
			if ff.color != ffMain.color {
				fmt.Fprintf(buf, `;fill:%s`, toCSSColor(ff.color))
			}
			buf.ReadByte()
			buf.WriteTo(w)
		}
	}

	ffMain := t.mostCommonFontFace()

	fmt.Fprintf(w, `<text x="%g" y="%g`, x, y)
	if rot != 0.0 {
		fmt.Fprintf(w, `" transform="rotate(%g,%g,%g)`, -rot, x, y)
	}
	fmt.Fprintf(w, `" style="font:`)
	if ffMain.style&FontItalic != 0 {
		fmt.Fprintf(w, ` italic`)
	}
	if boldness := ffMain.boldness(); boldness != 400 {
		fmt.Fprintf(w, ` %d`, boldness)
	}
	if ffMain.variant&FontSmallcaps != 0 {
		fmt.Fprintf(w, ` small-caps`)
	}
	fmt.Fprintf(w, ` %gpx %s`, ffMain.size*ffMain.scale, ffMain.font.name)
	if ffMain.color != Black {
		fmt.Fprintf(w, `;fill:%s`, toCSSColor(ffMain.color))
	}
	fmt.Fprintf(w, `">`)

	decorations := []pathLayer{}
	for _, line := range t.lines {
		for _, span := range line.spans {
			glyphSpacing := span.getGlyphSpacing(span.w)
			width := span.textWidth + float64(span.glyphSpacings)*glyphSpacing

			fmt.Fprintf(w, `<tspan x="%g" y="%g`, x+span.dx, y-line.y-span.ff.voffset)
			if glyphSpacing > 0.0 {
				fmt.Fprintf(w, `" textLength="%g`, width)
			}
			writeStyle(span.ff, ffMain)
			fmt.Fprintf(w, `">%s</tspan>`, span.text)

			deco := span.ff.Decorate(width).Transform(Identity.Translate(x+span.dx, -y+line.y+span.ff.voffset).RotateAt(rot, x, -y))
			decorations = append(decorations, pathLayer{deco, drawState{fillColor: span.ff.color}})
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
	dx, w          float64
	path           *Path // either path is set, or the text related attributes below
	text           string
	ff             FontFace
	textWidth      float64
	glyphSpacings  int
	wordBoundaries []textBoundary
}

// TODO: add newPathSpan

// TODO: proper transformation of typographic elements, ie. including surrounding text
func newTextSpan(ff FontFace, s string) textSpan {
	wordBoundaries, glyphSpacings := calcWordBoundaries(s)
	s = strings.ReplaceAll(s, "\u200b", "") // zero-width space
	textWidth := ff.TextWidth(s)
	return textSpan{
		dx:             0.0,
		w:              0.0,
		path:           nil,
		text:           s,
		ff:             ff,
		textWidth:      textWidth,
		glyphSpacings:  glyphSpacings,
		wordBoundaries: wordBoundaries,
	}
}

func (span textSpan) Bounds(width float64) Rect {
	p, deco, _ := span.ToPath(width)
	return p.Bounds().Add(deco.Bounds()) // TODO: make more efficient?
}

func (span textSpan) WidthRange() (float64, float64) {
	return span.textWidth, span.textWidth + float64(span.glyphSpacings)*span.ff.Metrics().XHeight*MaxGlyphSpacing
}

func (span textSpan) Heights() (float64, float64, float64) {
	return span.ff.Metrics().Ascent, span.ff.Metrics().Descent, span.ff.Metrics().LineHeight - span.ff.Metrics().Ascent - span.ff.Metrics().Descent
}

func (span textSpan) Split(width float64) ([]textSpan, bool) {
	if width == 0.0 || span.textWidth <= width {
		return []textSpan{span}, true
	}
	for i := len(span.wordBoundaries) - 1; i >= 0; i-- {
		boundary := span.wordBoundaries[i]
		s0 := span.text[:boundary.pos] + "-"
		if span.ff.TextWidth(s0) <= width {
			s1 := span.text[boundary.pos+boundary.size:]
			if boundary.pos == 0 {
				return []textSpan{span}, false
			}
			return []textSpan{newTextSpan(span.ff, s0), newTextSpan(span.ff, s1)}, true
		}
	}
	return []textSpan{span}, false
}

func (span textSpan) getGlyphSpacing(width float64) float64 {
	glyphSpacing := 0.0
	maxGlyphSpacing := span.ff.Metrics().XHeight * MaxGlyphSpacing
	if 0 < span.glyphSpacings && span.textWidth < width && width < span.textWidth+float64(span.glyphSpacings)*maxGlyphSpacing {
		glyphSpacing = (width - span.textWidth) / float64(span.glyphSpacings)
	}
	return glyphSpacing
}

// TODO: transform to Draw to canvas and cache the glyph rasterizations?
func (span textSpan) ToPath(width float64) (*Path, *Path, color.RGBA) {
	glyphSpacing := span.getGlyphSpacing(width)
	s := span.text

	x := 0.0
	p := &Path{}
	var rPrev rune
	for i, r := range s {

		if i > 0 {
			x += span.ff.Kerning(rPrev, r)
		}

		pr, advance := span.ff.ToPath(string(r))
		pr = pr.Translate(x, 0.0)
		p = p.Append(pr)
		x += advance

		x += glyphSpacing
		rPrev = r
	}
	return p, span.ff.Decorate(width), span.ff.color
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
