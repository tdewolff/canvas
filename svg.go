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

	"golang.org/x/net/html"
)

type svgDef func(string, *Canvas)

type svgElem struct {
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

type svgCanvas struct {
	c                       *Canvas
	ctx                     *Context
	width, height, diagonal float64
}

type svgParser struct {
	z   *parse.Input
	err error
	svgCanvas

	elemStack  []svgElem
	stateStack []svgState
	state      svgState

	cssRules []cssRule // from <style>
	defs     map[string]svgDef
	fonts    map[string]*FontFamily

	// active definitions for attributes
	activeDefs map[string]svgDef
}

func (svg *svgParser) parseViewBox(attrWidth, attrHeight, attrViewBox string) (float64, float64, [4]float64) {
	var err error
	var viewbox [4]float64
	var width, height float64
	if attrViewBox != "" {
		vals := strings.Split(attrViewBox, " ")
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
	if attrWidth != "" && !strings.HasSuffix(attrWidth, "%") {
		width = svg.parseDimension(attrWidth, 1.0)
	} else {
		width = (viewbox[2] - viewbox[0]) * 25.4 / 96.0
	}
	if attrHeight != "" && !strings.HasSuffix(attrHeight, "%") {
		height = svg.parseDimension(attrHeight, 1.0)
	} else {
		height = (viewbox[3] - viewbox[1]) * 25.4 / 96.0
	}
	return width, height, viewbox
}

func (svg *svgParser) init(width, height float64, viewbox [4]float64) {
	svg.width, svg.height = width*96.0/25.4, height*96.0/25.4
	svg.diagonal = math.Sqrt((svg.width*svg.width + svg.height*svg.height) / 2.0)

	svg.c = New(width, height)
	svg.ctx = NewContext(svg.c)
	svg.ctx.SetCoordSystem(CartesianIV)
	if 0.0 < (viewbox[2]-viewbox[0]) && 0.0 < (viewbox[3]-viewbox[1]) {
		m := Identity.Scale(width/(viewbox[2]-viewbox[0]), height/(viewbox[3]-viewbox[1])).Translate(-viewbox[0], -viewbox[1])
		svg.ctx.SetView(m)
	}
	svg.ctx.SetStrokeJoiner(MiterJoiner{BevelJoin, svgDefaultState.strokeMiterLimit})
	svg.state = svgDefaultState
}

func (svg *svgParser) push(tag string, attrs map[string]string) {
	svg.ctx.Push()
	svg.stateStack = append(svg.stateStack, svg.state)
	svg.elemStack = append(svg.elemStack, svgElem{tag, attrs["id"], attrs})
}

func (svg *svgParser) pop() {
	if len(svg.stateStack) == 0 {
		svg.err = parse.NewErrorLexer(svg.z, "invalid SVG")
		return
	}
	svg.elemStack = svg.elemStack[:len(svg.elemStack)-1]
	svg.state = svg.stateStack[len(svg.stateStack)-1]
	svg.stateStack = svg.stateStack[:len(svg.stateStack)-1]
	svg.ctx.Pop()
}

func (svg *svgParser) parseNumber(v string) float64 {
	if len(v) == 0 {
		return 0.0
	}
	percentage := v[len(v)-1] == '%'
	if percentage {
		v = v[:len(v)-1]
	}
	num, err := strconv.ParseFloat(v, 64)
	if err != nil {
		if svg.err == nil {
			svg.err = parse.NewErrorLexer(svg.z, "bad number: %w: %s", err, v)
		}
		return 0.0
	}
	if percentage {
		num /= 100.0
	}
	return num
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
	// lengths
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

	// angles
	case "deg":
		return num
	case "grad":
		return num / 400.0 * 360.0
	case "rad":
		return num / math.Pi * 180.0
	case "turn":
		return num * 360.0

	// other
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
	if v == "none" {
		return Paint{Color: Transparent}
	}
	return Paint{Color: svg.parseColor(v)}
}

func (svg *svgParser) parseColor(v string) color.RGBA {
	if len(v) == 0 {
		return Black
	} else if v[0] == '#' {
		return Hex(v)
	}
	v = strings.ToLower(v)
	if col, ok := cssColors[v]; ok {
		return col
	}
	var col color.RGBA
	if strings.HasPrefix(v, "rgb(") && strings.HasSuffix(v, ")") {
		comps := strings.Split(v[4:len(v)-1], ",")
		if len(comps) != 3 {
			if svg.err == nil {
				svg.err = parse.NewErrorLexer(svg.z, "bad rgb function: %s", v)
			}
			return Black
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
			return Black
		}
		col.A = svg.parseColorComponent(comps[3])
		col.R = uint8(float64(svg.parseColorComponent(comps[0]))*float64(col.A)/255.0 + 0.5)
		col.G = uint8(float64(svg.parseColorComponent(comps[1]))*float64(col.A)/255.0 + 0.5)
		col.B = uint8(float64(svg.parseColorComponent(comps[2]))*float64(col.A)/255.0 + 0.5)
	} else {
		col = Black
	}
	return col
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

func (svg *svgParser) parseAttributes(l *xml.Lexer) (xml.TokenType, []string, map[string]string) {
	// get all attributes
	var tt xml.TokenType
	attrs := map[string]string{}
	attrNames := []string{}
	for {
		// TODO: attribute errors point to wrong position
		tt, _ = l.Next()
		if tt != xml.AttributeToken {
			break
		}
		val := l.AttrVal()
		if len(val) < 2 {
			break
		}
		val = val[1 : len(val)-1]
		attrNames = append(attrNames, string(l.Text()))
		attrs[string(l.Text())] = string(val)
	}
	return tt, attrNames, attrs
}

type svgTag struct {
	parent    *svgTag
	name      string
	attrNames []string
	attrs     map[string]string
	content   []*svgTag
}

func (svg *svgParser) parseTag(l *xml.Lexer) *svgTag {
	var root, parent *svgTag
	for {
		tt, data := l.Next()
		if tt == xml.ErrorToken {
			if l.Err() != io.EOF {
				svg.err = l.Err()
			} else {
				svg.err = parse.NewErrorLexer(svg.z, "unexpected end-of-file")
			}
			break
		} else if tt == xml.StartTagToken {
			var attrNames []string
			var attrs map[string]string
			name := string(data[1:])
			tt, attrNames, attrs = svg.parseAttributes(l)
			tag := &svgTag{
				parent:    parent,
				name:      name,
				attrNames: attrNames,
				attrs:     attrs,
			}

			if parent == nil {
				root = tag
			} else {
				parent.content = append(parent.content, tag)
			}

			if tt == xml.StartTagCloseVoidToken {
				if parent == nil {
					break
				}
			} else {
				parent = tag
			}

			// Handle <style> being nested in <defs>. Adobe Illustrator
			// does this, for example.
			if name == "style" {
				tt, data = l.Next()
				if tt == xml.TextToken {
					svg.parseStyle(data)
					tt, data = l.Next()
				} else {
					svg.err = parse.NewErrorLexer(svg.z, "Bad style tag")
				}
				break
			}
		} else if tt == xml.EndTagToken {
			if parent == nil {
				break // when starting on an end tag
			}
			parent = parent.parent
			if parent == nil {
				break
			}
		}
	}
	return root
}

func (svg *svgParser) parseDefs(l *xml.Lexer) {
	for {
		tag := svg.parseTag(l)
		if tag == nil {
			break
		}
		id := tag.attrs["id"]
		if id == "" {
			continue
		}
		switch tag.name {
		case "linearGradient":
			if _, ok := tag.attrs["x2"]; !ok {
				tag.attrs["x2"] = "100%"
			}
			x1p := strings.HasSuffix(tag.attrs["x1"], "%")
			y1p := strings.HasSuffix(tag.attrs["y1"], "%")
			x2p := strings.HasSuffix(tag.attrs["x2"], "%")
			y2p := strings.HasSuffix(tag.attrs["y2"], "%")
			x1 := svg.parseDimension(tag.attrs["x1"], 1.0)
			x2 := svg.parseDimension(tag.attrs["x2"], 1.0)
			y1 := svg.parseDimension(tag.attrs["y1"], 1.0)
			y2 := svg.parseDimension(tag.attrs["y2"], 1.0)

			stops := Stops{}
			for _, tag := range tag.content {
				if tag.name != "stop" {
					continue
				}

				offset := svg.parseNumber(tag.attrs["offset"])
				stopColor := svg.parseColor(tag.attrs["stop-color"])
				if v, ok := tag.attrs["stop-opacity"]; ok {
					stopOpacity := svg.parseNumber(v)
					stopColor.R = uint8(float64(stopColor.R) / float64(stopColor.A) * stopOpacity * 255.0)
					stopColor.G = uint8(float64(stopColor.G) / float64(stopColor.A) * stopOpacity * 255.0)
					stopColor.B = uint8(float64(stopColor.B) / float64(stopColor.A) * stopOpacity * 255.0)
					stopColor.A = uint8(stopOpacity * 255.0)
				}
				stops.Add(offset, stopColor)
			}
			svg.defs[id] = func(attr string, c *Canvas) {
				layers := c.layers[c.zindex]
				if len(layers) == 0 || layers[len(layers)-1].path == nil {
					return
				}
				layer := &layers[len(layers)-1]

				rect := layer.path.FastBounds()
				x1t, y1t, x2t, y2t := x1, y1, x2, y2
				if x1p {
					x1t = (rect.X0 + rect.W()*x1t) * 25.4 / 96.0
				}
				if y1p {
					y1t = (rect.Y0 + rect.H()*y1t) * 25.4 / 96.0
				}
				if x2p {
					x2t = (rect.X0 + rect.W()*x2t) * 25.4 / 96.0
				}
				if y2p {
					y2t = (rect.Y0 + rect.H()*y2t) * 25.4 / 96.0
				}
				linearGradient := NewLinearGradient(Point{x1t, y1t}, Point{x2t, y2t})
				linearGradient.Stops = stops

				if attr == "fill" {
					layer.style.Fill = Paint{Gradient: linearGradient}
				} else if attr == "stroke" {
					layer.style.Stroke = Paint{Gradient: linearGradient}
				}
			}
		case "marker":
			width, height, viewbox := svg.parseViewBox(tag.attrs["markerWidth"], tag.attrs["markerHeight"], tag.attrs["viewBox"])
			if width == 0.0 {
				width = 3.0
			}
			if height == 0.0 {
				height = 3.0
			}

			units := tag.attrs["markerUnits"]
			if units != "userSpaceOnUse" {
				units = "strokeWidth"
			}

			var refx, refy float64
			switch tag.attrs["refX"] {
			case "left":
				refx = 0.0
			case "center":
				refx = width / 2.0
			case "right":
				refx = width
			default:
				refx = svg.parseDimension(tag.attrs["refX"], 0.0)
			}
			switch tag.attrs["refY"] {
			case "top":
				refy = 0.0
			case "center":
				refy = height / 2.0
			case "bottom":
				refy = height
			default:
				refy = svg.parseDimension(tag.attrs["refY"], 0.0)
			}

			angle := 0.0
			orient := tag.attrs["orient"]
			if orient != "auto" && orient != "auto-start-reverse" {
				angle = svg.parseDimension(tag.attrs["orient"], 0.0)
			}

			origSVGCanvas := svg.svgCanvas
			svg.push(tag.name, tag.attrs)
			svg.init(width, height, [4]float64{})
			for _, tag := range tag.content {
				svg.push(tag.name, tag.attrs)

				props := []cssProperty{}
				for _, key := range tag.attrNames {
					props = append(props, cssProperty{key, tag.attrs[key]})
				}
				svg.setStyling(props)

				svg.drawShape(tag.name, tag.attrs)

				svg.pop()
			}
			marker := svg.c
			svg.svgCanvas = origSVGCanvas
			svg.pop()

			svg.defs[id] = func(attr string, c *Canvas) {
				layers := c.layers[c.zindex]
				if len(layers) == 0 || layers[len(layers)-1].path == nil {
					return
				}
				layer := layers[len(layers)-1]
				path := layer.path
				strokeWidth := layer.style.StrokeWidth

				a := angle
				coordPos := path.Coords()
				coordDir := path.CoordDirections()
				for i := range coordPos {
					if attr == "marker-start" && i == 0 || attr == "marker-end" && i == len(coordPos)-1 || attr == "marker-mid" && i != 0 && i != len(coordPos)-1 {
						pos, dir := coordPos[i], coordDir[i]
						if orient == "auto" || orient == "auto-start-reverse" {
							a = dir.Angle()
							if orient == "auto-start-reverse" {
								a += 180.0
							}
						}

						view := Identity.ReflectYAbout(c.H/2.0).Mul(svg.ctx.View()).Translate(pos.X, pos.Y).Rotate(a * 180.0 / math.Pi)
						if units == "strokeWidth" {
							view = view.Scale(strokeWidth, strokeWidth)
						}

						f := height / (viewbox[3] - viewbox[1])
						view = view.Translate(-refx*f, -refy*f).Scale(f, f).ReflectYAbout(height / 2.0)
						marker.RenderViewTo(c, view)
					}
				}
			}
		}
	}
}

func (svg *svgParser) parseStyle(b []byte) {
	p := css.NewParser(parse.NewInputBytes(b), false)
	selectors := []cssSelector{}
	for {
		gt, _, _ := p.Next()
		if gt == css.ErrorGrammar {
			break
		} else if gt == css.BeginRulesetGrammar || gt == css.QualifiedRuleGrammar {
			selector := cssSelector{}
			node := cssSelectorNode{op: ' '}
			vals := p.Values()
			for i := 0; i < len(vals); i++ {
				t := vals[i]
				if t.TokenType == css.WhitespaceToken || t.TokenType == css.DelimToken && t.Data[0] == '>' {
					selector = append(selector, node)
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
			selector = append(selector, node)
			selectors = append(selectors, selector)
		}

		if gt == css.BeginRulesetGrammar {
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
			selectors = selectors[:0:0]
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

func (svg *svgParser) setStyling(props []cssProperty) {
	// apply CSS from <style>
	for _, rule := range svg.cssRules {
		// TODO: this is in order of appearance, use selector specificity/precedence?
		if rule.AppliesTo(svg.elemStack) {
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

func (svg *svgParser) parseUrlID(val string) string {
	if strings.HasPrefix(val, "url(") && strings.HasSuffix(val, ")") {
		if 6 < len(val) && (val[4] == '#' || val[5] == '#') {
			if val[4] == '#' {
				return val[5 : len(val)-1]
			} else {
				return val[6 : len(val)-2]
			}
		}
	}
	return ""
}

func (svg *svgParser) setAttribute(key, val string) {
	switch key {
	case "fill":
		if id := svg.parseUrlID(val); id != "" {
			svg.activeDefs["fill"] = svg.defs[id]
		} else {
			svg.ctx.SetFill(svg.parsePaint(val))
		}
	case "stroke":
		if id := svg.parseUrlID(val); id != "" {
			svg.activeDefs["stroke"] = svg.defs[id]
		} else {
			svg.ctx.SetStroke(svg.parsePaint(val))
		}
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
			svg.ctx.SetStrokeJoiner(MiterJoiner{nil, svg.state.strokeMiterLimit})
		} else if val == "round" {
			svg.ctx.SetStrokeJoiner(RoundJoin)
		}
	case "stroke-miterlimit":
		svg.state.strokeMiterLimit = svg.parseDimension(val, svg.diagonal)
		if miter, ok := svg.ctx.StrokeJoiner.(MiterJoiner); ok {
			miter.Limit = svg.state.strokeMiterLimit
		}
	case "transform":
		m := svg.parseTransform(val)
		svg.ctx.ComposeView(m)
	case "text-anchor":
		svg.state.textAnchor = val
	case "font-family":
		svg.state.fontFamily = val
	case "font-size":
		svg.state.fontSize = svg.parseDimension(val, svg.height)
	case "marker-start":
		if id := svg.parseUrlID(val); id != "" {
			svg.activeDefs["marker-start"] = svg.defs[id]
		}
	case "marker-mid":
		if id := svg.parseUrlID(val); id != "" {
			svg.activeDefs["marker-mid"] = svg.defs[id]
		}
	case "marker-end":
		if id := svg.parseUrlID(val); id != "" {
			svg.activeDefs["marker-end"] = svg.defs[id]
		}
	}
}

func (svg *svgParser) getFontFace() *FontFace {
	fontFamily, ok := svg.fonts[svg.state.fontFamily]
	if !ok {
		fontFamily = NewFontFamily(svg.state.fontFamily)
		if err := fontFamily.LoadSystemFont(svg.state.fontFamily, FontRegular); err != nil {
			panic(err)
		}
		svg.fonts[svg.state.fontFamily] = fontFamily
	}
	fontSize := svg.state.fontSize * ptPerMm
	return fontFamily.Face(fontSize, svg.ctx.Style.Fill.Color)
}

func (svg *svgParser) toPath(tag string, attrs map[string]string) (x float64, y float64, path *Path) {
	switch tag {
	case "circle":
		x = svg.parseDimension(attrs["cx"], svg.width)
		y = svg.parseDimension(attrs["cy"], svg.height)
		path = Circle(
			svg.parseDimension(attrs["r"], svg.diagonal),
		)
	case "ellipse":
		x = svg.parseDimension(attrs["cx"], svg.width)
		y = svg.parseDimension(attrs["cy"], svg.height)
		path = Ellipse(
			svg.parseDimension(attrs["rx"], svg.width),
			svg.parseDimension(attrs["ry"], svg.height),
		)
	case "path":
		var err error
		path, err = ParseSVGPath(attrs["d"])
		if err != nil && svg.err == nil {
			svg.err = parse.NewErrorLexer(svg.z, "bad path: %w", err)
		}
	case "polygon", "polyline":
		path = &Path{}
		points := svg.parsePoints(attrs["points"])
		for i := 0; i+1 < len(points); i += 2 {
			if i == 0 {
				path.MoveTo(points[0], points[1])
			} else {
				path.LineTo(points[i], points[i+1])
			}
		}
		if tag == "polygon" {
			path.Close()
		}
	case "line":
		x1 := svg.parseDimension(attrs["x1"], svg.width)
		y1 := svg.parseDimension(attrs["y1"], svg.height)
		x2 := svg.parseDimension(attrs["x2"], svg.width)
		y2 := svg.parseDimension(attrs["y2"], svg.height)
		path = &Path{}
		path.MoveTo(x1, y1)
		path.LineTo(x2, y2)
	case "rect":
		x = svg.parseDimension(attrs["x"], svg.width)
		y = svg.parseDimension(attrs["y"], svg.height)
		width := svg.parseDimension(attrs["width"], svg.width)
		height := svg.parseDimension(attrs["height"], svg.height)
		path = &Path{}
		if attrs["rx"] == "" && attrs["ry"] == "" {
			path = Rectangle(width, height)
		} else {
			// TODO: handle both rx and ry
			var r float64
			if attrs["ry"] == "" {
				r = svg.parseDimension(attrs["rx"], svg.width)
			} else {
				r = svg.parseDimension(attrs["ry"], svg.height)
			}
			path = RoundedRectangle(width, height, r)
		}
	case "text":
		svg.state.textX = svg.parseDimension(attrs["x"], svg.width)
		svg.state.textY = svg.parseDimension(attrs["y"], svg.height)
	}

	return
}

func (svg *svgParser) drawShape(tag string, attrs map[string]string) {
	svg.ctx.DrawPath(svg.toPath(tag, attrs))
}

type SVGPath struct {
	Tag string
	Attrs map[string]string
	X, Y float64
	*Path
}

func ParseSVG(r io.Reader) (*Canvas, error) {
	cvs, _, err := parseSVGFull(r)
	return cvs, err
}

func ParseSVGWithPaths(r io.Reader) (*Canvas, []SVGPath, error) {
	return parseSVGFull(r)
}

func parseSVGFull(r io.Reader) (*Canvas, []SVGPath, error) {
	z := parse.NewInput(r)
	defer z.Restore()

	l := xml.NewLexer(z)
	var paths []SVGPath
	svg := svgParser{
		z:          z,
		defs:       map[string]svgDef{},
		fonts:      map[string]*FontFamily{},
		activeDefs: map[string]svgDef{},
	}
	for {
		tt, data := l.Next()
		switch tt {
		case xml.ErrorToken:
			if l.Err() != io.EOF {
				return svg.c, paths, l.Err()
			} else if svg.err != nil {
				return svg.c, paths, svg.err
			} else if svg.c == nil {
				return svg.c, paths, fmt.Errorf("expected SVG tag")
			}
			if svg.c.W == 0.0 || svg.c.H == 0.0 {
				svg.c.Fit(0.0)
			}
			return svg.c, paths, nil
		case xml.StartTagToken:
			tag := string(data[1:])
			tt, attrNames, attrs := svg.parseAttributes(l)

			// handle SVG tag and create canvas
			if tag == "svg" && svg.c == nil {
				width, height, viewbox := svg.parseViewBox(attrs["width"], attrs["height"], attrs["viewBox"])
				svg.init(width, height, viewbox)
			} else if tag != "svg" && svg.c == nil {
				return svg.c, paths, fmt.Errorf("expected SVG tag")
			}

			// handle special tags
			if tag == "style" {
				tt, data = l.Next()
				if tt == xml.TextToken {
					svg.parseStyle(data)
					tt, data = l.Next() // end token
				} else {
					return svg.c, paths, fmt.Errorf("bad style tag")
				}
				break
			} else if tag == "defs" {
				if tt != xml.StartTagCloseVoidToken {
					svg.parseDefs(l)
				}
				break
			}

			// push new state
			svg.push(tag, attrs)

			// set styling and presentation attributes
			props := []cssProperty{}
			for _, key := range attrNames {
				props = append(props, cssProperty{key, attrs[key]})
			}
			svg.setStyling(props)

			pathX, pathY, path := svg.toPath(tag, attrs)
			if path != nil {
				// draw shapes such as circles, paths, etc.
				svg.ctx.DrawPath(pathX, pathY, path)

				// Copy tag attributes map, excluding `d` which is where
				// path data is stored. Since the path is already returned as
				// `*Path`, there's not much point to returning `d`, and for
				// large/many paths it can be wasteful of memory to return it.
				attrsNoD := map[string]string{}
				for key, value := range attrs {
					if key != "d" {
						attrsNoD[key] = value
					}
				}

				paths = append(paths, SVGPath{
					Tag: tag,
					Attrs: attrsNoD,
					X: pathX,
					Y: pathY,
					Path: path,
				})
			}

			// set linearGradient, markers, etc.
			// these defs depend on the shape or size of the path
			for attr, applyDef := range svg.activeDefs {
				if applyDef != nil {
					applyDef(attr, svg.c)
				}
				svg.activeDefs[attr] = nil
			}

			// tag is self-closing
			if tt == xml.StartTagCloseVoidToken {
				svg.pop()
			}
		case xml.TextToken:
			if 0 < len(svg.elemStack) {
				tag := svg.elemStack[len(svg.elemStack)-1].tag
				if tag == "text" {
					textAlign := Left
					if svg.state.textAnchor == "middle" {
						textAlign = Center
					} else if svg.state.textAnchor == "end" {
						textAlign = Right
					}
					t := html.UnescapeString(string(data))
					text := NewTextLine(svg.getFontFace(), t, textAlign)
					svg.ctx.DrawText(svg.state.textX, svg.state.textY, text)
				}
			}
		case xml.EndTagToken:
			svg.pop()
		}
	}
}

type cssAttrSelector struct {
	op   byte // empty, =, ~, |
	attr string
	val  string
}

func (sel cssAttrSelector) AppliesTo(elem svgElem) bool {
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

func (attr cssAttrSelector) String() string {
	sb := strings.Builder{}
	sb.WriteString(attr.attr)
	if attr.op != 0 {
		sb.WriteByte(attr.op)
		if attr.op != '=' {
			sb.WriteByte('=')
		}
		sb.WriteByte('"')
		sb.WriteString(attr.val)
		sb.WriteByte('"')
	}
	return sb.String()
}

type cssSelectorNode struct {
	op    byte   // space or >, first is always space
	typ   string // is * for universal
	attrs []cssAttrSelector
}

func (sel cssSelectorNode) AppliesTo(elem svgElem) bool {
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

func (sel cssSelectorNode) String() string {
	sb := strings.Builder{}
	sb.WriteByte(sel.op)
	sb.WriteString(sel.typ)
	for _, attr := range sel.attrs {
		if attr.attr == "id" && attr.op == '=' {
			sb.WriteByte('#')
			sb.WriteString(attr.val)
		} else if attr.attr == "class" && attr.op == '~' {
			sb.WriteByte('.')
			sb.WriteString(attr.val)
		} else {
			sb.WriteByte('[')
			sb.WriteString(attr.String())
			sb.WriteByte(']')
		}
	}
	return sb.String()
}

type cssSelector []cssSelectorNode

func (sels cssSelector) AppliesTo(elems []svgElem) bool {
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

func (sels cssSelector) String() string {
	if len(sels) == 0 {
		return ""
	}
	sb := strings.Builder{}
	for _, sel := range sels {
		sb.WriteString(sel.String())
	}
	return sb.String()[1:]
}

type cssProperty struct {
	key, val string
}

func (prop cssProperty) String() string {
	return prop.key + ":" + prop.val
}

type cssRule struct {
	selectors []cssSelector
	props     []cssProperty
}

func (rule cssRule) AppliesTo(elems []svgElem) bool {
	for _, sels := range rule.selectors {
		if sels.AppliesTo(elems) {
			return true
		}
	}
	return false
}

func (rule cssRule) String() string {
	sb := strings.Builder{}
	for i, sel := range rule.selectors {
		if i != 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(sel.String())
	}
	sb.WriteString(" { ")
	for _, prop := range rule.props {
		sb.WriteString(prop.String())
		sb.WriteString("; ")
	}
	sb.WriteString("}")
	return sb.String()
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
