![Canvas](https://raw.githubusercontent.com/tdewolff/canvas/master/resources/title/title.png)

[![API reference](https://img.shields.io/badge/godoc-reference-5272B4)](https://pkg.go.dev/github.com/tdewolff/canvas?tab=doc) [![User guide](https://img.shields.io/badge/user-guide-5272B4)](https://github.com/tdewolff/canvas/wiki) [![Go Report Card](https://goreportcard.com/badge/github.com/tdewolff/canvas)](https://goreportcard.com/report/github.com/tdewolff/canvas) [![Coverage Status](https://coveralls.io/repos/github/tdewolff/canvas/badge.svg?branch=master)](https://coveralls.io/github/tdewolff/canvas?branch=master) [![Donate](https://img.shields.io/badge/patreon-donate-DFB317)](https://www.patreon.com/tdewolff)

Canvas is a common vector drawing target that can output SVG, PDF, EPS, raster images (PNG, JPG, GIF, ...), HTML Canvas through WASM, OpenGL, and Gio. It has a wide range of path manipulation functionality such as flattening, stroking and dashing implemented. Additionally, it has a text formatter and embeds and subsets fonts (TTF, OTF, WOFF, WOFF2, or EOT) or converts them to outlines. It can be considered a Cairo or node-canvas alternative in Go. See the example below in Figure 1 for an overview of the functionality.

![Preview](https://raw.githubusercontent.com/tdewolff/canvas/master/resources/preview/preview.png)

**Figure 1**: top-left you can see text being fitted into a box, justified using Donald Knuth's linea breaking algorithm to stretch the spaces between words to fill the whole width. You can observe a variety of styles and text decorations applied, as well as support for LTR/RTL mixing and complex scripts. In the bottom-right the word "stroke" is being stroked and drawn as a path. Top-right we see a LaTeX formula that has been converted to a path. Left of that we see an ellipse showcasing precise dashing, notably the length of e.g. the short dash is equal wherever it is on the curve. Note that the dashes themselves are elliptical arcs as well (thus exactly precise even if magnified greatly). To the right we see a closed polygon of four points being smoothed by cubic Béziers that are smooth along the whole path, and the blue line on the left shows a smoothed open path. On the bottom you can see a rotated rasterized image. The result is equivalent for all renderers (PNG, PDF, SVG, etc.).

### Sponsors

Please see https://www.patreon.com/tdewolff for ways to contribute, otherwise please contact me directly!

## Recent changes
- Renderers have been moved from `github.com/tdewolff/canvas/.` to `github.com/tdewolff/canvas/renderers/.`
- `FontFamily.Use()` is deprecated, use `FontFamily.SetFeatures()` (not yet used)
- `DPMM` is now a function just like `DPI`: `rasterizer.PNGWriter(5.0 * canvas.DPMM)` => `rasterizer.PNGWriter(canvas.DPMM(5.0))`
- `FontFace` is now passed around as a pointer
- `NewRichText` now requires a default `*FontFace` to be passed
- Use the `latex` build tag to use the original LaTeX expression parser
- Renderer writers have been moved from `renderers/ABC/abc.Writer` to `renderers/ABC`
- `rasterizer.New` is renamed to `rasterizer.FromImage`

## Features
- Path segment types: MoveTo, LineTo, QuadTo, CubeTo, ArcTo, Close
- Precise path flattening, stroking, and dashing for all segment type uing papers (see below)
- Smooth spline generation through points for open and closed paths
- LaTeX to path conversion (native Go and CGO implementations available)
- Font formats support 
- - SFNT (such as TTF, OTF, WOFF, WOFF2, EOT) supporting TrueType, CFF, and CFF2 tables
- HarfBuzz for text shaping (native Go and CGO implementations available)
- FriBidi for text bidirectionality (native Go and CGO implementations available)
- Donald Knuth's line breaking algorithm for text layout
- sRGB compliance (use `SRGBColorSpace`, only available for rasterizer)
- Font rendering with gamma correction of 1.43 (WIP)
- Rendering targets
- - Raster images (PNG, GIF, JPEG, TIFF, BMP, WEBP)
- - PDF
- - SVG and SVGZ
- - PS and EPS
- - HTMLCanvas
- - OpenGL
- - [Gio](https://gioui.org/)
- - [Fyne](https://fyne.io/) (WIP)
- Rendering sources
- - Canvas itself
- - [go-chart](https://github.com/wcharczuk/go-chart)
- - [gonum/plot](https://github.com/gonum/plot)

## Documentation
**[API documentation](https://pkg.go.dev/github.com/tdewolff/canvas?tab=doc)**

**[User guide](https://github.com/tdewolff/canvas/wiki)**

### Examples
**[Street Map](https://github.com/tdewolff/canvas/tree/master/examples/map)**: the centre of Amsterdam is drawn from data loaded from the Open Street Map API.

**[Mauna-Loa CO2 conentration](https://github.com/tdewolff/canvas/tree/master/examples/graph)**: using data from the Mauna-Loa observatory, carbon dioxide concentrations over time are drawn

**[Document](https://github.com/tdewolff/canvas/tree/master/examples/document)**: an example of a text document.

**[OpenGL](https://github.com/tdewolff/canvas/tree/master/examples/opengl)**: an example using the OpenGL backend.

**[Gio](https://github.com/tdewolff/canvas/tree/master/examples/gio)**: an example using the Gio backend.

**[TeX/PGF](https://github.com/tdewolff/canvas/tree/master/examples/tex)**: an example showing the usage of the PGF (TikZ) LaTeX package as renderer in order to generated a PDF using LaTeX.

**[go-chart](https://github.com/tdewolff/canvas/tree/master/examples/go-chart)**: example usage of the [go-chart](https://github.com/wcharczuk/go-chart) library, plotting a financial graph.

**[gonum/plot](https://github.com/tdewolff/canvas/tree/master/examples/gonum-plot)**: example usage of the [gonum/plot](https://github.com/gonum/plot) library.

## Articles
* [Numerically stable quadratic formula](https://math.stackexchange.com/questions/866331/numerically-stable-algorithm-for-solving-the-quadratic-equation-when-a-is-very/2007723#2007723)
* [Quadratic Bézier length](https://malczak.linuxpl.com/blog/quadratic-bezier-curve-length/)
* [Bézier spline through open path](https://www.particleincell.com/2012/bezier-splines/)
* [Bézier spline through closed path](http://www.jacos.nl/jacos_html/spline/circular/index.html)
* [Point inclusion in polygon test](https://wrf.ecse.rpi.edu/Research/Short_Notes/pnpoly.html)

#### My own

* [Arc length parametrization](https://tacodewolff.nl/posts/20190525-arc-length/)

#### Papers

* [M. Walter, A. Fournier, Approximate Arc Length Parametrization, Anais do IX SIBGRAPHI (1996), p. 143--150](https://www.visgraf.impa.br/sibgrapi96/trabs/pdf/a14.pdf)
* [T.F. Hain, et al., Fast, precise flattening of cubic Bézier path and offset curves, Computers & Graphics 29 (2005). p. 656--666](https://doi.org/10.1016/j.cag.2005.08.002)
* [M. Goldapp, Approximation of circular arcs by cubic polynomials, Computer Aided Geometric Design 8 (1991), p. 227--238](https://doi.org/10.1016/0167-8396%2891%2990007-X)
* [L. Maisonobe, Drawing and elliptical arc using polylines, quadratic or cubic Bézier curves (2003)](https://spaceroots.org/documents/ellipse/elliptical-arc.pdf)
* [S.H. Kim and Y.J. Ahn, An approximation of circular arcs by quartic Bezier curves, Computer-Aided Design 39 (2007, p. 490--493)](https://doi.org/10.1016/j.cad.2007.01.004)
* [D.E. Knuth and M.F. Plass, Breaking Paragraphs into Lines, Software: Practive and Experience 11 (1981), p. 1119--1184]()

## License
Released under the [MIT license](LICENSE.md).
