![Canvas](https://raw.githubusercontent.com/tdewolff/canvas/master/examples/title/out.png)

[![GoDoc](http://godoc.org/github.com/tdewolff/canvas?status.svg)](http://godoc.org/github.com/tdewolff/canvas) [![Build Status](https://travis-ci.org/tdewolff/canvas.svg?branch=master)](https://travis-ci.org/tdewolff/canvas) [![Go Report Card](https://goreportcard.com/badge/github.com/tdewolff/canvas)](https://goreportcard.com/report/github.com/tdewolff/canvas) [![Coverage Status](https://coveralls.io/repos/github/tdewolff/canvas/badge.svg?branch=master)](https://coveralls.io/github/tdewolff/canvas?branch=master)

Canvas is a common vector drawing target that can output SVG, PDF, EPS and raster images (which can be saved as PNG, JPG, ...). It has a wide range of path manipulation functionality such as flattening, stroking and dashing implemented. Additionally, it has a good text formatter and embeds fonts (TTF, OTF or WOFF) or converts them to outlines. It can be considered a Cairo or node-canvas alternative in Go. See the example below in Fig. 1 and Fig. 2 for an overview of the functionality.

![Preview](https://raw.githubusercontent.com/tdewolff/canvas/master/examples/preview/out.png)

**Figure 1**: top-left you can see text being fitted into a box and their bounding box (orange-red), the spaces between the words on the first row are being stretched to fill the whole width. You can see all the possible styles and text decorations applied. Also note the typographic substitutions (the quotes) and ligature support (fi, ffi, ffl, ...). Below the text box, the word "stroke" is being stroked and drawn as a path. Top-right we see a LaTeX formula that has been converted to a path. Left of that we see ellipse support showcasing precise dashing, notably the length of e.g. the short dash is equal wherever it is (approximated through arc length parametrization) on the curve. It also shows support for alternating dash lengths, in this case (2.0, 4.0, 2.0) for dashes and for spaces. Note that the dashes themselves are elliptical arcs as well (thus exactly precise even if magnified greatly). In the bottom-right we see a closed polygon of four points being smoothed by cubic Béziers that are smooth along the whole path, and next to it on the left an open path. In the middle you can see a rasterized image painted.

<a href="https://raw.githubusercontent.com/tdewolff/canvas/master/examples/graph/out.png"><img src="https://raw.githubusercontent.com/tdewolff/canvas/master/examples/graph/out.png" height="250"></a>
<a href="https://raw.githubusercontent.com/tdewolff/canvas/master/examples/map/out.png"><img src="https://raw.githubusercontent.com/tdewolff/canvas/master/examples/map/out.png" height="250"></a>
<a href="https://raw.githubusercontent.com/tdewolff/canvas/master/examples/document/out.png"><img src="https://raw.githubusercontent.com/tdewolff/canvas/master/examples/document/out.png" height="250"></a>

**Figure 2abc**: Three examples of what is possible with this library, for example the plotting of graphs, maps and documents.

**Terminology**: a path is a sequence of drawing commands (MoveTo, LineTo, QuadTo, CubeTo, ArcTo, Close) that completely describe a path. QuadTo and CubeTo are quadratic and cubic Béziers respectively, ArcTo is an elliptical arc, and Close is a LineTo to the last MoveTo command and closes the path (sometimes this has a special meaning such as when stroking). A path can consist of several subpaths by having more than one MoveTo or Close command. A subpath consists of path segments which are defined by a command and some values or coordinates.

Flattening is the act of converting the QuadTo, CubeTo and ArcTo segments into LineTos so that all path segments are linear.

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
* [T.F. Hain, A.L. Ahmad, S.V.R. Racherla, D.D. Langan, Fast, precise flattening of cubic Bézier path and offset curves, Computers & Graphics 29 (2005). p. 656--666](https://www.sciencedirect.com/science/article/pii/S0097849305001287?via%3Dihub)
* [M. Goldapp, Approximation of circular arcs by cubic polynomials, Computer Aided Geometric Design 8 (1991), p. 227--238](https://www.sciencedirect.com/science/article/abs/pii/016783969190007X)
* [Drawing and elliptical arc using polylines, quadratic or cubic Bézier curves (2003), L. Maisonobe](https://spaceroots.org/documents/ellipse/elliptical-arc.pdf)

## Status
### Targets
| Feature | Image | SVG | PDF | EPS |
| ------- | ----- | --- | --- | --- |
| Draw path fill | yes | yes | yes | yes |
| Draw path stroke | yes | yes | yes | no |
| Draw path dash | yes | yes | yes | no |
| Embed fonts | | yes | yes (TTF & OTF) | no |
| Draw text | | yes | yes | as path |
| Draw image | yes | yes | yes | no |

* EPS does not support transparency
* PDF and EPS do not support line joins for last and first dash for dashed and closed path

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

* Fix slowness in the rasterizer (text_example.go is slow! use rasterized cache for each glyph/path)
* Add targets such as OpenGL and HTML5 Canvas (consider WASM output)
* Encoding forward and backward command in path so to easily reversely iterate the path

Fonts

* **Compressing fonts and embedding only used characters**
* **Use ligature tables**
* Font embedding for EPSs
* Support WOFF2 font format
* Support Type1 font format?
* Support font hinting (for the rasterizer)?
* Support LaTeX and other paths as text spans?

Paths

* **Intersection function between line, Bézier and ellipse and between themselves (for path merge, overlap/mask, clipping, etc.)**
* **Implement Bentley-Ottmann algorithm to find all line intersections (clipping)**
* Get position and derivative/normal at length L along the path
* Simplify polygons using the Ramer-Douglas-Peucker algorithm

Optimization

* **Avoid overlapping paths when offsetting in corners (needs path intersection code)**
* Approximate Béziers by elliptic arcs instead of lines when stroking, if number of path elements is reduced by more than 2 times (unsure if worth it)

Far future

* Support fill gradients and patterns (hard)
* Load in PDFs, SVGs and EPSs and turn to paths/texts
* Load in Markdown/HTML formatting and turn into texts


## Canvas
``` go
c := canvas.New(width, height float64)
c.PushState()          // save state set by function below on the stack
c.PopState()           // pop state from the stack
c.SetView(Matrix)      // set view transformation, all drawn elements are transformed by this matrix
c.ComposeView(Matrix)  // add transformation after the current view transformation
c.ResetView()          // use identity transformation matrix
c.SetFillColor(color.Color)
c.SetStrokeColor(color.Color)
c.SetStrokeCapper(Capper)
c.SetStrokeJoiner(Joiner)
c.SetStrokeWidth(width float64)
c.SetDashes(offset float64, lengths ...float64)

c.DrawPath(x, y float64, *Path)
c.DrawText(x, y float64, *Text)
c.DrawImage(x, y float64, image.Image, ImageEncoding, dpm float64)

c.Fit(margin float64)  // resize canvas to fit all elements with a given margin
c.WriteSVG(w io.Writer)
c.WriteEPS(w io.Writer)
c.WritePDF(w io.Writer)
c.WriteImage(dpm float64) *image.RGBA
```

Canvas allows to draw either paths, text or images. All positions and sizes are given in millimeters.

## Text
![Text Example](https://raw.githubusercontent.com/tdewolff/canvas/master/examples/text/out.png)

``` go
dejaVuSerif := NewFontFamily("dejavu-serif")
err := dejaVuSerif.LoadFontFile("DejaVuSerif.ttf", canvas.FontRegular)  // TTF, OTF or WOFF
ff := dejaVuSerif.Face(size float64, color.Color, FontStyle, FontVariant, ...FontDecorator)

text = NewTextLine(ff, "string\nsecond line", halign) // simple text line
text = NewTextBox(ff, "string", width, height, halign, valign, indent, lineStretch)  // split on word boundaries and specify text alignment

// rich text allowing different styles of text in one box
richText := NewRichText()  // allow different FontFaces in the same text block
richText.Add(ff, "string")
text = richText.ToText(width, height, halign, valign, indent, lineStretch)

c.DrawText(0.0, 0.0, text)
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

p = p.Optimize()  // optimize and shorten path
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


## Example
See https://github.com/tdewolff/canvas/tree/master/example for a working examples.

## License
Released under the [MIT license](LICENSE.md).
