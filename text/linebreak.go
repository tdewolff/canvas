package text

import (
	"fmt"
	"math"
	"unicode"
	"unicode/utf8"

	"github.com/tdewolff/canvas/font"
)

// See: Donald E. Knuth and Michael F. Plass, "Breaking Paragraphs into Lines", 1981
// Implementations:
//   https://github.com/bramstein/typeset (JavaScript) was of great help
//   https://github.com/robertknight/tex-linebreak (JavaScript)
//   https://github.com/akuchling/texlib (Python)

const FairyTales = "In olden times when wish\u200Bing still helped one there\u2001lived a king\u2001whose daugh\u200Bters were all beau\u200Bti\u200Bful; and the young\u200Best was so beautiful that the sun it\u200Bself, which has seen so much, was aston\u200Bished when\u200Bever it shone in her face. Close by the king's castle lay a great dark for\u200Best, and un\u200Bder an old lime-tree in the for\u200Best was a well, and when the day was very warm, the king's child went out into the for\u200Best and sat down by the side of the cool foun\u200Btain; and when she was bored she took a golden ball, and threw it up on high and caught it; and this ball was her favor\u200Bite play\u200Bthing."

// FrenchSpacing enforces equal widths for inter-word and inter-sentence spaces
var FrenchSpacing = false

// stretchability and shrinkability of spaces
var SpaceStretch = 1.0 / 2.0
var SpaceShrink = 1.0 / 3.0

// stretchability and shrinkability factors for inter-sentence and other types of spaces
// not used if FrenchSpacing is set
var SentenceFactor = 3.0
var ColonFactor = 2.0
var SemicolonFactor = 1.5
var CommaFactor = 1.25

// algorithmic parameters
var Tolerance = 10.0 // TeX uses 200
var DemeritsLine = 10.0
var DemeritsFlagged = 100.0 // TeX uses 10000
var DemeritsFitness = 100.0 // TeX uses 10000
var HyphenPenalty = 50.0
var Infinity = 1000.0 // in case of ratio, demerits becomes about 1e22

type Align int

const (
	Left Align = iota
	Right
	Centered
	Justified
)

type Type int

const (
	BoxType Type = iota
	GlueType
	PenaltyType
)

func (t Type) String() string {
	switch t {
	case BoxType:
		return "Box"
	case GlueType:
		return "Glue"
	case PenaltyType:
		return "Penalty"
	}
	return fmt.Sprintf("Type(%d)", t)
}

type Item struct {
	Type
	Width, Stretch, Shrink float64
	Penalty                float64
	Flagged                bool
	Size                   int // number of boxes (glyphs) compressed into one
}

func Box(width float64) Item {
	return Item{
		Type:  BoxType,
		Width: width,
	}
}

func Glue(width, stretch, shrink float64) Item {
	return Item{
		Type:    GlueType,
		Width:   width,
		Stretch: stretch,
		Shrink:  shrink,
	}
}

func Penalty(width, penalty float64, flagged bool) Item {
	return Item{
		Type:    PenaltyType,
		Width:   width,
		Penalty: penalty,
		Flagged: flagged,
	}
}

type Breakpoint struct {
	next, prev *Breakpoint
	parent     *Breakpoint

	Position int
	Line     int
	Fitness  int
	Width    float64
	W, Y, Z  float64
	Ratio    float64
	Demerits float64
}

func (br *Breakpoint) String() string {
	s := ""
	n := br
	for n != nil {
		s += fmt.Sprintf("%v>", n.Position)
		n = n.parent
	}
	return s[:len(s)-1]
}

type Breakpoints struct {
	head, tail *Breakpoint
}

func (list *Breakpoints) String() string {
	if list.head == nil {
		return "nil"
	}

	s := ""
	n := list.head
	for n != nil {
		s += fmt.Sprintf("%v ", n)
		n = n.next
	}
	return s[:len(s)-1]
}

func (list *Breakpoints) Has(b *Breakpoint) bool {
	return !(b.prev == nil && b.next == nil && (b != list.head || list.head == nil))
}

func (list *Breakpoints) Push(b *Breakpoint) {
	if list.head == nil {
		list.head = b
		list.tail = b
	} else if !list.Has(b) {
		b.prev = list.tail
		list.tail.next = b
		list.tail = b
	}
}

func (list *Breakpoints) InsertBefore(b *Breakpoint, at *Breakpoint) {
	if list.Has(b) || !list.Has(at) {
		return
	}
	b.next = at
	if at.prev == nil {
		list.head = b
	} else {
		b.prev = at.prev
		at.prev.next = b
	}
	at.prev = b
}

func (list *Breakpoints) Remove(b *Breakpoint) {
	if !list.Has(b) {
		return
	}
	if b.prev == nil {
		list.head = b.next
	} else {
		b.prev.next = b.next
	}
	if b.next == nil {
		list.tail = b.prev
	} else {
		b.next.prev = b.prev
	}
	b.prev = nil
	b.next = nil
}

type Linebreaker struct {
	items       []Item
	activeNodes *Breakpoints
	W, Y, Z     float64
	width       float64
}

func NewLinebreaker(items []Item, width float64) *Linebreaker {
	activeNodes := &Breakpoints{}
	activeNodes.Push(&Breakpoint{Fitness: 1})
	return &Linebreaker{
		items:       items,
		activeNodes: activeNodes,
		width:       width,
	}
}

func (lb *Linebreaker) computeAdjustmentRatio(b int, active *Breakpoint) float64 {
	// compute the adjustment ratio r from a to b
	L := lb.W - active.W
	if lb.items[b].Type == PenaltyType {
		L += lb.items[b].Width
	}
	//j := active.line + 1
	// using a finite value for infinity allows to break lines without a glue (one word)
	// allowing negative ratios will break up words that are too long
	ratio := 0.0
	if L < lb.width {
		ratio = (lb.width - L) / (lb.Y - active.Y)
	} else if lb.width < L {
		ratio = (lb.width - L) / (lb.Z - active.Z)
	}
	return math.Min(math.Max(ratio, -Infinity), Infinity)
}

func (lb *Linebreaker) computeSum(b int) (float64, float64, float64) {
	// compute tw=(sum w)after(b), ty=(sum y)after(b), and tz=(sum z)after(b)
	W, Y, Z := lb.W, lb.Y, lb.Z
	for i, item := range lb.items[b:] {
		if item.Type == BoxType || (item.Type == PenaltyType && item.Penalty <= -Infinity && 0 < i) {
			break
		} else if item.Type == GlueType {
			W += item.Width
			Y += item.Stretch
			Z += item.Shrink
		}
	}
	return W, Y, Z
}

func (lb *Linebreaker) mainLoop(b int, tolerance float64) {
	item := lb.items[b]
	active := lb.activeNodes.head

	// the inner loop iterates through active nodes at a certain line, while the outer loop iterates over lines
	for active != nil {
		Dmin := math.Inf(1.0)
		// per fitness class, we have demerits (D), active nodes (A), and ratios (R)
		D := [4]float64{Dmin, Dmin, Dmin, Dmin}
		A := [4]*Breakpoint{}
		R := [4]float64{}
		for active != nil {
			next := active.next
			ratio := lb.computeAdjustmentRatio(b, active)
			// permit exceeding lines for desperate second try with infinite tolerance
			ratioLB := -1.0 <= ratio || math.IsInf(tolerance, 1.0) && ratio == -Infinity
			if !ratioLB || item.Type == PenaltyType && item.Penalty <= -Infinity {
				lb.activeNodes.Remove(active)
			}
			if ratioLB && ratio <= tolerance {
				// compute demerits d and fitness class c
				badness := 100.0 * math.Pow(math.Abs(ratio), 3.0)
				demerits := 0.0
				if item.Type == PenaltyType && 0.0 <= item.Penalty {
					// positive penalty
					demerits = math.Pow(DemeritsLine+badness+item.Penalty, 2.0)
				} else if item.Type == PenaltyType && -Infinity < item.Penalty {
					// negative but not a forced break
					demerits = math.Pow(DemeritsLine+badness, 2.0) - math.Pow(item.Penalty, 2.0)
				} else {
					// other cases
					demerits = math.Pow(DemeritsLine+badness, 2.0)
				}
				if lb.items[active.Position].Flagged && item.Flagged {
					demerits += DemeritsFlagged
				}

				c := 3
				if ratio < -0.5 {
					c = 0
				} else if ratio <= 0.5 {
					c = 1
				} else if ratio <= 1.0 {
					c = 2
				}
				if 1.0 < math.Abs(float64(c-active.Fitness)) {
					demerits += DemeritsFitness
				}
				demerits += active.Demerits

				if demerits < D[c] {
					D[c] = demerits
					A[c] = active
					R[c] = ratio
					if demerits < Dmin {
						Dmin = demerits
					}
				}
			}

			j := active.Line + 1
			active = next

			// stop adding candidates of the current line and move on to the next line
			if active != nil && j <= active.Line {
				// we omitted (j < j0) as j0 is difficult to know for complex cases
				break
			}
		}

		if Dmin < math.Inf(1.0) {
			// insert new active node for break from A[c] to the current item
			W, Y, Z := lb.computeSum(b)
			width := lb.W
			if lb.items[b].Type == PenaltyType {
				width += lb.items[b].Width
			}
			for c := 0; c < len(D); c++ {
				if D[c] <= Dmin+DemeritsFitness {
					breakpoint := &Breakpoint{
						parent:   A[c],
						Position: b,
						Line:     A[c].Line + 1,
						Fitness:  c,
						Width:    width,
						W:        W,
						Y:        Y,
						Z:        Z,
						Ratio:    R[c],
						Demerits: D[c],
					}
					if active == nil {
						lb.activeNodes.Push(breakpoint)
					} else {
						lb.activeNodes.InsertBefore(breakpoint, active)
					}
				}
			}
		}
	}
}

func Linebreak(items []Item, width float64, looseness int) []*Breakpoint {
	tolerance := Tolerance

START:
	// create an active node representing the beginning of the paragraph
	lb := NewLinebreaker(items, width)
	// if index is a legal breakpoint then main loop
	for b, item := range lb.items {
		if item.Type == BoxType {
			lb.W += item.Width
		} else if item.Type == GlueType {
			if 0 < b && lb.items[b-1].Type == BoxType {
				lb.mainLoop(b, tolerance)
			}
			lb.W += item.Width
			lb.Y += item.Stretch
			lb.Z += item.Shrink
		} else if item.Type == PenaltyType && item.Penalty < Infinity {
			lb.mainLoop(b, tolerance)
		}

		// do something drastic since there is no feasible solution
		if lb.activeNodes.head == nil {
			if !math.IsInf(tolerance, 1.0) {
				tolerance = math.Inf(1.0)
				goto START
			} else {
				break
			}
		}
	}

	// either len(items)==0 or we couldn't find feasible line breaks
	if lb.activeNodes.head == nil {
		return []*Breakpoint{&Breakpoint{
			Position: len(lb.items) - 1,
		}}
	}

	// choose the active node with fewest total demerits
	b := &Breakpoint{Demerits: math.Inf(1.0)}
	for a := lb.activeNodes.head; a != nil; a = a.next {
		if a.Demerits < b.Demerits {
			b = a
		}
	}

	// choose the appropriate active node
	if looseness != 0 {
		s := 0
		k := b.Line
		for a := lb.activeNodes.head; a != nil; a = a.next {
			delta := a.Line - k
			if looseness <= delta && delta < s || s < delta && delta <= looseness {
				s = delta
				b = a
			} else if delta == s && a.Demerits < b.Demerits {
				b = a
			}
		}
	}

	// use the chosen node to determine the optimum breakpoint sequence
	breaks := make([]*Breakpoint, b.Line+1)
	for b != nil {
		if b.Line+1 < len(breaks) {
			breaks[b.Line+1].Width -= b.W
		}
		if b.Ratio < -1.0 || Tolerance < b.Ratio {
			b.Ratio = 0.0
		}
		breaks[b.Line] = b
		b = b.parent
	}
	breaks = breaks[1:]
	return breaks
}

func isUpper(s string) bool {
	for _, r := range s {
		if !unicode.IsUpper(r) {
			return false
		}
	}
	return true
}

func isSpace(s string) bool {
	// no-break spaces such as U+00A0, U+202F, U+180E, and U+FEFF are used as boxes
	spaces := []rune(" \u2000\u2001\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200A\u205F")
	if r, size := utf8.DecodeRuneInString(s); size == len(s) {
		for _, space := range spaces {
			if r == space {
				return true
			}
		}
	}
	return false
}

func isNewline(s string) bool {
	newlines := []rune("\r\n\f\v\u0085\u2028\u2029")
	if r, size := utf8.DecodeRuneInString(s); size == len(s) {
		for _, newline := range newlines {
			if r == newline {
				return true
			}
		}
	}
	return false
}

// GlyphsToItems converts a slice of glyphs into the box/glue/penalty items model as used by Knuth's line breaking algorithm. The SFNT and Size of each glyph must be set. Indent and align specify the indentation width of the first line and the alignment (left, right, centered, justified) of the lines respectively. Finally, stretchWidth is typically twice the width of a space and is inserted as stretchable glue for alignment purposes.
func GlyphsToItems(glyphs []Glyph, indent float64, align Align, vertical bool) []Item {
	if len(glyphs) == 0 {
		return []Item{}
	}

	stretchWidth := 0.0 // the average space width used for left, right, centered alignment
	if align != Justified {
		n := 0.0
		for _, glyph := range glyphs {
			if isSpace(glyph.Text) {
				if !vertical {
					stretchWidth += float64(glyph.XAdvance) * glyph.Size / float64(glyph.SFNT.Head.UnitsPerEm)
				} else {
					stretchWidth += float64(-glyph.YAdvance) * glyph.Size / float64(glyph.SFNT.Head.UnitsPerEm)
				}
				n += 1.0
			}
		}
		stretchWidth /= n
	}

	items := []Item{}
	items = append(items, Box(indent))
	if align == Centered {
		items = append(items, Glue(0.0, stretchWidth, 0.0))
	}
	for i, glyph := range glyphs {
		if isSpace(glyph.Text) {
			var spaceWidth float64
			if !vertical {
				spaceWidth = float64(glyph.XAdvance) * glyph.Size / float64(glyph.SFNT.Head.UnitsPerEm)
			} else {
				spaceWidth = float64(-glyph.YAdvance) * glyph.Size / float64(glyph.SFNT.Head.UnitsPerEm)
			}
			spaceFactor := 1.0
			if !FrenchSpacing && align == Justified {
				j := i - 1
				if 0 <= j && (glyphs[j].Text == ")" || glyphs[j].Text == "]" || glyphs[j].Text == "'" || glyphs[j].Text == "\"") {
					j--
				}
				if 0 <= j && (j == 0 || !isUpper(glyphs[j-1].Text)) {
					switch glyphs[j].Text {
					case ".", "!", "?":
						spaceFactor = SentenceFactor
					case ":":
						spaceFactor = ColonFactor
					case ";":
						spaceFactor = SemicolonFactor
					case ",":
						spaceFactor = CommaFactor
					}
				}
			}
			var w, y, z float64
			if align == Justified {
				w = spaceWidth
				y = spaceWidth * SpaceStretch * spaceFactor
				z = spaceWidth * SpaceShrink / spaceFactor
			} else if align == Left || align == Right || align == Centered {
				w = 0.0
				y = stretchWidth
				z = 0.0
			}
			if items[len(items)-1].Type == GlueType {
				items[len(items)-1].Width += w
				items[len(items)-1].Stretch += y
				items[len(items)-1].Shrink += z
			} else {
				items = append(items, Glue(w, y, z))
			}
			if align == Justified {
				items[len(items)-1].Size++
			} else if align == Left || align == Right {
				items = append(items, Penalty(0.0, 0.0, false))
				items = append(items, Glue(spaceWidth, -stretchWidth, 0.0))
				items[len(items)-1].Size++
			} else if align == Centered {
				items = append(items, Penalty(0.0, 0.0, false))
				items = append(items, Glue(spaceWidth, -stretchWidth, 0.0))
				items[len(items)-1].Size++
				items = append(items, Box(0.0))
				items = append(items, Penalty(0.0, Infinity, false))
				items = append(items, Glue(0.0, stretchWidth, 0.0))
			}
		} else if isNewline(glyph.Text) {
			if glyph.Text != "\n" || i == 0 || glyphs[i-1].Text != "\r" {
				items = append(items, Penalty(0.0, -Infinity, false))
			}
			items[len(items)-1].Size++
		} else if glyph.Text == "\u200B" {
			// optional hyphens
			var hyphenWidth float64
			if !vertical {
				hyphenWidth = float64(glyph.SFNT.GlyphAdvance(glyph.SFNT.GlyphIndex('-')))
			} else {
				hyphenWidth = float64(glyph.SFNT.GlyphVerticalAdvance(glyph.SFNT.GlyphIndex('-')))
			}
			hyphenWidth *= glyph.Size / float64(glyph.SFNT.Head.UnitsPerEm)
			if align == Justified {
				items = append(items, Penalty(hyphenWidth, HyphenPenalty, true))
				items[len(items)-1].Size++
			} else if align == Left || align == Right {
				items = append(items, Penalty(0.0, Infinity, false))
				items = append(items, Glue(0.0, stretchWidth, 0.0))
				items = append(items, Penalty(hyphenWidth, 10.0*HyphenPenalty, true))
				items[len(items)-1].Size++
				items = append(items, Glue(0.0, -stretchWidth, 0.0))
			} else if align == Centered {
				// nothing
			}
		} else {
			// glyphs
			var width float64
			if !vertical {
				width = float64(glyph.XAdvance) * glyph.Size / float64(glyph.SFNT.Head.UnitsPerEm)
			} else {
				width = float64(-glyph.YAdvance) * glyph.Size / float64(glyph.SFNT.Head.UnitsPerEm)
			}
			if 1 < len(items) && items[len(items)-1].Type == BoxType {
				// merge with previous box only if it's not indent
				items[len(items)-1].Width += width
			} else {
				items = append(items, Box(width))
			}
			items[len(items)-1].Size++
		}
		if glyph.Text == "-" {
			// optional break after hyphen
			items = append(items, Penalty(0.0, HyphenPenalty, true))
		}
	}
	if align == Centered {
		items = append(items, Glue(0.0, stretchWidth, 0.0))
		items = append(items, Penalty(0.0, -Infinity, false))
	} else {
		items = append(items, Glue(0.0, 1.0e6, 0.0)) // using inf can causes NaNs
		items = append(items, Penalty(0.0, -Infinity, true))
	}
	return items
}

func LinebreakGlyphs(sfnt *font.SFNT, size float64, glyphs []Glyph, indent, width float64, align Align, looseness int) [][]Glyph {
	if len(glyphs) == 0 {
		return [][]Glyph{}
	}
	for i, _ := range glyphs {
		glyphs[i].SFNT = sfnt
		glyphs[i].Size = size
	}
	spaceID := sfnt.GlyphIndex(' ')
	hyphenID := sfnt.GlyphIndex('-')
	toUnits := float64(sfnt.Head.UnitsPerEm) / size

	vertical := false
	items := GlyphsToItems(glyphs, indent, align, vertical)
	breaks := Linebreak(items, width, looseness)

	i, j := 0, 0 // index into: glyphs, breaks/lines
	atStart := true
	glyphLines := [][]Glyph{[]Glyph{}}
	if align == Right {
		glyphLines[j] = append(glyphLines[j], Glyph{SFNT: sfnt, Size: size, ID: spaceID, Text: " ", XAdvance: int32((width - breaks[j].Width) * toUnits)})
	}
	for position, item := range items {
		if position == breaks[j].Position {
			if item.Type == PenaltyType && item.Flagged && item.Width != 0.0 {
				if 0 < len(glyphLines[j]) && glyphLines[j][len(glyphLines[j])-1].ID == spaceID {
					glyphLines[j] = glyphLines[j][:len(glyphLines[j])-1]
				}
				glyphLines[j] = append(glyphLines[j], Glyph{SFNT: sfnt, Size: size, ID: hyphenID, Text: "-", XAdvance: int32(item.Width * toUnits)})
			}
			glyphLines = append(glyphLines, []Glyph{})
			if j+1 < len(breaks) {
				j++
			}
			if align == Right {
				glyphLines[j] = append(glyphLines[j], Glyph{SFNT: sfnt, Size: size, ID: spaceID, Text: " ", XAdvance: int32((width - breaks[j].Width) * toUnits)})
			}
			atStart = true
		} else if item.Type == BoxType {
			glyphLines[j] = append(glyphLines[j], glyphs[i:i+item.Size]...)
			atStart = false
			i += item.Size
		} else if item.Type == GlueType && !atStart {
			width := item.Width
			if 0.0 <= breaks[j].Ratio {
				if !math.IsInf(item.Stretch, 0.0) {
					width += breaks[j].Ratio * item.Stretch
				}
			} else if !math.IsInf(item.Shrink, 0.0) {
				width += breaks[j].Ratio * item.Shrink
			}
			if 0 < len(glyphLines[j]) && glyphLines[j][len(glyphLines[j])-1].ID == spaceID {
				glyphLines[j][len(glyphLines[j])-1].XAdvance += int32(width * toUnits)
			} else {
				glyphLines[j] = append(glyphLines[j], Glyph{SFNT: sfnt, Size: size, ID: spaceID, Text: " ", XAdvance: int32(width * toUnits)})
			}
		}
	}
	if 0 < len(glyphLines[j]) && glyphLines[j][len(glyphLines[j])-1].ID == spaceID {
		glyphLines[j] = glyphLines[j][:len(glyphLines[j])-1]
	}
	return glyphLines
}
