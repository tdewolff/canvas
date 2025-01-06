package main

import (
	"image/color"
	"math"
	"sort"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers"
)

func hue(col color.RGBA) float64 {
	var hue float64
	a := float64(col.A) / 255.0
	r := float64(col.R) / 255.0 / a
	g := float64(col.G) / 255.0 / a
	b := float64(col.B) / 255.0 / a

	min := math.Min(r, math.Min(g, b))
	max := math.Max(r, math.Max(g, b))
	if min == max {
		return -1.0
	} else if r == max {
		hue = (g - b) / (max - min)
	} else if g == max {
		hue = 2.0 + (b-r)/(max-min)
	} else if b == max {
		hue = 4.0 + (r-g)/(max-min)
	}
	hue *= 60.0
	if hue < 0.0 {
		hue += 360.0
	}
	return hue
}

func lum(col color.RGBA) float64 {
	a := float64(col.A) / 255.0
	r := float64(col.R) / 255.0 / a
	g := float64(col.G) / 255.0 / a
	b := float64(col.B) / 255.0 / a
	return 0.2126*r + 0.7152*g + 0.0722*b
}

func main() {
	font := canvas.NewFontFamily("latin")
	if err := font.LoadSystemFont("serif", canvas.FontRegular); err != nil {
		panic(err)
	}

	// sort by hue
	indices := make([]int, len(colors))
	for i := range colors {
		indices[i] = i
	}
	sort.Slice(indices, func(i, j int) bool {
		if hue(colors[indices[i]]) == hue(colors[indices[j]]) {
			return lum(colors[indices[i]]) < lum(colors[indices[j]])
		}
		return hue(colors[indices[i]]) < hue(colors[indices[j]])
	})

	c := canvas.New(0.0, 0.0)
	ctx := canvas.NewContext(c)
	face := font.Face(2.0)

	for j, idx := range indices {
		i := j / 25
		j -= 25 * i

		ctx.SetFillColor(colors[idx])
		ctx.DrawPath(0.5+10.0*float64(i), -9.5-10.0*float64(j), canvas.Rectangle(9.0, 9.0))

		p, w, _ := face.ToPath(names[idx])
		p = p.Offset(0.05, 0.01)
		p = p.Translate(-w/2.0, 0.0)
		ctx.SetFillColor(canvas.Hex("#ffffff88"))
		ctx.DrawPath(5.0+10.0*float64(i), -9.0-10.0*float64(j), p)

		text := canvas.NewTextLine(face, names[idx], canvas.Center)
		ctx.DrawText(5.0+10.0*float64(i), -9.0-10.0*float64(j), text)
	}

	c.Fit(1.0)
	c.SetZIndex(-1)
	ctx.SetFillColor(canvas.White)
	ctx.SetStrokeColor(canvas.Transparent)
	ctx.DrawPath(0, 0, canvas.Rectangle(c.W, c.H))

	renderers.Write("colors.png", c, canvas.DPMM(20.0))
}

var names = []string{
	"Aliceblue",
	"Antiquewhite",
	"Aqua",
	"Aquamarine",
	"Azure",
	"Beige",
	"Bisque",
	"Black",
	"Blanchedalmond",
	"Blue",
	"Blueviolet",
	"Brown",
	"Burlywood",
	"Cadetblue",
	"Chartreuse",
	"Chocolate",
	"Coral",
	"Cornflowerblue",
	"Cornsilk",
	"Crimson",
	"Cyan",
	"Darkblue",
	"Darkcyan",
	"Darkgoldenrod",
	"Darkgray",
	"Darkgreen",
	"Darkgrey",
	"Darkkhaki",
	"Darkmagenta",
	"Darkolivegreen",
	"Darkorange",
	"Darkorchid",
	"Darkred",
	"Darksalmon",
	"Darkseagreen",
	"Darkslateblue",
	"Darkslategray",
	"Darkslategrey",
	"Darkturquoise",
	"Darkviolet",
	"Deeppink",
	"Deepskyblue",
	"Dimgray",
	"Dimgrey",
	"Dodgerblue",
	"Firebrick",
	"Floralwhite",
	"Forestgreen",
	"Fuchsia",
	"Gainsboro",
	"Ghostwhite",
	"Gold",
	"Goldenrod",
	"Gray",
	"Green",
	"Greenyellow",
	"Grey",
	"Honeydew",
	"Hotpink",
	"Indianred",
	"Indigo",
	"Ivory",
	"Khaki",
	"Lavender",
	"Lavenderblush",
	"Lawngreen",
	"Lemonchiffon",
	"Lightblue",
	"Lightcoral",
	"Lightcyan",
	"Lightgoldenrodyellow",
	"Lightgray",
	"Lightgreen",
	"Lightgrey",
	"Lightpink",
	"Lightsalmon",
	"Lightseagreen",
	"Lightskyblue",
	"Lightslategray",
	"Lightslategrey",
	"Lightsteelblue",
	"Lightyellow",
	"Lime",
	"Limegreen",
	"Linen",
	"Magenta",
	"Maroon",
	"Mediumaquamarine",
	"Mediumblue",
	"Mediumorchid",
	"Mediumpurple",
	"Mediumseagreen",
	"Mediumslateblue",
	"Mediumspringgreen",
	"Mediumturquoise",
	"Mediumvioletred",
	"Midnightblue",
	"Mintcream",
	"Mistyrose",
	"Moccasin",
	"Navajowhite",
	"Navy",
	"Oldlace",
	"Olive",
	"Olivedrab",
	"Orange",
	"Orangered",
	"Orchid",
	"Palegoldenrod",
	"Palegreen",
	"Paleturquoise",
	"Palevioletred",
	"Papayawhip",
	"Peachpuff",
	"Peru",
	"Pink",
	"Plum",
	"Powderblue",
	"Purple",
	"Red",
	"Rosybrown",
	"Royalblue",
	"Saddlebrown",
	"Salmon",
	"Sandybrown",
	"Seagreen",
	"Seashell",
	"Sienna",
	"Silver",
	"Skyblue",
	"Slateblue",
	"Slategray",
	"Slategrey",
	"Snow",
	"Springgreen",
	"Steelblue",
	"Tan",
	"Teal",
	"Thistle",
	"Tomato",
	"Turquoise",
	"Violet",
	"Wheat",
	"White",
	"Whitesmoke",
	"Yellow",
	"Yellowgreen",
}

var colors = []color.RGBA{
	canvas.Aliceblue,
	canvas.Antiquewhite,
	canvas.Aqua,
	canvas.Aquamarine,
	canvas.Azure,
	canvas.Beige,
	canvas.Bisque,
	canvas.Black,
	canvas.Blanchedalmond,
	canvas.Blue,
	canvas.Blueviolet,
	canvas.Brown,
	canvas.Burlywood,
	canvas.Cadetblue,
	canvas.Chartreuse,
	canvas.Chocolate,
	canvas.Coral,
	canvas.Cornflowerblue,
	canvas.Cornsilk,
	canvas.Crimson,
	canvas.Cyan,
	canvas.Darkblue,
	canvas.Darkcyan,
	canvas.Darkgoldenrod,
	canvas.Darkgray,
	canvas.Darkgreen,
	canvas.Darkgrey,
	canvas.Darkkhaki,
	canvas.Darkmagenta,
	canvas.Darkolivegreen,
	canvas.Darkorange,
	canvas.Darkorchid,
	canvas.Darkred,
	canvas.Darksalmon,
	canvas.Darkseagreen,
	canvas.Darkslateblue,
	canvas.Darkslategray,
	canvas.Darkslategrey,
	canvas.Darkturquoise,
	canvas.Darkviolet,
	canvas.Deeppink,
	canvas.Deepskyblue,
	canvas.Dimgray,
	canvas.Dimgrey,
	canvas.Dodgerblue,
	canvas.Firebrick,
	canvas.Floralwhite,
	canvas.Forestgreen,
	canvas.Fuchsia,
	canvas.Gainsboro,
	canvas.Ghostwhite,
	canvas.Gold,
	canvas.Goldenrod,
	canvas.Gray,
	canvas.Green,
	canvas.Greenyellow,
	canvas.Grey,
	canvas.Honeydew,
	canvas.Hotpink,
	canvas.Indianred,
	canvas.Indigo,
	canvas.Ivory,
	canvas.Khaki,
	canvas.Lavender,
	canvas.Lavenderblush,
	canvas.Lawngreen,
	canvas.Lemonchiffon,
	canvas.Lightblue,
	canvas.Lightcoral,
	canvas.Lightcyan,
	canvas.Lightgoldenrodyellow,
	canvas.Lightgray,
	canvas.Lightgreen,
	canvas.Lightgrey,
	canvas.Lightpink,
	canvas.Lightsalmon,
	canvas.Lightseagreen,
	canvas.Lightskyblue,
	canvas.Lightslategray,
	canvas.Lightslategrey,
	canvas.Lightsteelblue,
	canvas.Lightyellow,
	canvas.Lime,
	canvas.Limegreen,
	canvas.Linen,
	canvas.Magenta,
	canvas.Maroon,
	canvas.Mediumaquamarine,
	canvas.Mediumblue,
	canvas.Mediumorchid,
	canvas.Mediumpurple,
	canvas.Mediumseagreen,
	canvas.Mediumslateblue,
	canvas.Mediumspringgreen,
	canvas.Mediumturquoise,
	canvas.Mediumvioletred,
	canvas.Midnightblue,
	canvas.Mintcream,
	canvas.Mistyrose,
	canvas.Moccasin,
	canvas.Navajowhite,
	canvas.Navy,
	canvas.Oldlace,
	canvas.Olive,
	canvas.Olivedrab,
	canvas.Orange,
	canvas.Orangered,
	canvas.Orchid,
	canvas.Palegoldenrod,
	canvas.Palegreen,
	canvas.Paleturquoise,
	canvas.Palevioletred,
	canvas.Papayawhip,
	canvas.Peachpuff,
	canvas.Peru,
	canvas.Pink,
	canvas.Plum,
	canvas.Powderblue,
	canvas.Purple,
	canvas.Red,
	canvas.Rosybrown,
	canvas.Royalblue,
	canvas.Saddlebrown,
	canvas.Salmon,
	canvas.Sandybrown,
	canvas.Seagreen,
	canvas.Seashell,
	canvas.Sienna,
	canvas.Silver,
	canvas.Skyblue,
	canvas.Slateblue,
	canvas.Slategray,
	canvas.Slategrey,
	canvas.Snow,
	canvas.Springgreen,
	canvas.Steelblue,
	canvas.Tan,
	canvas.Teal,
	canvas.Thistle,
	canvas.Tomato,
	canvas.Turquoise,
	canvas.Violet,
	canvas.Wheat,
	canvas.White,
	canvas.Whitesmoke,
	canvas.Yellow,
	canvas.Yellowgreen,
}
