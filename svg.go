package canvas

import (
	"bytes"
	"fmt"
	"image/color"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
	"github.com/tdewolff/parse/v2/xml"
)

type elem struct {
	tag   string
	id    string
	attrs map[string]string
}

type svgState struct {
	strokeMiterLimit float64
	textX            float64
	textY            float64
	textAnchor       string
	fontFamily       string
	fontSize         float64
}

var svgDefaultState = svgState{
	strokeMiterLimit: 4.0,
	textAnchor:       "start",
	fontFamily:       "serif",
	fontSize:         16.0, // in px
}

type svgParser struct {
	z                       *parse.Input
	c                       *Canvas
	ctx                     *Context
	width, height, diagonal float64
	err                     error

	stateStack []svgState
	state      svgState

	cssRules []cssRule
	fonts    map[string]*FontFamily
}

func (svg *svgParser) init(width, height float64, viewbox [4]float64) {
	svg.c = New(width, height)
	svg.ctx = NewContext(svg.c)
	svg.ctx.SetCoordSystem(CartesianIV)
	if 0.0 < (viewbox[2]-viewbox[0]) && 0.0 < (viewbox[3]-viewbox[1]) {
		m := Identity.Scale(width/(viewbox[2]-viewbox[0]), height/(viewbox[3]-viewbox[1])).Translate(-viewbox[0], -viewbox[1])
		svg.ctx.SetView(m)
		svg.ctx.SetCoordView(m)
	}
	svg.ctx.SetStrokeJoiner(MiterJoiner{BevelJoin, svgDefaultState.strokeMiterLimit})
	svg.state = svgDefaultState
	svg.fonts = map[string]*FontFamily{}
}

func (svg *svgParser) push() {
	svg.ctx.Push()
	svg.stateStack = append(svg.stateStack, svg.state)
}

func (svg *svgParser) pop() {
	svg.state = svg.stateStack[len(svg.stateStack)-1]
	svg.stateStack = svg.stateStack[:len(svg.stateStack)-1]
	svg.ctx.Pop()
}

func (svg *svgParser) parseDimension(v string, parent float64) float64 {
	if len(v) == 0 {
		return 0.0
	}

	nn, _ := parse.Dimension([]byte(v))
	num, err := strconv.ParseFloat(v[:nn], 64)
	if err != nil {
		if svg.err == nil {
			svg.err = parse.NewErrorLexer(svg.z, "bad dimension: %w: %s", err, v)
		}
		return 0.0
	}

	dim := v[nn:]
	switch strings.ToLower(dim) {
	case "cm":
		return num * 10.0 * 96.0 / 25.4
	case "mm":
		return num * 96.0 / 25.4
	case "q":
		return num * 0.25 * 96.0 / 25.4
	case "in":
		return num * 96.0
	case "pc":
		return num * 96.0 / 6.0
	case "pt":
		return num * 96.0 / 72.0
	case "", "px":
		return num
	case "%":
		return num * parent / 100.0
	}
	if svg.err == nil {
		svg.err = parse.NewErrorLexer(svg.z, "unknown dimension: %s", dim)
	}
	return 0.0
}

func (svg *svgParser) parseColorComponent(v string) uint8 {
	v = strings.TrimSpace(v)
	if len(v) == 0 {
		return 0
	} else if v[len(v)-1] == '%' {
		num, err := strconv.ParseFloat(v[:len(v)-1], 64)
		if err != nil && svg.err == nil {
			svg.err = parse.NewErrorLexer(svg.z, "bad color component: %w: %s", err, v)
		}
		return uint8(num*255.0 + 0.5)
	}
	num, err := strconv.ParseUint(v, 10, 8)
	if err != nil && svg.err == nil {
		svg.err = parse.NewErrorLexer(svg.z, "bad color component: %w: %s", err, v)
	}
	return uint8(num)
}

func (svg *svgParser) parsePaint(v string) Paint {
	if len(v) == 0 {
		return Paint{Color: Black}
	} else if v[0] == '#' {
		return Paint{Color: Hex(v)}
	}
	v = strings.ToLower(v)
	if col, ok := cssColors[v]; ok {
		return Paint{Color: col}
	}
	var col color.RGBA
	if strings.HasPrefix(v, "rgb(") && strings.HasSuffix(v, ")") {
		comps := strings.Split(v[4:len(v)-1], ",")
		if len(comps) != 3 {
			if svg.err == nil {
				svg.err = parse.NewErrorLexer(svg.z, "bad rgb function: %s", v)
			}
			return Paint{Color: Black}
		}
		col.R = svg.parseColorComponent(comps[0])
		col.G = svg.parseColorComponent(comps[1])
		col.B = svg.parseColorComponent(comps[2])
		col.A = 255
	} else if strings.HasPrefix(v, "rgba(") && strings.HasSuffix(v, ")") {
		comps := strings.Split(v[5:len(v)-1], ",")
		if len(comps) != 4 {
			if svg.err == nil {
				svg.err = parse.NewErrorLexer(svg.z, "bad rgba function: %s", v)
			}
			return Paint{Color: Black}
		}
		col.A = svg.parseColorComponent(comps[3])
		col.R = uint8(float64(svg.parseColorComponent(comps[0]))*float64(col.A)/255.0 + 0.5)
		col.G = uint8(float64(svg.parseColorComponent(comps[1]))*float64(col.A)/255.0 + 0.5)
		col.B = uint8(float64(svg.parseColorComponent(comps[2]))*float64(col.A)/255.0 + 0.5)
	}
	return Paint{Color: col}
}

func (svg *svgParser) parsePoints(v string) []float64 {
	v = strings.ReplaceAll(v, "\n", ",")
	v = strings.ReplaceAll(v, "\t", ",")
	v = strings.ReplaceAll(v, " ", ",")

	vals := []float64{}
	for _, item := range strings.Split(v, ",") {
		if 0 < len(item) {
			val, err := strconv.ParseFloat(item, 64)
			if err != nil && svg.err == nil {
				svg.err = parse.NewErrorLexer(svg.z, "bad number array: %w: %s", err, v)
			}
			vals = append(vals, val)
		}
	}
	return vals
}

func (svg *svgParser) parseTransform(v string) Matrix {
	i, j := 0, 0
	m := Identity
	var fun string
	for i < len(v) {
		if v[i] == '(' {
			fun = strings.ToLower(strings.TrimSpace(v[j:i]))
			j = i + 1
		} else if v[i] == ')' {
			d := svg.parsePoints(v[j:i])
			switch fun {
			case "matrix":
				if len(d) != 6 {
					svg.err = parse.NewErrorLexer(svg.z, "bad transform matrix")
				} else {
					m = m.Mul(Matrix{{d[0], d[2], d[4]}, {d[1], d[3], d[5]}})
				}
			case "translate":
				if len(d) != 1 && len(d) != 2 {
					svg.err = parse.NewErrorLexer(svg.z, "bad transform translate")
				} else if len(d) == 1 {
					m = m.Translate(d[0], 0.0)
				} else {
					m = m.Translate(d[0], d[1])
				}
			case "scale":
				if len(d) != 1 && len(d) != 2 {
					svg.err = parse.NewErrorLexer(svg.z, "bad transform scale")
				} else if len(d) == 1 {
					m = m.Scale(d[0], d[0])
				} else {
					m = m.Scale(d[0], d[1])
				}
			case "rotate":
				if len(d) != 1 && len(d) != 3 {
					svg.err = parse.NewErrorLexer(svg.z, "bad transform rotate")
				} else if len(d) == 1 {
					m = m.Rotate(d[0])
				} else {
					m = m.RotateAbout(d[0], d[1], d[2])
				}
			case "skewx":
				if len(d) != 1 {
					svg.err = parse.NewErrorLexer(svg.z, "bad transform skewX")
				} else {
					// TODO
				}
			case "skewy":
				if len(d) != 1 {
					svg.err = parse.NewErrorLexer(svg.z, "bad transform skewY")
				} else {
					// TODO
				}
			}
			j = i + 1
		}
		i++
	}
	return m
}

func (svg *svgParser) parseStyle(b []byte) {
	p := css.NewParser(parse.NewInputBytes(b), false)
	for {
		gt, _, _ := p.Next()
		if gt == css.ErrorGrammar {
			break
		} else if gt == css.BeginRulesetGrammar {
			selectors := []cssSelector{cssSelector{}}
			node := cssSelectorNode{op: ' '}
			vals := p.Values()
			for i := 0; i < len(vals); i++ {
				t := vals[i]
				if t.TokenType == css.DelimToken && t.Data[0] == ',' {
					selectors = append(selectors, cssSelector{})
				} else if t.TokenType == css.WhitespaceToken || t.TokenType == css.DelimToken && t.Data[0] == '>' {
					selectors[len(selectors)-1] = append(selectors[len(selectors)-1], node)
					node = cssSelectorNode{op: ' '}
					if t.TokenType == css.DelimToken {
						node.op = '>'
					}
				} else if t.TokenType == css.IdentToken || t.TokenType == css.DelimToken && t.Data[0] == '*' {
					node.typ = string(t.Data)
				} else if t.TokenType == css.DelimToken && (t.Data[0] == '.' || t.Data[0] == '#') && i+1 < len(vals) && vals[i+1].TokenType == css.IdentToken {
					if t.Data[0] == '#' {
						node.attrs = append(node.attrs, cssAttrSelector{op: '=', attr: "id", val: string(vals[i+1].Data)})
					} else {
						node.attrs = append(node.attrs, cssAttrSelector{op: '~', attr: "class", val: string(vals[i+1].Data)})
					}
					i++
				} else if t.TokenType == css.DelimToken && t.Data[0] == '[' && i+2 < len(vals) && vals[i+1].TokenType == css.IdentToken && vals[i+2].TokenType == css.DelimToken {
					if vals[i+2].Data[0] == ']' {
						node.attrs = append(node.attrs, cssAttrSelector{op: 0, attr: string(vals[i+1].Data)})
						i += 2
					} else if i+4 < len(vals) && vals[i+3].TokenType == css.IdentToken && vals[i+4].TokenType == css.DelimToken && vals[i+4].Data[0] == ']' {
						node.attrs = append(node.attrs, cssAttrSelector{op: vals[i+2].Data[0], attr: string(vals[i+1].Data), val: string(vals[i+3].Data)})
						i += 4
					}
				}
			}
			selectors[len(selectors)-1] = append(selectors[len(selectors)-1], node)

			props := []cssProperty{}
			for {
				gt, _, data := p.Next()
				if gt != css.DeclarationGrammar {
					break
				}

				val := strings.Builder{}
				for _, t := range p.Values() {
					val.Write(t.Data)
				}
				props = append(props, cssProperty{string(data), val.String()})
			}
			svg.cssRules = append(svg.cssRules, cssRule{
				selectors: selectors,
				props:     props,
			})
		}
	}
}

func (svg *svgParser) parseStyleAttribute(style string) []cssProperty {
	props := []cssProperty{}
	p := css.NewParser(parse.NewInput(bytes.NewBufferString(style)), true)
	for {
		gt, _, data := p.Next()
		if gt == css.ErrorGrammar {
			break
		} else if gt == css.DeclarationGrammar {
			val := strings.Builder{}
			for _, t := range p.Values() {
				val.Write(t.Data)
			}
			props = append(props, cssProperty{string(data), val.String()})
		}
	}
	return props
}

func (svg *svgParser) setStyling(elems []elem, props []cssProperty) {
	// apply CSS from <style>
	for _, rule := range svg.cssRules {
		// TODO: this is in order of appearance, use selector specificity/precedence?
		if rule.AppliesTo(elems) {
			for _, prop := range rule.props {
				svg.setAttribute(prop.key, prop.val)
			}
		}
	}

	// apply attributes in order
	for _, prop := range props {
		if prop.key == "style" {
			for _, styleProp := range svg.parseStyleAttribute(prop.val) {
				svg.setAttribute(styleProp.key, styleProp.val)
			}
		} else {
			svg.setAttribute(prop.key, prop.val)
		}
	}
}

func (svg *svgParser) setAttribute(key, val string) {
	switch key {
	case "fill":
		svg.ctx.SetFill(svg.parsePaint(val))
	case "stroke":
		svg.ctx.SetStroke(svg.parsePaint(val))
	case "stroke-width":
		svg.ctx.SetStrokeWidth(svg.parseDimension(val, svg.diagonal))
	case "stroke-dashoffset":
		svg.ctx.Style.DashOffset = svg.parseDimension(val, svg.diagonal)
	case "stroke-dasharray":
		if val == "none" {
			svg.ctx.Style.Dashes = []float64{}
		} else {
			svg.ctx.Style.Dashes = svg.parsePoints(val)
		}
	case "stroke-linecap":
		if val == "butt" {
			svg.ctx.SetStrokeCapper(ButtCap)
		} else if val == "round" {
			svg.ctx.SetStrokeCapper(RoundCap)
		} else if val == "square" {
			svg.ctx.SetStrokeCapper(SquareCap)
		}
	case "stroke-linejoin":
		if val == "arcs" {
			svg.ctx.SetStrokeJoiner(ArcsJoin)
		} else if val == "bevel" {
			svg.ctx.SetStrokeJoiner(BevelJoin)
		} else if val == "miter" {
			svg.ctx.SetStrokeJoiner(MiterJoiner{BevelJoin, svg.state.strokeMiterLimit})
		} else if val == "miter-clip" {
			// not exactly correct
			svg.ctx.SetStrokeJoiner(MiterJoiner{BevelJoin, svg.state.strokeMiterLimit})
		} else if val == "round" {
			svg.ctx.SetStrokeJoiner(RoundJoin)
		}
	case "stroke-miterlimit":
		svg.state.strokeMiterLimit = svg.parseDimension(val, svg.diagonal)
		if miter, ok := svg.ctx.StrokeJoiner.(MiterJoiner); ok {
			miter.Limit = svg.state.strokeMiterLimit
		}
	case "transform":
		svg.ctx.ComposeView(svg.parseTransform(val))
	case "text-anchor":
		svg.state.textAnchor = val
	case "font-family":
		svg.state.fontFamily = val
	case "font-size":
		svg.state.fontSize = svg.parseDimension(val, svg.height)
	}
}

func (svg *svgParser) getFontFace() *FontFace {
	fontFamily, ok := svg.fonts[svg.state.fontFamily]
	if !ok {
		fontFamily = NewFontFamily(svg.state.fontFamily)
		fontFamily.LoadLocalFont(svg.state.fontFamily, FontRegular)
		svg.fonts[svg.state.fontFamily] = fontFamily
	}
	fontSize := svg.state.fontSize * 72.0 / 25.4 // pt/mm
	return fontFamily.Face(fontSize)
}

func ParseSVG(r io.Reader) (*Canvas, error) {
	z := parse.NewInput(r)
	defer z.Restore()

	l := xml.NewLexer(z)
	svg := svgParser{
		z: z,
	}
	elemStack := []elem{}
	for {
		tt, data := l.Next()
		switch tt {
		case xml.ErrorToken:
			if l.Err() != io.EOF {
				return svg.c, l.Err()
			} else if svg.err != nil {
				return svg.c, svg.err
			} else if svg.c == nil {
				return svg.c, fmt.Errorf("expected SVG tag")
			}
			if svg.c.W == 0.0 || svg.c.H == 0.0 {
				svg.c.Fit(0.0)
			}
			return svg.c, nil
		case xml.StartTagToken:
			// get all attributes
			attrs := map[string]string{}
			attrNames := []string{}
			for {
				// TODO: attribute errors point to wrong position
				tt, _ = l.Next()
				if tt != xml.AttributeToken {
					break
				}
				val := l.AttrVal()
				val = val[1 : len(val)-1]
				attrNames = append(attrNames, string(l.Text()))
				attrs[string(l.Text())] = string(val)
			}

			// handle SVG tag and create canvas
			tag := string(data[1:])
			if tag == "svg" && svg.c == nil {
				var err error
				var viewbox [4]float64
				var width, height float64
				if _, ok := attrs["viewBox"]; ok {
					vals := strings.Split(attrs["viewBox"], " ")
					if len(vals) != 4 {
						svg.err = parse.NewErrorLexer(svg.z, "bad viewBox")
					} else {
						for i := 0; i < 4; i++ {
							viewbox[i], err = strconv.ParseFloat(vals[i], 64)
							if err != nil && svg.err == nil {
								svg.err = parse.NewErrorLexer(svg.z, "bad viewBox: %w", err)
							}
						}
					}
				}
				if _, ok := attrs["width"]; ok {
					width = svg.parseDimension(attrs["width"], 0.0)
				} else {
					width = (viewbox[2] - viewbox[0]) * 25.4 / 96.0
				}
				if _, ok := attrs["height"]; ok {
					height = svg.parseDimension(attrs["height"], 0.0)
				} else {
					height = (viewbox[3] - viewbox[1]) * 25.4 / 96.0
				}

				svg.width = width * 96.0 / 25.4
				svg.height = height * 96.0 / 25.4
				svg.diagonal = math.Sqrt((svg.width*svg.width + svg.height*svg.height) / 2.0)
				svg.init(width, height, viewbox)
			} else if tag != "svg" && svg.c == nil {
				return svg.c, fmt.Errorf("expected SVG tag")
			}

			// handle style tag
			if tag == "style" {
				tt, data = l.Next()
				if tt == xml.TextToken {
					svg.parseStyle(data)
					tt, data = l.Next() // end token
				} else {
					return svg.c, fmt.Errorf("bad style tag")
				}
				break
			}

			// push new state
			svg.push()
			elemStack = append(elemStack, elem{tag, attrs["id"], attrs})

			// set styling and presentation attributes
			props := []cssProperty{}
			for _, key := range attrNames {
				props = append(props, cssProperty{key, attrs[key]})
			}
			svg.setStyling(elemStack, props)

			switch tag {
			case "circle":
				cx := svg.parseDimension(attrs["cx"], svg.width)
				cy := svg.parseDimension(attrs["cy"], svg.height)
				r := svg.parseDimension(attrs["r"], svg.diagonal)
				svg.ctx.DrawPath(cx, cy, Circle(r))
			case "ellipse":
				cx := svg.parseDimension(attrs["cx"], svg.width)
				cy := svg.parseDimension(attrs["cy"], svg.height)
				rx := svg.parseDimension(attrs["rx"], svg.width)
				ry := svg.parseDimension(attrs["ry"], svg.height)
				svg.ctx.DrawPath(cx, cy, Ellipse(rx, ry))
			case "path":
				p, err := ParseSVGPath(attrs["d"])
				if err != nil && svg.err == nil {
					svg.err = parse.NewErrorLexer(svg.z, "bad path: %w", err)
				}
				svg.ctx.DrawPath(0, 0, p)
			case "polygon", "polyline":
				points := svg.parsePoints(attrs["points"])
				p := &Path{}
				for i := 0; i+1 < len(points); i += 2 {
					if i == 0 {
						p.MoveTo(points[0], points[1])
					} else {
						p.LineTo(points[i], points[i+1])
					}
				}
				if tag == "polygon" {
					p.Close()
				}
				svg.ctx.DrawPath(0.0, 0.0, p)
			case "line":
				p := &Path{}
				x1, _ := strconv.ParseInt(attrs["x1"], 10, 64)
				y1, _ := strconv.ParseInt(attrs["y1"], 10, 64)
				x2, _ := strconv.ParseInt(attrs["x2"], 10, 64)
				y2, _ := strconv.ParseInt(attrs["y2"], 10, 64)

				p.MoveTo(float64(x1), float64(y1))
				p.LineTo(float64(x2), float64(y2))
				svg.ctx.DrawPath(0.0, 0.0, p)
			case "rect":
				x := svg.parseDimension(attrs["x"], svg.width)
				y := svg.parseDimension(attrs["y"], svg.height)
				width := svg.parseDimension(attrs["width"], svg.width)
				height := svg.parseDimension(attrs["height"], svg.height)
				svg.ctx.DrawPath(x, y, Rectangle(width, height))
			case "text":
				svg.state.textX = svg.parseDimension(attrs["x"], svg.width)
				svg.state.textY = svg.parseDimension(attrs["y"], svg.height)
			}

			// tag is self-closing
			if tt == xml.StartTagCloseVoidToken {
				elemStack = elemStack[:len(elemStack)-1]
				svg.pop()
			}
		case xml.TextToken:
			if 0 < len(elemStack) {
				tag := elemStack[len(elemStack)-1].tag
				if tag == "text" {
					textAlign := Left
					if svg.state.textAnchor == "middle" {
						textAlign = Center
					} else if svg.state.textAnchor == "end" {
						textAlign = Right
					}
					text := NewTextLine(svg.getFontFace(), string(data), textAlign)
					svg.ctx.DrawText(svg.state.textX, svg.state.textY, text)
				}
			}
		case xml.EndTagToken:
			if len(elemStack) != 0 {
				elemStack = elemStack[:len(elemStack)-1]
			}
			svg.pop()
		}
	}
}

type cssAttrSelector struct {
	op   byte // empty, =, ~, |
	attr string
	val  string
}

func (sel cssAttrSelector) AppliesTo(elem elem) bool {
	switch sel.op {
	case 0:
		_, ok := elem.attrs[sel.attr]
		return ok
	case '=':
		return elem.attrs[sel.attr] == sel.val
	case '~':
		vals := strings.Split(elem.attrs[sel.attr], " ")
		for _, val := range vals {
			if val != "" && val == sel.val {
				return true
			}
		}
		return false
	case '|':
		return elem.attrs[sel.attr] == sel.val || strings.HasPrefix(elem.attrs[sel.attr], sel.val+"-")
	}
	return false
}

type cssSelectorNode struct {
	op    byte   // space or >, first is always space
	typ   string // is * for universal
	attrs []cssAttrSelector
}

func (sel cssSelectorNode) AppliesTo(elem elem) bool {
	if sel.typ != "*" && sel.typ != "" && sel.typ != elem.tag {
		return false
	}
	for _, attr := range sel.attrs {
		if !attr.AppliesTo(elem) {
			return false
		}
	}
	return true
}

type cssSelector []cssSelectorNode

func (sels cssSelector) AppliesTo(elems []elem) bool {
	ielem := 0
Retry:
	isel := 0
	ielemNext := len(elems)
	for isel < len(sels) && ielem < len(elems) {
		switch sels[isel].op {
		case ' ':
			for {
				if ielem == len(elems) {
					ielem = ielemNext
					goto Retry
				} else if sels[isel].AppliesTo(elems[ielem]) {
					ielem++
					break
				}
				ielem++
			}
			if ielemNext == len(elems) {
				ielemNext = ielem
			}
		case '>':
			if !sels[isel].AppliesTo(elems[ielem]) {
				ielem = ielemNext
				goto Retry
			}
			ielem++
		default:
			return false
		}
		isel++
	}
	return len(sels) != 0 && isel == len(sels)
}

type cssProperty struct {
	key, val string
}

type cssRule struct {
	selectors []cssSelector
	props     []cssProperty
}

func (rule cssRule) AppliesTo(elems []elem) bool {
	for _, sels := range rule.selectors {
		if sels.AppliesTo(elems) {
			return true
		}
	}
	return false
}

var cssColors = map[string]color.RGBA{
	"aliceblue":            color.RGBA{240, 248, 255, 255},
	"antiquewhite":         color.RGBA{250, 235, 215, 255},
	"aqua":                 color.RGBA{0, 255, 255, 255},
	"aquamarine":           color.RGBA{127, 255, 212, 255},
	"azure":                color.RGBA{240, 255, 255, 255},
	"beige":                color.RGBA{245, 245, 220, 255},
	"bisque":               color.RGBA{255, 228, 196, 255},
	"black":                color.RGBA{0, 0, 0, 255},
	"blanchedalmond":       color.RGBA{255, 235, 205, 255},
	"blue":                 color.RGBA{0, 0, 255, 255},
	"blueviolet":           color.RGBA{138, 43, 226, 255},
	"brown":                color.RGBA{165, 42, 42, 255},
	"burlywood":            color.RGBA{222, 184, 135, 255},
	"cadetblue":            color.RGBA{95, 158, 160, 255},
	"chartreuse":           color.RGBA{127, 255, 0, 255},
	"chocolate":            color.RGBA{210, 105, 30, 255},
	"coral":                color.RGBA{255, 127, 80, 255},
	"cornflowerblue":       color.RGBA{100, 149, 237, 255},
	"cornsilk":             color.RGBA{255, 248, 220, 255},
	"crimson":              color.RGBA{220, 20, 60, 255},
	"cyan":                 color.RGBA{0, 255, 255, 255},
	"darkblue":             color.RGBA{0, 0, 139, 255},
	"darkcyan":             color.RGBA{0, 139, 139, 255},
	"darkgoldenrod":        color.RGBA{184, 134, 11, 255},
	"darkgray":             color.RGBA{169, 169, 169, 255},
	"darkgreen":            color.RGBA{0, 100, 0, 255},
	"darkgrey":             color.RGBA{169, 169, 169, 255},
	"darkkhaki":            color.RGBA{189, 183, 107, 255},
	"darkmagenta":          color.RGBA{139, 0, 139, 255},
	"darkolivegreen":       color.RGBA{85, 107, 47, 255},
	"darkorange":           color.RGBA{255, 140, 0, 255},
	"darkorchid":           color.RGBA{153, 50, 204, 255},
	"darkred":              color.RGBA{139, 0, 0, 255},
	"darksalmon":           color.RGBA{233, 150, 122, 255},
	"darkseagreen":         color.RGBA{143, 188, 143, 255},
	"darkslateblue":        color.RGBA{72, 61, 139, 255},
	"darkslategray":        color.RGBA{47, 79, 79, 255},
	"darkslategrey":        color.RGBA{47, 79, 79, 255},
	"darkturquoise":        color.RGBA{0, 206, 209, 255},
	"darkviolet":           color.RGBA{148, 0, 211, 255},
	"deeppink":             color.RGBA{255, 20, 147, 255},
	"deepskyblue":          color.RGBA{0, 191, 255, 255},
	"dimgray":              color.RGBA{105, 105, 105, 255},
	"dimgrey":              color.RGBA{105, 105, 105, 255},
	"dodgerblue":           color.RGBA{30, 144, 255, 255},
	"firebrick":            color.RGBA{178, 34, 34, 255},
	"floralwhite":          color.RGBA{255, 250, 240, 255},
	"forestgreen":          color.RGBA{34, 139, 34, 255},
	"fuchsia":              color.RGBA{255, 0, 255, 255},
	"gainsboro":            color.RGBA{220, 220, 220, 255},
	"ghostwhite":           color.RGBA{248, 248, 255, 255},
	"gold":                 color.RGBA{255, 215, 0, 255},
	"goldenrod":            color.RGBA{218, 165, 32, 255},
	"gray":                 color.RGBA{128, 128, 128, 255},
	"green":                color.RGBA{0, 128, 0, 255},
	"greenyellow":          color.RGBA{173, 255, 47, 255},
	"grey":                 color.RGBA{128, 128, 128, 255},
	"honeydew":             color.RGBA{240, 255, 240, 255},
	"hotpink":              color.RGBA{255, 105, 180, 255},
	"indianred":            color.RGBA{205, 92, 92, 255},
	"indigo":               color.RGBA{75, 0, 130, 255},
	"ivory":                color.RGBA{255, 255, 240, 255},
	"khaki":                color.RGBA{240, 230, 140, 255},
	"lavender":             color.RGBA{230, 230, 250, 255},
	"lavenderblush":        color.RGBA{255, 240, 245, 255},
	"lawngreen":            color.RGBA{124, 252, 0, 255},
	"lemonchiffon":         color.RGBA{255, 250, 205, 255},
	"lightblue":            color.RGBA{173, 216, 230, 255},
	"lightcoral":           color.RGBA{240, 128, 128, 255},
	"lightcyan":            color.RGBA{224, 255, 255, 255},
	"lightgoldenrodyellow": color.RGBA{250, 250, 210, 255},
	"lightgray":            color.RGBA{211, 211, 211, 255},
	"lightgreen":           color.RGBA{144, 238, 144, 255},
	"lightgrey":            color.RGBA{211, 211, 211, 255},
	"lightpink":            color.RGBA{255, 182, 193, 255},
	"lightsalmon":          color.RGBA{255, 160, 122, 255},
	"lightseagreen":        color.RGBA{32, 178, 170, 255},
	"lightskyblue":         color.RGBA{135, 206, 250, 255},
	"lightslategray":       color.RGBA{119, 136, 153, 255},
	"lightslategrey":       color.RGBA{119, 136, 153, 255},
	"lightsteelblue":       color.RGBA{176, 196, 222, 255},
	"lightyellow":          color.RGBA{255, 255, 224, 255},
	"lime":                 color.RGBA{0, 255, 0, 255},
	"limegreen":            color.RGBA{50, 205, 50, 255},
	"linen":                color.RGBA{250, 240, 230, 255},
	"magenta":              color.RGBA{255, 0, 255, 255},
	"maroon":               color.RGBA{128, 0, 0, 255},
	"mediumaquamarine":     color.RGBA{102, 205, 170, 255},
	"mediumblue":           color.RGBA{0, 0, 205, 255},
	"mediumorchid":         color.RGBA{186, 85, 211, 255},
	"mediumpurple":         color.RGBA{147, 112, 219, 255},
	"mediumseagreen":       color.RGBA{60, 179, 113, 255},
	"mediumslateblue":      color.RGBA{123, 104, 238, 255},
	"mediumspringgreen":    color.RGBA{0, 250, 154, 255},
	"mediumturquoise":      color.RGBA{72, 209, 204, 255},
	"mediumvioletred":      color.RGBA{199, 21, 133, 255},
	"midnightblue":         color.RGBA{25, 25, 112, 255},
	"mintcream":            color.RGBA{245, 255, 250, 255},
	"mistyrose":            color.RGBA{255, 228, 225, 255},
	"moccasin":             color.RGBA{255, 228, 181, 255},
	"navajowhite":          color.RGBA{255, 222, 173, 255},
	"navy":                 color.RGBA{0, 0, 128, 255},
	"oldlace":              color.RGBA{253, 245, 230, 255},
	"olive":                color.RGBA{128, 128, 0, 255},
	"olivedrab":            color.RGBA{107, 142, 35, 255},
	"orange":               color.RGBA{255, 165, 0, 255},
	"orangered":            color.RGBA{255, 69, 0, 255},
	"orchid":               color.RGBA{218, 112, 214, 255},
	"palegoldenrod":        color.RGBA{238, 232, 170, 255},
	"palegreen":            color.RGBA{152, 251, 152, 255},
	"paleturquoise":        color.RGBA{175, 238, 238, 255},
	"palevioletred":        color.RGBA{219, 112, 147, 255},
	"papayawhip":           color.RGBA{255, 239, 213, 255},
	"peachpuff":            color.RGBA{255, 218, 185, 255},
	"peru":                 color.RGBA{205, 133, 63, 255},
	"pink":                 color.RGBA{255, 192, 203, 255},
	"plum":                 color.RGBA{221, 160, 221, 255},
	"powderblue":           color.RGBA{176, 224, 230, 255},
	"purple":               color.RGBA{128, 0, 128, 255},
	"red":                  color.RGBA{255, 0, 0, 255},
	"rosybrown":            color.RGBA{188, 143, 143, 255},
	"royalblue":            color.RGBA{65, 105, 225, 255},
	"saddlebrown":          color.RGBA{139, 69, 19, 255},
	"salmon":               color.RGBA{250, 128, 114, 255},
	"sandybrown":           color.RGBA{244, 164, 96, 255},
	"seagreen":             color.RGBA{46, 139, 87, 255},
	"seashell":             color.RGBA{255, 245, 238, 255},
	"sienna":               color.RGBA{160, 82, 45, 255},
	"silver":               color.RGBA{192, 192, 192, 255},
	"skyblue":              color.RGBA{135, 206, 235, 255},
	"slateblue":            color.RGBA{106, 90, 205, 255},
	"slategray":            color.RGBA{112, 128, 144, 255},
	"slategrey":            color.RGBA{112, 128, 144, 255},
	"snow":                 color.RGBA{255, 250, 250, 255},
	"springgreen":          color.RGBA{0, 255, 127, 255},
	"steelblue":            color.RGBA{70, 130, 180, 255},
	"tan":                  color.RGBA{210, 180, 140, 255},
	"teal":                 color.RGBA{0, 128, 128, 255},
	"thistle":              color.RGBA{216, 191, 216, 255},
	"tomato":               color.RGBA{255, 99, 71, 255},
	"turquoise":            color.RGBA{64, 224, 208, 255},
	"violet":               color.RGBA{238, 130, 238, 255},
	"wheat":                color.RGBA{245, 222, 179, 255},
	"white":                color.RGBA{255, 255, 255, 255},
	"whitesmoke":           color.RGBA{245, 245, 245, 255},
	"yellow":               color.RGBA{255, 255, 0, 255},
	"yellowgreen":          color.RGBA{154, 205, 50, 255},
}
