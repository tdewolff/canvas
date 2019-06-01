package canvas

import (
	"image/color"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"
)

const MaxSentenceSpacing = 3.0 // times width of space
const MaxWordSpacing = 2.5     // times width of space
const MaxGlyphSpacing = 0.5    // times x-height

type TextAlign int

const (
	Left TextAlign = iota
	Right
	Center
	Top
	Bottom
	Justify
)

type Text struct {
	lines []line
	fonts map[*Font]bool
}

type line struct {
	lineSpans []lineSpan
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

type span interface {
	WidthRange() (float64, float64)       // min-width and max-width
	Heights() (float64, float64, float64) // ascent, descent, line spacing
	Bounds(float64) Rect
	Split(float64) (span, span)
	ToPath(float64) *Path
}

////////////////////////////////

type RichText struct {
	spans      []span
	positions  []int
	boundaries []textBoundary
	fonts      map[*Font]bool
	text       string
}

func NewRichText() *RichText {
	return &RichText{
		fonts: map[*Font]bool{},
	}
}

func (rt *RichText) Add(ff FontFace, s string) *RichText {
	start := len(rt.text)
	rt.text += s

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
	rt.fonts[ff.f] = true
	return rt
}

// TODO: see if can be simplified + more documentation
func (rt *RichText) ToText(width, height float64, halign, valign TextAlign, indent float64) *Text {
	var span0, span1 span
	if 0 < len(rt.spans) {
		span1 = rt.spans[0]
	}

	k := 0 // index into rt.spans and rt.positions
	lines := []line{}
	kFirst := []int{0} // index (k) of the first span per line
	iBoundary := 0
	y, prevLineSpacing := 0.0, 0.0
	for k < len(rt.spans) {
		dx := indent
		indent = 0.0

		// accumulate line spans for a full line, ie. either split span1 to fit or if it fits retrieve the next span1 and repeat
		lss := []lineSpan{}
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

		l := line{lss, 0.0}
		top, ascent, descent, bottom := l.Heights()
		lineSpacing := math.Max(top-ascent, prevLineSpacing)
		if len(lines) != 0 {
			y -= lineSpacing
		}
		y -= ascent
		l.y = y
		y -= descent
		prevLineSpacing = bottom - descent

		kFirst = append(kFirst, k)
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
				firstPos := rt.positions[kFirst[j]]
				if 0 < j && 0 < len(rt.boundaries) && rt.boundaries[iBoundary].pos < firstPos {
					iBoundary++ // skip first boundary on line for all but the first line
				}

				// find boundaries on this line and their width range (word and sentence boundaries can expand to fit line)
				iFirstBoundary := iBoundary
				lastPos := rt.positions[kFirst[j+1]-1] // position of last word (first character)
				minBoundaryWidth, maxBoundaryWidth := 0.0, 0.0
				for ; iBoundary < len(rt.boundaries) && rt.boundaries[iBoundary].pos < lastPos; iBoundary++ {
					boundary := rt.boundaries[iBoundary]
					minBoundaryWidth += boundary.width
					if boundary.kind == sentenceBoundary {
						maxBoundaryWidth += boundary.width + boundary.xheight*MaxSentenceSpacing
					} else if boundary.kind == wordBoundary {
						maxBoundaryWidth += boundary.width + boundary.xheight*MaxWordSpacing
					}
				}
				lineBoundaries := rt.boundaries[iFirstBoundary:iBoundary]

				// get the width range of our spans (eg. for text width can increase with extra character spacing)
				minWidth, maxWidth := 0.0, 0.0
				for i, ls := range l.lineSpans {
					spanWidth, spanMaxWidth := ls.span.WidthRange()
					if i == 0 {
						minWidth += ls.dx
						maxWidth += ls.dx
					}
					minWidth += spanWidth
					maxWidth += spanMaxWidth
				}

				// only expand if we can reach the line width
				expandBoundaryWidth := minWidth + maxBoundaryWidth
				minWidth += minBoundaryWidth
				maxWidth += maxBoundaryWidth
				if minWidth < width && width < maxWidth {
					dx := 0.0

					// see if boundary expanding alone is enough to fit the line, otherwise also expand spans
					boundaryFactor, spanFactor := 0.0, 0.0
					if width < expandBoundaryWidth {
						boundaryFactor = (width - minWidth) / (expandBoundaryWidth - minWidth)
					} else {
						boundaryFactor = 1.0
						spanFactor = (width - expandBoundaryWidth) / (maxWidth - expandBoundaryWidth)
					}
					for i, ls := range l.lineSpans {
						if 0 < i {
							boundary := lineBoundaries[i-1]
							var boundaryExpansion float64
							if boundary.kind == sentenceBoundary {
								boundaryExpansion = boundary.xheight * MaxSentenceSpacing
							} else if boundary.kind == wordBoundary {
								boundaryExpansion = boundary.xheight * MaxWordSpacing
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
	}
	return &Text{lines, rt.fonts}
}

func NewText(ff FontFace, s string) *Text {
	ss := splitNewlines(s)
	y := 0.0
	lines := []line{}
	for _, s := range ss {
		span := lineSpan{newTextSpan(ff, s), 0.0, 0.0}
		lines = append(lines, line{[]lineSpan{span}, y})

		ascent, descent, spacing := span.Heights()
		y -= spacing + ascent + descent + spacing
	}
	return &Text{lines, map[*Font]bool{ff.f: true}}
}

func NewTextBox(ff FontFace, s string, width, height float64, halign, valign TextAlign, indent float64) *Text {
	return NewRichText().Add(ff, s).ToText(width, height, halign, valign, indent)
}

// Bounds returns the rectangle that contains the entire text box.
func (t *Text) Bounds() Rect {
	if len(t.lines) == 0 {
		return Rect{}
	}
	x0, y0, x1, y1 := math.Inf(1.0), math.Inf(-1.0), math.Inf(-1.0), math.Inf(1.0)
	for _, line := range t.lines {
		for _, ls := range line.lineSpans {
			spanBounds := ls.span.Bounds(ls.w)
			x0 = math.Min(x0, ls.dx+spanBounds.X)
			x1 = math.Max(x1, ls.dx+spanBounds.X+spanBounds.W)
			y0 = math.Max(y0, line.y+spanBounds.H+spanBounds.Y)
			y1 = math.Min(y1, line.y+spanBounds.Y)
		}
	}
	return Rect{x0, y0, x1 - x0, y1 - y0}
}

// ToPath makes a path out of the text, with x,y the top-left point of the rectangle that fits the text (ie. y is not the text base)
func (t *Text) ToPath() *Path {
	p := &Path{}
	for _, line := range t.lines {
		for _, ls := range line.lineSpans {
			ps := ls.span.ToPath(ls.w)
			ps.Translate(ls.dx, line.y)
			p.Append(ps)
		}
	}
	return p
}

func (t *Text) ToPathDecorations() *Path {
	p := &Path{}
	for _, line := range t.lines {
		if 0 < len(line.lineSpans) {
			var ff FontFace
			x0, x1 := 0.0, 0.0
			for i, ls := range line.lineSpans {
				ts, ok := ls.span.(textSpan)
				if 0.0 < x1-x0 && (!ok || ts.ff != ff || i+1 == len(line.lineSpans)) {
					ps := ff.ToPathDecoration(x1 - x0)
					if !ps.Empty() {
						ps.Translate(x0, line.y)
						p.Append(ps)
					}
					x0 = x1
				}
				ff = ts.ff
				if x0 == x1 {
					x0 = ls.dx
				}
				x1 = ls.dx + ls.w
			}
			if 0.0 < x1-x0 {
				ps := ff.ToPathDecoration(x1 - x0)
				if !ps.Empty() {
					ps.Translate(x0, line.y)
					p.Append(ps)
				}
			}
		}
	}
	return p
}

func (t *Text) ToSVG(x, y, rot float64, c color.Color) string {
	sb := strings.Builder{}
	sb.WriteString("<text x=\"")
	writeFloat64(&sb, x)
	sb.WriteString("\" y=\"")
	writeFloat64(&sb, y)
	if rot != 0.0 {
		sb.WriteString("\" transform=\"rotate(")
		writeFloat64(&sb, -rot)
		sb.WriteString(",")
		writeFloat64(&sb, x)
		sb.WriteString(",")
		writeFloat64(&sb, y)
		sb.WriteString(")")
	}
	if c != color.Black {
		sb.WriteString("\" fill=\"")
		writeCSSColor(&sb, c)
	}
	sb.WriteString("\">")

	for _, line := range t.lines {
		for _, ls := range line.lineSpans {
			switch span := ls.span.(type) {
			case textSpan:
				name, size, style, _ := span.ff.Info() // TODO: use decoration
				span.splitAtSpacings(ls.dx, ls.w, func(dx, w, glyphSpacing float64, s string) {
					sb.WriteString("<tspan x=\"")
					writeFloat64(&sb, x+dx)
					sb.WriteString("\" y=\"")
					writeFloat64(&sb, y-line.y)
					if glyphSpacing > 0.0 {
						sb.WriteString("\" textLength=\"")
						writeFloat64(&sb, w)
					}
					sb.WriteString("\" font-family=\"")
					sb.WriteString(name)
					sb.WriteString("\" font-size=\"")
					writeFloat64(&sb, size)
					if style&Italic != 0 {
						sb.WriteString("\" font-style=\"italic")
					}
					if style&Bold != 0 {
						sb.WriteString("\" font-weight=\"bold")
					}
					sb.WriteString("\">")
					s = span.ff.f.transform(s, w == 0.0)
					sb.WriteString(s)
					sb.WriteString("</tspan>")
				})
			default:
				panic("unsupported span type")
			}
		}
	}
	sb.WriteString("</text>")
	return sb.String()
}

type textSpan struct {
	ff        FontFace
	s         string
	textWidth float64
	//sentenceSpacings int
	//wordSpacings     int
	glyphSpacings  int
	wordBoundaries []textBoundary
}

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

func (ts textSpan) ToPath(width float64) *Path {
	//sentenceSpacing := 0.0
	//wordSpacing := 0.0
	glyphSpacing := 0.0
	if width > ts.textWidth {
		widthLeft := width - ts.textWidth
		xHeight := ts.ff.Metrics().XHeight
		//if ts.sentenceSpacings > 0 {
		//	sentenceSpacing = math.Min(widthLeft/float64(ts.sentenceSpacings), xHeight*MaxSentenceSpacing)
		//	widthLeft -= float64(ts.sentenceSpacings) * sentenceSpacing
		//}
		//if ts.wordSpacings > 0 {
		//	wordSpacing = math.Min(widthLeft/float64(ts.wordSpacings), xHeight*MaxWordSpacing)
		//	widthLeft -= float64(ts.wordSpacings) * wordSpacing
		//}
		if ts.glyphSpacings > 0 {
			glyphSpacing = math.Min(widthLeft/float64(ts.glyphSpacings), xHeight*MaxGlyphSpacing)
		}
	}
	s := ts.ff.f.transform(ts.s, glyphSpacing == 0.0)

	x := 0.0
	p := &Path{}
	var rPrev rune
	//iBoundary := 0
	for i, r := range s {
		if i > 0 {
			x += ts.ff.Kerning(rPrev, r)
		}

		pr, advance := ts.ff.ToPath(r)
		pr.Translate(x, 0.0)
		p.Append(pr)
		x += advance

		spacing := glyphSpacing
		//if iBoundary < len(ts.textBoundaries) && ts.textBoundaries[iBoundary].pos == i {
		//	if ts.textBoundaries[iBoundary].kind == wordBoundary {
		//		spacing = wordSpacing
		//	} else if ts.textBoundaries[iBoundary].kind == sentenceBoundary {
		//		spacing = sentenceSpacing
		//	}
		//	iBoundary++
		//}
		x += spacing
		rPrev = r
	}
	return p
}

// TODO: remove
func (ts textSpan) splitAtSpacings(spanDx, width float64, f func(float64, float64, float64, string)) {
	//spaceWidth := ts.ff.TextWidth(" ")
	//sentenceSpacing := 0.0
	//wordSpacing := 0.0
	glyphSpacing := 0.0
	if width > ts.textWidth {
		widthLeft := width - ts.textWidth
		xHeight := ts.ff.Metrics().XHeight
		//	if ts.sentenceSpacings > 0 {
		//		sentenceSpacing = math.Min(widthLeft/float64(ts.sentenceSpacings), xHeight*MaxSentenceSpacing)
		//		widthLeft -= float64(ts.sentenceSpacings) * sentenceSpacing
		//	}
		//	if ts.wordSpacings > 0 {
		//		wordSpacing = math.Min(widthLeft/float64(ts.wordSpacings), xHeight*MaxWordSpacing)
		//		widthLeft -= float64(ts.wordSpacings) * wordSpacing
		//	}
		if ts.glyphSpacings > 0 {
			glyphSpacing = math.Min(widthLeft/float64(ts.glyphSpacings), xHeight*MaxGlyphSpacing)
		}
	}
	//if sentenceSpacing > 0.0 || wordSpacing > 0.0 {
	//	prevPos := 0
	//	dx := spanDx
	//	for _, textBoundary := range ts.textBoundaries {
	//		s := ts.s[prevPos:textBoundary.pos]
	//		w := ts.ff.TextWidth(s)
	//		if glyphSpacing > 0.0 {
	//			w += float64(utf8.RuneCountInString(s)-1) * glyphSpacing
	//		}
	//		f(dx, w, glyphSpacing, s)
	//		prevPos = textBoundary.pos + 1
	//		dx += ts.ff.TextWidth(s) + spaceWidth + float64(utf8.RuneCountInString(s))*glyphSpacing
	//		if textBoundary.kind == wordBoundary {
	//			dx += wordSpacing
	//		} else if textBoundary.kind == sentenceBoundary {
	//			dx += sentenceSpacing
	//		}
	//	}
	//} else {
	f(spanDx, width, glyphSpacing, ts.s)
	//}
}

type textBoundaryKind int

const (
	lineBoundary textBoundaryKind = iota
	sentenceBoundary
	wordBoundary
	breakBoundary // zero-width space indicates word boundary
)

type textBoundary struct {
	kind           textBoundaryKind
	pos            int
	size           int
	xheight, width float64
}

func mergeBoundaries(a, b []textBoundary) []textBoundary {
	if 0 < len(a) && 0 < len(b) && a[len(a)-1].pos+a[len(a)-1].size == b[0].pos {
		if b[0].kind < a[len(a)-1].kind {
			a[len(a)-1].kind = b[0].kind
		}
		a[len(a)-1].size += b[0].size
		a[len(a)-1].xheight = math.Max(a[len(a)-1].xheight, b[0].xheight)
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
			boundaries = append(boundaries, textBoundary{breakBoundary, i, size, 0.0, 0.0})
		} else {
			glyphSpacings++
		}
	}
	return boundaries, glyphSpacings
}

func calcTextBoundaries(ff FontFace, s string, a, b int) []textBoundary {
	xheight := ff.Metrics().XHeight

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
				boundaries = mergeBoundaries(boundaries, []textBoundary{{lineBoundary, a + i, size, 0.0, 0.0}})
			}
		} else if isWhitespace(r) {
			width := ff.TextWidth(string(r))
			if (rPrev == '.' && !unicode.IsUpper(rPrevPrev) && !isWhitespace(rPrevPrev)) || rPrev == '!' || rPrev == '?' {
				boundaries = mergeBoundaries(boundaries, []textBoundary{{sentenceBoundary, a + i, size, xheight, width}})
			} else {
				boundaries = mergeBoundaries(boundaries, []textBoundary{{wordBoundary, a + i, size, xheight, width}})
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

//func splitWhitespaces(s string) ([]string, int) {
//	ss := []string{}
//	i := 0
//	inWhitespace := false
//	firstNonWhitespace := 0
//	for j, r := range s {
//		ws := isWhitespace(r)
//		if ws && !inWhitespace {
//			if 0 < j {
//				ss = append(ss, s[i:j])
//			} else {
//				firstNonWhitespace++
//			}
//			inWhitespace = true
//			i = j
//		} else if !ws && inWhitespace {
//			ss = append(ss, s[i:j])
//			inWhitespace = false
//			i = j
//		}
//	}
//	if i < j {
//		ss = append(ss, s[i:])
//	}
//	return ss, firstNonWhitespace
//}

func isNewline(r rune) bool {
	return r == '\n' || r == '\r' || r == '\f' || r == '\v' || r == '\u2028' || r == '\u2029'
}

func isWhitespace(r rune) bool {
	// TODO: add breaking spaces such as en quad, em space, hair space, ...
	// see https://unicode.org/reports/tr14/#Properties
	return r == ' ' || r == '\t' || isNewline(r)
}
