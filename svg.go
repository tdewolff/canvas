package canvas

import (
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

type svgParserContext struct {
	c                       *Canvas
	ctx                     *Context
	width, height, diagonal float64
}

type markerContext struct {
	c          *Canvas
	refX, refY float64
}

type svgParser struct {
	z *parse.Input

	c []*svgParserContext

	defs map[string]interface{}

	styles          map[string][]string
	stylesSelectors []string

	err error

	instyle bool
	tags    []string
	classes [][]string

	intext       bool
	x            float64
	y            float64
	fontfamilies map[string]*FontFamily
	fontfamily   string
	fontsize     float64
	textanchor   string

	id string

	markerstart *markerContext
	markerend   *markerContext
	markermid   *markerContext
	refX        float64
	refY        float64
}

func (svg *svgParser) ctx() *svgParserContext {
	return svg.c[len(svg.c)-1]
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
				svg.err = parse.NewErrorLexer(svg.z, "bad rgb function")
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
				svg.err = parse.NewErrorLexer(svg.z, "bad rgba function")
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

func (svg *svgParser) setAttribute(key, val string) {
	switch key {
	case "id":
		svg.id = val
	case "fill":
		svg.ctx().ctx.SetFill(svg.parsePaint(val))
	case "stroke":
		svg.ctx().ctx.SetStroke(svg.parsePaint(val))
	case "stroke-width":
		svg.ctx().ctx.SetStrokeWidth(svg.parseDimension(val, svg.ctx().diagonal))
	case "stroke-dashoffset":
		svg.ctx().ctx.Style.DashOffset = svg.parseDimension(val, svg.ctx().diagonal)
	case "stroke-dasharray":
		if val == "none" {
			svg.ctx().ctx.Style.Dashes = []float64{}
			break
		}
		svg.ctx().ctx.Style.Dashes = svg.parsePoints(val)
	case "stroke-linecap":
		if val == "butt" {
			svg.ctx().ctx.SetStrokeCapper(ButtCap)
		} else if val == "round" {
			svg.ctx().ctx.SetStrokeCapper(RoundCap)
		} else if val == "square" {
			svg.ctx().ctx.SetStrokeCapper(SquareCap)
		}
	case "stroke-linejoin":
		if val == "arcs" {
			svg.ctx().ctx.SetStrokeJoiner(ArcsJoin)
		} else if val == "bevel" {
			svg.ctx().ctx.SetStrokeJoiner(BevelJoin)
		} else if val == "miter" {
			// TODO: not exactly correct
			svg.ctx().ctx.SetStrokeJoiner(MiterJoiner{BevelJoin, 4.0})
		} else if val == "miter-clip" {
			svg.ctx().ctx.SetStrokeJoiner(MiterJoiner{BevelJoin, 4.0})
		} else if val == "round" {
			svg.ctx().ctx.SetStrokeJoiner(RoundJoin)
		}
	case "stroke-miterlimit":
	// TODO: keep in state?
	case "transform":
		svg.ctx().ctx.ComposeView(svg.parseTransform(val))
		// TODO: add more: stroke-opacity, fill-opacity
	case "font-family":
		svg.fontfamily = val
	case "font-size":
		// TODO come up with better relational font sizes
		if val == "medium" {
			val = "14pt"
		}
		svg.fontsize = svg.parseDimension(val, svg.ctx().height)
	case "marker-start":
		if strings.HasPrefix(val, "url(") {
			id := val[5 : len(val)-1]
			if marker, ok := svg.defs[id]; ok {
				svg.markerstart = marker.(*markerContext)
			}
		}
	case "marker-end":
		if strings.HasPrefix(val, "url(") {
			id := val[5 : len(val)-1]
			if marker, ok := svg.defs[id]; ok {
				svg.markerend = marker.(*markerContext)
			}
		}
	case "marker-mid":
		if strings.HasPrefix(val, "url(") {
			id := val[5 : len(val)-1]
			if marker, ok := svg.defs[id]; ok {
				svg.markermid = marker.(*markerContext)
			}
		}
	case "refX":
		svg.refX = svg.parseDimension(val, 0.0)
	case "refY":
		svg.refY = svg.parseDimension(val, 0.0)
	case "text-anchor":
		svg.textanchor = val
	}
}

func (svg *svgParser) loadFontFamily(family string) *FontFamily {
	if ff, ok := svg.fontfamilies[family]; !ok {
		ff = NewFontFamily(family)
		ff.LoadLocalFont(family, FontRegular)
		svg.fontfamilies[family] = ff
		return ff
	} else {
		return ff
	}
}

func (svg *svgParser) markPath(p *Path) {
	for vi, vp := range p.Coords() {
		var marker *markerContext
		if vi == 0 {
			marker = svg.markerstart
		} else if vi == len(p.Coords())-1 {
			marker = svg.markerend
		} else {
			marker = svg.markermid
		}

		if marker != nil {
			view := Identity.ReflectYAbout(svg.ctx().ctx.Height() / 2.0)
			// TODO figure out why the slight precision and scale loss here
			view = view.Translate(vp.X-marker.refX-1, vp.Y-marker.refY-1)
			view = view.Scale(1.5, 1.5)
			marker.c.RenderViewTo(svg.ctx().c, view)
		}
	}
}

func ParseSVG(r io.Reader) (*Canvas, error) {
	z := parse.NewInput(r)
	defer z.Restore()

	l := xml.NewLexer(z)
	svg := svgParser{
		z: z,
	}
	for {
		tt, data := l.Next()
		switch tt {
		case xml.ErrorToken:
			if l.Err() != io.EOF {
				if len(svg.c) == 0 {
					return New(0.0, 0.0), l.Err()
				}

				return svg.ctx().c, l.Err()
			} else if svg.err != nil {
				if len(svg.c) == 0 {
					return New(0.0, 0.0), svg.err
				}

				return svg.c[0].c, svg.err
			} else if len(svg.c) == 0 {
				return New(0.0, 0.0), fmt.Errorf("expected SVG tag")
			}
			if svg.ctx().c.W == 0.0 || svg.ctx().c.H == 0.0 {
				svg.ctx().c.Fit(0.0)
			}
			return svg.c[0].c, nil
		case xml.StartTagToken:
			// TODO: attribute errors point to wrong position
			attrs := map[string]string{}
			attrNames := []string{}
			for {
				tt, _ = l.Next()
				if tt != xml.AttributeToken {
					break
				}
				val := l.AttrVal()
				val = val[1 : len(val)-1]
				attrNames = append(attrNames, string(l.Text()))
				attrs[string(l.Text())] = string(val)
			}

			tag := string(data[1:])
			if (tag == "svg" && len(svg.c) == 0) || tag == "marker" {
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
				} else if _, ok := attrs["markerWidth"]; ok {
					width = svg.parseDimension(attrs["markerWidth"], 0.0)
				} else {
					width = (viewbox[2] - viewbox[0]) * 25.4 / 96.0
				}
				if _, ok := attrs["height"]; ok {
					height = svg.parseDimension(attrs["height"], 0.0)
				} else if _, ok := attrs["markerHeight"]; ok {
					height = svg.parseDimension(attrs["markerHeight"], 0.0)
				} else {
					height = (viewbox[3] - viewbox[1]) * 25.4 / 96.0
				}

				spc := svgParserContext{}
				svg.c = append(svg.c, &spc)

				spc.width = width * 96.0 / 25.4
				spc.height = height * 96.0 / 25.4

				spc.diagonal = math.Sqrt((spc.width*spc.width + spc.height*spc.height) / 2.0)

				spc.c = New(width, height)
				spc.ctx = NewContext(spc.c)
				spc.ctx.SetCoordSystem(CartesianIV)
				if 0.0 < (viewbox[2]-viewbox[0]) && 0.0 < (viewbox[3]-viewbox[1]) {
					m := Identity.Scale(width/(viewbox[2]-viewbox[0]), height/(viewbox[3]-viewbox[1])).Translate(-viewbox[0], -viewbox[1])
					spc.ctx.SetView(m)
					spc.ctx.SetCoordView(m)
				}
				spc.ctx.SetStrokeJoiner(MiterJoiner{BevelJoin, 4.0})

				if svg.defs == nil {
					svg.defs = make(map[string]interface{})
					svg.fontfamilies = make(map[string]*FontFamily)
					svg.styles = make(map[string][]string)
				}
			} else if tag != "svg" && len(svg.c) == 0 {
				return New(0.0, 0.0), fmt.Errorf("expected SVG tag")
			}

			svg.ctx().ctx.Push()
			svg.tags = append(svg.tags, tag)
			classes := []string{}
			for _, key := range attrNames {
				val := attrs[key]
				if key == "class" {
					classes = strings.Split(val, " ")
				}
			}
			svg.classes = append(svg.classes, classes)

			// Check for matching styles
			for _, s := range svg.stylesSelectors {
				if styles, ok := svg.styles[s]; ok {
					selector := strings.Split(s, " ")
					i := len(svg.tags) - 1

					for len(selector) != 0 && i >= 0 {
						parts := strings.Split(selector[len(selector)-1], ".")
						t := parts[0]
						c := ""
						if len(parts) == 2 {
							c = parts[1]
						}

						tagMatches := false
						classMatches := false
						if t == "" {
							tagMatches = true
						} else if t == svg.tags[i] {
							tagMatches = true
						}

						if c == "" {
							classMatches = true
						} else {
							for _, class := range svg.classes[i] {
								if class == c {
									classMatches = true
									break
								}
							}
						}

						if tagMatches && classMatches {
							selector = selector[:len(selector)-1]
						}

						i -= 1
					}

					if len(selector) == 0 {
						for _, style := range styles {
							values := strings.Split(style, ":")
							svg.setAttribute(values[0], values[1])
						}
					}
				}
			}

			for _, key := range attrNames {
				val := attrs[key]
				if key == "style" {
					for _, item := range strings.Split(val, ";") {
						if keyVal := strings.Split(item, ":"); len(keyVal) == 2 {
							svg.setAttribute(strings.TrimSpace(keyVal[0]), strings.TrimSpace(keyVal[1]))
						}
					}
				} else {
					svg.setAttribute(key, val)
				}
			}

			switch tag {
			case "circle":
				cx := svg.parseDimension(attrs["cx"], svg.ctx().width)
				cy := svg.parseDimension(attrs["cy"], svg.ctx().height)
				r := svg.parseDimension(attrs["r"], svg.ctx().diagonal)
				svg.ctx().ctx.DrawPath(cx, cy, Circle(r))
			case "ellipse":
				cx := svg.parseDimension(attrs["cx"], svg.ctx().width)
				cy := svg.parseDimension(attrs["cy"], svg.ctx().height)
				rx := svg.parseDimension(attrs["rx"], svg.ctx().width)
				ry := svg.parseDimension(attrs["ry"], svg.ctx().height)
				p := Ellipse(rx, ry)
				svg.ctx().ctx.DrawPath(cx, cy, p)
			case "path":
				p, err := ParseSVGPath(attrs["d"])
				if err != nil && svg.err == nil {
					svg.err = parse.NewErrorLexer(svg.z, "bad path: %w", err)
				}

				svg.ctx().ctx.DrawPath(0, 0, p)
				svg.markPath(p)
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
				svg.ctx().ctx.DrawPath(0.0, 0.0, p)
				svg.markPath(p)
			case "line":
				p := &Path{}
				x1, _ := strconv.ParseInt(attrs["x1"], 10, 64)
				y1, _ := strconv.ParseInt(attrs["y1"], 10, 64)
				x2, _ := strconv.ParseInt(attrs["x2"], 10, 64)
				y2, _ := strconv.ParseInt(attrs["y2"], 10, 64)

				p.MoveTo(float64(x1), float64(y1))
				p.LineTo(float64(x2), float64(y2))
				svg.ctx().ctx.DrawPath(0.0, 0.0, p)
				svg.markPath(p)
			case "rect":
				x := svg.parseDimension(attrs["x"], svg.ctx().width)
				y := svg.parseDimension(attrs["y"], svg.ctx().height)
				width := svg.parseDimension(attrs["width"], svg.ctx().width)
				height := svg.parseDimension(attrs["height"], svg.ctx().height)
				p := Rectangle(width, height)
				svg.ctx().ctx.DrawPath(x, y, p)
				// TODO check if this works when x, y are not in the path
				svg.markPath(p)
			case "text":
				svg.intext = true
				svg.x = svg.parseDimension(attrs["x"], svg.ctx().width)
				svg.y = svg.parseDimension(attrs["y"], svg.ctx().height)
			case "style":
				svg.instyle = true
			}

			if tt == xml.StartTagCloseVoidToken {
				svg.ctx().ctx.Pop()
				if len(svg.tags) > 0 {
					svg.tags = svg.tags[:len(svg.tags)-1]
					svg.classes = svg.classes[:len(svg.classes)-1]
				}
			}
		case xml.TextToken:
			if svg.intext {
				// TODO come up with a better default font and size
				family := svg.fontfamily
				size := svg.fontsize
				if family == "" {
					family = "monospace"
				}
				if size == 0 {
					size = svg.parseDimension("14pt", svg.ctx().height)
				}
				// TODO why is the size so small and needs this kind of scaling?
				size = size * 2.5
				ff := svg.loadFontFamily(family)
				face := ff.Face(size, FontRegular)
				text := NewTextLine(face, string(data), Left)
				xadj := 0.0
				if svg.textanchor == "middle" {
					xadj = text.Bounds().W / 2
				} else if svg.textanchor == "end" {
					xadj = text.Bounds().W
				}
				svg.ctx().ctx.DrawText(svg.x-xadj, svg.y, text)
			}
			if svg.instyle {
				parser := css.NewParser(parse.NewInputString(string(data)), false)
				selectors := []string{}
				styles := []string{}
				for {
					gt, _, data := parser.Next()
					if gt == css.QualifiedRuleGrammar || gt == css.BeginRulesetGrammar {
						selector := []string{}
						for _, v := range parser.Values() {
							if v.TokenType == css.DelimToken || v.TokenType == css.IdentToken {
								selector = append(selector, string(v.Data))
							} else if v.TokenType == css.WhitespaceToken {
								selector = append(selector, " ")
							}
						}
						if len(selector) != 0 {
							selectors = append(selectors, strings.Join(selector, ""))
						}
					}

					if gt == css.DeclarationGrammar {
						values := ""
						for _, value := range parser.Values() {
							values += string(value.Data)
						}
						styles = append(styles, fmt.Sprintf("%s:%s", string(data), values))
					}

					if gt == css.ErrorGrammar || gt == css.EndRulesetGrammar {
						for _, sel := range selectors {
							svg.styles[sel] = styles
							svg.stylesSelectors = append(svg.stylesSelectors, sel)
						}
						selectors = []string{}
						styles = []string{}
					}

					if gt == css.ErrorGrammar {
						break
					}
				}
			}
		case xml.EndTagToken:
			svg.ctx().ctx.Pop()
			svg.instyle = false
			svg.intext = false
			svg.tags = svg.tags[:len(svg.tags)-1]
			svg.classes = svg.classes[:len(svg.classes)-1]
			svg.markerstart = nil
			svg.markerend = nil
			svg.markermid = nil

			tag := string(data[2 : len(data)-1])
			if tag == "marker" {
				canvas := svg.c[len(svg.c)-1].c
				svg.c = svg.c[:len(svg.c)-1]
				mc := markerContext{c: canvas, refX: svg.refX, refY: svg.refY}
				if svg.id != "" {
					svg.defs[svg.id] = &mc
				}
			}
		}
	}
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
