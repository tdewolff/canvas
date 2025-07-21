package text

import (
	"fmt"
	"math"

	"github.com/tdewolff/font"
)

var Epsilon = 1e-6

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
var SpaceStretch = 1.0 / 2.0 // ratio of the space that can be added

// SpaceShrink is the shrinkability of spaces.
var SpaceShrink = 1.0 / 3.0 // ratio of the space that can be removed

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

func (a Align) String() string {
	switch a {
	case Left:
		return "Left"
	case Right:
		return "Right"
	case Centered:
		return "Centered"
	case Justified:
		return "Justified"
	}
	return fmt.Sprint(int(a))
}

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
	Width, Stretch, Shrink float64 // Width is the natural width, Stretch the width that can be added, and Shrink that can be removed
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

type Items []Item

// Box adds a box item (a word) of the given fixed width.
func (items Items) Box(width float64) Items {
	items = append(items, Item{
		Type:  BoxType,
		Width: width,
	})
	return items
}

// Glue adds a glue item (a space) where width is the default width, stretch*Tolerance the maximum width, and shrink*Tolerance the minimum width.
func (items Items) Glue(width, stretch, shrink float64) Items {
	items = append(items, Item{
		Type:    GlueType,
		Width:   width,
		Stretch: stretch,
		Shrink:  shrink,
	})
	return items
}

// Penalty adds a penalty item (explicit or possible newline, hyphen) with a given penalization factor. For hyphen insertion, width is the hyphen width and flagged should be set to discourage multiple hyphened lines next to each other. For explicit newlines the penalty is -Infinity.
func (items Items) Penalty(width, penalty float64, flagged bool) Items {
	items = append(items, Item{
		Type:    PenaltyType,
		Width:   width,
		Penalty: penalty,
		Flagged: flagged,
	})
	return items
}

type Break struct {
	Position               int
	Width, Stretch, Shrink float64
	Ratio                  float64
}

// Breakpoint is a (possible) break point in the string.
type Breakpoint struct {
	next, prev *Breakpoint
	parent     *Breakpoint

	Break
	Line     int
	Fitness  int
	W, Y, Z  float64
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
		if lb.Y-active.Y == 0.0 {
			// unstretchable line, act as if we have a very small stretchable space.
			// this helps to distinguish between between smaller and longer lines if both would
			// need to stretched beyond the tolerance
			return Infinity * (1.0 + (lb.width-L)/lb.width)
		}
		ratio = (lb.width - L) / (lb.Y - active.Y)
	} else if lb.width < L {
		ratio = (lb.width - L) / (lb.Z - active.Z)
	}
	// limiting positive ratios gives space to distinguish non-stretchable lines
	// allowing negative ratios will break up words that are too long
	return math.Min(ratio, Infinity) // range [-inf,1000]
}

func (lb *linebreaker) sumBreakGlues(b int) (float64, float64, float64) {
	// compute tw=(sum w)after(b), ty=(sum y)after(b), and tz=(sum z)after(b)
	// count the glue at or right after the break, they disappear (this is different from the original algorithm)
	W, Y, Z := lb.W, lb.Y, lb.Z
	if lb.items[b].Type == GlueType {
		W += lb.items[b].Width
		Y += lb.items[b].Stretch
		Z += lb.items[b].Shrink
	}
	if b+1 < len(lb.items) && lb.items[b+1].Type == GlueType {
		W += lb.items[b+1].Width
		Y += lb.items[b+1].Stretch
		Z += lb.items[b+1].Shrink
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
			W, Y, Z := lb.sumBreakGlues(b)
			width := lb.W
			if lb.items[b].Type == PenaltyType {
				width += lb.items[b].Width
			}
			for c := 0; c < len(D); c++ {
				if D[c] <= Dmin+DemeritsFitness {
					breakpoint := &Breakpoint{
						parent: A[c],
						Break: Break{
							Position: b,
							Width:    width,
							Stretch:  lb.Y,
							Shrink:   lb.Z,
							Ratio:    R[c],
						},
						Line:     A[c].Line + 1,
						Fitness:  c,
						W:        W,
						Y:        Y,
						Z:        Z,
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

// KnuthLinebreak breaks a list of items using Donald Knuth's line breaking algorithm. See Donald E. Knuth and Michael F. Plass, "Breaking Paragraphs into Lines", 1981
func KnuthLinebreak(items Items, width float64, looseness int) ([]Break, bool) {
	overflows := false
	tolerance := Tolerance
	var breaks []Break

	// break up paragraphs, ie. between explicit newlines, to somewhat linearise this function
	for first, last := 0, 0; first < len(items); first = last {
		if items[first].Type == GlueType {
			first++
		}
		for last < len(items) && (items[last].Type != PenaltyType || items[last].Penalty != -Infinity) {
			last++
		}
		last++

	START:
		// create an active node representing the beginning of the paragraph
		lb := newLinebreaker(items[first:last], width)
		// if index is a legal breakpoint then main loop
		for b, item := range lb.items {
			if item.Type == BoxType {
				lb.W += item.Width
			} else if item.Type == GlueType {
				// additionally don't check glue if it has zero width (eg. before a penalty, not in the original algorithm), this optimizes the search space
				if 0 < b && lb.items[b-1].Type == BoxType && 0.0 < item.Width {
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
								parent: parent,
								Break: Break{
									Position: b,
									Width:    lb.W,
									Stretch:  lb.Y,
									Shrink:   lb.Z,
									Ratio:    0.0,
								},
								Line:     parent.Line + 1,
								Fitness:  1,
								W:        lb.W,
								Y:        lb.Y,
								Z:        lb.Z,
								Demerits: parent.Demerits + 1000.0,
							}
							lb.activeNodes.Push(breakpoint)
						}
					}
				}
			}
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
		line0 := len(breaks)
		breaks = append(breaks, make([]Break, b.Line)...)
		for b != nil && b.Position != 0 {
			if line0+b.Line < len(breaks) {
				// values are cumulative, take their difference
				breaks[line0+b.Line].Width -= b.W
				breaks[line0+b.Line].Stretch -= b.Y
				breaks[line0+b.Line].Shrink -= b.Z
			}
			if b.Ratio < -1.0 || Tolerance < b.Ratio {
				b.Ratio = 0.0
			}
			breaks[line0+b.Line-1] = b.Break
			breaks[line0+b.Line-1].Position += first
			b = b.parent
		}
	}
	return breaks, !overflows
}

// GreedyLinebreak breaks a list of items using a greedy line breaking algorithm. This is much faster than Knuth's algorithm.
func GreedyLinebreak(items Items, width float64) ([]Break, bool) {
	overflows := false
	breaks := []Break{}
	w, y, z := 0.0, 0.0, 0.0 // of glues between boxes
	W, Y, Z := 0.0, 0.0, 0.0 // of line

	i, b := 0, -1
	for ; i < len(items); i++ {
		if b != -1 && items[i].Type == BoxType && width < W+w+items[i].Width-Z-z-items[i].Shrink || items[i].Penalty == -Infinity {
			// break
			if items[i].Penalty == -Infinity {
				// move breakpoint to this
				W += w
				Y += y
				Z += z
				w, y, z = 0.0, 0.0, 0.0
				b = i
			}
			if items[b].Type == PenaltyType {
				// add width of used potential hyphen
				W += items[b].Width
			} else if items[b].Type == GlueType {
				// subtract disappeared glue used as break
				W -= items[b].Width
				Y -= items[b].Stretch
				Z -= items[b].Shrink
			}

			ratio := 0.0
			if W-Z <= width && width < W {
				// shrink
				ratio = (width - W) / Z
			} else if W < width && width <= W+Y {
				// stretch
				ratio = (width - W) / Y
			} else if width < W-Z {
				overflows = true
			}
			breaks = append(breaks, Break{
				Position: b,
				Width:    W,
				Stretch:  Y,
				Shrink:   Z,
				Ratio:    ratio,
			})

			W, Y, Z = 0.0, 0.0, 0.0
			if b+1 < len(items) && items[b+1].Type == GlueType {
				// skip glue after break
				W -= items[b+1].Width
				Y -= items[b+1].Stretch
				Z -= items[b+1].Shrink
			}
		}

		if items[i].Type == BoxType {
			w += items[i].Width
		} else if items[i].Type == GlueType {
			w += items[i].Width
			y += items[i].Stretch
			z += items[i].Shrink
		}

		if items[i].Type == PenaltyType && items[i].Penalty != Infinity || 0 < i && items[i].Type == GlueType && items[i-1].Type == BoxType && (i+1 == len(items) || items[i+1].Type != PenaltyType) {
			// possible breakpoint:
			// - penalty that is not infinity
			// - glue after box and not before penalty
			W += w
			Y += y
			Z += z
			w, y, z = 0.0, 0.0, 0.0
			b = i
		}
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

func SpaceGlue(glyphs []Glyph, i int) (float64, float64, float64) {
	spaceFactor := 1.0
	spaceWidth := glyphs[i].Advance()
	if !FrenchSpacing {
		i--
		if 0 <= i && (glyphs[i].Text == ')' || glyphs[i].Text == ']' || glyphs[i].Text == '\'' || glyphs[i].Text == '"') {
			i--
		}

		// TODO: add support for surpressing extra spacing with U+E000 and forcing extra spacing with U+E001
		// don't add extra spacing after uppercase + period
		//if i == 0 || glyphs[i].Text != '.' || 0 < i && !unicode.IsUpper(glyphs[i-1].Text) {
		if 0 <= i {
			switch glyphs[i].Text {
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
	return spaceWidth, spaceWidth * SpaceStretch * spaceFactor, spaceWidth * SpaceShrink / spaceFactor
}

// GlyphsToItems converts a slice of glyphs into the box/glue/penalty items model as used by Knuth's line breaking algorithm. The SFNT and Size of each glyph must be set. Indent and align specify the indentation width of the first line and the alignment (left, right, centered, justified) of the lines respectively.
func GlyphsToItems(glyphs []Glyph, indent float64, align Align) Items {
	// Notes:
	// - Penalties with a cost less than Infinity are potential breakpoints
	// - Glues directly after a box are potential breakpoints (a penalty with cost Infinity would prohibit this)
	// - Only the Justified and Left alignments are really used, Left is used for Right and Centered as well
	if len(glyphs) == 0 {
		return Items{}
	}

	// trim spaces from start and end
	first, last := 0, len(glyphs)

	items := Items{}
	items = items.Box(indent)
	for i := first; i < last; i++ {
		glyph := glyphs[i]
		if IsSpace(glyph.Text) {
			w, y, z := SpaceGlue(glyphs, i)
			if align == Justified {
				items = items.Glue(w, y, z)
				items[len(items)-1].Size++
			} else {
				items = items.Glue(0.0, Infinity, 0.0)
				items = items.Penalty(0.0, 0.0, false) // breakable
				items = items.Glue(w, y-Infinity, z)
				items[len(items)-1].Size++
			}
		} else if IsNewline(glyph.Text) {
			// only add one penalty for \r\n
			items = items.Glue(0.0, Infinity, 0.0)
			items = items.Penalty(0.0, -Infinity, false) // forced breakpoint
			items[len(items)-1].Size++
			if glyph.Text == '\r' && i+1 < len(glyphs) && glyphs[i+1].Text == '\n' {
				items[len(items)-1].Size++
				i++
			}
			items = items.Glue(0.0, -Infinity, 0.0)
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
				items = items.Penalty(hyphenWidth, HyphenPenalty, true) // breakable
				items[len(items)-1].Size++
			} else {
				items = items.Penalty(0.0, Infinity, false)
				items = items.Glue(0.0, Infinity, 0.0)
				items = items.Penalty(hyphenWidth, 10.0*HyphenPenalty, true) // breakable
				items[len(items)-1].Size++
				items = items.Glue(0.0, -Infinity, 0.0)
			}
		} else {
			// glyphs
			width := glyph.Advance()
			if 1 < len(items) && items[len(items)-1].Type == BoxType {
				if IsSpacelessScript(glyph.Script) || 0 < i && IsSpacelessScript(glyphs[i-1].Script) {
					// allow breaks around spaceless script glyphs, most commonly CJK
					items = items.Penalty(0.0, 0.0, false) // breakable
					items = items.Box(width)
				} else {
					// merge with previous box only if it's not indent
					items[len(items)-1].Width += width
				}
			} else {
				items = items.Box(width)
			}
			items[len(items)-1].Size++
		}
		if glyph.Text == '-' {
			// optional break after hyphen
			if align == Justified {
				items = items.Penalty(0.0, HyphenPenalty, true) // breakable
			} else {
				items = items.Penalty(0.0, Infinity, false)
				items = items.Glue(0.0, Infinity, 0.0)
				items = items.Penalty(0.0, HyphenPenalty, true) // breakable
				items = items.Glue(0.0, -Infinity, 0.0)
			}
		}
	}
	items = items.Glue(0.0, Infinity, 0.0)
	items = items.Penalty(0.0, -Infinity, false) // forced breakpoint
	return items
}

// Linebreaker is an interface for line breaking algorithms. Given a set of items and a desired text width, it will break lines to remain within the given width. It returns the breakpoints and whether it succesfully broke all lines to fit within the width; it returns falso if it overflows.
type Linebreaker interface {
	Linebreak([]Item, float64) ([]Break, bool)
}

// LinebreakGlyphs breaks a slice of glyphs uing the given SFNT font and font size. The indent and width specify the first line's indentation and the maximum line's width respectively. Align sets the horizontal alignment of the text. The looseness specifies whether it is desirable to have less or more lines than optimal.
func LinebreakGlyphs(sfnt *font.SFNT, size float64, glyphs []Glyph, indent, width float64, align Align, linebreaker Linebreaker) [][]Glyph {
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
	breaks, _ := linebreaker.Linebreak(items, width)

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
