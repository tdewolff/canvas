package text

import (
	"fmt"
	"math"
	"unicode"

	"github.com/tdewolff/font"
)

// See: Donald E. Knuth and Michael F. Plass, "Breaking Paragraphs into Lines", 1981
// Implementations:
//   https://github.com/bramstein/typeset (JavaScript) was of great help
//   https://github.com/robertknight/tex-linebreak (JavaScript)
//   https://github.com/akuchling/texlib (Python)

// Special characters:
//   \x09 TAB - breakpoint space
//   \x0A LINE FEED - line separator
//   \x0B VERTICAL TAB - line separator
//   \x0C FORM FEED - line separator
//   \x0D CARRIAGE RETURN - line separator
//   \x20 SPACE - breakpoint space
//   \u00A0 NO-BREAK SPACE - space but not breakpoint
//   \u00AD SOFT HYPHEN - breakpoint with hyphen insertion
//   \u180E MONGOLIAN VOWEL SEPARATOR - space but not breakpoint
//   \u2000 EN QUAD - breakpoint space
//   \u2001 EM QUAD - breakpoint space
//   \u2002 EN SPACE - breakpoint space
//   \u2003 EM SPACE - breakpoint space
//   \u2004 THREE-PER-EM SPACE - breakpoint space
//   \u2005 FOUR-PER-EM SPACE - breakpoint space
//   \u2006 SIX-PER-EM SPACE - breakpoint space
//   \u2007 FIGURE SPACE - breakpoint space
//   \u2008 PUNCTUATION SPACE - breakpoint space
//   \u2009 THIN SPACE - breakpoint space
//   \u200A HAIR SPACE - breakpoint space
//   \u200B ZERO WIDTH SPACE - breakpoint without hyphen insertion
//   \u2028 LINE SEPARATOR - line separator
//   \u2029 PARAGRAPH SEPARATOR - line separator
//   \u202F NARROW NO-BREAK SPACE - space but not breakpoint
//   \u2058 MEDIUM MATHEMATICAL SPACE - breakpoint space
//   \u2060 WORD JOINER - no breakpoint
//   \u3000 IDEOGRAPHIC SPACE - breakpoint space
//   \uFEFF ZERO WIDTH NO-BREAK SPACE - no breakpoint
//
// When to use what?
//   Start a new line: \n
//   Space that doesn't break: \u00A0
//   Word break opportunity with hyphenation: \u00AD
//   Word break opportunity without hyphenation: \u200B
//   Prevent word break: \u2060

// FairyTales is an example text.
const FairyTales = "In olden times when wish\u00ADing still helped one there\u2001lived a king\u2001whose daugh\u00ADters were all beau\u00ADti\u00ADful; and the young\u00ADest was so beautiful that the sun it\u00ADself, which has seen so much, was aston\u00ADished when\u00ADever it shone in her face. Close by the king's castle lay a great dark for\u00ADest, and un\u00ADder an old lime-tree in the for\u00ADest was a well, and when the day was very warm, the king's child went out into the for\u00ADest and sat down by the side of the cool foun\u00ADtain; and when she was bored she took a golden ball, and threw it up on high and caught it; and this ball was her favor\u00ADite play\u00ADthing."

// SpaceStretch is the stretchability of spaces.
var SpaceStretch = 1.0 / 2.0

// SpaceShrink is the shrinkability of spaces.
var SpaceShrink = 1.0 / 3.0

// FrenchSpacing enforces equal widths for inter-word and inter-sentence spaces.
var FrenchSpacing = false

// Stretchability and shrinkability factors for inter-sentence and other types of spaces, not used if FrenchSpacing is set.
var (
	SentenceFactor  = 3.0
	ColonFactor     = 2.0
	SemicolonFactor = 1.5
	CommaFactor     = 1.25
)

// Tolerance is the maximum stretchability of the spaces of a line.
var Tolerance = 2.0

// DemeritsLine is the badness rating for an extra line.
var DemeritsLine = 10.0

// DemeritsFlagged is the badness rating for two consecutive lines ending in hyphens.
var DemeritsFlagged = 100.0

// DemeritsFitness is the badness rating for very different fitness ratings for consecutive lines. Fitness is a categorization of four types for ratio ranges.
var DemeritsFitness = 100.0

// HyphenPenalty is the aesthetic cost of ending a line in a hyphen.
var HyphenPenalty = 50.0

// Infinity specifies infinity as something finite to prevent numerical errors.
var Infinity = 1000.0 // in case of ratio, demerits become about 1e22

// Align is te text alignment.
type Align int

// see Align
const (
	Left Align = iota
	Right
	Centered
	Justified
)

// Type is the item type.
type Type int

// see Type
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

// Item is a box, glue or penalty item.
type Item struct {
	Type
	Width, Stretch, Shrink float64
	Penalty                float64
	Flagged                bool
	Size                   int // number of boxes (glyphs) compressed into one
}

func (item Item) String() string {
	if item.Type == BoxType {
		return fmt.Sprintf("Box[w=%.6g]", item.Width)
	} else if item.Type == GlueType {
		return fmt.Sprintf("Glue[w=%.6g y=%.6g z=%.6g]", item.Width, item.Stretch, item.Shrink)
	} else if item.Type == PenaltyType {
		return fmt.Sprintf("Penalty[p=%.6g]", item.Penalty)
	}
	return "?"
}

// Box returns a box item of given width.
func Box(width float64) Item {
	return Item{
		Type:  BoxType,
		Width: width,
	}
}

// Glue returns a glue item where width is the default width, stretch*Tolerance the maximum width, and shrink*Tolerance the minimum width.
func Glue(width, stretch, shrink float64) Item {
	return Item{
		Type:    GlueType,
		Width:   width,
		Stretch: stretch,
		Shrink:  shrink,
	}
}

// Penalty returns a panalty item with a given penalization factor. For hyphen insertion, width is the hyphen width and flagged should be set to discourage multiple hyphened lines next to each other.
func Penalty(width, penalty float64, flagged bool) Item {
	return Item{
		Type:    PenaltyType,
		Width:   width,
		Penalty: penalty,
		Flagged: flagged,
	}
}

// Breakpoint is a (possible) break point in the string.
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

// Breakpoints is a list of break points.
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

// Has return true if it contains break point b.
func (list *Breakpoints) Has(b *Breakpoint) bool {
	return !(b.prev == nil && b.next == nil && (b != list.head || list.head == nil))
}

// Push adds break point b to the end of the list.
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

// InsertBefore inserts break point b before at.
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

// Remove removes break point b.
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

type linebreaker struct {
	items         []Item
	activeNodes   *Breakpoints
	inactiveNodes *Breakpoints
	W, Y, Z       float64
	width         float64
	nextTolerance float64
}

func newLinebreaker(items []Item, width float64) *linebreaker {
	activeNodes := &Breakpoints{}
	activeNodes.Push(&Breakpoint{Fitness: 1})
	return &linebreaker{
		items:         items,
		activeNodes:   activeNodes,
		inactiveNodes: &Breakpoints{},
		width:         width,
		nextTolerance: math.Inf(1.0),
	}
}

func (lb *linebreaker) computeAdjustmentRatio(b int, active *Breakpoint) float64 {
	// compute the adjustment ratio r from a to b
	L := lb.W - active.W
	if lb.items[b].Type == PenaltyType {
		L += lb.items[b].Width
	}
	ratio := 0.0
	if L < lb.width {
		ratio = (lb.width - L) / (lb.Y - active.Y)
	} else if lb.width < L {
		ratio = (lb.width - L) / (lb.Z - active.Z)
	}
	// limiting positive ratios gives space to distinguish non-stretchable lines
	// allowing negative ratios will break up words that are too long
	return math.Min(ratio, Infinity) // range [-inf,1000]
}

func (lb *linebreaker) computeSum(b int) (float64, float64, float64) {
	// compute tw=(sum w)after(b), ty=(sum y)after(b), and tz=(sum z)after(b)
	// count all glues after the proposed break at b until a box, compulsory break, or explicit space
	// these glues are eaten up by the proposed break and should be taken into account
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

func (lb *linebreaker) mainLoop(b int, tolerance float64) {
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
			if ratio < -1.0 || item.Type == PenaltyType && item.Penalty <= -Infinity {
				lb.activeNodes.Remove(active)
				lb.inactiveNodes.Push(active)
			}
			if -1.0 <= ratio && ratio <= tolerance {
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
			} else if tolerance < ratio {
				// set the next tolerance to the minimum value that changes results
				lb.nextTolerance = math.Min(lb.nextTolerance, ratio)
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

// Linebreak breaks a list of items using Donald Knuth's line breaking algorithm. See Donald E. Knuth and Michael F. Plass, "Breaking Paragraphs into Lines", 1981
func Linebreak(items []Item, width float64, looseness int) ([]*Breakpoint, bool) {
	overflows := false
	tolerance := Tolerance

START:
	// create an active node representing the beginning of the paragraph
	lb := newLinebreaker(items, width)
	// if index is a legal breakpoint then main loop
	for b, item := range lb.items {
		if item.Type == BoxType {
			lb.W += item.Width
		} else if item.Type == GlueType {
			// additionally don't check glue if the next element is a penalty (not in the original algorithm), this optimizes the search space
			if 0 < b && lb.items[b-1].Type == BoxType && lb.items[b+1].Type != PenaltyType {
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
			if tolerance != lb.nextTolerance {
				tolerance = lb.nextTolerance
				goto START
			} else {
				// line doesn't fit, amongst the rejected active set we choose the ones that stick out the least and continue
				overflows = true
				minWidth := math.Inf(1.0)
				for parent := lb.inactiveNodes.head; parent != nil; parent = parent.next {
					minWidth = math.Min(minWidth, lb.W-parent.W)
				}
				for parent := lb.inactiveNodes.head; parent != nil; parent = parent.next {
					if lb.W-parent.W == minWidth {
						breakpoint := &Breakpoint{
							parent:   parent,
							Position: b,
							Line:     parent.Line + 1,
							Fitness:  1,
							Width:    lb.W,
							W:        lb.W,
							Y:        lb.Y,
							Z:        lb.Z,
							Ratio:    0.0,
							Demerits: parent.Demerits + 1000.0,
						}
						lb.activeNodes.Push(breakpoint)
					}
				}
			}
		}
	}

	// either len(items)==0 or we couldn't find feasible line breaks
	if lb.activeNodes.head == nil {
		return []*Breakpoint{{
			Position: len(lb.items) - 1,
		}}, !overflows
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
	if 1 < len(breaks) {
		breaks = breaks[1:]
	}
	return breaks, !overflows
}

func IsSpace(r rune) bool {
	// no-break spaces such as U+00A0, U+180E, U+202F, and U+FEFF are used as boxes
	spaces := []rune(" \t\u2000\u2001\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200A\u205F\u3000")
	for _, space := range spaces {
		if r == space {
			return true
		}
	}
	return false
}

func IsNewline(r rune) bool {
	newlines := []rune("\r\n\f\v\u0085\u2028\u2029")
	for _, newline := range newlines {
		if r == newline {
			return true
		}
	}
	return false
}

// GlyphsToItems converts a slice of glyphs into the box/glue/penalty items model as used by Knuth's line breaking algorithm. The SFNT and Size of each glyph must be set. Indent and align specify the indentation width of the first line and the alignment (left, right, centered, justified) of the lines respectively.
func GlyphsToItems(glyphs []Glyph, indent float64, align Align) []Item {
	if len(glyphs) == 0 {
		return []Item{}
	}

	// the average space width used for left, right, centered alignment
	stretchWidth := 0.0
	if align != Justified {
		n := 0.0
		for _, glyph := range glyphs {
			if IsSpace(glyph.Text) {
				stretchWidth += glyph.Advance()
				n += 1.0
			}
		}
		if 1e-8 < n {
			stretchWidth /= n
		}
	}

	// trim spaces from start and end
	padStart, padEnd := Box(0.0), Box(0.0)
	first, last := 0, len(glyphs)
	for i := 0; i < len(glyphs); i++ {
		if IsSpace(glyphs[i].Text) {
			padStart.Width += glyphs[i].Advance()
			padStart.Size++
			first++
		} else {
			break
		}
	}
	for i := len(glyphs) - 1; first <= i; i-- {
		if IsSpace(glyphs[i].Text) {
			padEnd.Width += glyphs[i].Advance()
			padEnd.Size++
			last--
		} else {
			break
		}
	}

	items := []Item{}
	items = append(items, Box(indent))
	if padStart.Size != 0 {
		items[0].Width += padStart.Width
		items[0].Size += padStart.Size
		items = append(items, Penalty(0, 0, false))
	}
	if align == Centered {
		items = append(items, Glue(0.0, stretchWidth, 0.0))
	}
	for i := first; i < last; i++ {
		glyph := glyphs[i]
		if IsSpace(glyph.Text) {
			spaceWidth := glyph.Advance()
			spaceFactor := 1.0
			if !FrenchSpacing && align == Justified {
				j := i - 1
				if 0 <= j && (glyphs[j].Text == ')' || glyphs[j].Text == ']' || glyphs[j].Text == '\'' || glyphs[j].Text == '"') {
					j--
				}
				if 0 <= j && (j == 0 || !unicode.IsUpper(glyphs[j-1].Text)) {
					switch glyphs[j].Text {
					case '.', '!', '?':
						spaceFactor = SentenceFactor
					case ':':
						spaceFactor = ColonFactor
					case ';':
						spaceFactor = SemicolonFactor
					case ',':
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
		} else if IsNewline(glyph.Text) {
			// only add one penalty for \r\n
			if glyph.Text != '\n' || i == 0 || glyphs[i-1].Text != '\r' {
				items = append(items, Penalty(0.0, -Infinity, false))
			}
			items[len(items)-1].Size++
		} else if glyph.Text == '\u00AD' || glyph.Text == '\u200B' {
			// optional hyphens
			var hyphenWidth float64
			if glyph.Text == '\u00AD' {
				if !glyph.Vertical {
					hyphenWidth = float64(glyph.SFNT.GlyphAdvance(glyph.SFNT.GlyphIndex('-')))
				} else {
					hyphenWidth = float64(glyph.SFNT.GlyphVerticalAdvance(glyph.SFNT.GlyphIndex('-')))
				}
				hyphenWidth *= glyph.Size / float64(glyph.SFNT.Head.UnitsPerEm)
			}
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
			width := glyph.Advance()
			if 1 < len(items) && items[len(items)-1].Type == BoxType {
				if IsSpacelessScript(glyph.Script) || IsSpacelessScript(glyphs[i-1].Script) {
					// allow breaks around spaceless script glyphs, most commonly CJK
					items = append(items, Penalty(0.0, 0.0, false))
					items = append(items, Box(width))
				} else {
					// merge with previous box only if it's not indent
					items[len(items)-1].Width += width
				}
			} else {
				items = append(items, Box(width))
			}
			items[len(items)-1].Size++
		}
		if glyph.Text == '-' {
			// optional break after hyphen
			items = append(items, Penalty(0.0, HyphenPenalty, true))
		}
	}
	if padEnd.Size != 0 {
		items = append(items, padEnd)
	}
	if align == Centered {
		items = append(items, Glue(0.0, stretchWidth, 0.0))
		items = append(items, Penalty(0.0, -Infinity, false))
	} else {
		items = append(items, Glue(0.0, math.Inf(1.0), 0.0))
		items = append(items, Penalty(0.0, -Infinity, true))
	}
	return items
}

// LinebreakGlyphs breaks a slice of glyphs uing the given SFNT font and font size. The indent and width specify the first line's indentation and the maximum line's width respectively. Align sets the horizontal alignment of the text. The looseness specifies whether it is desirable to have less or more lines than optimal.
func LinebreakGlyphs(sfnt *font.SFNT, size float64, glyphs []Glyph, indent, width float64, align Align, looseness int) [][]Glyph {
	if len(glyphs) == 0 {
		return [][]Glyph{}
	}
	for i := range glyphs {
		glyphs[i].SFNT = sfnt
		glyphs[i].Size = size
	}
	spaceID := sfnt.GlyphIndex(' ')
	hyphenID := sfnt.GlyphIndex('-')
	toUnits := float64(sfnt.Head.UnitsPerEm) / size

	items := GlyphsToItems(glyphs, indent, align)
	breaks, _ := Linebreak(items, width, looseness)

	i, j := 0, 0 // index into: glyphs, breaks/lines
	atStart := true
	glyphLines := [][]Glyph{{}}
	if align == Right {
		glyphLines[j] = append(glyphLines[j], Glyph{SFNT: sfnt, Size: size, ID: spaceID, Text: ' ', XAdvance: int32((width - breaks[j].Width) * toUnits)})
	}
	for position, item := range items {
		if position == breaks[j].Position {
			if item.Type == PenaltyType && item.Flagged && item.Width != 0.0 {
				if 0 < len(glyphLines[j]) && glyphLines[j][len(glyphLines[j])-1].ID == spaceID {
					glyphLines[j] = glyphLines[j][:len(glyphLines[j])-1]
				}
				glyphLines[j] = append(glyphLines[j], Glyph{SFNT: sfnt, Size: size, ID: hyphenID, Text: '-', XAdvance: int32(item.Width * toUnits)})
			}
			glyphLines = append(glyphLines, []Glyph{})
			if j+1 < len(breaks) {
				j++
			}
			if align == Right {
				glyphLines[j] = append(glyphLines[j], Glyph{SFNT: sfnt, Size: size, ID: spaceID, Text: ' ', XAdvance: int32((width - breaks[j].Width) * toUnits)})
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
				glyphLines[j] = append(glyphLines[j], Glyph{SFNT: sfnt, Size: size, ID: spaceID, Text: ' ', XAdvance: int32(width * toUnits)})
			}
		}
	}
	if 0 < len(glyphLines[j]) && glyphLines[j][len(glyphLines[j])-1].ID == spaceID {
		glyphLines[j] = glyphLines[j][:len(glyphLines[j])-1]
	}
	return glyphLines
}
