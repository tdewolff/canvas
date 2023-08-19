package canvas

import (
	"fmt"
	"image/color"
	"io"
	"strconv"
	"strings"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/xml"
)

func parseCSSDimension(v string, parent float64) (float64, error) {
	// to px
	if len(v) == 0 {
		return 0.0, nil
	}

	nn, _ := parse.Dimension([]byte(v))
	num, err := strconv.ParseFloat(v[:nn], 64)
	if err != nil {
		return 0.0, err
	}

	dim := v[nn:]
	switch strings.ToLower(dim) {
	case "cm":
		return num * 10.0 * 96.0 / 25.4, nil
	case "mm":
		return num * 96.0 / 25.4, nil
	case "q":
		return num * 0.25 * 96.0 / 25.4, nil
	case "in":
		return num * 96.0, nil
	case "pc":
		return num * 96.0 / 6.0, nil
	case "pt":
		return num * 96.0 / 72.0, nil
	case "", "px":
		return num, nil
	case "%":
		return num * parent / 100.0, nil
	}
	return 0.0, fmt.Errorf("unknown dimension: %s", dim)
}

func parseCSSColorComponent(v string) uint8 {
	v = strings.TrimSpace(v)
	if len(v) == 0 {
		return 0
	} else if v[len(v)-1] == '%' {
		num, _ := strconv.ParseFloat(v[:len(v)-1], 64)
		return uint8(num*255.0 + 0.5)
	}
	num, _ := strconv.ParseUint(v, 10, 8)
	return uint8(num)
}

func parseCSSColor(v string) color.RGBA {
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
			return Black
		}
		col.R = parseCSSColorComponent(comps[0])
		col.G = parseCSSColorComponent(comps[1])
		col.B = parseCSSColorComponent(comps[2])
		col.A = 255
	} else if strings.HasPrefix(v, "rgba(") && strings.HasSuffix(v, ")") {
		comps := strings.Split(v[4:len(v)-1], ",")
		if len(comps) != 4 {
			return Black
		}
		col.A = parseCSSColorComponent(comps[3])
		col.R = uint8(float64(parseCSSColorComponent(comps[0]))*float64(col.A)/255.0 + 0.5)
		col.G = uint8(float64(parseCSSColorComponent(comps[1]))*float64(col.A)/255.0 + 0.5)
		col.B = uint8(float64(parseCSSColorComponent(comps[2]))*float64(col.A)/255.0 + 0.5)
	}
	return col
}

func parseCSSPoints(v string) ([]float64, error) {
	v = strings.ReplaceAll(v, "\n", ",")
	v = strings.ReplaceAll(v, "\t", ",")
	v = strings.ReplaceAll(v, " ", ",")

	vals := []float64{}
	for _, item := range strings.Split(v, ",") {
		if 0 < len(item) {
			val, err := strconv.ParseFloat(item, 64)
			if err != nil {
				return nil, err
			}
			vals = append(vals, val)
		}
	}
	if len(vals)%2 == 1 {
		vals = vals[:len(vals)-1]
	}
	return vals, nil
}

func parseCSSAttribute(ctx *Context, key, val string) {
	switch key {
	case "fill":
		ctx.SetFillColor(parseCSSColor(val))
	case "stroke":
		ctx.SetStrokeColor(parseCSSColor(val))
	case "stroke-width":
		v, _ := parseCSSDimension(val)
		ctx.SetStrokeWidth(v)
	case "stroke-dashoffset":
		v, _ := parseCSSDimension(val)
		ctx.Style.DashOffset = v
	case "stroke-dasharray":
		v, _ := parseCSSPoints(val)
		ctx.Style.Dashes = v
	case "stroke-linecap":
		if val == "butt" {
			ctx.SetStrokeCapper(ButtCap)
		} else if val == "round" {
			ctx.SetStrokeCapper(RoundCap)
		} else if val == "square" {
			ctx.SetStrokeCapper(SquareCap)
		}
	case "stroke-linejoin":
		if val == "arcs" {
			ctx.SetStrokeJoiner(ArcsJoin)
		} else if val == "bevel" {
			ctx.SetStrokeJoiner(BevelJoin)
		} else if val == "miter" {
			ctx.SetStrokeJoiner(MiterJoin)
		} else if val == "miter-clip" {
			ctx.SetStrokeJoiner(MiterJoin)
		} else if val == "round" {
			ctx.SetStrokeJoiner(RoundJoin)
		}
		// TODO: add more: stroke-miterlimit, stroke-opacity, fill-opacity, transform
	}
}

func ParseSVG(r io.Reader) (*Canvas, error) {
	var c *Canvas
	var ctx *Context

	var viewBoxWidth, viewBoxHeight float64

	l := xml.NewLexer(parse.NewInput(r))
	for {
		tt, data := l.Next()
		switch tt {
		case xml.ErrorToken:
			if l.Err() != io.EOF {
				return nil, l.Err()
			} else if c == nil {
				return nil, fmt.Errorf("expected SVG tag")
			}
			if c.W == 0.0 || c.H == 0.0 {
				c.Fit(0.0)
			}
			return c, nil
		case xml.StartTagToken:
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
			if tag == "svg" {
				if c != nil {
					return nil, fmt.Errorf("unexpected SVG tag")
				}

				var err error
				var viewbox [4]float64
				var width, height float64
				if _, ok := attrs["viewBox"]; ok {
					vals := strings.Split(attrs["viewBox"], " ")
					if len(vals) != 4 {
						return nil, fmt.Errorf("svg viewBox attribute invalid")
					}
					for i := 0; i < 4; i++ {
						viewbox[i], err = strconv.ParseFloat(vals[i], 64)
						if err != nil {
							return nil, fmt.Errorf("svg viewBox attribute: %w", err)
						}
					}
				}
				if _, ok := attrs["width"]; ok {
					width, err = parseCSSDimension(attrs["width"], 0.0)
					if err != nil {
						return nil, fmt.Errorf("svg width attribute: %w", err)
					}
				} else {
					width = (viewbox[2] - viewbox[0]) * 25.4 / 96.0
				}
				if _, ok := attrs["height"]; ok {
					height, err = parseCSSDimension(attrs["height"], 0.0)
					if err != nil {
						return nil, fmt.Errorf("svg height attribute: %w", err)
					}
				} else {
					height = (viewbox[3] - viewbox[1]) * 25.4 / 96.0
				}

				viewBoxWidth = width * 96.0 / 25.4
				viewBoxHeight = height * 96.0 / 25.4

				c = New(width, height)
				ctx = NewContext(c)
				ctx.SetCoordSystem(CartesianIV)
				if 0.0 < (viewbox[2]-viewbox[0]) && 0.0 < (viewbox[3]-viewbox[1]) {
					m := Identity.Scale(width/(viewbox[2]-viewbox[0]), height/(viewbox[3]-viewbox[1])).Translate(-viewbox[0], -viewbox[1])
					ctx.SetView(m)
					ctx.SetCoordView(m)
				}
			} else if c == nil {
				return nil, fmt.Errorf("expected SVG tag")
			}

			ctx.Push()
			for _, key := range attrNames {
				val := attrs[key]
				if key == "style" {
					for _, item := range strings.Split(val, ";") {
						if keyVal := strings.Split(item, ":"); len(keyVal) == 2 {
							parseCSSAttribute(ctx, strings.TrimSpace(keyVal[0]), strings.TrimSpace(keyVal[1]))
						}
					}
				} else {
					parseCSSAttribute(ctx, key, val)
				}
			}

			switch tag {
			case "circle":
				cx, err := parseCSSDimension(attrs["cx"], viewBoxWidth)
				if err != nil {
					return nil, fmt.Errorf("circle cx attribute: %w", err)
				}
				cy, err := parseCSSDimension(attrs["cy"], viewBoxHeight)
				if err != nil {
					return nil, fmt.Errorf("circle cy attribute: %w", err)
				}
				r, err := parseCSSDimension(attrs["r"], 0.0)
				if err != nil {
					return nil, fmt.Errorf("circle r attribute: %w", err)
				}
				ctx.DrawPath(cx, cy, Circle(r))
			case "ellipse":
				cx, err := parseCSSDimension(attrs["cx"], viewBoxWidth)
				if err != nil {
					return nil, fmt.Errorf("ellipse cx attribute: %w", err)
				}
				cy, err := parseCSSDimension(attrs["cy"], viewBoxHeight)
				if err != nil {
					return nil, fmt.Errorf("ellipse cy attribute: %w", err)
				}
				rx, err := parseCSSDimension(attrs["rx"], viewBoxWidth)
				if err != nil {
					return nil, fmt.Errorf("ellipse rx attribute: %w", err)
				}
				ry, err := parseCSSDimension(attrs["ry"], viewBoxHeight)
				if err != nil {
					return nil, fmt.Errorf("ellipse ry attribute: %w", err)
				}
				ctx.DrawPath(cx, cy, Ellipse(rx, ry))
			case "path":
				p, err := ParseSVGPath(attrs["d"])
				if err != nil {
					return nil, fmt.Errorf("path d attribute: %w", err)
				}
				ctx.DrawPath(0, 0, p)
			case "polygon", "polyline":
				points, err := parseCSSPoints(attrs["points"])
				if err != nil {
					return nil, fmt.Errorf("%s points attribute: %w", tag, err)
				}
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
				ctx.DrawPath(0.0, 0.0, p)
			case "rect":
				x, err := parseCSSDimension(attrs["x"], viewBoxWidth)
				if err != nil {
					return nil, fmt.Errorf("rect x attribute: %w", err)
				}
				y, err := parseCSSDimension(attrs["y"], viewBoxHeight)
				if err != nil {
					return nil, fmt.Errorf("rect y attribute: %w", err)
				}
				width, err := parseCSSDimension(attrs["width"], viewBoxWidth)
				if err != nil {
					return nil, fmt.Errorf("rect width attribute: %w", err)
				}
				height, err := parseCSSDimension(attrs["height"], viewBoxHeight)
				if err != nil {
					return nil, fmt.Errorf("rect height attribute: %w", err)
				}
				ctx.DrawPath(x, y, Rectangle(width, height))
			}

			if tt == xml.StartTagCloseVoidToken {
				ctx.Pop()
			}
		case xml.EndTagToken:
			ctx.Pop()
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
