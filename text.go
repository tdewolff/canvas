package canvas

import (
	"image/color"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"
)

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
	fonts []*Font
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

func calcSpanPosition(textWidth, maxTextWidth float64, halign TextAlign, indent, width float64) (float64, float64) {
	dx := indent
	spanWidth := textWidth
	if halign == Right {
		dx = width - textWidth - indent
	} else if halign == Center {
		dx = (width - textWidth) / 2.0
	} else if halign == Justify {
		spanWidth = math.Min(maxTextWidth, width-indent)
	}
	return dx, spanWidth
}

type RichText struct {
	spans []span
	fonts []*Font
}

func NewRichText() *RichText {
	return &RichText{}
}

func (rt *RichText) Add(ff FontFace, s string) *RichText {
	rt.spans = append(rt.spans, newTextSpan(ff, s))
	found := false
	for _, font := range rt.fonts {
		if font == ff.f {
			found = true
			break
		}
	}
	if !found {
		rt.fonts = append(rt.fonts, ff.f)
	}
	return rt
}

func (rt *RichText) ToText(width, height float64, halign, valign TextAlign, indent float64) *Text {
	j := 0
	lines := []line{}
	var span0, span1 span
	if 0 < len(rt.spans) {
		span1 = rt.spans[0]
	}
	h, prevBottom := 0.0, 0.0
	for (height == 0.0 || h < height) && j < len(rt.spans) {
		dx := indent
		indent = 0.0
		lss := []lineSpan{}
		for {
			span0, span1 = span1.Split(width - dx)
			if span0 == nil {
				break // span starts with newline or cannot be broken up to fit
			}
			spanWidth, _ := span0.WidthRange()
			lss = append(lss, lineSpan{span0, dx, spanWidth})
			dx += spanWidth
			if span1 != nil {
				break // span couldn't fully fit, we have a full line
			} else {
				j++
				if j == len(rt.spans) {
					break
				}
				span1 = rt.spans[j]
			}
		}

		l := line{lss, 0.0}
		top, ascent, descent, bottom := l.Heights()
		top = math.Max(top, prevBottom)
		if len(lines) != 0 {
			h += top
		}
		h += ascent
		l.y = -h
		h += descent
		prevBottom = bottom

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
		} else if halign == Justify {
			for _, l := range lines[:len(lines)-1] {
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
				if minWidth < width && width < maxWidth {
					dx := 0.0
					f := (width - minWidth) / (maxWidth - minWidth)
					for i, ls := range l.lineSpans {
						spanWidth, spanMaxWidth := ls.span.WidthRange()
						w := spanWidth + (spanMaxWidth-spanWidth)*f
						l.lineSpans[i].dx += dx
						dx += w - ls.w
						l.lineSpans[i].w = w
					}
				}
			}
		}

		// apply vertical alignment
		dy := 0.0
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
	return &Text{lines, []*Font{ff.f}}
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
				name, style, size := span.ff.Info()
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

const MaxSentenceSpacing = 2.0
const MaxWordSpacing = 1.5
const MaxGlyphSpacing = 1.0

type textSpan struct {
	ff               FontFace
	s                string
	textWidth        float64
	sentenceSpacings int
	wordSpacings     int
	glyphSpacings    int
	textBoundaries   []textBoundary
}

func newTextSpan(ff FontFace, s string) textSpan {
	textWidth := ff.TextWidth(s)
	textBoundaries, sentenceSpacings, wordSpacings, glyphSpacings := calcTextBoundaries(s)
	return textSpan{
		ff:               ff,
		s:                s,
		textWidth:        textWidth,
		sentenceSpacings: sentenceSpacings,
		wordSpacings:     wordSpacings,
		glyphSpacings:    glyphSpacings,
		textBoundaries:   textBoundaries,
	}
}

func (ts textSpan) Bounds(width float64) Rect {
	return ts.ToPath(width).Bounds() // TODO: make more efficient?
}

func (ts textSpan) WidthRange() (float64, float64) {
	spacings := float64(ts.sentenceSpacings) * MaxSentenceSpacing
	spacings += float64(ts.wordSpacings) * MaxWordSpacing
	spacings += float64(ts.glyphSpacings) * MaxGlyphSpacing
	return ts.textWidth, ts.textWidth + spacings
}

func (ts textSpan) Heights() (float64, float64, float64) {
	return ts.ff.Metrics().Ascent, ts.ff.Metrics().Descent, ts.ff.Metrics().LineHeight - ts.ff.Metrics().Ascent - ts.ff.Metrics().Descent
}

func (ts textSpan) Split(width float64) (span, span) {
	if width == 0.0 || ts.textWidth < width {
		return ts, nil
	}
	for i, textBoundary := range ts.textBoundaries {
		s := ts.s[:textBoundary.pos]
		if textBoundary.kind == breakBoundary {
			s += "-"
		}
		if textBoundary.kind == lineBoundary || width != 0.0 && width < ts.ff.TextWidth(s) {
			if i == 0 {
				return nil, ts
			}
			textBoundary = ts.textBoundaries[i-1]
			s0 := ts.s[:textBoundary.pos]
			s1 := ts.s[textBoundary.pos+textBoundary.size:]
			if textBoundary.pos == 0 {
				return nil, newTextSpan(ts.ff, s1)
			}
			if textBoundary.kind == breakBoundary {
				s0 += "-"
			}
			return newTextSpan(ts.ff, s0), newTextSpan(ts.ff, s1)
		}
	}
	return ts, nil
}

func (ts textSpan) ToPath(width float64) *Path {
	sentenceSpacing := 0.0
	wordSpacing := 0.0
	glyphSpacing := 0.0
	if width > ts.textWidth {
		widthLeft := width - ts.textWidth
		xHeight := ts.ff.Metrics().XHeight
		if ts.sentenceSpacings > 0 {
			sentenceSpacing = math.Min(widthLeft/float64(ts.sentenceSpacings), xHeight*MaxSentenceSpacing)
			widthLeft -= float64(ts.sentenceSpacings) * sentenceSpacing
		}
		if ts.wordSpacings > 0 {
			wordSpacing = math.Min(widthLeft/float64(ts.wordSpacings), xHeight*MaxWordSpacing)
			widthLeft -= float64(ts.wordSpacings) * wordSpacing
		}
		if ts.glyphSpacings > 0 {
			glyphSpacing = math.Min(widthLeft/float64(ts.glyphSpacings), xHeight*MaxGlyphSpacing)
		}
	}
	s := ts.ff.f.transform(ts.s, glyphSpacing == 0.0)

	x := 0.0
	p := &Path{}
	var rPrev rune
	iBoundary := 0
	for i, r := range s {
		if i > 0 {
			x += ts.ff.Kerning(rPrev, r)
		}

		pr, advance := ts.ff.ToPath(r)
		pr.Translate(x, 0.0)
		p.Append(pr)
		x += advance

		spacing := glyphSpacing
		if iBoundary < len(ts.textBoundaries) && ts.textBoundaries[iBoundary].pos == i {
			if ts.textBoundaries[iBoundary].kind == wordBoundary {
				spacing = wordSpacing
			} else if ts.textBoundaries[iBoundary].kind == sentenceBoundary {
				spacing = sentenceSpacing
			}
			iBoundary++
		}
		x += spacing
		rPrev = r
	}
	return p
}

func (ts textSpan) splitAtSpacings(spanDx, width float64, f func(float64, float64, float64, string)) {
	spaceWidth := ts.ff.TextWidth(" ")
	sentenceSpacing := 0.0
	wordSpacing := 0.0
	glyphSpacing := 0.0
	if width > ts.textWidth {
		widthLeft := width - ts.textWidth
		xHeight := ts.ff.Metrics().XHeight
		if ts.sentenceSpacings > 0 {
			sentenceSpacing = math.Min(widthLeft/float64(ts.sentenceSpacings), xHeight*MaxSentenceSpacing)
			widthLeft -= float64(ts.sentenceSpacings) * sentenceSpacing
		}
		if ts.wordSpacings > 0 {
			wordSpacing = math.Min(widthLeft/float64(ts.wordSpacings), xHeight*MaxWordSpacing)
			widthLeft -= float64(ts.wordSpacings) * wordSpacing
		}
		if ts.glyphSpacings > 0 {
			glyphSpacing = math.Min(widthLeft/float64(ts.glyphSpacings), xHeight*MaxGlyphSpacing)
		}
	}
	if sentenceSpacing > 0.0 || wordSpacing > 0.0 {
		prevPos := 0
		dx := spanDx
		for _, textBoundary := range ts.textBoundaries {
			s := ts.s[prevPos:textBoundary.pos]
			w := ts.ff.TextWidth(s)
			if glyphSpacing > 0.0 {
				w += float64(utf8.RuneCountInString(s)-1) * glyphSpacing
			}
			f(dx, w, glyphSpacing, s)
			prevPos = textBoundary.pos + 1
			dx += ts.ff.TextWidth(s) + spaceWidth + float64(utf8.RuneCountInString(s))*glyphSpacing
			if textBoundary.kind == wordBoundary {
				dx += wordSpacing
			} else if textBoundary.kind == sentenceBoundary {
				dx += sentenceSpacing
			}
		}
	} else {
		f(spanDx, width, glyphSpacing, ts.s)
	}
}

type textBoundaryKind int

const (
	endBoundary textBoundaryKind = iota
	lineBoundary
	sentenceBoundary
	wordBoundary
	breakBoundary // zero-width space indicates word boundary
)

type textBoundary struct {
	kind textBoundaryKind
	pos  int
	size int
}

func calcTextBoundaries(s string) ([]textBoundary, int, int, int) {
	boundaries := []textBoundary{}
	sentenceSpacings, wordSpacings, glyphSpacings := 0, 0, 0

	var rPrev, rPrevPrev rune
	for i, r := range s {
		glyphSpacings++
		size := utf8.RuneLen(r)
		if r == '\r' || r == '\n' || r == '\f' || r == '\v' || r == '\u2028' || r == '\u2029' {
			if r == '\n' && 0 < i && s[i-1] == '\r' {
				boundaries[len(boundaries)-1].size++
			} else {
				boundaries = append(boundaries, textBoundary{lineBoundary, i, size})
			}
		} else if r == ' ' {
			if (rPrev == '.' && !unicode.IsUpper(rPrevPrev) && rPrevPrev != ' ') || rPrev == '!' || rPrev == '?' {
				boundaries = append(boundaries, textBoundary{sentenceBoundary, i, 1})
				sentenceSpacings++
			} else if rPrev != ' ' {
				// TODO: add breaking spaces such as en quad, em space, hair space, ...
				// see https://unicode.org/reports/tr14/#Properties
				boundaries = append(boundaries, textBoundary{wordBoundary, i, 1})
				wordSpacings++
			}
		} else if r == '\u200b' {
			boundaries = append(boundaries, textBoundary{breakBoundary, i, size})
		}
		rPrevPrev = rPrev
		rPrev = r
	}
	boundaries = append(boundaries, textBoundary{endBoundary, len(s), 0})

	glyphSpacings -= wordSpacings + sentenceSpacings + 1
	return boundaries, sentenceSpacings, wordSpacings, glyphSpacings
}
