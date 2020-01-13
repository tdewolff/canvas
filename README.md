![Canvas](https://raw.githubusercontent.com/tdewolff/canvas/master/examples/title/out.png)

[![GoDoc](http://godoc.org/github.com/tdewolff/canvas?status.svg)](http://godoc.org/github.com/tdewolff/canvas) [![Build Status](https://travis-ci.org/tdewolff/canvas.svg?branch=master)](https://travis-ci.org/tdewolff/canvas) [![Go Report Card](https://goreportcard.com/badge/github.com/tdewolff/canvas)](https://goreportcard.com/report/github.com/tdewolff/canvas) [![Coverage Status](https://coveralls.io/repos/github/tdewolff/canvas/badge.svg?branch=master)](https://coveralls.io/github/tdewolff/canvas?branch=master)

Canvas is a common vector drawing target that can output SVG, PDF, EPS, raster images (PNG, JPG, GIF, ...), HTML Canvas through WASM, and OpenGL. It has a wide range of path manipulation functionality such as flattening, stroking and dashing implemented. Additionally, it has a good text formatter and embeds fonts (TTF, OTF, WOFF, or WOFF2) or converts them to outlines. It can be considered a Cairo or node-canvas alternative in Go. See the example below in Fig. 1 and Fig. 2 for an overview of the functionality.

![Preview](https://raw.githubusercontent.com/tdewolff/canvas/master/examples/preview/out.png)

**Figure 1**: top-left you can see text being fitted into a box and their bounding box (orange-red), the spaces between the words on the first row are being stretched to fill the whole width. You can see all the possible styles and text decorations applied. Also note the typographic substitutions (the quotes) and ligature support (fi, ffi, ffl, ...). Below the text box, the word "stroke" is being stroked and drawn as a path. Top-right we see a LaTeX formula that has been converted to a path. Left of that we see ellipse support showcasing precise dashing, notably the length of e.g. the short dash is equal wherever it is (approximated through arc length parametrization) on the curve. It also shows support for alternating dash lengths, in this case (2.0, 4.0, 2.0) for dashes and for spaces. Note that the dashes themselves are elliptical arcs as well (thus exactly precise even if magnified greatly). In the bottom-right we see a closed polygon of four points being smoothed by cubic Béziers that are smooth along the whole path, and next to it on the left an open path. In the middle you can see a rasterized image painted.

<a href="https://raw.githubusercontent.com/tdewolff/canvas/master/examples/map/out.png"><img src="https://raw.githubusercontent.com/tdewolff/canvas/master/examples/map/out.png" height="250"></a>
<a href="https://raw.githubusercontent.com/tdewolff/canvas/master/examples/document/out.png"><img src="https://raw.githubusercontent.com/tdewolff/canvas/master/examples/document/out.png" height="250"></a>
<a href="https://raw.githubusercontent.com/tdewolff/canvas/master/examples/go-chart/output.png"><img src="https://raw.githubusercontent.com/tdewolff/canvas/master/examples/go-chart/output.png" height="200" style="padding:25px 0;"></a>

**Figure 2abc**: Three examples of what is possible with this library, for example the plotting of graphs, maps and documents.

[Live WASM HTML Canvas example](https://tdewolff.github.io/canvas/examples/html-canvas/index.html)

**Terminology**: a path is a sequence of drawing commands (MoveTo, LineTo, QuadTo, CubeTo, ArcTo, Close) that completely describe a path. QuadTo and CubeTo are quadratic and cubic Béziers respectively, ArcTo is an elliptical arc, and Close is a LineTo to the last MoveTo command and closes the path (sometimes this has a special meaning such as when stroking). A path can consist of several subpaths by having more than one MoveTo or Close command. A subpath consists of path segments which are defined by a command and some values or coordinates.

Flattening is the act of converting the QuadTo, CubeTo and ArcTo segments into LineTos so that all path segments are linear.

### Getting Started
With modules enabled, add the following imports and run the project with `go get`

```go
import (
    "github.com/tdewolff/canvas"
)
```

#### Examples
**[Preview](https://github.com/tdewolff/canvas/tree/master/examples/preview)**: canvas preview (as shown above) showing most of the functionality and exporting as PNG, SVG, PDF and EPS. It shows image and text rendering as well as LaTeX support and path functionality.

**[Map](https://github.com/tdewolff/canvas/tree/master/examples/map)**: data is loaded from Open Street Map of the city centre of Amsterdam and rendered to a PNG.

**[Graph](https://github.com/tdewolff/canvas/tree/master/examples/graph)**: a simple graph is being plotted using the CO2 data from the Mauna Loa observatory.

**[Text document](https://github.com/tdewolff/canvas/tree/master/examples/document)**: a simple text document is rendered to PNG.

**[go-chart](https://github.com/tdewolff/canvas/tree/master/examples/go-chart)**: using the [go-chart](https://github.com/wcharczuk/go-chart) library a financial graph is plotted.

**[HTML Canvas](https://github.com/tdewolff/canvas/tree/master/examples/html-canvas)**: using WASM, a HTML Canvas is used as target. [Live demo](https://tdewolff.github.io/canvas/examples/html-canvas/index.html).

**[TeX/PGF](https://github.com/tdewolff/canvas/tree/master/examples/tex)**: using the PGF (TikZ) LaTeX package, the output can be directly included in the main TeX file.

**[OpenGL](https://github.com/tdewolff/canvas/tree/master/examples/opengl)**: rendering example to an OpenGL target (WIP).

### Articles
* [Numerically stable quadratic formula](https://math.stackexchange.com/questions/866331/numerically-stable-algorithm-for-solving-the-quadratic-equation-when-a-is-very/2007723#2007723)
* [Quadratic Bézier length](https://malczak.linuxpl.com/blog/quadratic-bezier-curve-length/)
* [Bézier spline through open path](https://www.particleincell.com/2012/bezier-splines/)
* [Bézier spline through closed path](http://www.jacos.nl/jacos_html/spline/circular/index.html)
* [Point inclusion in polygon test](https://wrf.ecse.rpi.edu/Research/Short_Notes/pnpoly.html)

My own

* [Arc length parametrization](https://tacodewolff.nl/posts/20190525-arc-length/)

Papers

* [M. Walter, A. Fournier, Approximate Arc Length Parametrization, Anais do IX SIBGRAPHI (1996), p. 143--150](https://www.visgraf.impa.br/sibgrapi96/trabs/pdf/a14.pdf)
* [T.F. Hain, et al., Fast, precise flattening of cubic Bézier path and offset curves, Computers & Graphics 29 (2005). p. 656--666](https://doi.org/10.1016/j.cag.2005.08.002)
* [M. Goldapp, Approximation of circular arcs by cubic polynomials, Computer Aided Geometric Design 8 (1991), p. 227--238](https://doi.org/10.1016/0167-8396%2891%2990007-X)
* [L. Maisonobe, Drawing and elliptical arc using polylines, quadratic or cubic Bézier curves (2003)](https://spaceroots.org/documents/ellipse/elliptical-arc.pdf)
* [S.H. Kim and Y.J. Ahn, An approximation of circular arcs by quartic Bezier curves, Computer-Aided Design 39 (2007, p. 490--493)](https://doi.org/10.1016/j.cad.2007.01.004)

## Status
### Targets
| Feature | Image | SVG | PDF | EPS | WASM Canvas | OpenGL |
| ------- | ----- | --- | --- | --- | ----------------- | ------ |
| Draw path fill | yes | yes | yes | yes | yes | no |
| Draw path stroke | yes | yes | yes | no | yes | no |
| Draw path dash | yes | yes | yes | no | yes | no |
| Embed fonts | | yes | yes | no | no | no |
| Draw text | path | yes | yes | path | path | path |
| Draw image | yes | yes | yes | no | yes | no |
| EvenOdd fill rule | no | yes | yes | no | no | no |

* EPS does not support transparency
* PDF and EPS do not support line joins for last and first dash for closed dashed path
* OpenGL proper tessellation is missing

### Path
| Command | Flatten | Stroke | Length | SplitAt |
| ------- | ------- | ------ | ------ | ------- |
| LineTo  | yes     | yes    | yes    | yes     |
| QuadTo  | yes (CubeTo) | yes (CubeTo) | yes | yes (GL5 + Chebyshev10) |
| CubeTo  | yes     | yes    | yes (GL5) | yes (GL5 + Chebyshev10) |
| ArcTo   | yes | yes | yes (GL5) | yes (GL5 + Chebyshev10) |

* Ellipse => Cubic Bézier: used by rasterizer and PDF targets (see Maisonobe paper)

NB: GL5 means a Gauss-Legendre n=5, which is an numerical approximation as there is no analytical solution. Chebyshev is a converging way to approximate a function by an n=10 degree polynomial. It uses the bisection method as well to determine the polynomial points.


## Planning
Features that are planned to be implemented in the future, with important issues in bold. Also see the TODOs in the code.

General

* Fix slowness in the rasterizer (text\_example.go is slow! use rasterized cache for each glyph/path)
* Use general span placement algorithm (like CSS flexbox) that replace the current Text placer, to allow for text, image, path elements (e.g. inline formulas, inline icons or emoticons, ...)
* Use word breaking algorithm from [Knuth & Plass](http://defoe.sourceforge.net/folio/knuth-plass.html), implemented in JS in [typeset](http://www.bramstein.com/projects/typeset/). Use letter stretching and shrinking, shrinking by using ligatures, space shrinking and stretching (depending if space is between words or after comma or dot), and spacing or shrinking between glyphs. Use a point system of how ugly breaks are on a paragraph basis. Also see [Justify Just or Just Justify](https://quod.lib.umich.edu/j/jep/3336451.0013.105?view=text;rgn=main).
* Load in Markdown/HTML formatting and turn into text
* Add OpenGL target, needs tessellation (see Delaunay triangulation). See [Resolution independent NURBS curves rendering using programmable graphics pipeline](http://jogamp.com/doc/gpunurbs2011/p70-santina.pdf) and [poly2tri-go](https://github.com/ByteArena/poly2tri-go). Use rational quadratic Beziérs to represent quadratic Beziérs and elliptic arcs exactly, and reduce degree of cubic Beziérs. Using a fragment shader we can draw all curves exactly. Or use rational cubic Beziérs to represent them all exactly?

Fonts

* **Compressing fonts and embedding only used characters**
* **Use ligature and OS/2 tables**
* Support EOT font format
* Font embedding for EPS
* Support font hinting (for the rasterizer)?

Paths

* **Avoid overlapping paths when offsetting in corners**
* Get position and derivative/normal at length L along the path
* Simplify polygons using the Ramer-Douglas-Peucker algorithm
* Intersection function between line, Bézier and ellipse and between themselves (for path merge, overlap/mask, clipping, etc.)
* Implement Bentley-Ottmann algorithm to find all line intersections (clipping)

Far future

* Support fill gradients and patterns (hard)
* Load in PDF, SVG and EPS and turn to paths/text
* Generate TeX-like formulas in pure Go, use OpenType math font such as STIX or TeX Gyre


## Canvas
``` go
c := canvas.New(width, height float64)

ctx := canvas.NewContext(c)
ctx.Push()               // save state set by function below on the stack
ctx.Pop()                // pop state from the stack
ctx.SetView(Matrix)      // set view transformation, all drawn elements are transformed by this matrix
ctx.ComposeView(Matrix)  // add transformation after the current view transformation
ctx.ResetView()          // use identity transformation matrix
ctx.SetFillColor(color.Color)
ctx.SetStrokeColor(color.Color)
ctx.SetStrokeCapper(Capper)
ctx.SetStrokeJoiner(Joiner)
ctx.SetStrokeWidth(width float64)
ctx.SetDashes(offset float64, lengths ...float64)

ctx.DrawPath(x, y float64, *Path)
ctx.DrawText(x, y float64, *Text)
ctx.DrawImage(x, y float64, image.Image, ImageEncoding, dpm float64)

c.Fit(margin float64)  // resize canvas to fit all elements with a given margin
c.SaveSVG(filename string)
c.SaveEPS(filename string)
c.SavePDF(filename string)
c.SavePNG(filename string)
c.SaveJPG(filename string)
c.SaveGIF(filename string)
c.WriteImage(dpm float64) *image.RGBA
```

Canvas allows to draw either paths, text or images. All positions and sizes are given in millimeters.

## Text
![Text Example](https://raw.githubusercontent.com/tdewolff/canvas/master/examples/text/out.png)

``` go
dejaVuSerif := NewFontFamily("dejavu-serif")
err := dejaVuSerif.LoadFontFile("DejaVuSerif.ttf", canvas.FontRegular)  // TTF, OTF, WOFF, or WOFF2
ff := dejaVuSerif.Face(size float64, color.Color, FontStyle, FontVariant, ...FontDecorator)

text = NewTextLine(ff, "string\nsecond line", halign) // simple text line
text = NewTextBox(ff, "string", width, height, halign, valign, indent, lineStretch)  // split on word boundaries and specify text alignment

// rich text allowing different styles of text in one box
richText := NewRichText()  // allow different FontFaces in the same text block
richText.Add(ff, "string")
text = richText.ToText(width, height, halign, valign, indent, lineStretch)

ctx.DrawText(0.0, 0.0, text)
```


## Paths
A large deal of this library implements functionality for building paths. Any path can be constructed from a few basic commands, see below. Successive commands build up segments that start from the current pen position (which is the previous segments's end point) and are drawn towards a new end point. A path can consist of multiple subpaths which each start with a MoveTo command (there is an implicit MoveTo after each Close command), but be aware that overlapping paths can cancel each other depending on the FillRule.

``` go
p := &Path{}
p.MoveTo(x, y float64)                                            // new subpath starting at (x,y)
p.LineTo(x, y float64)                                            // straight line to (x,y)
p.QuadTo(cpx, cpy, x, y float64)                                  // a quadratic Bézier with control point (cpx,cpy) and end point (x,y)
p.CubeTo(cp1x, cp1y, cp2x, cp2y, x, y float64)                    // a cubic Bézier with control points (cp1x,cp1y), (cp2x,cp2y) and end point (x,y)
p.ArcTo(rx, ry, rot float64, largeArc, sweep bool, x, y float64)  // an arc of an ellipse with radii (rx,ry), rotated by rot (in degrees CCW), with flags largeArc and sweep (booleans, see https://www.w3.org/TR/SVG/paths.html#PathDataEllipticalArcCommands)
p.Arc(rx, ry, rot float64, theta0, theta1 float64)                // an arc of an ellipse with radii (rx,ry), rotated by rot (in degrees CCW), beginning at angle theta0 and ending at angle theta1
p.Close()                                                         // close the path, essentially a LineTo to the last MoveTo location

p = Rectangle(w, h float64)
p = RoundedRectangle(w, h, r float64)
p = BeveledRectangle(w, h, r float64)
p = Circle(r float64)
p = Ellipse(rx, ry float64)
p = RegularPolygon(n int, r float64, up bool)
p = RegularStarPolygon(n, d int, r float64, up bool)
p = StarPolygon(n int, R, r float64, up bool)
```

We can extract information from these paths using:

``` go
p.Empty() bool                 // true if path contains no segments (ie. no commands other than MoveTo or Close)
p.Pos() (x, y float64)         // current pen position
p.StartPos() (x, y float64)    // position of last MoveTo
p.Coords() []Point             // start/end positions of all segments
p.CCW() bool                   // true if the path is (mostly) counter clockwise
p.Interior(x, y float64) bool  // true if (x,y) is in the interior of the path, ie. gets filled (depends on FillRule)
p.Filling() []bool             // for all subpaths, true if the subpath is filling (depends on FillRule)
p.Bounds() Rect                // bounding box of path
p.Length() float64             // length of path in millimeters
```

These paths can be manipulated and transformed with the following commands. Each will return a pointer to the path.

``` go
p = p.Copy()
p = p.Append(q *Path)                 // append path q to p and return a new path
p = p.Join(q *Path)                   // join path q to p and return a new path
p = p.Reverse()                       // reverse the direction of the path
ps = p.Split() []*Path                // split the subpaths, ie. at Close/MoveTo
ps = p.SplitAt(d ...float64) []*Path  // split the path at certain lengths d

p = p.Transform(Matrix)               // apply multiple transformations at once and return a new path
p = p.Translate(x, y float64)

p = p.Flatten()                                            // flatten Bézier and arc segments to straight lines
p = p.Offset(width float64)                                // offset the path outwards (width > 0) or inwards (width < 0), depends on FillRule
p = p.Stroke(width float64, capper Capper, joiner Joiner)  // create a stroke from a path of certain width, using capper and joiner for caps and joins
p = p.Dash(offset float64, d ...float64)                   // create dashed path with lengths d which are alternating the dash and the space, start at an offset into the given pattern (can be negative)
```

### Polylines
Some operations on paths only work when it consists of linear segments only. We can either flatten an existing path or use the start/end coordinates of the segments to create a polyline.

``` go
polyline := PolylineFromPath(p)       // create by flattening p
polyline = PolylineFromPathCoords(p)  // create from the start/end coordinates of the segments of p

polyline.Smoothen()              // smoothen it by cubic Béziers
polyline.FillCount() int         // returns the fill count as dictated by the FillRule
polyline.Interior(x, y float64)  // returns true if (x,y) is in the interior of the polyline
```


### Path stroke
Below is an illustration of the different types of Cappers and Joiners you can use when creating a stroke of a path:

![Stroke example](https://raw.githubusercontent.com/tdewolff/canvas/master/examples/stroke/out.png)


## LaTeX
To generate outlines generated by LaTeX, you need `latex` and `dvisvgm` installed on your system.

``` go
p, err := ParseLaTeX(`$y=\sin\(\frac{x}{180}\pi\)$`)
if err != nil {
    panic(err)
}
```

Where the provided string gets inserted into the following document template:

``` latex
\documentclass{article}
\begin{document}
\thispagestyle{empty}
{{input}}
\end{document}
```

### Examples
See https://github.com/tdewolff/canvas/tree/master/examples for a working examples.

## License
Released under the [MIT license](LICENSE.md).
